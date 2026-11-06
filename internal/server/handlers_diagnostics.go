package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/models"
)

// handleDiagnostics serves GET /api/diagnostics, evaluating live health checks across daemon, storage, remotes, safety, and system runtime.
func (s *Server) handleDiagnostics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var checks []models.DiagnosticCheck

	// 1. Daemon Health Check
	checks = append(checks, s.checkDaemonHealth(ctx))

	// 2. Storage & Base Directory Check
	checks = append(checks, s.checkStorageAccess()...)

	// 3. Canary & Safety Protections Check
	checks = append(checks, s.checkSafetyProtection(ctx))

	// 4. Remote Authentication Check
	checks = append(checks, s.checkRemotesHealth(ctx))

	// 5. Host Runtime & Environment Check
	checks = append(checks, s.checkSystemRuntime())

	// Summarize counts
	summary := map[string]int{
		"pass":    0,
		"warning": 0,
		"fail":    0,
	}
	for _, c := range checks {
		switch c.Status {
		case models.DiagnosticStatusPass:
			summary["pass"]++
		case models.DiagnosticStatusWarning:
			summary["warning"]++
		case models.DiagnosticStatusFail:
			summary["fail"]++
		}
	}

	overallStatus := "ok"
	if summary["fail"] > 0 {
		overallStatus = "error"
	} else if summary["warning"] > 0 {
		overallStatus = "warning"
	}

	uptimeSec := int64(0)
	if !s.StartTime.IsZero() {
		uptimeSec = int64(time.Since(s.StartTime).Seconds())
	}

	baseDir := ""
	logPath := ""
	if s.Config != nil {
		baseDir = s.Config.BaseDir
		logPath = s.Config.LogFile
	}

	isSystemd := os.Getenv("INVOCATION_ID") != "" || os.Getenv("JOURNAL_STREAM") != ""

	resp := models.DiagnosticsResponse{
		Status:  overallStatus,
		Summary: summary,
		Checks:  checks,
		SystemInfo: models.DiagnosticSystemInfo{
			Version:     s.Version,
			GoVersion:   runtime.Version(),
			OS:          runtime.GOOS,
			Arch:        runtime.GOARCH,
			PID:         os.Getpid(),
			UptimeSec:   uptimeSec,
			BaseDir:     baseDir,
			LogFilePath: logPath,
			IsSystemd:   isSystemd,
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) checkDaemonHealth(ctx context.Context) models.DiagnosticCheck {
	check := models.DiagnosticCheck{
		ID:       "daemon_connectivity",
		Category: "daemon",
		Name:     "Rclone RC Daemon",
	}

	if s.TelemetryService == nil {
		check.Status = models.DiagnosticStatusFail
		check.Message = "Telemetry service uninitialized"
		check.Detail = "Cannot probe Rclone RC daemon"
		return check
	}

	health := s.TelemetryService.GetDaemonHealth(ctx)
	statusStr, _ := health["status"].(string)
	latency, _ := health["response_time_ms"].(float64)

	switch statusStr {
	case "healthy":
		if latency > 500.0 {
			check.Status = models.DiagnosticStatusWarning
			check.Message = "Daemon responding with high latency"
			check.Detail = fmt.Sprintf("Response time: %.1f ms (>500ms)", latency)
		} else {
			check.Status = models.DiagnosticStatusPass
			check.Message = "Daemon is active and responsive"
			check.Detail = fmt.Sprintf("Response time: %.1f ms", latency)
		}
	case "reconnecting":
		check.Status = models.DiagnosticStatusWarning
		check.Message = "Daemon is reconnecting"
		check.Detail = "Connection dropped; retry grace period in progress"
		check.ActionTitle = "View Daemon Logs"
		check.ActionURL = "/api/logs/daemon"
	default:
		check.Status = models.DiagnosticStatusFail
		check.Message = "Daemon is offline or unreachable"
		check.Detail = "Verify Rclone RC service is running and RC API credentials are valid"
		check.ActionTitle = "View Daemon Logs"
		check.ActionURL = "/api/logs/daemon"
	}

	return check
}

func (s *Server) checkStorageAccess() []models.DiagnosticCheck {
	var checks []models.DiagnosticCheck

	baseDir := ""
	if s.Config != nil {
		baseDir = s.Config.BaseDir
	}
	if baseDir == "" {
		home, _ := os.UserHomeDir()
		baseDir = filepath.Join(home, ".config", "rclone-bisync")
	}

	baseCheck := models.DiagnosticCheck{
		ID:       "storage_base_dir",
		Category: "storage",
		Name:     "Config & State Storage Directory",
		Detail:   baseDir,
	}

	if info, err := os.Stat(baseDir); err != nil {
		if os.IsNotExist(err) {
			baseCheck.Status = models.DiagnosticStatusWarning
			baseCheck.Message = "Storage directory does not exist yet"
		} else {
			baseCheck.Status = models.DiagnosticStatusFail
			baseCheck.Message = fmt.Sprintf("Cannot access storage directory: %v", err)
		}
	} else if !info.IsDir() {
		baseCheck.Status = models.DiagnosticStatusFail
		baseCheck.Message = "Config path exists but is not a directory"
	} else {
		testFile := filepath.Join(baseDir, fmt.Sprintf(".diag_probe_%d.tmp", time.Now().UnixNano()))
		if err := os.WriteFile(testFile, []byte("diag"), 0600); err != nil {
			baseCheck.Status = models.DiagnosticStatusFail
			baseCheck.Message = fmt.Sprintf("Storage directory is not writable: %v", err)
		} else {
			_ = os.Remove(testFile)
			baseCheck.Status = models.DiagnosticStatusPass
			baseCheck.Message = "Storage directory is accessible and writable"
		}
	}
	checks = append(checks, baseCheck)

	logPath := ""
	if s.Config != nil {
		logPath = s.Config.LogFile
	}
	if logPath != "" {
		logCheck := models.DiagnosticCheck{
			ID:       "storage_log_file",
			Category: "storage",
			Name:     "Bisync Log Stream File",
			Detail:   logPath,
		}
		if fi, err := os.Stat(logPath); err != nil {
			if os.IsNotExist(err) {
				logCheck.Status = models.DiagnosticStatusPass
				logCheck.Message = "Log file will be created on first sync run"
			} else {
				logCheck.Status = models.DiagnosticStatusWarning
				logCheck.Message = fmt.Sprintf("Cannot stat log file: %v", err)
			}
		} else {
			sizeMB := float64(fi.Size()) / (1024 * 1024)
			logCheck.Status = models.DiagnosticStatusPass
			logCheck.Message = fmt.Sprintf("Log file active (%.2f MB)", sizeMB)
		}
		checks = append(checks, logCheck)
	}

	return checks
}

func (s *Server) checkSafetyProtection(_ context.Context) models.DiagnosticCheck {
	check := models.DiagnosticCheck{
		ID:       "safety_canary_guard",
		Category: "safety",
		Name:     "Safety Canary Protection",
	}

	if s.CanaryService == nil {
		check.Status = models.DiagnosticStatusWarning
		check.Message = "Canary service uninitialized"
		return check
	}

	err := s.CanaryService.SafeguardCanary("RCLONE_TEST")
	if err == nil {
		check.Status = models.DiagnosticStatusFail
		check.Message = "Canary safeguard is improperly configured (allowed RCLONE_TEST deletion)"
		check.Detail = "Critical circuit breaker failure"
		return check
	}

	check.Status = models.DiagnosticStatusPass
	check.Message = "Canary safeguards active (RCLONE_TEST deletion protection enforced)"
	check.Detail = "Circuit breakers operational against accidental bulk wiping"
	return check
}


func (s *Server) checkRemotesHealth(ctx context.Context) models.DiagnosticCheck {
	check := models.DiagnosticCheck{
		ID:       "remotes_auth_health",
		Category: "remotes",
		Name:     "Remote Authentication & Token Health",
	}

	if s.AuthService != nil {
		status, err := s.AuthService.GetAuthStatusCached(ctx, false)
		if err != nil {
			check.Status = models.DiagnosticStatusWarning
			check.Message = fmt.Sprintf("Unable to probe remotes auth status: %v", err)
			check.ActionTitle = "View Remotes"
			check.ActionURL = "/#remotes"
			return check
		}
		if status.Summary.TotalRemotes == 0 {
			check.Status = models.DiagnosticStatusWarning
			check.Message = "No remotes configured"
			check.Detail = "Add remote storage endpoints to enable synchronization"
			check.ActionTitle = "Configure Remotes"
			check.ActionURL = "/#remotes"
			return check
		}
		actionNeeded := status.Summary.AuthorizationRequired + status.Summary.ConfigurationRequired + status.Summary.MigrationRequired + status.Summary.Errors
		if actionNeeded > 0 {
			check.Status = models.DiagnosticStatusWarning
			check.Message = fmt.Sprintf("%d remote(s) require re-authentication or configuration", actionNeeded)
			check.Detail = fmt.Sprintf("Connected: %d, Action Required: %d", status.Summary.Connected, actionNeeded)
			check.ActionTitle = "Review Remotes"
			check.ActionURL = "/#remotes"
			return check
		}

		check.Status = models.DiagnosticStatusPass
		check.Message = fmt.Sprintf("All %d configured remote(s) authenticated", status.Summary.TotalRemotes)
		check.Detail = fmt.Sprintf("%d active / connected remotes", status.Summary.Connected)
		return check
	}

	if s.TelemetryService != nil {
		remotesSummary, err := s.TelemetryService.GetRemotesSummaryCached(ctx, false)
		if err != nil {
			check.Status = models.DiagnosticStatusWarning
			check.Message = fmt.Sprintf("Failed to query remotes: %v", err)
			return check
		}
		if remotesSummary.Count == 0 {
			check.Status = models.DiagnosticStatusWarning
			check.Message = "No remotes configured in Rclone"
			return check
		}
		check.Status = models.DiagnosticStatusPass
		check.Message = fmt.Sprintf("%d remote(s) discovered in Rclone configuration", remotesSummary.Count)
		return check
	}

	check.Status = models.DiagnosticStatusWarning
	check.Message = "Remote probing services unavailable"
	return check
}

func (s *Server) checkSystemRuntime() models.DiagnosticCheck {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	allocMB := float64(mem.Alloc) / (1024 * 1024)

	return models.DiagnosticCheck{
		ID:       "system_runtime",
		Category: "system",
		Name:     "Host Runtime & Memory Health",
		Status:   models.DiagnosticStatusPass,
		Message:  fmt.Sprintf("Hub process PID %d healthy (%.1f MB allocated, %d goroutines)", os.Getpid(), allocMB, runtime.NumGoroutine()),
		Detail:   fmt.Sprintf("OS: %s/%s, Go: %s", runtime.GOOS, runtime.GOARCH, runtime.Version()),
	}
}

