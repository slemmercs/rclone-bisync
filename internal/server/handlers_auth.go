package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/slemmercs/rclone-bisync/internal/models"
)

func (s *Server) handleRemotes(w http.ResponseWriter, r *http.Request) {
	summary, err := s.TelemetryService.GetRemotesSummary(r.Context())
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
	resp, err := s.AuthService.GetAuthStatus(r.Context())
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
