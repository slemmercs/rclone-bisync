package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/slemmercs/rclone-bisync/internal/models"
)

// handleRemoteProviders retrieves categorized Rclone storage backends and schemas.
func (s *Server) handleRemoteProviders(w http.ResponseWriter, r *http.Request) {
	if s.RemoteWizardService == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"error": "remote wizard service unavailable",
		})
		return
	}

	refresh := r.URL.Query().Get("refresh") == "true" || r.URL.Query().Get("refresh") == "1"
	catalog, err := s.RemoteWizardService.GetProvidersCatalog(r.Context(), refresh)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success":    true,
		"providers":  catalog.Providers,
		"categories": catalog.Categories,
	})
}

// handleCreateRemote handles remote creation with optional profile provisioning.
func (s *Server) handleCreateRemote(w http.ResponseWriter, r *http.Request) {
	if s.RemoteWizardService == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"error": "remote wizard service unavailable",
		})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req models.CreateRemoteBridgeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "invalid request payload: " + err.Error(),
		})
		return
	}

	actor := s.resolveActor(r)
	res, err := s.RemoteWizardService.CreateRemote(r.Context(), req, actor)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "cannot be empty") ||
			strings.Contains(err.Error(), "already exists") ||
			strings.Contains(err.Error(), "invalid remote name") {
			status = http.StatusBadRequest
		}
		writeJSON(w, status, map[string]any{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusCreated, res)
}

// handleDeleteRemote removes a configured remote with profile dependency conflict guards.
func (s *Server) handleDeleteRemote(w http.ResponseWriter, r *http.Request) {
	if s.RemoteWizardService == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"error": "remote wizard service unavailable",
		})
		return
	}

	remoteName := r.PathValue("name")
	if strings.TrimSpace(remoteName) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "remote name is required",
		})
		return
	}

	force := r.URL.Query().Get("force") == "true" || r.URL.Query().Get("force") == "1"
	actor := s.resolveActor(r)

	if err := s.RemoteWizardService.DeleteRemote(r.Context(), remoteName, force, actor); err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "referenced by active profile") {
			status = http.StatusConflict
		} else if strings.Contains(err.Error(), "cannot be empty") {
			status = http.StatusBadRequest
		}
		writeJSON(w, status, map[string]any{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "Remote deleted successfully",
	})
}
