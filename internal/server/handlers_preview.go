package server

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
)

// handleExplorerPreview serves file previews (UTF-8 text or base64 image data URIs) with jail confinement.
func (s *Server) handleExplorerPreview(w http.ResponseWriter, r *http.Request) {
	if s.ExplorerService == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "explorer service is not available"})
		return
	}

	remote := r.URL.Query().Get("remote")
	path := r.URL.Query().Get("path")
	profileID := r.URL.Query().Get("profile_id")

	if strings.TrimSpace(path) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "path parameter is required"})
		return
	}

	localBase := s.resolveLocalPath(profileID)
	resp, err := s.ExplorerService.PreviewItem(r.Context(), remote, path, localBase)
	if err != nil {
		status := http.StatusInternalServerError
		errLower := strings.ToLower(err.Error())
		if strings.Contains(errLower, "access denied") || strings.Contains(errLower, "traversal") {
			status = http.StatusForbidden
		} else if strings.Contains(errLower, "not found") {
			status = http.StatusNotFound
		} else if strings.Contains(errLower, "cannot preview directory") {
			status = http.StatusBadRequest
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// handleExplorerDownload streams a single file directly to the client with appropriate headers.
func (s *Server) handleExplorerDownload(w http.ResponseWriter, r *http.Request) {
	if s.ExplorerService == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "explorer service is not available"})
		return
	}

	remote := r.URL.Query().Get("remote")
	path := r.URL.Query().Get("path")
	profileID := r.URL.Query().Get("profile_id")

	if strings.TrimSpace(path) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "path parameter is required"})
		return
	}

	localBase := s.resolveLocalPath(profileID)
	stream, name, size, err := s.ExplorerService.DownloadStream(r.Context(), remote, path, localBase)
	if err != nil {
		status := http.StatusInternalServerError
		errLower := strings.ToLower(err.Error())
		if strings.Contains(errLower, "access denied") || strings.Contains(errLower, "traversal") {
			status = http.StatusForbidden
		} else if strings.Contains(errLower, "not found") {
			status = http.StatusNotFound
		} else if strings.Contains(errLower, "cannot download directory") {
			status = http.StatusBadRequest
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	defer stream.Close()

	if name == "" {
		name = filepath.Base(path)
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filepath.Base(name)))
	w.Header().Set("Content-Type", "application/octet-stream")
	if size > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	}
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, stream)
}
