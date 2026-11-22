package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/slemmercs/rclone-bisync/internal/models"
)

// handleHasherWrap provisions a Hasher overlay remote wrapping an existing remote.
func (s *Server) handleHasherWrap(w http.ResponseWriter, r *http.Request) {
	if s.HasherService == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"error": "hasher service unavailable",
		})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req models.HasherWrapRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "invalid request payload: " + err.Error(),
		})
		return
	}

	actor := s.resolveActor(r)
	res, err := s.HasherService.WrapRemoteInHasher(r.Context(), req, actor)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "cannot be empty") ||
			strings.Contains(err.Error(), "already exists") ||
			strings.Contains(err.Error(), "invalid hasher remote name") {
			status = http.StatusBadRequest
		}
		writeJSON(w, status, map[string]any{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusCreated, res)
}

// handleHasherCommand dispatches supported Hasher backend commands.
func (s *Server) handleHasherCommand(w http.ResponseWriter, r *http.Request) {
	if s.HasherService == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"error": "hasher service unavailable",
		})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req models.HasherCommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "invalid request payload: " + err.Error(),
		})
		return
	}

	res, err := s.HasherService.ExecuteHasherCommand(r.Context(), req)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "cannot be empty") ||
			strings.Contains(err.Error(), "unsupported hasher command") {
			status = http.StatusBadRequest
		}
		writeJSON(w, status, map[string]any{
			"error": err.Error(),
			"data":  res,
		})
		return
	}

	writeJSON(w, http.StatusOK, res)
}

// handleCryptWrap provisions a Crypt encryption overlay wrapping an existing remote.
func (s *Server) handleCryptWrap(w http.ResponseWriter, r *http.Request) {
	if s.CryptService == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"error": "crypt service unavailable",
		})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req models.CryptWrapRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "invalid request payload: " + err.Error(),
		})
		return
	}

	actor := s.resolveActor(r)
	res, err := s.CryptService.WrapRemoteInCrypt(r.Context(), req, actor)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "cannot be empty") ||
			strings.Contains(err.Error(), "already exists") ||
			strings.Contains(err.Error(), "invalid crypt remote name") ||
			strings.Contains(err.Error(), "invalid filename_encryption") {
			status = http.StatusBadRequest
		}
		writeJSON(w, status, map[string]any{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusCreated, res)
}
