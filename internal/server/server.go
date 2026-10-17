package server

import (
		"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/core/security"
	"github.com/slemmercs/rclone-bisync/internal/models"
	"github.com/slemmercs/rclone-bisync/internal/services"
)

type Server struct {
	Config      *config.Config
	SyncService *services.SyncService
	LogService  *services.LogService
	FrontendFS  embed.FS
	Mux         *http.ServeMux
}

func NewServer(cfg *config.Config, syncService *services.SyncService, frontendFS embed.FS) *Server {
	s := &Server{
		Config:      cfg,
		SyncService: syncService,
		LogService:  services.NewLogService(),
		FrontendFS:  frontendFS,
		Mux:         http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// CSRF & Security middleware
	if (r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodDelete || r.Method == http.MethodPatch) && strings.HasPrefix(r.URL.Path, "/api/") {
		if r.Header.Get("X-Dashboard-Request") != "1" {
			writeJSON(w, http.StatusForbidden, map[string]interface{}{
				"status": "error",
				"msg":    "Security policy violation: Missing required X-Dashboard-Request header.",
			})
			return
		}

		origin := r.Header.Get("Origin")
		if origin != "" && !isAllowedOrigin(origin) {
			writeJSON(w, http.StatusForbidden, map[string]interface{}{
				"status": "error",
				"msg":    fmt.Sprintf("Security policy violation: Untrusted request origin '%s'.", origin),
			})
			return
		}

		referer := r.Header.Get("Referer")
		if referer != "" && !isAllowedOrigin(referer) {
			writeJSON(w, http.StatusForbidden, map[string]interface{}{
				"status": "error",
				"msg":    "Security policy violation: Untrusted request referer.",
			})
			return
		}
	}

	s.Mux.ServeHTTP(w, r)
}

func isAllowedOrigin(rawURL string) bool {
	if rawURL == "" {
		return true
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	hostname := u.Hostname()
	return hostname == "localhost" || hostname == "127.0.0.1"
}

func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	redacted := security.RedactSecrets(data)
	_ = json.NewEncoder(w).Encode(redacted)
}

func (s *Server) routes() {
	s.Mux.HandleFunc("GET /api/logs/bisync/raw", s.handleRawBisyncLog)
	s.Mux.HandleFunc("GET /api/logs/daemon/raw", s.handleRawDaemonLog)
	s.Mux.HandleFunc("GET /api/logs/daemon", s.handleDaemonLogs)
	s.Mux.HandleFunc("GET /api/telemetry", s.handleTelemetry)
	s.Mux.HandleFunc("POST /api/action", s.handleAction)
	s.Mux.HandleFunc("GET /api/deletions/pending", s.handlePendingDeletions)
	s.Mux.HandleFunc("GET /api/remotes", s.handleRemotes)
	s.Mux.HandleFunc("POST /api/browse", s.handleBrowse)
	s.Mux.HandleFunc("POST /api/browse/canary", s.handleProvisionCanary)
	s.Mux.HandleFunc("POST /api/browse/copy", s.handleCopyItem)
	s.Mux.HandleFunc("POST /api/browse/delete", s.handleDeleteItem)

	// Static assets & SPA
	staticSubFS, err := fs.Sub(s.FrontendFS, "dashboard/frontend")
	if err == nil {
		fileServer := http.FileServer(http.FS(staticSubFS))
		s.Mux.Handle("GET /static/", http.StripPrefix("/static/", fileServer))

		s.Mux.HandleFunc("GET /favicon.ico", func(w http.ResponseWriter, r *http.Request) {
			favicon, err := fs.ReadFile(staticSubFS, "favicon.svg")
			if err != nil {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "image/svg+xml")
			_, _ = w.Write(favicon)
		})

		s.Mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
			indexHTML, err := fs.ReadFile(staticSubFS, "index.html")
			if err != nil {
				http.Error(w, "index.html not found", http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(indexHTML)
		})
	}
}

func (s *Server) handleRawBisyncLog(w http.ResponseWriter, r *http.Request) {
	if _, err := os.Stat(s.Config.LogFile); os.IsNotExist(err) {
		writeJSON(w, http.StatusOK, map[string]string{"error": "Log file not found"})
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Disposition", "attachment; filename=bisync.log")
	http.ServeFile(w, r, s.Config.LogFile)
}

func (s *Server) handleRawDaemonLog(w http.ResponseWriter, r *http.Request) {
	daemonLog := filepath.Join(s.Config.BaseDir, "rclone-daemon.log")
	if _, err := os.Stat(daemonLog); os.IsNotExist(err) {
		writeJSON(w, http.StatusOK, map[string]string{"error": "Daemon log file not found"})
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Disposition", "attachment; filename=rclone-daemon.log")
	http.ServeFile(w, r, daemonLog)
}

func (s *Server) handleDaemonLogs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	runID := q.Get("run_id")
	jobIDStr := q.Get("job_id")
	jobID := 0
	if jobIDStr != "" {
		if parsed, err := strconv.Atoi(jobIDStr); err == nil {
			jobID = parsed
		}
	}

	startStr := q.Get("start_time")
	if startStr == "" {
		startStr = q.Get("start")
	}
	endStr := q.Get("end_time")
	if endStr == "" {
		endStr = q.Get("end")
	}

	var startTime, endTime time.Time
	if startStr != "" {
		if t, err := time.Parse(time.RFC3339Nano, startStr); err == nil {
			startTime = t
		} else if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			startTime = t
		}
	}
	if endStr != "" {
		if t, err := time.Parse(time.RFC3339Nano, endStr); err == nil {
			endTime = t
		} else if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			endTime = t
		}
	}

	limit := 200
	if qLimit := q.Get("limit"); qLimit != "" {
		if parsed, err := strconv.Atoi(qLimit); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	daemonLog := filepath.Join(s.Config.BaseDir, "rclone-daemon.log")
	logs, err := s.LogService.FilterDaemonLogs(daemonLog, startTime, endTime, limit)
	if err != nil {
		writeJSON(w, http.StatusOK, models.DaemonLogsResponse{
			RunID: runID,
			JobID: jobID,
			Error: err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, models.DaemonLogsResponse{
		RunID:      runID,
		JobID:      jobID,
		Count:      len(logs),
		Logs:       logs,
		Correlated: !startTime.IsZero(),
	})
}

func (s *Server) handleTelemetry(w http.ResponseWriter, r *http.Request) {
	limit := 40
	if qLimit := r.URL.Query().Get("limit"); qLimit != "" {
		if parsed, err := strconv.Atoi(qLimit); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	payload := s.SyncService.GetTelemetryPayload(r.Context(), limit)
	writeJSON(w, http.StatusOK, payload)
}

func (s *Server) handleAction(w http.ResponseWriter, r *http.Request) {
	var req models.ActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Invalid request body"})
		return
	}
	res, err := s.SyncService.HandleAction(r.Context(), req.Action)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": "error", "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handlePendingDeletions(w http.ResponseWriter, r *http.Request) {
	data := s.SyncService.GetPendingDeletions()
	writeJSON(w, http.StatusOK, data)
}

func (s *Server) handleRemotes(w http.ResponseWriter, r *http.Request) {
	summary, err := s.SyncService.GetRemotesSummary(r.Context())
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"remotes": []interface{}{},
			"count":   0,
			"summary": map[string]interface{}{},
			"error":   err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (s *Server) handleBrowse(w http.ResponseWriter, r *http.Request) {
	var req models.BrowseRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	remote := req.Remote
	if remote == "" {
		remote = "local"
	}
	res := s.SyncService.BrowsePath(r.Context(), remote, req.Path)
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleProvisionCanary(w http.ResponseWriter, r *http.Request) {
	var req models.BrowseRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	remote := req.Remote
	if remote == "" {
		remote = "local"
	}
	res, err := s.SyncService.CreateCanary(r.Context(), remote, req.Path)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": "error", "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleCopyItem(w http.ResponseWriter, r *http.Request) {
	var req models.CopyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Invalid request body"})
		return
	}
	res, err := s.SyncService.CopyItem(r.Context(), req.SrcRemote, req.SrcPath, req.DstRemote, req.DstPath, req.IsDir)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": "error", "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleDeleteItem(w http.ResponseWriter, r *http.Request) {
	var req models.DeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Invalid request body"})
		return
	}
	res, err := s.SyncService.DeleteItem(r.Context(), req.Remote, req.Path, req.IsDir)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": "error", "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, res)
}
