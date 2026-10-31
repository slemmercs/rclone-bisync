package runner

import (
	"context"
	"crypto/rand"
	"fmt"
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
	if runLogger != nil && cfg.DebugMode {
		runLogger.SetDebug(true)
	}

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
		var randBytes [2]byte
		_, _ = rand.Read(randBytes[:])
		runID = fmt.Sprintf("sync_%s_%x", time.Now().Format("060102_150405"), randBytes)
	}

	startTime := time.Now()

	// Acquire lock
	locked, err := r.LockMgr.Acquire()
	if err != nil || !locked {
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
		// Non-mutating aborts are recorded in execution logs, skipping immutable audit ledger.
		return false
	}
	defer r.LockMgr.Release()

	// Rate limit backoff check
	if !force && state.IsRateLimitActive(r.Config.RateLimitBackoffFile) {
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
		// Non-mutating aborts are recorded in execution logs, skipping immutable audit ledger.
		return false
	}

	notificationState := state.LoadNotificationState(r.Config.NotificationStateFile)

	// Propagate debug verbose flag to rclone command options if DebugMode is active
	commonArgs := r.Config.RcloneCommonArgs
	if r.Config.DebugMode {
		if r.Logger != nil {
			r.Logger.SetDebug(true)
			r.Logger.Debug("Debug logging enabled for sync run", "run_id", runID, "origin", origin)
		}
		hasVerbose := false
		for _, a := range commonArgs {
			if a == "--verbose" || a == "-v" || a == "-vv" {
				hasVerbose = true
				break
			}
		}
		if !hasVerbose {
			commonArgs = append(append([]string{}, commonArgs...), "--verbose")
		}
	}

	// Compile bisync payload
	params, err := compiler.CompileBisyncPayload(
		commonArgs,
		r.Config.LocalPath,
		r.Config.RemotePath,
		resync,
		force,
		dryRun,
	)
	if err != nil {
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
		// Non-mutating aborts are recorded in execution logs, skipping immutable audit ledger.
		if !noNotify {
			_ = r.Notifier.SendDesktop(r.notificationTitle("Sync Aborted - Canary Safety Guard"), err.Error(), "critical")
		}
		return false
	}

	jobID, err := r.Client.TriggerBisync(ctx, params, fmt.Sprintf("bisync-%s", runID))
	if err != nil {
		errStr := err.Error()
		errType := classifier.ClassifyError(errStr)
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
		// Non-mutating aborts are recorded in execution logs, skipping immutable audit ledger.
		if !noNotify && r.Notifier.ShouldNotifyError(errType, notificationState.LastErrorType, notificationState.LastErrorTimestamp) {
			_ = r.Notifier.SendDesktop(r.notificationTitle(fmt.Sprintf("Sync Dispatch Error - %s", errType)), errStr, "critical")
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
		return r.handleSyncSuccess(runID, jobID, origin, startTime, duration, monitorRes, resync, force, dryRun, noNotify, notificationState)
	}

	return r.handleSyncFailure(runID, jobID, origin, startTime, duration, monitorRes, resync, force, dryRun, noNotify, notificationState)
}

// notificationTitle prefixes the notification title with [<ProfileName>] if available.
func (r *SyncRunner) notificationTitle(base string) string {
	name := strings.TrimSpace(r.Config.ProfileName)
	if name == "" && r.Config.ProfileID != "" && r.Config.ProfileID != "default" {
		name = r.Config.ProfileID
	}
	if name != "" && name != "default" {
		return fmt.Sprintf("[%s] %s", name, base)
	}
	return base
}
