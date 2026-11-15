package services

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/audit"
	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/models"
	"github.com/slemmercs/rclone-bisync/internal/profiles"
)

// DedupeService coordinates atomic batch deduplication actions (stage renames/deletions) and post-dedupe resync triggers.
type DedupeService struct {
	Config          *config.Config
	ExplorerService *ExplorerService
	AuditService    *AuditService
	ProfileService  *profiles.Service
}

// NewDedupeService creates a new DedupeService instance.
func NewDedupeService(cfg *config.Config, expSvc *ExplorerService, auditSvc *AuditService, profSvc *profiles.Service) *DedupeService {
	return &DedupeService{
		Config:          cfg,
		ExplorerService: expSvc,
		AuditService:    auditSvc,
		ProfileService:  profSvc,
	}
}

// ExecuteDedupeBatch executes staged duplicate resolution decisions (delete/rename/keep) with safety checks.
func (s *DedupeService) ExecuteDedupeBatch(ctx context.Context, req models.DedupeBatchApplyRequest, localBase string) (models.DedupeBatchApplyResponse, error) {
	remoteClean := strings.TrimSpace(req.Remote)
	if remoteClean == "" {
		return models.DedupeBatchApplyResponse{Status: "error", Error: "remote parameter required"}, nil
	}

	runID := fmt.Sprintf("dedupe-batch-%d", time.Now().Unix())
	resp := models.DedupeBatchApplyResponse{Status: "success", RunID: runID}

	if len(req.Actions) == 0 {
		return resp, nil
	}
	if s.ExplorerService == nil {
		return models.DedupeBatchApplyResponse{Status: "error", Error: "explorer service is unavailable"}, nil
	}

	reqPath := strings.Trim(strings.TrimSpace(req.Path), "/")
	for _, action := range req.Actions {
		cleanPath := strings.Trim(strings.TrimSpace(action.Path), "/")
		if cleanPath == "" {
			resp.FailureCount++
			resp.Errors = append(resp.Errors, "empty path in action")
			continue
		}
		if reqPath != "" && !strings.HasPrefix(cleanPath, reqPath+"/") && cleanPath != reqPath {
			cleanPath = fmt.Sprintf("%s/%s", reqPath, cleanPath)
		}
		if strings.Contains(cleanPath, "..") {
			resp.FailureCount++
			resp.Errors = append(resp.Errors, fmt.Sprintf("path traversal rejected: %s", cleanPath))
			continue
		}
		if filepath.Base(cleanPath) == CanaryFilename {
			resp.FailureCount++
			resp.Errors = append(resp.Errors, fmt.Sprintf("canary (%s) cannot be modified or deleted", CanaryFilename))
			continue
		}

		switch strings.ToLower(strings.TrimSpace(action.Action)) {
		case "keep":
			resp.SuccessCount++
		case "delete":
			delRes, err := s.ExplorerService.DeleteItem(ctx, remoteClean, cleanPath, false, localBase)
			if err != nil || delRes.Status == "error" {
				errMsg := delRes.Error
				if errMsg == "" && err != nil {
					errMsg = err.Error()
				}
				resp.FailureCount++
				resp.Errors = append(resp.Errors, fmt.Sprintf("delete %s: %s", cleanPath, errMsg))
			} else {
				resp.SuccessCount++
			}
		case "rename":
			newName := strings.TrimSpace(action.NewName)
			if newName == "" || strings.Contains(newName, "/") || strings.Contains(newName, "\\") || strings.Contains(newName, "..") {
				resp.FailureCount++
				resp.Errors = append(resp.Errors, fmt.Sprintf("invalid rename target %q for %s", newName, cleanPath))
				continue
			}
			if newName == CanaryFilename {
				resp.FailureCount++
				resp.Errors = append(resp.Errors, fmt.Sprintf("cannot rename %s to canary (%s)", cleanPath, CanaryFilename))
				continue
			}

			dstDir := filepath.Dir(cleanPath)
			dstPath := newName
			if dstDir != "." && dstDir != "" && dstDir != "/" {
				dstPath = fmt.Sprintf("%s/%s", strings.Trim(dstDir, "/"), newName)
			}

			moveRes, err := s.ExplorerService.MoveItem(ctx, models.MoveRequest{
				Remote:    remoteClean,
				SrcRemote: remoteClean,
				DstRemote: remoteClean,
				SrcPath:   cleanPath,
				DstPath:   dstPath,
				ProfileID: req.ProfileID,
			}, localBase)

			if err != nil || moveRes.Status == "error" {
				errMsg := moveRes.Error
				if errMsg == "" && err != nil {
					errMsg = err.Error()
				}
				resp.FailureCount++
				resp.Errors = append(resp.Errors, fmt.Sprintf("rename %s -> %s: %s", cleanPath, newName, errMsg))
			} else {
				resp.SuccessCount++
			}
		default:
			resp.FailureCount++
			resp.Errors = append(resp.Errors, fmt.Sprintf("unknown action %q for %s", action.Action, cleanPath))
		}
	}

	profileID := req.ProfileID
	if profileID == "" {
		profileID = remoteClean
	}

	if s.AuditService != nil {
		if s.AuditService.Logger != nil {
			var level, summary string
			if resp.SuccessCount > 0 {
				level = "notice"
				summary = fmt.Sprintf("[%s] Dedupe batch resolution applied: %d succeeded, %d failed.", profileID, resp.SuccessCount, resp.FailureCount)
			} else {
				level = "error"
				summary = fmt.Sprintf("[%s] Dedupe batch resolution failed: %d failed out of %d.", profileID, resp.FailureCount, len(req.Actions))
			}
			_ = s.AuditService.Logger.RecordSyncRun(models.BisyncLogEntry{
				Time:        time.Now().Format(time.RFC3339),
				Level:       level,
				Type:        "dedupe",
				Msg:         summary,
				RunID:       runID,
				Origin:      "dedupe_batch",
				ProfileID:   profileID,
				ProfileName: profileID,
				Action:      "DEDUPE_BATCH_APPLIED",
			})
		}
		if resp.SuccessCount > 0 {
			_, _ = s.AuditService.RecordEvent(profileID, audit.ActionDedupeExecuted, "system", map[string]any{
				"run_id":        runID,
				"remote":        remoteClean,
				"path":          req.Path,
				"success_count": resp.SuccessCount,
				"failure_count": resp.FailureCount,
				"total_actions": len(req.Actions),
			})
		}
	}

	if req.TriggerResync {
		if resp.SuccessCount == 0 {
			resp.ResyncQueued = false
			resp.ResyncMessage = "Resync skipped: no dedupe actions succeeded"
		} else if s.ProfileService != nil && req.ProfileID != "" {
			err := s.ProfileService.TriggerSyncWithOptions(ctx, req.ProfileID, profiles.SyncOptions{
				Resync: true,
				Origin: "dedupe_batch",
			})
			if err != nil {
				resp.ResyncQueued = false
				resp.ResyncMessage = fmt.Sprintf("Resync trigger failed: %v", err)
			} else {
				resp.ResyncQueued = true
				resp.ResyncMessage = "Post-dedupe resync successfully queued."
			}
		} else {
			resp.ResyncQueued = false
			resp.ResyncMessage = "Profile not specified or profile service unavailable for resync"
		}
	}

	if resp.FailureCount > 0 && resp.SuccessCount == 0 {
		resp.Status = "error"
		if len(resp.Errors) > 0 {
			resp.Error = fmt.Sprintf("All staged dedupe actions failed: %s", resp.Errors[0])
		} else {
			resp.Error = "All staged dedupe actions failed"
		}
	} else if resp.FailureCount > 0 {
		resp.Status = "partial_success"
		if len(resp.Errors) > 0 {
			resp.Error = fmt.Sprintf("%d of %d actions failed: %s", resp.FailureCount, len(req.Actions), resp.Errors[0])
		} else {
			resp.Error = fmt.Sprintf("%d of %d actions failed", resp.FailureCount, len(req.Actions))
		}
	} else {
		resp.Status = "success"
	}

	return resp, nil
}
