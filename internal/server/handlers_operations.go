package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/slemmercs/rclone-bisync/internal/audit"
	"github.com/slemmercs/rclone-bisync/internal/models"
)

func (s *Server) registerOperationsRoutes() {
	s.Mux.HandleFunc("POST /api/operations/move", s.handleMoveItem)
	s.Mux.HandleFunc("POST /api/operations/purge", s.handlePurgeDirectory)
	s.Mux.HandleFunc("GET /api/operations/publiclink", s.handlePublicLink)
	s.Mux.HandleFunc("POST /api/operations/publiclink", s.handlePublicLink)
	s.Mux.HandleFunc("GET /api/explorer/preview", s.handleExplorerPreview)
	s.Mux.HandleFunc("POST /api/browse/upload", s.handleExplorerUpload)
	s.Mux.HandleFunc("POST /api/explorer/upload", s.handleExplorerUpload)
	s.Mux.HandleFunc("GET /api/explorer/download", s.handleExplorerDownload)
	s.Mux.HandleFunc("GET /api/explorer/dock", s.handleExplorerDock)
	s.Mux.HandleFunc("POST /api/explorer/dock", s.handleExplorerDock)
	s.Mux.HandleFunc("DELETE /api/explorer/dock", s.handleExplorerDock)
}

func (s *Server) handleMoveItem(w http.ResponseWriter, r *http.Request) {
	if s.ExplorerService == nil {
		writeJSON(w, http.StatusInternalServerError, models.MoveResponse{Status: "error", Error: "explorer service not configured"})
		return
	}

	var req models.MoveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, models.MoveResponse{Status: "error", Error: "Invalid request payload"})
		return
	}

	if strings.Contains(req.SrcPath, "..") || strings.Contains(req.DstPath, "..") {
		writeJSON(w, http.StatusOK, models.MoveResponse{Status: "error", Error: "Path traversal forbidden"})
		return
	}

	localBase := s.resolveLocalPath(req.ProfileID)
	resp, err := s.ExplorerService.MoveItem(r.Context(), req, localBase)
	if err != nil {
		writeJSON(w, http.StatusOK, models.MoveResponse{Status: "error", Error: err.Error()})
		return
	}

	if resp.Status == "success" && s.AuditService != nil {
		_, _ = s.AuditService.RecordEvent(req.ProfileID, audit.ActionWorkspaceItemMoved, s.resolveActor(r), map[string]any{
			"src_remote": req.SrcRemote,
			"src_path":   req.SrcPath,
			"dst_remote": req.DstRemote,
			"dst_path":   req.DstPath,
			"is_dir":     req.IsDir,
			"source":     "WORKSPACE_STAGING",
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handlePurgeDirectory(w http.ResponseWriter, r *http.Request) {
	if s.ExplorerService == nil {
		writeJSON(w, http.StatusInternalServerError, models.PurgeResponse{Status: "error", Error: "explorer service not configured"})
		return
	}

	var req models.PurgeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, models.PurgeResponse{Status: "error", Error: "Invalid request payload"})
		return
	}

	if strings.Contains(req.Path, "..") {
		writeJSON(w, http.StatusOK, models.PurgeResponse{Status: "error", Error: "Path traversal forbidden"})
		return
	}

	localBase := s.resolveLocalPath(req.ProfileID)
	resp, err := s.ExplorerService.PurgeDir(r.Context(), req, localBase)
	if err != nil {
		writeJSON(w, http.StatusOK, models.PurgeResponse{Status: "error", Error: err.Error()})
		return
	}

	if resp.Status == "success" && s.AuditService != nil {
		_, _ = s.AuditService.RecordEvent(req.ProfileID, audit.ActionWorkspaceDirPurged, s.resolveActor(r), map[string]any{
			"remote": req.Remote,
			"path":   req.Path,
			"source": "WORKSPACE_STAGING",
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handlePublicLink(w http.ResponseWriter, r *http.Request) {
	if s.ExplorerService == nil {
		writeJSON(w, http.StatusInternalServerError, models.PublicLinkResponse{Status: "error", Error: "explorer service not configured"})
		return
	}

	var req models.PublicLinkRequest
	if r.Method == http.MethodGet {
		q := r.URL.Query()
		req.ProfileID = q.Get("profile_id")
		req.Remote = q.Get("remote")
		req.Path = q.Get("path")
		req.Unlink = q.Get("unlink") == "true" || q.Get("unlink") == "1"
	} else {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, models.PublicLinkResponse{Status: "error", Error: "Invalid request payload"})
			return
		}
	}

	if strings.Contains(req.Path, "..") {
		writeJSON(w, http.StatusOK, models.PublicLinkResponse{Status: "error", Error: "Path traversal forbidden"})
		return
	}

	resp, err := s.ExplorerService.GetPublicLink(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusOK, models.PublicLinkResponse{Status: "error", Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleExplorerDock(w http.ResponseWriter, r *http.Request) {
	if s.UploadDock == nil {
		writeJSON(w, http.StatusOK, models.UploadDockStatus{
			PendingCount: 0,
			IsSyncActive: false,
			Items:        []models.UploadQueueItem{},
		})
		return
	}

	switch r.Method {
	case http.MethodGet:
		profileID := r.URL.Query().Get("profile_id")
		status := s.UploadDock.GetStatus(profileID)
		writeJSON(w, http.StatusOK, status)
	case http.MethodPost:
		var item models.UploadQueueItem
		if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
			return
		}
		enqueued, err := s.UploadDock.Enqueue(item)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, enqueued)
	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		profileID := r.URL.Query().Get("profile_id")
		if id != "" {
			s.UploadDock.RemoveItem(id)
		} else {
			s.UploadDock.ClearCompleted(profileID)
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "success"})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
	}
}
