package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/slemmercs/rclone-bisync/internal/models"
)

// handleGetConfigEncryption reports the encryption status of rclone.conf.
func (s *Server) handleGetConfigEncryption(w http.ResponseWriter, r *http.Request) {
	if s.ConfigEncryptionService == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "config encryption service is not available"})
		return
	}

	status, err := s.ConfigEncryptionService.GetStatus(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, status)
}

// handleSetConfigEncryption sets or updates the password on rclone.conf.
func (s *Server) handleSetConfigEncryption(w http.ResponseWriter, r *http.Request) {
	if s.ConfigEncryptionService == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "config encryption service is not available"})
		return
	}

	var req models.SetEncryptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request payload"})
		return
	}

	if req.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "password cannot be empty"})
		return
	}

	resp, err := s.ConfigEncryptionService.SetPassword(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// handleRemoveConfigEncryption removes the encryption password from rclone.conf.
func (s *Server) handleRemoveConfigEncryption(w http.ResponseWriter, r *http.Request) {
	if s.ConfigEncryptionService == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "config encryption service is not available"})
		return
	}

	var req models.RemoveEncryptionRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	if strings.TrimSpace(req.Password) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"status": "warning",
			"error":  "Current passphrase is required to remove encryption.",
		})
		return
	}

	resp, err := s.ConfigEncryptionService.RemovePassword(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, resp)
}
