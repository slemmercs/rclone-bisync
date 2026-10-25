package runner

import (
	"context"
	"fmt"
	"log"
	"math/rand"
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
		return r.handleSyncSuccess(runID, jobID, origin, startTime, duration, monitorRes, resync, force, dryRun, noNotify, notificationState)
	}

	return r.handleSyncFailure(runID, jobID, origin, startTime, duration, monitorRes, resync, force, dryRun, noNotify, notificationState)
}
