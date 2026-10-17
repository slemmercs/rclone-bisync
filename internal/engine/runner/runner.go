package runner

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/core/classifier"
	"github.com/slemmercs/rclone-bisync/internal/core/client"
	"github.com/slemmercs/rclone-bisync/internal/core/compiler"
	"github.com/slemmercs/rclone-bisync/internal/core/lock"
	"github.com/slemmercs/rclone-bisync/internal/core/notifier"
	"github.com/slemmercs/rclone-bisync/internal/core/state"
)

// SyncRunner coordinates bidirectional synchronization runs.
type SyncRunner struct {
	Config   *config.Config
	LockMgr  *lock.LockManager
	Client   *client.RcloneClient
	Notifier *notifier.Dispatcher
}

// NewSyncRunner creates a configured SyncRunner instance.
func NewSyncRunner(cfg *config.Config) *SyncRunner {
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

	return &SyncRunner{
		Config:   cfg,
		LockMgr:  lockMgr,
		Client:   rcClient,
		Notifier: dispatcher,
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

	// Acquire lock
	locked, err := r.LockMgr.Acquire()
	if err != nil || !locked {
		log.Printf("Another sync process is already running or lock acquisition failed: %v", err)
		return false
	}
	defer r.LockMgr.Release()

	// Rate limit backoff check
	if !force && state.IsRateLimitActive(r.Config.RateLimitBackoffFile) {
		log.Printf("Sync skipped: Rate limit backoff is active. Use --force to override.")
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
		if !noNotify {
			_ = r.Notifier.SendDesktop("Sync Aborted - Canary Safety Guard", err.Error(), "critical")
		}
		return false
	}

	startTime := time.Now()
	jobID, err := r.Client.TriggerBisync(ctx, params, fmt.Sprintf("bisync-%s", runID))
	if err != nil {
		errStr := err.Error()
		errType := classifier.ClassifyError(errStr)
		log.Printf("Failed to dispatch bisync to daemon: %v", err)
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
		return true
	}

	errType := monitorRes.ErrorType
	if errType == "" && monitorRes.ErrorMessage != "" {
		errType = classifier.ClassifyError(monitorRes.ErrorMessage)
	}

	log.Printf("Sync failed [%s]: %s", errType, monitorRes.ErrorMessage)

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

	if errType == "TOO_MANY_DELETES" {
		_ = state.WriteJSONAtomic(r.Config.PendingDeletionsFile, map[string]interface{}{
			"aborted_at":     float64(time.Now().Unix()),
			"has_deletions":  true,
			"error":          monitorRes.ErrorMessage,
			"message":        monitorRes.ErrorMessage,
		})
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
