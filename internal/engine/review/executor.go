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
	LocalRoot   string
	RemoteName  string
	BackupDir   string
	Client      RemoteClient
	WALPath     string
	QueueConfig *QueueConfig
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

func recordAction(session *models.ReviewSession, action models.ResolutionAction) {
	for i := range session.Actions {
		if session.Actions[i].ID == action.ID || session.Actions[i].ItemPath == action.ItemPath {
			session.Actions[i] = action
			return
		}
	}
	session.Actions = append(session.Actions, action)
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

	completedActions := make(map[string]models.ResolutionAction)
	for _, a := range session.Actions {
		if a.Status == "completed" {
			completedActions[a.ItemPath] = a
		}
	}
	session.Actions = make([]models.ResolutionAction, 0)

	allWorkItems := BuildWorkItems(session.Deletions, session.Conflicts)
	pendingItems := make([]WorkItem, 0, len(allWorkItems))

	for _, item := range allWorkItems {
		if prev, ok := completedActions[item.Path]; ok && prev.Type == item.Decision {
			recordAction(session, prev)
			continue
		}
		pendingItems = append(pendingItems, item)
	}

	queue := NewReviewQueue(e.QueueConfig)
	summary := queue.Process(ctx, pendingItems, func(opCtx context.Context, item WorkItem) error {
		if item.Kind == WorkItemKindDeletion && item.Deletion != nil {
			return e.executeDeletion(opCtx, session, *item.Deletion)
		}
		if item.Kind == WorkItemKindConflict && item.Conflict != nil {
			return e.executeConflict(opCtx, session, *item.Conflict)
		}
		return nil
	})

	if summary.HasErrors() {
		_ = Transition(session, models.ReviewStatusFailed)
		return fmt.Errorf("session execution finished with %d error(s): %s", summary.Failed, summary.FormatSummary())
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

func (e *ResolutionExecutor) executeDeletion(ctx context.Context, session *models.ReviewSession, cand models.DeletionCandidate) (err error) {
	actionID := fmt.Sprintf("act-del-%d", len(session.Actions)+1)
	action := models.ResolutionAction{
		ID:       actionID,
		Type:     cand.Decision,
		ItemPath: cand.Path,
		IsDir:    cand.IsDir,
		Status:   "pending",
	}

	defer func() {
		if err != nil && ClassifyError(err) != ErrorClassIdempotent {
			action.Status = "failed"
			action.Error = err.Error()
			recordAction(session, action)
		} else {
			action.Status = "completed"
			action.Error = ""
			recordAction(session, action)
		}
	}()

	actualPath := cand.Path
	fullLocal, rErr := ResolveSafePath(e.LocalRoot, cand.Path)
	if rErr != nil {
		return rErr
	}

	// Reconcile trailing-space directory/file if trimmed during discovery
	if _, statErr := os.Stat(fullLocal); statErr != nil && os.IsNotExist(statErr) {
		if _, trailErr := os.Stat(fullLocal + " "); trailErr == nil {
			if safeTrail, safeErr := ResolveSafePath(e.LocalRoot, cand.Path+" "); safeErr == nil {
				actualPath = cand.Path + " "
				fullLocal = safeTrail
			}
		}
	}

	isDir := cand.IsDir
	if fi, statErr := os.Stat(fullLocal); statErr == nil && fi.IsDir() {
		isDir = true
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
				var rmErr error
				if isDir {
					rmErr = os.RemoveAll(fullLocal)
				} else {
					rmErr = os.Remove(fullLocal)
				}
				if rmErr != nil {
					e.logWAL(session.ID, actionID, "MUTATION_FAILED", cand.Path, cand.Decision, rmErr.Error())
					return rmErr
				}
				e.logWAL(session.ID, actionID, "MUTATION_COMPLETED", cand.Path, cand.Decision, "")
			}
		} else {
			if e.Client != nil {
				if !isDir {
					backupPath, bErr := RemoteBackup(ctx, e.Client, e.RemoteName, cand.Path, session.ID)
					if bErr == nil {
						action.BackupPath = backupPath
						e.logWAL(session.ID, actionID, "BACKUP_COMPLETED", cand.Path, cand.Decision, "")
					}
				}
				e.logWAL(session.ID, actionID, "MUTATION_STARTED", cand.Path, cand.Decision, "")
				if isDir {
					if ok, err := e.Client.PurgeDir(ctx, e.RemoteFs(), cand.Path); err != nil || !ok {
						if err != nil && (strings.Contains(err.Error(), "404") || strings.Contains(strings.ToLower(err.Error()), "not found")) {
							e.logWAL(session.ID, actionID, "MUTATION_COMPLETED", cand.Path, cand.Decision, "already absent on remote")
						} else {
							errMsg := fmt.Sprintf("remote purge failed: %v", err)
							e.logWAL(session.ID, actionID, "MUTATION_FAILED", cand.Path, cand.Decision, errMsg)
							return errors.New(errMsg)
						}
					}
				} else {
					if ok, err := e.Client.DeleteFile(ctx, e.RemoteFs(), cand.Path); err != nil || !ok {
						if err != nil && (strings.Contains(err.Error(), "404") || strings.Contains(strings.ToLower(err.Error()), "not found")) {
							e.logWAL(session.ID, actionID, "MUTATION_COMPLETED", cand.Path, cand.Decision, "already absent on remote")
						} else {
							errMsg := fmt.Sprintf("remote delete failed: %v", err)
							e.logWAL(session.ID, actionID, "MUTATION_FAILED", cand.Path, cand.Decision, errMsg)
							return errors.New(errMsg)
						}
					}
				}
				e.logWAL(session.ID, actionID, "MUTATION_COMPLETED", cand.Path, cand.Decision, "")
			}
		}

	case models.DecisionRestoreItem:
		if cand.Source == models.SideRemote {
			if isDir {
				if e.Client != nil {
					e.logWAL(session.ID, actionID, "MUTATION_STARTED", actualPath, cand.Decision, "")
					if ok, err := e.Client.Mkdir(ctx, e.RemoteFs(), actualPath); err != nil || !ok {
						errMsg := fmt.Sprintf("restore mkdir on remote failed: %v", err)
						e.logWAL(session.ID, actionID, "MUTATION_FAILED", actualPath, cand.Decision, errMsg)
						return errors.New(errMsg)
					}
					e.logWAL(session.ID, actionID, "MUTATION_COMPLETED", actualPath, cand.Decision, "")
				}
			} else {
				if e.Client != nil {
					e.logWAL(session.ID, actionID, "MUTATION_STARTED", actualPath, cand.Decision, "")
					if ok, err := e.Client.CopyFile(ctx, e.LocalRoot, actualPath, e.RemoteFs(), actualPath); err != nil || !ok {
						if err != nil && (strings.Contains(err.Error(), "404") || strings.Contains(strings.ToLower(err.Error()), "not found")) {
							e.logWAL(session.ID, actionID, "MUTATION_COMPLETED", actualPath, cand.Decision, "already absent on local disk")
						} else {
							errMsg := fmt.Sprintf("restore copy to remote failed: %v", err)
							e.logWAL(session.ID, actionID, "MUTATION_FAILED", actualPath, cand.Decision, errMsg)
							return errors.New(errMsg)
						}
					} else {
						e.logWAL(session.ID, actionID, "MUTATION_COMPLETED", actualPath, cand.Decision, "")
					}
				}
			}
		} else {
			if isDir {
				e.logWAL(session.ID, actionID, "MUTATION_STARTED", actualPath, cand.Decision, "")
				// #nosec G301 - restored local directory permissions
				if err := os.MkdirAll(fullLocal, 0750); err != nil {
					errMsg := fmt.Sprintf("restore mkdir on local failed: %v", err)
					e.logWAL(session.ID, actionID, "MUTATION_FAILED", actualPath, cand.Decision, errMsg)
					return errors.New(errMsg)
				}
				e.logWAL(session.ID, actionID, "MUTATION_COMPLETED", actualPath, cand.Decision, "")
			} else {
				if e.Client != nil {
					e.logWAL(session.ID, actionID, "MUTATION_STARTED", actualPath, cand.Decision, "")
					if ok, err := e.Client.CopyFile(ctx, e.RemoteFs(), actualPath, e.LocalRoot, actualPath); err != nil || !ok {
						if err != nil && (strings.Contains(err.Error(), "404") || strings.Contains(strings.ToLower(err.Error()), "not found")) {
							e.logWAL(session.ID, actionID, "MUTATION_COMPLETED", actualPath, cand.Decision, "already absent on remote")
						} else {
							errMsg := fmt.Sprintf("restore copy from remote failed: %v", err)
							e.logWAL(session.ID, actionID, "MUTATION_FAILED", actualPath, cand.Decision, errMsg)
							return errors.New(errMsg)
						}
					} else {
						e.logWAL(session.ID, actionID, "MUTATION_COMPLETED", actualPath, cand.Decision, "")
					}
				}
			}
		}
	}

	return nil
}

func (e *ResolutionExecutor) executeConflict(ctx context.Context, session *models.ReviewSession, cand models.ConflictCandidate) (err error) {
	actionID := fmt.Sprintf("act-conf-%d", len(session.Actions)+1)
	action := models.ResolutionAction{
		ID:       actionID,
		Type:     cand.Decision,
		ItemPath: cand.Path,
		Status:   "pending",
	}

	defer func() {
		if err != nil && ClassifyError(err) != ErrorClassIdempotent {
			action.Status = "failed"
			action.Error = err.Error()
			recordAction(session, action)
		} else {
			action.Status = "completed"
			action.Error = ""
			recordAction(session, action)
		}
	}()

	fullLocal, rErr := ResolveSafePath(e.LocalRoot, cand.Path)
	if rErr != nil {
		return rErr
	}

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
			if ok, err := e.Client.CopyFile(ctx, e.LocalRoot, cand.Path, e.RemoteFs(), cand.Path); err != nil || !ok {
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
			if ok, err := e.Client.CopyFile(ctx, e.RemoteFs(), cand.Path, e.LocalRoot, cand.Path); err != nil || !ok {
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
			if ok, err := e.Client.CopyFile(ctx, e.LocalRoot, cand.Path, e.RemoteFs(), cand.Path); err != nil || !ok {
				errMsg := fmt.Sprintf("overwrite primary remote with local failed: %v", err)
				e.logWAL(session.ID, actionID, "MUTATION_FAILED", cand.Path, cand.Decision, errMsg)
				return errors.New(errMsg)
			}
			e.logWAL(session.ID, actionID, "MUTATION_COMPLETED", cand.Path, cand.Decision, "")
		}
	}

	return nil
}
