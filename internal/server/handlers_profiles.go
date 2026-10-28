package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/models"
	"github.com/slemmercs/rclone-bisync/internal/profiles"
)

func (s *Server) getGlobalDefaults() config.GlobalDefaults {
	def := config.DefaultGlobalDefaults()
	if s.Config == nil {
		return def
	}
	g := s.Config.GlobalDefaults
	if g.MaxDelete <= 0 {
		g.MaxDelete = def.MaxDelete
	}
	if g.ConflictResolve == "" {
		g.ConflictResolve = def.ConflictResolve
	}
	if g.CompareMode == "" {
		g.CompareMode = def.CompareMode
	}
	return g
}

func (s *Server) handleListProfiles(w http.ResponseWriter, r *http.Request) {
	homeDir, _ := os.UserHomeDir()
	globals := s.getGlobalDefaults()
	if s.ProfileService == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":          "success",
			"profiles":        []*profiles.Profile{},
			"total":           0,
			"home_dir":        homeDir,
			"global_defaults": globals,
		})
		return
	}
	includeArchived := r.URL.Query().Get("archived") == "true"
	profs := s.ProfileService.List(includeArchived)
	writeJSON(w, http.StatusOK, map[string]any{
		"status":          "success",
		"profiles":        profs,
		"total":           len(profs),
		"home_dir":        homeDir,
		"global_defaults": globals,
	})
}

func (s *Server) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	if s.ProfileService == nil {
		writeJSON(w, http.StatusServiceUnavailable, models.ErrorResponse{Status: "error", Error: "Profile service not initialized"})
		return
	}
	id := r.PathValue("id")
	prof, err := s.ProfileService.Get(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, models.ErrorResponse{Status: "error", Error: err.Error()})
		return
	}
	globals := s.getGlobalDefaults()
	resolved := profiles.ResolveFlags(prof, globals, nil)
	writeJSON(w, http.StatusOK, map[string]any{
		"status":          "success",
		"profile":         prof,
		"resolved_flags":  resolved,
		"global_defaults": globals,
	})
}

func (s *Server) handleCreateProfile(w http.ResponseWriter, r *http.Request) {
	if s.ProfileService == nil {
		writeJSON(w, http.StatusServiceUnavailable, models.ErrorResponse{Status: "error", Error: "Profile service not initialized"})
		return
	}
	var p profiles.Profile
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Status: "error", Error: "Invalid JSON payload"})
		return
	}
	if strings.Contains(p.LocalPath, "..") {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Status: "error", Error: "Relative path traversal forbidden in local path"})
		return
	}
	if p.Flags.CustomFlags != "" {
		if _, err := profiles.ValidateCustomFlags(p.Flags.CustomFlags); err != nil {
			writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Status: "error", Error: err.Error()})
			return
		}
	}
	created, err := s.ProfileService.Create(r.Context(), &p)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, profiles.ErrProfileAlreadyExists) {
			status = http.StatusConflict
		}
		writeJSON(w, status, models.ErrorResponse{Status: "error", Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"status":  "success",
		"profile": created,
	})
}

func (s *Server) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	if s.ProfileService == nil {
		writeJSON(w, http.StatusServiceUnavailable, models.ErrorResponse{Status: "error", Error: "Profile service not initialized"})
		return
	}
	id := r.PathValue("id")
	var req struct {
		profiles.Profile
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Status: "error", Error: "Invalid JSON payload"})
		return
	}
	if strings.Contains(req.Profile.LocalPath, "..") {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Status: "error", Error: "Relative path traversal forbidden in local path"})
		return
	}
	if req.Profile.Flags.CustomFlags != "" {
		if _, err := profiles.ValidateCustomFlags(req.Profile.Flags.CustomFlags); err != nil {
			writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Status: "error", Error: err.Error()})
			return
		}
	}
	req.Profile.ID = id
	updated, err := s.ProfileService.Update(r.Context(), &req.Profile, req.Reason)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, profiles.ErrProfileNotFound) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, models.ErrorResponse{Status: "error", Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "success",
		"profile": updated,
	})
}

func (s *Server) handleDeleteProfile(w http.ResponseWriter, r *http.Request) {
	if s.ProfileService == nil {
		writeJSON(w, http.StatusServiceUnavailable, models.ErrorResponse{Status: "error", Error: "Profile service not initialized"})
		return
	}
	id := r.PathValue("id")
	if err := s.ProfileService.Delete(r.Context(), id); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, profiles.ErrProfileNotFound) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, models.ErrorResponse{Status: "error", Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "success",
		"msg":    fmt.Sprintf("Profile %s deleted", id),
	})
}

func (s *Server) handleCloneProfile(w http.ResponseWriter, r *http.Request) {
	if s.ProfileService == nil {
		writeJSON(w, http.StatusServiceUnavailable, models.ErrorResponse{Status: "error", Error: "Profile service not initialized"})
		return
	}
	sourceID := r.PathValue("id")
	var req struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Status: "error", Error: "Invalid JSON payload"})
		return
	}
	cloned, err := s.ProfileService.Clone(r.Context(), sourceID, req.ID, req.Name)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, profiles.ErrProfileNotFound) {
			status = http.StatusNotFound
		} else if errors.Is(err, profiles.ErrProfileAlreadyExists) {
			status = http.StatusConflict
		}
		writeJSON(w, status, models.ErrorResponse{Status: "error", Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"status":  "success",
		"profile": cloned,
	})
}

func (s *Server) handleRollbackProfile(w http.ResponseWriter, r *http.Request) {
	if s.ProfileService == nil {
		writeJSON(w, http.StatusServiceUnavailable, models.ErrorResponse{Status: "error", Error: "Profile service not initialized"})
		return
	}
	id := r.PathValue("id")
	var req struct {
		TargetVersion int `json:"target_version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Status: "error", Error: "Invalid JSON payload"})
		return
	}
	restored, err := s.ProfileService.Rollback(r.Context(), id, req.TargetVersion)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, profiles.ErrProfileNotFound) || errors.Is(err, profiles.ErrVersionNotFound) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, models.ErrorResponse{Status: "error", Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "success",
		"profile": restored,
	})
}

func (s *Server) handleProfileSync(w http.ResponseWriter, r *http.Request) {
	if s.ProfileService == nil {
		writeJSON(w, http.StatusServiceUnavailable, models.ErrorResponse{Status: "error", Error: "Profile service not initialized"})
		return
	}
	id := r.PathValue("id")
	var req struct {
		Force  bool `json:"force"`
		Resync bool `json:"resync"`
		DryRun bool `json:"dry_run"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	opts := profiles.SyncOptions{
		Force:  req.Force,
		Resync: req.Resync,
		DryRun: req.DryRun,
	}
	err := s.ProfileService.TriggerSyncWithOptions(r.Context(), id, opts)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, profiles.ErrProfileNotActive) || errors.Is(err, profiles.ErrProfileNotFound) {
			status = http.StatusNotFound
		} else if errors.Is(err, profiles.ErrSyncInProgress) {
			status = http.StatusConflict
		}
		writeJSON(w, status, models.ErrorResponse{Status: "error", Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "success",
		"msg":    fmt.Sprintf("Sync triggered for profile %s", id),
	})
}

func (s *Server) handleProfileHistory(w http.ResponseWriter, r *http.Request) {
	if s.ProfileService == nil {
		writeJSON(w, http.StatusServiceUnavailable, models.ErrorResponse{Status: "error", Error: "Profile service not initialized"})
		return
	}
	id := r.PathValue("id")
	history, err := s.ProfileService.GetHistory(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, models.ErrorResponse{Status: "error", Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "success",
		"history": history,
	})
}
