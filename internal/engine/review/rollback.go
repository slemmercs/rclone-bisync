package review

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/models"
)

// RollbackSession performs Saga compensation in reverse order for all actions executed in the session.
func RollbackSession(ctx context.Context, session *models.ReviewSession, executor *ResolutionExecutor) error {
	if session == nil {
		return fmt.Errorf("session is nil")
	}
	if executor == nil {
		return fmt.Errorf("executor is nil")
	}

	if err := Transition(session, models.ReviewStatusRollback); err != nil {
		return err
	}

	var rollbackErr error

	// Revert actions in reverse chronological order
	for i := len(session.Actions) - 1; i >= 0; i-- {
		action := &session.Actions[i]
		if action.Status != "completed" && action.Status != "interrupted" {
			continue
		}

		fullLocal, _ := ResolveSafePath(executor.LocalRoot, action.ItemPath)

		// 1. Restore from backup if backup was captured
		if action.BackupPath != "" {
			if strings.HasPrefix(action.BackupPath, ".review_backups") && executor.Client != nil {
				// Remote server-side backup
				if ok, err := executor.Client.CopyFile(ctx, executor.RemoteFs(), action.BackupPath, executor.RemoteFs(), action.ItemPath); err != nil || !ok {
					rollbackErr = fmt.Errorf("failed restoring remote backup %s: %w", action.BackupPath, err)
				}
			} else {
				// Local file backup
				if _, err := os.Stat(action.BackupPath); err == nil && fullLocal != "" {
					if err := RestoreLocalBackup(action.BackupPath, fullLocal); err != nil {
						rollbackErr = fmt.Errorf("failed restoring local backup %s: %w", action.BackupPath, err)
					}
				}
			}
		} else if action.Type == models.DecisionRestoreItem {
			// Restore created a new file where none existed; remove the newly created copy
			if fullLocal != "" {
				_ = os.Remove(fullLocal)
			}
			if executor.Client != nil {
				_, _ = executor.Client.DeleteFile(ctx, executor.RemoteFs(), action.ItemPath)
			}
		}

		if executor.WALPath != "" {
			_ = AppendWAL(executor.WALPath, models.WALEntry{
				SessionID: session.ID,
				ActionID:  action.ID,
				Step:      "ROLLBACK_COMPLETED",
				ItemPath:  action.ItemPath,
				Type:      action.Type,
				Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
			})
		}
		action.Status = "rolled_back"
	}

	// Session transitions back to REVIEWING for user inspection and adjustments
	_ = Transition(session, models.ReviewStatusReviewing)
	return rollbackErr
}
