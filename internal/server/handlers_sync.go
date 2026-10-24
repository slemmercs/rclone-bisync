package server

import (
	"encoding/json"
	"net/http"

	"github.com/slemmercs/rclone-bisync/internal/models"
)

func (s *Server) handleAction(w http.ResponseWriter, r *http.Request) {
	var req models.ActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, models.ActionResponse{Status: "error", Msg: "Invalid request body", Error: "Invalid request body"})
		return
	}
	if s.SyncService == nil {
		writeJSON(w, http.StatusOK, models.ActionResponse{Status: "error", Msg: "SyncService not initialized", Error: "SyncService not initialized"})
		return
	}
	res, err := s.SyncService.HandleAction(r.Context(), req.Action)
	if err != nil {
		writeJSON(w, http.StatusOK, models.ActionResponse{Status: "error", Msg: err.Error(), Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handlePendingDeletions(w http.ResponseWriter, r *http.Request) {
	data := s.DeletionService.GetPendingDeletions()
	writeJSON(w, http.StatusOK, data)
}
