package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/slemmercs/rclone-bisync/internal/audit"
	"github.com/slemmercs/rclone-bisync/internal/core/client"
	"github.com/slemmercs/rclone-bisync/internal/models"
)

func (s *Server) handleRemotes(w http.ResponseWriter, r *http.Request) {
	refresh := r.URL.Query().Get("refresh") == "true"
	summary, err := s.TelemetryService.GetRemotesSummaryCached(r.Context(), refresh)
	if err != nil {
		writeJSON(w, http.StatusOK, models.RemotesResponse{
			Remotes: []map[string]interface{}{},
			Count:   0,
			Summary: map[string]interface{}{},
			Error:   err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (s *Server) handleRemotesAuth(w http.ResponseWriter, r *http.Request) {
	if s.AuthService == nil {
		writeJSON(w, http.StatusOK, models.AuthStatusResponse{})
		return
	}
	refresh := r.URL.Query().Get("refresh") == "true"
	resp, err := s.AuthService.GetAuthStatusCached(r.Context(), refresh)
	if err != nil {
		writeJSON(w, http.StatusOK, models.AuthStatusResponse{
			Summary: models.AuthHealthSummary{},
			Remotes: []models.RemoteAuthInfo{},
		})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleRemoteReconnect(w http.ResponseWriter, r *http.Request) {
	if s.AuthService == nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Status: "error", Error: "AuthService not initialized"})
		return
	}
	name := r.PathValue("name")
	if strings.TrimSpace(name) == "" {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Status: "error", Error: "Remote name parameter required"})
		return
	}

	resp, err := s.AuthService.ReconnectRemote(r.Context(), name)
	if err != nil {
		writeJSON(w, http.StatusOK, resp)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleRemoteReconnectCancel(w http.ResponseWriter, r *http.Request) {
	if s.AuthService == nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Status: "error", Error: "AuthService not initialized"})
		return
	}
	name := r.PathValue("name")
	if strings.TrimSpace(name) == "" {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Status: "error", Error: "Remote name parameter required"})
		return
	}

	cancelled := s.AuthService.CancelReconnect(name)
	writeJSON(w, http.StatusOK, models.CancelReconnectResponse{
		Status:    "success",
		Remote:    name,
		Cancelled: cancelled,
		Msg:       "Reconnect process cancelled.",
	})
}

func (s *Server) handleRemoteCredentials(w http.ResponseWriter, r *http.Request) {
	if s.AuthService == nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Status: "error", Error: "AuthService not initialized"})
		return
	}
	name := r.PathValue("name")
	if strings.TrimSpace(name) == "" {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Status: "error", Error: "Remote name parameter required"})
		return
	}

	var req models.UpdateCredentialsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Status: "error", Error: "Invalid request payload"})
		return
	}

	err := s.AuthService.UpdateCredentials(r.Context(), name, req.ClientID, req.ClientSecret)
	if err != nil {
		writeJSON(w, http.StatusOK, models.UpdateCredentialsResponse{Status: "error", Error: err.Error()})
		return
	}

	// Immediate validation check post-update
	val, _ := s.AuthService.ValidateRemote(r.Context(), name)

	writeJSON(w, http.StatusOK, models.UpdateCredentialsResponse{
		Status:     "success",
		Remote:     name,
		Msg:        "Credentials successfully saved.",
		Validation: &val,
	})
}

func (s *Server) handleRemotesAuthCheck(w http.ResponseWriter, r *http.Request) {
	if s.AuthService == nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Status: "error", Error: "AuthService not initialized"})
		return
	}

	targetRemote := strings.TrimSpace(r.URL.Query().Get("remote"))
	if targetRemote != "" {
		targetRemote = strings.TrimRight(targetRemote, ":")
		_, _ = s.AuthService.ValidateRemote(r.Context(), targetRemote)
		res, err := s.AuthService.GetAuthStatusCached(r.Context(), true)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Status: "error", Error: err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, res)
		return
	}

	res, err := s.AuthService.CheckAllRemotesAuth(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Status: "error", Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleRemoteValidate(w http.ResponseWriter, r *http.Request) {
	if s.AuthService == nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Status: "error", Error: "AuthService not initialized"})
		return
	}
	name := r.PathValue("name")
	if strings.TrimSpace(name) == "" {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Status: "error", Error: "Remote name parameter required"})
		return
	}

	res, err := s.AuthService.ValidateRemote(r.Context(), name)
	if err != nil {
		writeJSON(w, http.StatusOK, models.ValidateResponse{Status: "error", Error: err.Error(), Valid: false})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleRemotesAuthAudit(w http.ResponseWriter, r *http.Request) {
	if s.AuthService == nil {
		writeJSON(w, http.StatusOK, models.AuditEventsResponse{})
		return
	}
	const defaultLimit = 50
	const maxLimit = 500
	limit := defaultLimit
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			if parsed > maxLimit {
				limit = maxLimit
			} else {
				limit = parsed
			}
		}
	}
	events := s.AuthService.GetAuditEvents(limit)
	writeJSON(w, http.StatusOK, models.AuditEventsResponse{
		Events: events,
		Count:  len(events),
	})
}

func (s *Server) handleOAuthStop(w http.ResponseWriter, r *http.Request) {
	if s.AuthService == nil {
		writeJSON(w, http.StatusInternalServerError, models.OAuthStopResponse{Status: "error", Error: "AuthService not initialized"})
		return
	}
	_ = s.AuthService.StopOAuth(r.Context())
	writeJSON(w, http.StatusOK, models.OAuthStopResponse{
		Status: "success",
		Msg:    "OAuth processes stopped",
	})
}

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
		out = fmt.Sprintf("[SIMULATED] rclone dedupe --dedupe-mode %s %s:%s completed. No duplicate files found.", mode, remoteClean, req.Path)
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

	if s.AuditService != nil && mode != "list" {
		profileID := remoteClean
		if s.ProfileService != nil {
			for _, p := range s.ProfileService.List(true) {
				if strings.EqualFold(p.Remote, remoteClean) || strings.EqualFold(p.Remote, remoteClean+":") {
					profileID = p.ID
					break
				}
			}
		}
		_, _ = s.AuditService.RecordEvent(profileID, audit.ActionDedupeExecuted, "USER", map[string]any{
			"remote":  remoteClean,
			"path":    req.Path,
			"mode":    mode,
			"by_hash": req.ByHash,
			"output":  strings.TrimSpace(out),
		})
	}

	writeJSON(w, http.StatusOK, models.DedupeResponse{
		Status: "success",
		Remote: remoteClean,
		Path:   req.Path,
		Mode:   mode,
		ByHash: req.ByHash,
		Output: out,
	})
}
