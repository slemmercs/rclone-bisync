package server

import (
	"encoding/json"
	"net/http"

	"github.com/slemmercs/rclone-bisync/internal/models"
)

func (s *Server) resolveLocalPath(profileID string) string {
	if s.ProfileService != nil && profileID != "" {
		if p, err := s.ProfileService.Get(profileID); err == nil && p != nil && p.LocalPath != "" {
			return p.LocalPath
		}
	}
	if s.Config != nil && s.Config.LocalPath != "" {
		return s.Config.LocalPath
	}
	return "~/gdrive"
}

func (s *Server) resolveExcludePatterns(profileID string) []string {
	if s.ProfileService != nil && profileID != "" {
		if p, err := s.ProfileService.Get(profileID); err == nil && p != nil && len(p.ExcludePatterns) > 0 {
			return p.ExcludePatterns
		}
	}
	if s.Config != nil && len(s.Config.RcloneCommonArgs) > 0 {
		var rules []string
		for i := 0; i < len(s.Config.RcloneCommonArgs); i++ {
			if s.Config.RcloneCommonArgs[i] == "--exclude" && i+1 < len(s.Config.RcloneCommonArgs) {
				rules = append(rules, s.Config.RcloneCommonArgs[i+1])
				i++
			}
		}
		return rules
	}
	return nil
}

func (s *Server) handleBrowse(w http.ResponseWriter, r *http.Request) {
	var req models.BrowseRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	remote := req.Remote
	if remote == "" {
		remote = "local"
	}
	res := s.ExplorerService.BrowsePath(r.Context(), remote, req.Path, s.resolveLocalPath(req.ProfileID))
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleProvisionCanary(w http.ResponseWriter, r *http.Request) {
	var req models.BrowseRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	remote := req.Remote
	if remote == "" {
		remote = "local"
	}
	res, err := s.CanaryService.CreateCanary(r.Context(), remote, req.Path, s.resolveLocalPath(req.ProfileID))
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
	localBase := s.resolveLocalPath(req.ProfileID)
	excludes := s.resolveExcludePatterns(req.ProfileID)
	res, err := s.ExplorerService.CopyItemWithOptions(r.Context(), req.SrcRemote, req.SrcPath, req.DstRemote, req.DstPath, req.IsDir, localBase, excludes)
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
	res, err := s.ExplorerService.DeleteItem(r.Context(), req.Remote, req.Path, req.IsDir, s.resolveLocalPath(req.ProfileID))
	if err != nil {
		writeJSON(w, http.StatusOK, models.DeleteResponse{Status: "error", Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, res)
}
