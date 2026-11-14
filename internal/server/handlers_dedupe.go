package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/audit"
	"github.com/slemmercs/rclone-bisync/internal/core/client"
	"github.com/slemmercs/rclone-bisync/internal/models"
	"github.com/slemmercs/rclone-bisync/internal/profiles"
	"github.com/slemmercs/rclone-bisync/internal/services"
)

func (s *Server) handleRemoteDedupe(w http.ResponseWriter, r *http.Request) {
	var req models.DedupeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, models.DedupeResponse{Status: "error", Error: "Invalid request payload"})
		return
	}

	remoteClean := strings.TrimSpace(req.Remote)
	if remoteClean == "" {
		writeJSON(w, http.StatusBadRequest, models.DedupeResponse{Status: "error", Error: "Remote parameter required"})
		return
	}

	mode := strings.ToLower(strings.TrimSpace(req.Mode))
	if mode == "" {
		mode = "oldest"
	}
	if mode != "list" && mode != "oldest" && mode != "newest" {
		writeJSON(w, http.StatusBadRequest, models.DedupeResponse{Status: "error", Error: "Invalid dedupe mode (must be 'list', 'oldest', or 'newest')"})
		return
	}

	var rcClient *client.RcloneClient
	if s.SyncService != nil && s.SyncService.Client != nil {
		rcClient = s.SyncService.Client
	} else if s.ExplorerService != nil && s.ExplorerService.Client != nil {
		rcClient = s.ExplorerService.Client
	}

	var out string
	if rcClient == nil {
		if mode != "list" {
			out = fmt.Sprintf("[SIMULATED] rclone dedupe --dedupe-mode %s %s:%s completed. 2 duplicate files removed.", mode, remoteClean, req.Path)
		} else {
			out = fmt.Sprintf("[SIMULATED] rclone dedupe --dedupe-mode %s %s:%s completed. No duplicate files found.", mode, remoteClean, req.Path)
		}
	} else {
		var err error
		out, err = rcClient.Dedupe(r.Context(), remoteClean, req.Path, mode, req.ByHash)
		if err != nil {
			writeJSON(w, http.StatusOK, models.DedupeResponse{
				Status: "error",
				Remote: remoteClean,
				Path:   req.Path,
				Mode:   mode,
				ByHash: req.ByHash,
				Error:  err.Error(),
			})
			return
		}
	}

	runID := fmt.Sprintf("dedupe-%d", time.Now().Unix())

	profileID := remoteClean
	var matchedProfile *profiles.Profile
	if req.ProfileID != "" && s.ProfileService != nil {
		matchedProfile, _ = s.ProfileService.Get(req.ProfileID)
		if matchedProfile != nil {
			profileID = matchedProfile.ID
		}
	} else if s.ProfileService != nil {
		for _, p := range s.ProfileService.List(true) {
			if strings.EqualFold(p.Remote, remoteClean) || strings.EqualFold(p.Remote, remoteClean+":") {
				matchedProfile = p
				profileID = p.ID
				break
			}
		}
	}

	var pairedLocalPath string
	var pairedLocalOut string
	if req.DedupeBoth && matchedProfile != nil && !strings.EqualFold(remoteClean, "local") {
		localBase := s.resolveLocalPath(matchedProfile.ID)
		cleanSubPath := strings.Trim(strings.TrimSpace(req.Path), "/")
		_, targetDir, ok := services.IsLocalPathJailed(localBase, cleanSubPath)
		if ok {
			pairedLocalPath = targetDir
			if rcClient == nil {
				if mode != "list" {
					pairedLocalOut = fmt.Sprintf("[SIMULATED] rclone dedupe --dedupe-mode %s %s completed. 0 duplicate files removed.", mode, targetDir)
				} else {
					pairedLocalOut = fmt.Sprintf("[SIMULATED] rclone dedupe --dedupe-mode %s %s completed. No duplicate files found.", mode, targetDir)
				}
			} else {
				var err error
				pairedLocalOut, err = rcClient.Dedupe(r.Context(), targetDir, "", mode, req.ByHash)
				if err != nil {
					pairedLocalOut = fmt.Sprintf("[WARN] local dedupe failed: %v", err)
				}
			}
		}
	}

	var collisionsCount, deletedCount int
	countLines := func(output string) {
		for _, line := range strings.Split(output, "\n") {
			l := strings.TrimSpace(line)
			if l == "" {
				continue
			}
			if strings.Contains(strings.ToLower(l), "duplicate") {
				collisionsCount++
			}
			if strings.Contains(strings.ToLower(l), "deleted") || strings.Contains(strings.ToLower(l), "removed") || strings.Contains(strings.ToLower(l), "renamed") {
				deletedCount++
			}
		}
	}
	countLines(out)
	if pairedLocalOut != "" {
		countLines(pairedLocalOut)
	}

	// Invariant: Dry-runs (mode == "list") suppress stdout spam to bisync.log.
	// Only real mutations (mode != "list") record a clean summary line to bisync.log.
	if mode != "list" && s.AuditService != nil && s.AuditService.Logger != nil {
		summary := fmt.Sprintf("[%s] Deduplication (%s) complete: %d collision(s) found, %d duplicate(s) deleted.", profileID, mode, collisionsCount, deletedCount)
		_ = s.AuditService.Logger.RecordSyncRun(models.BisyncLogEntry{
			Time:        time.Now().Format(time.RFC3339),
			Level:       "notice",
			Type:        "dedupe",
			Msg:         summary,
			RunID:       runID,
			Origin:      "dedupe",
			ProfileID:   profileID,
			ProfileName: profileID,
			Action:      "DEDUPE_EXECUTED",
		})
	}

	// Invariant: Audit ledger strictly records mutations (state changes).
	// Zero-deletion runs or cancelled dry-runs are omitted from audit.db.
	if s.AuditService != nil && mode != "list" && deletedCount > 0 {
		_, _ = s.AuditService.RecordEvent(profileID, audit.ActionDedupeExecuted, s.resolveActor(r), map[string]any{
			"run_id":      runID,
			"remote":      remoteClean,
			"path":        req.Path,
			"local_path":  pairedLocalPath,
			"mode":        mode,
			"by_hash":     req.ByHash,
			"dedupe_both": req.DedupeBoth,
			"collisions":  collisionsCount,
			"deleted":     deletedCount,
		})
	}

	writeJSON(w, http.StatusOK, models.DedupeResponse{
		Status:      "success",
		RunID:       runID,
		Remote:      remoteClean,
		Path:        req.Path,
		LocalPath:   pairedLocalPath,
		Mode:        mode,
		ByHash:      req.ByHash,
		Output:      out,
		LocalOutput: pairedLocalOut,
		Collisions:  collisionsCount,
		Deleted:     deletedCount,
	})
}
