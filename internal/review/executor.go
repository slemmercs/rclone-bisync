package review

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/models"
)

// ResolutionExecutor coordinates executing staged review decisions via reversible operations.
type ResolutionExecutor struct {
	LocalRoot  string
	RemoteName string
	BackupDir  string
	Client     RemoteClient
	WALPath    string
}

// NewResolutionExecutor creates a new configured executor instance.
func NewResolutionExecutor(localRoot, remoteName, backupDir, walPath string, client RemoteClient) *ResolutionExecutor {
	return &ResolutionExecutor{
		LocalRoot:  filepath.Clean(localRoot),
		RemoteName: strings.TrimSuffix(remoteName, ":"),
		BackupDir:  filepath.Clean(backupDir),
		Client:     client,
		WALPath:    walPath,
	}
}

// RemoteFs returns a properly formatted rclone fs string (supporting cloud remotes and local paths).
func (e *ResolutionExecutor) RemoteFs() string {
	return FormatRemoteFs(e.RemoteName)
}

// ExecuteSession executes all staged decisions for candidates in the session.
func (e *ResolutionExecutor) ExecuteSession(ctx context.Context, session *models.ReviewSession) error {
	if session == nil {
		return fmt.Errorf("session is nil")
	}

	if session.Status == models.ReviewStatusCreated {
		if err := Transition(session, models.ReviewStatusReviewing); err != nil {
			return err
		}
	}

	if err := Transition(session, models.ReviewStatusApplying); err != nil {
		return err
	}

	session.Actions = make([]models.ResolutionAction, 0)

	// Execute staged deletions
	for _, cand := range session.Deletions {
		if !cand.Resolved || cand.Decision == "" || cand.Decision == models.DecisionSkip {
			continue
		}
		if err := e.executeDeletion(ctx, session, cand); err != nil {
			_ = Transition(session, models.ReviewStatusFailed)
			return fmt.Errorf("failed executing deletion decision for %s: %w", cand.Path, err)
		}
	}

	// Execute staged conflicts
	for _, cand := range session.Conflicts {
		if !cand.Resolved || cand.Decision == "" || cand.Decision == models.DecisionSkip {
			continue
		}
		if err := e.executeConflict(ctx, session, cand); err != nil {
			_ = Transition(session, models.ReviewStatusFailed)
			return fmt.Errorf("failed executing conflict decision for %s: %w", cand.Path, err)
		}
	}

	return Transition(session, models.ReviewStatusReady)
}

func (e *ResolutionExecutor) logWAL(sessionID, actionID, step, itemPath string, decision models.ActionDecision, errStr string) {
	if e.WALPath == "" {
		return
	}
	_ = AppendWAL(e.WALPath, models.WALEntry{
		SessionID: sessionID,
		ActionID:  actionID,
		Step:      step,
		ItemPath:  itemPath,
		Type:      decision,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Error:     errStr,
	})
}

func (e *ResolutionExecutor) executeDeletion(ctx context.Context, session *models.ReviewSession, cand models.DeletionCandidate) error {
	actionID := fmt.Sprintf("act-del-%d", len(session.Actions)+1)
	action := models.ResolutionAction{
		ID:       actionID,
		Type:     cand.Decision,
		ItemPath: cand.Path,
		Status:   "pending",
	}

	fullLocal, err := ResolveSafePath(e.LocalRoot, cand.Path)
	if err != nil {
		action.Status = "failed"
		action.Error = err.Error()
		session.Actions = append(session.Actions, action)
		return err
	}

	switch cand.Decision {
	case models.DecisionAcceptDelete:
		if cand.Source == models.SideRemote {
			if _, statErr := os.Stat(fullLocal); statErr == nil {
				backupPath, bErr := LocalBackup(fullLocal, e.BackupDir, session.ID, cand.Path)
				if bErr != nil {
					return bErr
				}
				action.BackupPath = backupPath
				e.logWAL(session.ID, actionID, "BACKUP_COMPLETED", cand.Path, cand.Decision, "")

				e.logWAL(session.ID, actionID, "MUTATION_STARTED", cand.Path, cand.Decision, "")
				if err := os.Remove(fullLocal); err != nil {
					e.logWAL(session.ID, actionID, "MUTATION_FAILED", cand.Path, cand.Decision, err.Error())
					return err
				}
				e.logWAL(session.ID, actionID, "MUTATION_COMPLETED", cand.Path, cand.Decision, "")
			}
		} else {
			if e.Client != nil {
				backupPath, bErr := RemoteBackup(ctx, e.Client, e.RemoteName, cand.Path, session.ID)
				if bErr == nil {
					action.BackupPath = backupPath
					e.logWAL(session.ID, actionID, "BACKUP_COMPLETED", cand.Path, cand.Decision, "")
				}
				e.logWAL(session.ID, actionID, "MUTATION_STARTED", cand.Path, cand.Decision, "")
				if ok, err := e.Client.DeleteFile(ctx, e.RemoteFs(), cand.Path); err != nil || !ok {
					errMsg := fmt.Sprintf("remote delete failed: %v", err)
					e.logWAL(session.ID, actionID, "MUTATION_FAILED", cand.Path, cand.Decision, errMsg)
					return errors.New(errMsg)
				}
				e.logWAL(session.ID, actionID, "MUTATION_COMPLETED", cand.Path, cand.Decision, "")
			}
		}

	case models.DecisionRestoreItem:
		if cand.Source == models.SideRemote {
			if e.Client != nil {
				e.logWAL(session.ID, actionID, "MUTATION_STARTED", cand.Path, cand.Decision, "")
				localDir := filepath.Dir(fullLocal)
				baseName := filepath.Base(fullLocal)
				if ok, err := e.Client.CopyFile(ctx, localDir, baseName, e.RemoteFs(), cand.Path); err != nil || !ok {
					errMsg := fmt.Sprintf("restore copy to remote failed: %v", err)
					e.logWAL(session.ID, actionID, "MUTATION_FAILED", cand.Path, cand.Decision, errMsg)
					return errors.New(errMsg)
				}
				e.logWAL(session.ID, actionID, "MUTATION_COMPLETED", cand.Path, cand.Decision, "")
			}
		} else {
			if e.Client != nil {
				e.logWAL(session.ID, actionID, "MUTATION_STARTED", cand.Path, cand.Decision, "")
				localDir := filepath.Dir(fullLocal)
				baseName := filepath.Base(fullLocal)
				if ok, err := e.Client.CopyFile(ctx, e.RemoteFs(), cand.Path, localDir, baseName); err != nil || !ok {
					errMsg := fmt.Sprintf("restore copy from remote failed: %v", err)
					e.logWAL(session.ID, actionID, "MUTATION_FAILED", cand.Path, cand.Decision, errMsg)
					return errors.New(errMsg)
				}
				e.logWAL(session.ID, actionID, "MUTATION_COMPLETED", cand.Path, cand.Decision, "")
			}
		}
	}

	action.Status = "completed"
	session.Actions = append(session.Actions, action)
	return nil
}

func (e *ResolutionExecutor) executeConflict(ctx context.Context, session *models.ReviewSession, cand models.ConflictCandidate) error {
	actionID := fmt.Sprintf("act-conf-%d", len(session.Actions)+1)
	action := models.ResolutionAction{
		ID:       actionID,
		Type:     cand.Decision,
		ItemPath: cand.Path,
		Status:   "pending",
	}

	fullLocal, err := ResolveSafePath(e.LocalRoot, cand.Path)
	if err != nil {
		action.Status = "failed"
		action.Error = err.Error()
		session.Actions = append(session.Actions, action)
		return err
	}

	localDir := filepath.Dir(fullLocal)
	baseName := filepath.Base(fullLocal)

	switch cand.Decision {
	case models.DecisionKeepLocal:
		if e.Client != nil {
			backupPath, bErr := RemoteBackup(ctx, e.Client, e.RemoteName, cand.Path, session.ID)
			if bErr == nil {
				action.BackupPath = backupPath
				e.logWAL(session.ID, actionID, "BACKUP_COMPLETED", cand.Path, cand.Decision, "")
			}
			e.logWAL(session.ID, actionID, "MUTATION_STARTED", cand.Path, cand.Decision, "")
			if ok, err := e.Client.CopyFile(ctx, localDir, baseName, e.RemoteFs(), cand.Path); err != nil || !ok {
				errMsg := fmt.Sprintf("copy local to remote failed: %v", err)
				e.logWAL(session.ID, actionID, "MUTATION_FAILED", cand.Path, cand.Decision, errMsg)
				return errors.New(errMsg)
			}
			e.logWAL(session.ID, actionID, "MUTATION_COMPLETED", cand.Path, cand.Decision, "")
		}

	case models.DecisionKeepRemote:
		if _, statErr := os.Stat(fullLocal); statErr == nil {
			backupPath, bErr := LocalBackup(fullLocal, e.BackupDir, session.ID, cand.Path)
			if bErr != nil {
				return bErr
			}
			action.BackupPath = backupPath
			e.logWAL(session.ID, actionID, "BACKUP_COMPLETED", cand.Path, cand.Decision, "")
		}
		if e.Client != nil {
			e.logWAL(session.ID, actionID, "MUTATION_STARTED", cand.Path, cand.Decision, "")
			if ok, err := e.Client.CopyFile(ctx, e.RemoteFs(), cand.Path, localDir, baseName); err != nil || !ok {
				errMsg := fmt.Sprintf("copy remote to local failed: %v", err)
				e.logWAL(session.ID, actionID, "MUTATION_FAILED", cand.Path, cand.Decision, errMsg)
				return errors.New(errMsg)
			}
			e.logWAL(session.ID, actionID, "MUTATION_COMPLETED", cand.Path, cand.Decision, "")
		}

	case models.DecisionKeepBoth:
		ext := filepath.Ext(cand.Path)
		baseWithoutExt := strings.TrimSuffix(baseName, ext)
		altName := fmt.Sprintf("%s.remote-conflict%s", baseWithoutExt, ext)
		altRemotePath := filepath.Join(filepath.Dir(cand.Path), altName)

		if e.Client != nil {
			e.logWAL(session.ID, actionID, "MUTATION_STARTED", cand.Path, cand.Decision, "")
			if ok, err := e.Client.CopyFile(ctx, e.RemoteFs(), cand.Path, e.RemoteFs(), altRemotePath); err != nil || !ok {
				errMsg := fmt.Sprintf("preserve both on remote failed: %v", err)
				e.logWAL(session.ID, actionID, "MUTATION_FAILED", cand.Path, cand.Decision, errMsg)
				return errors.New(errMsg)
			}
			if ok, err := e.Client.CopyFile(ctx, localDir, baseName, e.RemoteFs(), cand.Path); err != nil || !ok {
				errMsg := fmt.Sprintf("overwrite primary remote with local failed: %v", err)
				e.logWAL(session.ID, actionID, "MUTATION_FAILED", cand.Path, cand.Decision, errMsg)
				return errors.New(errMsg)
			}
			e.logWAL(session.ID, actionID, "MUTATION_COMPLETED", cand.Path, cand.Decision, "")
		}
	}

	action.Status = "completed"
	session.Actions = append(session.Actions, action)
	return nil
}
