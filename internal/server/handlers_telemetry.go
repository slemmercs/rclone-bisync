package server

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/models"
)

func (s *Server) handleDaemonHealth(w http.ResponseWriter, r *http.Request) {
	health := s.TelemetryService.GetDaemonHealth(r.Context())
	statusStr, _ := health["status"].(string)
	latency, _ := health["response_time_ms"].(float64)
	lastCheck, _ := health["last_check"].(float64)
	writeJSON(w, http.StatusOK, models.DaemonHealthResponse{
		Status:         statusStr,
		ResponseTimeMs: latency,
		LastCheck:      lastCheck,
	})
}

func (s *Server) handleRawBisyncLog(w http.ResponseWriter, r *http.Request) {
	if _, err := os.Stat(s.Config.LogFile); os.IsNotExist(err) {
		writeJSON(w, http.StatusOK, models.ErrorResponse{Status: "error", Error: "Log file not found"})
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Disposition", "attachment; filename=bisync.log")
	http.ServeFile(w, r, s.Config.LogFile)
}

func (s *Server) handleRawDaemonLog(w http.ResponseWriter, r *http.Request) {
	daemonLog := filepath.Join(s.Config.BaseDir, "rclone-daemon.log")
	if _, err := os.Stat(daemonLog); os.IsNotExist(err) {
		writeJSON(w, http.StatusOK, models.ErrorResponse{Status: "error", Error: "Daemon log file not found"})
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
	isSyncing := false
	if s.SyncService != nil {
		isSyncing = s.SyncService.IsSyncing()
	}
	if !isSyncing && s.ProfileService != nil {
		isSyncing = s.ProfileService.IsAnySyncing()
	}
	payload := s.TelemetryService.GetTelemetryPayload(r.Context(), isSyncing, limit)
	writeJSON(w, http.StatusOK, payload)
}
