package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/slemmercs/rclone-bisync/internal/models"
	"github.com/slemmercs/rclone-bisync/internal/profiles"
)

func (s *Server) handleListProfiles(w http.ResponseWriter, r *http.Request) {
	if s.ProfileService == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":   "success",
			"profiles": []*profiles.Profile{},
			"total":    0,
		})
		return
	}
	includeArchived := r.URL.Query().Get("archived") == "true"
	profs := s.ProfileService.List(includeArchived)
	writeJSON(w, http.StatusOK, map[string]any{
		"status":   "success",
		"profiles": profs,
		"total":    len(profs),
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
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "success",
		"profile": prof,
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
		Force bool `json:"force"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	err := s.ProfileService.TriggerSync(r.Context(), id, req.Force)
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
