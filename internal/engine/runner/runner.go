package runner

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/audit"
	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/core/classifier"
	"github.com/slemmercs/rclone-bisync/internal/core/client"
	"github.com/slemmercs/rclone-bisync/internal/core/compiler"
	"github.com/slemmercs/rclone-bisync/internal/core/lock"
	"github.com/slemmercs/rclone-bisync/internal/core/logger"
	"github.com/slemmercs/rclone-bisync/internal/core/notifier"
	"github.com/slemmercs/rclone-bisync/internal/core/state"
	"github.com/slemmercs/rclone-bisync/internal/models"
	"github.com/slemmercs/rclone-bisync/internal/review"
	"github.com/slemmercs/rclone-bisync/internal/services"
)

// AuditRecorder provides audit event recording capability for sync runs.
type AuditRecorder interface {
	RecordEvent(profileID string, action audit.Action, actor string, details map[string]any) (*audit.AuditRecord, error)
}

// SyncRunner coordinates bidirectional synchronization runs.
type SyncRunner struct {
	Config   *config.Config
	LockMgr  *lock.LockManager
	Client   *client.RcloneClient
	Notifier *notifier.Dispatcher
	Logger   *logger.Logger
	Audit    AuditRecorder
}

// NewSyncRunner creates a configured SyncRunner instance.
func NewSyncRunner(cfg *config.Config) *SyncRunner {
	if cfg == nil {
		cfg = &config.Config{}
	}
	user := ""
	pass := ""
	if cfg.RCAuthEnabled {
		user = cfg.RCUser
		pass = cfg.RCPass
	}
	rcClient := client.NewRcloneClient(cfg.RCAPIURL, user, pass)
	lockMgr := lock.NewLockManager(cfg.LockFile, cfg.LockStaleTimeoutSeconds, cfg.CheckStaleLocks)
	dispatcher := notifier.NewDispatcher(
		cfg.NotificationAppName,
		cfg.WebhookURL,
		cfg.WebhookTimeoutSec,
		cfg.Features["smart_notifications"],
		cfg.ErrorNotificationRepeatHours,
		"",
	)
	runLogger, _ := logger.New(cfg.LogFile, true)

	var auditRec AuditRecorder
	if cfg.AuditDBPath != "" {
		if db, err := audit.OpenDB(cfg.AuditDBPath); err == nil && db != nil {
			auditRec = services.NewAuditService(db, runLogger)
		}
	}

	return &SyncRunner{
		Config:   cfg,
		LockMgr:  lockMgr,
		Client:   rcClient,
		Notifier: dispatcher,
		Logger:   runLogger,
		Audit:    auditRec,
	}
}

// Run executes the full synchronization cycle.
func (r *SyncRunner) Run(
	ctx context.Context,
	resync bool,
	force bool,
	dryRun bool,
	origin string,
	runID string,
	noNotify bool,
) bool {
	if runID == "" {
		runID = fmt.Sprintf("sync_%s_%x", time.Now().Format("060102_150405"), rand.Intn(0xffff))
	}

	startTime := time.Now()

	// Acquire lock
	locked, err := r.LockMgr.Acquire()
	if err != nil || !locked {
		log.Printf("Another sync process is already running or lock acquisition failed: %v", err)
		if r.Logger != nil {
			_ = r.Logger.RecordSyncRun(models.BisyncLogEntry{
				Time:        time.Now().Format(time.RFC3339),
				Level:       "warn",
				Msg:         "Sync skipped: another sync process is already running or lock acquisition failed",
				RunID:       runID,
				Origin:      origin,
				Status:      "ABORTED",
				ErrorType:   "LOCK_CONTENTION",
				DurationSec: 0,
				StartTime:   startTime.Format(time.RFC3339),
				EndTime:     time.Now().Format(time.RFC3339),
			})
		}
		r.recordAuditAborted(runID, origin, "LOCK_CONTENTION", "Another sync process is already running or lock acquisition failed", resync, force, dryRun)
		return false
	}
	defer r.LockMgr.Release()

	// Rate limit backoff check
	if !force && state.IsRateLimitActive(r.Config.RateLimitBackoffFile) {
		log.Printf("Sync skipped: Rate limit backoff is active. Use --force to override.")
		if r.Logger != nil {
			_ = r.Logger.RecordSyncRun(models.BisyncLogEntry{
				Time:        time.Now().Format(time.RFC3339),
				Level:       "warn",
				Msg:         "Sync skipped: rate limit backoff is active (use --force to override)",
				RunID:       runID,
				Origin:      origin,
				Status:      "ABORTED",
				ErrorType:   "RATE_LIMIT_BACKOFF",
				DurationSec: 0,
				StartTime:   startTime.Format(time.RFC3339),
				EndTime:     time.Now().Format(time.RFC3339),
			})
		}
		r.recordAuditAborted(runID, origin, "RATE_LIMIT_BACKOFF", "Rate limit backoff is active (use --force to override)", resync, force, dryRun)
		return false
	}

	notificationState := state.LoadNotificationState(r.Config.NotificationStateFile)

	// Compile bisync payload
	params, err := compiler.CompileBisyncPayload(
		r.Config.RcloneCommonArgs,
		r.Config.LocalPath,
		r.Config.RemotePath,
		resync,
		force,
		dryRun,
	)
	if err != nil {
		log.Printf("Payload compilation failed: %v", err)
		if r.Logger != nil {
			_ = r.Logger.RecordSyncRun(models.BisyncLogEntry{
				Time:        time.Now().Format(time.RFC3339),
				Level:       "error",
				Msg:         fmt.Sprintf("Sync aborted: %v", err),
				RunID:       runID,
				Origin:      origin,
				Status:      "ABORTED",
				ErrorType:   "CANARY_GUARD",
				DurationSec: 0,
				StartTime:   startTime.Format(time.RFC3339),
				EndTime:     time.Now().Format(time.RFC3339),
			})
		}
		r.recordAuditAborted(runID, origin, "CANARY_GUARD", err.Error(), resync, force, dryRun)
		if !noNotify {
			_ = r.Notifier.SendDesktop("Sync Aborted - Canary Safety Guard", err.Error(), "critical")
		}
		return false
	}

	jobID, err := r.Client.TriggerBisync(ctx, params, fmt.Sprintf("bisync-%s", runID))
	if err != nil {
		errStr := err.Error()
		errType := classifier.ClassifyError(errStr)
		log.Printf("Failed to dispatch bisync to daemon: %v", err)
		if r.Logger != nil {
			_ = r.Logger.RecordSyncRun(models.BisyncLogEntry{
				Time:        time.Now().Format(time.RFC3339),
				Level:       "error",
				Msg:         fmt.Sprintf("Sync dispatch failed [%s]: %s", errType, errStr),
				RunID:       runID,
				Origin:      origin,
				Status:      "FAULT",
				ErrorType:   errType,
				DurationSec: time.Since(startTime).Seconds(),
				StartTime:   startTime.Format(time.RFC3339),
				EndTime:     time.Now().Format(time.RFC3339),
			})
		}
		r.recordAuditAborted(runID, origin, errType, errStr, resync, force, dryRun)
		if !noNotify && r.Notifier.ShouldNotifyError(errType, notificationState.LastErrorType, notificationState.LastErrorTimestamp) {
			_ = r.Notifier.SendDesktop(fmt.Sprintf("Sync Dispatch Error - %s", errType), errStr, "critical")
		}
		return false
	}

	monitorRes := r.Client.MonitorJob(
		ctx,
		jobID,
		time.Duration(r.Config.JobPollInitialSleepSec)*time.Second,
		time.Duration(r.Config.JobPollMaxSleepSec)*time.Second,
		r.Config.JobPollBackoffMultiplier,
		3,
		time.Duration(r.Config.JobTimeoutSeconds)*time.Second,
	)

	duration := time.Since(startTime).Seconds()

	if monitorRes.Success {
		log.Printf("Sync succeeded in %.2fs (%d files synced)", duration, monitorRes.FilesSynced)
		if r.Logger != nil {
			_ = r.Logger.RecordSyncRun(models.BisyncLogEntry{
				Time:        time.Now().Format(time.RFC3339),
				Level:       "info",
				Msg:         fmt.Sprintf("Sync succeeded in %.1fs (%d file(s) transferred)", duration, monitorRes.FilesSynced),
				RunID:       runID,
				JobID:       jobID,
				Origin:      origin,
				Status:      "SUCCESS",
				DurationSec: duration,
				FilesSynced: monitorRes.FilesSynced,
				StartTime:   startTime.Format(time.RFC3339),
				EndTime:     time.Now().Format(time.RFC3339),
			})
		}
		state.ClearPendingDeletions(r.Config.PendingDeletionsFile)
		state.ClearRateLimitBackoff(r.Config.RateLimitBackoffFile)

		notificationState.LastSuccessTimestamp = float64(time.Now().Unix())
		notificationState.LastErrorType = ""
		notificationState.LastErrorTimestamp = 0
		_ = state.WriteNotificationState(r.Config.NotificationStateFile, notificationState)

		if r.Config.EnableMetrics {
			_ = state.AppendMetricsEntry(r.Config.MetricsFile, state.MetricsEntry{
				Timestamp:       float64(time.Now().Unix()),
				Datetime:        time.Now().Format(time.RFC3339),
				RunID:           runID,
				JobID:           jobID,
				Origin:          origin,
				Success:         true,
				DurationSeconds: duration,
				FilesSynced:     monitorRes.FilesSynced,
			})
		}

		if !noNotify {
			_ = r.Notifier.SendDesktop("Rclone Bisync Successful", fmt.Sprintf("Synced %d files in %.1fs", monitorRes.FilesSynced, duration), "normal")
		}

		// Option 2: Emit SYNC_EXECUTED only when actual state mutations occurred (FilesSynced > 0).
		// Zero-transfer runs remain purely in bisync.log telemetry.
		if monitorRes.FilesSynced > 0 {
			r.recordAuditExecuted(runID, jobID, origin, duration, monitorRes.FilesSynced, resync, force, dryRun)
		}

		return true
	}

	errType := monitorRes.ErrorType
	if errType == "" && monitorRes.ErrorMessage != "" {
		errType = classifier.ClassifyError(monitorRes.ErrorMessage)
	}

	r.recordAuditAborted(runID, origin, errType, monitorRes.ErrorMessage, resync, force, dryRun)

	log.Printf("Sync failed [%s]: %s", errType, monitorRes.ErrorMessage)
	if r.Logger != nil {
		_ = r.Logger.RecordSyncRun(models.BisyncLogEntry{
			Time:        time.Now().Format(time.RFC3339),
			Level:       "error",
			Msg:         fmt.Sprintf("Sync failed [%s]: %s", errType, monitorRes.ErrorMessage),
			RunID:       runID,
			JobID:       jobID,
			Origin:      origin,
			Status:      "FAULT",
			ErrorType:   errType,
			DurationSec: duration,
			FilesSynced: monitorRes.FilesSynced,
			StartTime:   startTime.Format(time.RFC3339),
			EndTime:     time.Now().Format(time.RFC3339),
		})
	}

	if r.Config.EnableMetrics {
		_ = state.AppendMetricsEntry(r.Config.MetricsFile, state.MetricsEntry{
			Timestamp:       float64(time.Now().Unix()),
			Datetime:        time.Now().Format(time.RFC3339),
			RunID:           runID,
			JobID:           jobID,
			Origin:          origin,
			Success:         false,
			DurationSeconds: duration,
			ErrorType:       errType,
			FilesSynced:     monitorRes.FilesSynced,
		})
	}

	if errType == "RATE_LIMIT" {
		_ = state.SetRateLimitBackoff(r.Config.RateLimitBackoffFile, 3600)
	}

	if errType == "TOO_MANY_DELETES" || errType == "CONFLICT" {
		var logLines []string
		if r.Config.BaseDir != "" {
			daemonLog := filepath.Join(r.Config.BaseDir, "rclone-daemon.log")
			if data, err := os.ReadFile(daemonLog); err == nil {
				logLines = append(logLines, strings.Split(string(data), "\n")...)
			}
		}
		if r.Config.LogFile != "" {
			if data, err := os.ReadFile(r.Config.LogFile); err == nil {
				logLines = append(logLines, strings.Split(string(data), "\n")...)
			}
		}
		if len(logLines) == 0 && monitorRes.ErrorMessage != "" {
			logLines = []string{monitorRes.ErrorMessage}
		}

		deletions, conflicts, _ := review.DiscoverCandidates(
			logLines,
			r.Config.PendingDeletionsFile,
			r.Config.LocalPath,
		)

		var fileList []string
		for _, d := range deletions {
			fileList = append(fileList, d.Path)
		}

		_ = state.WriteJSONAtomic(r.Config.PendingDeletionsFile, map[string]interface{}{
			"aborted_at":    float64(time.Now().Unix()),
			"has_deletions": errType == "TOO_MANY_DELETES",
			"error":         monitorRes.ErrorMessage,
			"message":       monitorRes.ErrorMessage,
			"error_type":    errType,
			"files":         fileList,
			"total_count":   len(fileList),
		})

		if r.Config.ActiveReviewFile != "" && (len(deletions) > 0 || len(conflicts) > 0) {
			sess := review.NewSession(errType, monitorRes.ErrorMessage, deletions, conflicts)
			_ = review.Transition(sess, models.ReviewStatusReviewing)
			_ = review.SaveActiveSession(r.Config.ActiveReviewFile, sess)
		}
	}

	if !noNotify && r.Notifier.ShouldNotifyError(errType, notificationState.LastErrorType, notificationState.LastErrorTimestamp) {
		urgency := "normal"
		if errType == "CHECK_ACCESS_FAILED" || errType == "TOO_MANY_DELETES" || errType == "PERMISSION" {
			urgency = "critical"
		}
		_ = r.Notifier.SendDesktop(fmt.Sprintf("Sync Failure - %s", errType), monitorRes.ErrorMessage, urgency)
		notificationState.LastErrorType = errType
		notificationState.LastErrorTimestamp = float64(time.Now().Unix())
		_ = state.WriteNotificationState(r.Config.NotificationStateFile, notificationState)
	}

	return false
}

func (r *SyncRunner) recordAuditExecuted(runID string, jobID int, origin string, durationSec float64, filesSynced int, resync, force, dryRun bool) {
	if r.Audit == nil {
		return
	}
	profileID := "default"
	if r.Config != nil && r.Config.RemotePath != "" {
		profileID = r.Config.RemotePath
	}
	actor := "SYSTEM"
	if origin == "dashboard" || origin == "cli" {
		actor = "USER"
	}
	_, _ = r.Audit.RecordEvent(profileID, audit.ActionSyncExecuted, actor, map[string]any{
		"run_id":       runID,
		"job_id":       jobID,
		"origin":       origin,
		"duration_sec": durationSec,
		"files_synced": filesSynced,
		"resync":       resync,
		"force":        force,
		"dry_run":      dryRun,
	})
}

func (r *SyncRunner) recordAuditAborted(runID string, origin string, errType string, errMsg string, resync, force, dryRun bool) {
	if r.Audit == nil {
		return
	}
	profileID := "default"
	if r.Config != nil && r.Config.RemotePath != "" {
		profileID = r.Config.RemotePath
	}
	actor := "SYSTEM"
	if origin == "dashboard" || origin == "cli" {
		actor = "USER"
	}
	_, _ = r.Audit.RecordEvent(profileID, audit.ActionSyncAborted, actor, map[string]any{
		"run_id":     runID,
		"origin":     origin,
		"error_type": errType,
		"error":      errMsg,
		"resync":     resync,
		"force":      force,
		"dry_run":    dryRun,
	})
}
