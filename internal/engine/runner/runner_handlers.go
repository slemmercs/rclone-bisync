package runner

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/core/classifier"
	"github.com/slemmercs/rclone-bisync/internal/core/client"
	"github.com/slemmercs/rclone-bisync/internal/core/state"
	"github.com/slemmercs/rclone-bisync/internal/engine/review"
	"github.com/slemmercs/rclone-bisync/internal/models"
)

func (r *SyncRunner) handleSyncSuccess(
	runID string,
	jobID int,
	origin string,
	startTime time.Time,
	duration float64,
	monitorRes client.MonitorJobResult,
	resync, force, dryRun, noNotify bool,
	notificationState state.NotificationState,
) bool {
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

func (r *SyncRunner) handleSyncFailure(
	runID string,
	jobID int,
	origin string,
	startTime time.Time,
	duration float64,
	monitorRes client.MonitorJobResult,
	resync, force, dryRun, noNotify bool,
	notificationState state.NotificationState,
) bool {
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
			// #nosec G304 - reading configured daemon log file
			if data, err := os.ReadFile(filepath.Clean(daemonLog)); err == nil {
				logLines = append(logLines, strings.Split(string(data), "\n")...)
			}
		}
		if r.Config.LogFile != "" {
			// #nosec G304 - reading configured log file
			if data, err := os.ReadFile(filepath.Clean(r.Config.LogFile)); err == nil {
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
			"has_deletions": errType == "TOO_MANY_DELETES" && len(fileList) > 0,
			"error":         monitorRes.ErrorMessage,
			"message":       monitorRes.ErrorMessage,
			"error_type":    errType,
			"files":         fileList,
			"total_count":   len(fileList),
		})

		if r.Config.ActiveReviewFile != "" && (len(deletions) > 0 || len(conflicts) > 0) {
			sess := review.NewSession(errType, monitorRes.ErrorMessage, deletions, conflicts)
			_ = review.Transition(sess, models.ReviewStatusReviewing)
			if existing, err := review.LoadActiveSession(r.Config.ActiveReviewFile); err == nil && existing != nil &&
				(existing.Status == models.ReviewStatusReviewing || existing.Status == models.ReviewStatusFailed || existing.Status == models.ReviewStatusApplying) {
				sess.ID = existing.ID
				sess.CreatedAt = existing.CreatedAt
				sess.Actions = existing.Actions
				existingDel := make(map[string]models.DeletionCandidate, len(existing.Deletions))
				for _, d := range existing.Deletions {
					existingDel[d.Path] = d
				}
				for i := range sess.Deletions {
					if prev, ok := existingDel[sess.Deletions[i].Path]; ok && prev.Resolved {
						sess.Deletions[i].Decision = prev.Decision
						sess.Deletions[i].Resolved = prev.Resolved
					}
				}
				existingConf := make(map[string]models.ConflictCandidate, len(existing.Conflicts))
				for _, c := range existing.Conflicts {
					existingConf[c.Path] = c
				}
				for i := range sess.Conflicts {
					if prev, ok := existingConf[sess.Conflicts[i].Path]; ok && prev.Resolved {
						sess.Conflicts[i].Decision = prev.Decision
						sess.Conflicts[i].Resolved = prev.Resolved
					}
				}
			}
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
