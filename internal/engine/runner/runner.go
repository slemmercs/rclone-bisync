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
	"github.com/slemmercs/rclone-bisync/internal/engine"
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
	FSM      *engine.SyncFSM
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

	profID := "default"
	if cfg.ProfileID != "" {
		profID = cfg.ProfileID
	}
	fsm := engine.NewSyncFSM(profID, engine.StateIdle)

	return &SyncRunner{
		Config:   cfg,
		LockMgr:  lockMgr,
		Client:   rcClient,
		Notifier: dispatcher,
		Logger:   runLogger,
		Audit:    auditRec,
		FSM:      fsm,
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
	ok, _ := r.RunWithError(ctx, resync, force, dryRun, origin, runID, noNotify)
	return ok
}

// RunWithError executes the full synchronization cycle and returns detailed error diagnostics upon failure.
func (r *SyncRunner) RunWithError(
	ctx context.Context,
	resync bool,
	force bool,
	dryRun bool,
	origin string,
	runID string,
	noNotify bool,
) (bool, error) {
	if runID == "" {
		var randBytes [2]byte
		_, _ = rand.Read(randBytes[:])
		runID = fmt.Sprintf("sync_%s_%x", time.Now().Format("060102_150405"), randBytes)
	}

	startTime := time.Now()

	// Acquire lock
	profID := "default"
	profName := ""
	if r.Config != nil {
		if r.Config.ProfileID != "" {
			profID = r.Config.ProfileID
		}
		profName = r.Config.ProfileName
	}

	if r.FSM == nil {
		r.FSM = engine.NewSyncFSM(profID, engine.StateIdle)
	} else if r.FSM.Current() == engine.StateCompleted || r.FSM.Current() == engine.StateAborted {
		_ = r.FSM.Reset()
	}

	_, _ = r.FSM.Transition(engine.StateQueued, runID, "sync dispatched", "")

	locked, err := r.LockMgr.Acquire()
	if err != nil || !locked {
		if r.FSM != nil {
			_, _ = r.FSM.Transition(engine.StateAborted, runID, "lock acquisition failed: another sync process is already running", "LOCK_CONTENTION")
		}
		if r.Logger != nil {
			_ = r.Logger.RecordSyncRun(models.BisyncLogEntry{
				Time:        time.Now().Format(time.RFC3339),
				Level:       "warn",
				Type:        "sync",
				ProfileID:   profID,
				ProfileName: profName,
				Msg:         r.formatLogMsg("Sync skipped: another sync process is already running or lock acquisition failed"),
				RunID:       runID,
				Origin:      origin,
				Status:      "ABORTED",
				Action:      "SYNC_ABORTED",
				ErrorType:   "LOCK_CONTENTION",
				DurationSec: 0,
				StartTime:   startTime.Format(time.RFC3339),
				EndTime:     time.Now().Format(time.RFC3339),
			})
		}
		// Non-mutating aborts are recorded in execution logs, skipping immutable audit ledger.
		return false, fmt.Errorf("lock acquisition failed: another sync process is already running")
	}
	defer r.LockMgr.Release()

	// Rate limit backoff check
	if !force && state.IsRateLimitActive(r.Config.RateLimitBackoffFile) {
		if r.FSM != nil {
			_, _ = r.FSM.Transition(engine.StateAborted, runID, "rate limit backoff is active", "RATE_LIMIT_BACKOFF")
		}
		if r.Logger != nil {
			_ = r.Logger.RecordSyncRun(models.BisyncLogEntry{
				Time:        time.Now().Format(time.RFC3339),
				Level:       "warn",
				Type:        "sync",
				ProfileID:   profID,
				ProfileName: profName,
				Msg:         r.formatLogMsg("Sync skipped: rate limit backoff is active (use --force to override)"),
				RunID:       runID,
				Origin:      origin,
				Status:      "ABORTED",
				Action:      "SYNC_ABORTED",
				ErrorType:   "RATE_LIMIT_BACKOFF",
				DurationSec: 0,
				StartTime:   startTime.Format(time.RFC3339),
				EndTime:     time.Now().Format(time.RFC3339),
			})
		}
		// Non-mutating aborts are recorded in execution logs, skipping immutable audit ledger.
		return false, fmt.Errorf("rate limit backoff is active (use --force to override)")
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
		if r.FSM != nil {
			_, _ = r.FSM.Transition(engine.StateAborted, runID, fmt.Sprintf("configuration/canary check failed: %v", err), "CANARY_GUARD")
		}
		if r.Logger != nil {
			_ = r.Logger.RecordSyncRun(models.BisyncLogEntry{
				Time:        time.Now().Format(time.RFC3339),
				Level:       "error",
				Type:        "sync",
				ProfileID:   profID,
				ProfileName: profName,
				Msg:         r.formatLogMsg(fmt.Sprintf("Sync aborted: %v", err)),
				RunID:       runID,
				Origin:      origin,
				Status:      "ABORTED",
				Action:      "SYNC_ABORTED",
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
		return false, fmt.Errorf("configuration/canary check failed: %w", err)
	}

	jobID, err := r.Client.TriggerBisync(ctx, params, fmt.Sprintf("bisync-%s", runID))
	if err != nil {
		errStr := err.Error()
		errType := classifier.ClassifyError(errStr)
		if r.FSM != nil {
			_, _ = r.FSM.Transition(engine.StateAborted, runID, fmt.Sprintf("sync dispatch failed [%s]: %s", errType, errStr), errType)
		}
		if r.Logger != nil {
			_ = r.Logger.RecordSyncRun(models.BisyncLogEntry{
				Time:        time.Now().Format(time.RFC3339),
				Level:       "error",
				Type:        "sync",
				ProfileID:   profID,
				ProfileName: profName,
				Msg:         r.formatLogMsg(fmt.Sprintf("Sync dispatch failed [%s]: %s", errType, errStr)),
				RunID:       runID,
				Origin:      origin,
				Status:      "FAULT",
				Action:      "SYNC_FAILED",
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
		return false, fmt.Errorf("sync dispatch failed [%s]: %s", errType, errStr)
	}

	if r.FSM != nil {
		_, _ = r.FSM.Transition(engine.StateRunning, runID, fmt.Sprintf("sync started (job ID %d)", jobID), "")
	}

	if r.Logger != nil {
		action := "Sync"
		if dryRun {
			action = "Dry run"
		} else if resync {
			action = "Resync"
		}
		_ = r.Logger.RecordSyncRun(models.BisyncLogEntry{
			Time:        time.Now().Format(time.RFC3339),
			Level:       "info",
			Type:        "sync",
			ProfileID:   profID,
			ProfileName: profName,
			Msg:         r.formatLogMsg(fmt.Sprintf("%s started (job ID %d)", action, jobID)),
			RunID:       runID,
			JobID:       jobID,
			Origin:      origin,
			Status:      "RUNNING",
			Action:      "SYNC_STARTED",
			StartTime:   startTime.Format(time.RFC3339),
		})
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
		return r.handleSyncSuccess(runID, jobID, origin, startTime, duration, monitorRes, resync, force, dryRun, noNotify, notificationState), nil
	}

	r.handleSyncFailure(runID, jobID, origin, startTime, duration, monitorRes, resync, force, dryRun, noNotify, notificationState)
	errType := monitorRes.ErrorType
	if errType == "" && monitorRes.ErrorMessage != "" {
		errType = classifier.ClassifyError(monitorRes.ErrorMessage)
	}
	return false, fmt.Errorf("sync execution failed [%s]: %s", errType, monitorRes.ErrorMessage)
}

// profileLabel returns the friendly ProfileName if available, else ProfileID if not "default", else "".
func (r *SyncRunner) profileLabel() string {
	if r.Config == nil {
		return ""
	}
	name := strings.TrimSpace(r.Config.ProfileName)
	if name != "" && !strings.EqualFold(name, "default") {
		return name
	}
	id := strings.TrimSpace(r.Config.ProfileID)
	if id != "" && !strings.EqualFold(id, "default") {
		return id
	}
	return ""
}

// formatLogMsg prepends [<ProfileLabel>] to msg if a profile label is available.
func (r *SyncRunner) formatLogMsg(msg string) string {
	label := r.profileLabel()
	if label != "" {
		return fmt.Sprintf("[%s] %s", label, msg)
	}
	return msg
}

// notificationTitle prefixes the notification title with [<ProfileName>] if available.
func (r *SyncRunner) notificationTitle(base string) string {
	label := r.profileLabel()
	if label != "" {
		return fmt.Sprintf("[%s] %s", label, base)
	}
	return base
}
