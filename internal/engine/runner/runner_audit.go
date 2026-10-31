package runner

import (
	"github.com/slemmercs/rclone-bisync/internal/audit"
)

func (r *SyncRunner) recordAuditExecuted(runID string, jobID int, origin string, durationSec float64, filesSynced int, resync, force, dryRun bool) {
	if r.Audit == nil {
		return
	}
	profileID := "default"
	if r.Config != nil && r.Config.ProfileID != "" {
		profileID = r.Config.ProfileID
	} else if r.Config != nil && r.Config.RemotePath != "" {
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

