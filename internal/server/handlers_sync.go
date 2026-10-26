package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/slemmercs/rclone-bisync/internal/models"
	"github.com/slemmercs/rclone-bisync/internal/profiles"
)

func (s *Server) handleAction(w http.ResponseWriter, r *http.Request) {
	var req models.ActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, models.ActionResponse{Status: "error", Msg: "Invalid request body", Error: "Invalid request body"})
		return
	}

	if req.ProfileID != "" && s.ProfileService != nil {
		switch req.Action {
		case "start_sync", "resync", "force_resync", "force_sync", "dry_run":
			opts := profiles.SyncOptions{
				Force:  req.Action == "force_sync",
				Resync: req.Action == "resync" || req.Action == "force_resync",
				DryRun: req.Action == "dry_run",
			}
			if err := s.ProfileService.TriggerSyncWithOptions(r.Context(), req.ProfileID, opts); err != nil {
				writeJSON(w, http.StatusOK, models.ActionResponse{Status: "error", Msg: err.Error(), Error: err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, models.ActionResponse{Status: "success", Msg: fmt.Sprintf("Action %s initiated for profile %s", req.Action, req.ProfileID)})
			return
		}
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
