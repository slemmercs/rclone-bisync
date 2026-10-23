package server

import (
	"encoding/json"
	"net/http"

	"github.com/slemmercs/rclone-bisync/internal/models"
)

func (s *Server) handleBrowse(w http.ResponseWriter, r *http.Request) {
	var req models.BrowseRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	remote := req.Remote
	if remote == "" {
		remote = "local"
	}
	res := s.ExplorerService.BrowsePath(r.Context(), remote, req.Path)
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleProvisionCanary(w http.ResponseWriter, r *http.Request) {
	var req models.BrowseRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	remote := req.Remote
	if remote == "" {
		remote = "local"
	}
	res, err := s.CanaryService.CreateCanary(r.Context(), remote, req.Path)
	if err != nil {
		writeJSON(w, http.StatusOK, models.CanaryResponse{Status: "error", Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleCopyItem(w http.ResponseWriter, r *http.Request) {
	var req models.CopyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, models.CopyResponse{Status: "error", Error: "Invalid request body"})
		return
	}
	res, err := s.ExplorerService.CopyItem(r.Context(), req.SrcRemote, req.SrcPath, req.DstRemote, req.DstPath, req.IsDir)
	if err != nil {
		writeJSON(w, http.StatusOK, models.CopyResponse{Status: "error", Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleDeleteItem(w http.ResponseWriter, r *http.Request) {
	var req models.DeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, models.DeleteResponse{Status: "error", Error: "Invalid request body"})
		return
	}
	res, err := s.ExplorerService.DeleteItem(r.Context(), req.Remote, req.Path, req.IsDir)
	if err != nil {
		writeJSON(w, http.StatusOK, models.DeleteResponse{Status: "error", Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, res)
}
