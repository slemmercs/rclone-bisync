package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/models"
)

type ConfigDaemonSettings struct {
	JobPollInitialSleepSec  float64 `json:"job_poll_initial_sleep_sec"`
	JobPollMaxSleepSec      float64 `json:"job_poll_max_sleep_sec"`
	JobTimeoutSeconds       int     `json:"job_timeout_seconds"`
	LockStaleTimeoutSeconds int     `json:"lock_stale_timeout_seconds"`
	RateLimitBackoffHours   float64 `json:"rate_limit_backoff_hours"`
}

type ConfigNotificationSettings struct {
	WebhookURL      string `json:"webhook_url"`
	WatchdogPingURL string `json:"watchdog_ping_url"`
}

type GlobalConfigResponse struct {
	Status         string                     `json:"status"`
	GlobalDefaults config.GlobalDefaults      `json:"global_defaults"`
	Daemon         ConfigDaemonSettings       `json:"daemon"`
	Notifications  ConfigNotificationSettings `json:"notifications"`
}

type UpdateConfigRequest struct {
	GlobalDefaults *config.GlobalDefaults      `json:"global_defaults,omitempty"`
	Daemon         *ConfigDaemonSettings       `json:"daemon,omitempty"`
	Notifications  *ConfigNotificationSettings `json:"notifications,omitempty"`
}

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	globals := s.getGlobalDefaults()

	daemonSettings := ConfigDaemonSettings{
		JobPollInitialSleepSec:  2,
		JobPollMaxSleepSec:      10,
		JobTimeoutSeconds:       1800,
		LockStaleTimeoutSeconds: 3600,
		RateLimitBackoffHours:   1,
	}

	notifSettings := ConfigNotificationSettings{}

	if s.Config != nil {
		if s.Config.JobPollInitialSleepSec > 0 {
			daemonSettings.JobPollInitialSleepSec = s.Config.JobPollInitialSleepSec
		}
		if s.Config.JobPollMaxSleepSec > 0 {
			daemonSettings.JobPollMaxSleepSec = s.Config.JobPollMaxSleepSec
		}
		if s.Config.JobTimeoutSeconds > 0 {
			daemonSettings.JobTimeoutSeconds = s.Config.JobTimeoutSeconds
		}
		if s.Config.LockStaleTimeoutSeconds > 0 {
			daemonSettings.LockStaleTimeoutSeconds = s.Config.LockStaleTimeoutSeconds
		}
		if s.Config.RateLimitBackoffHours > 0 {
			daemonSettings.RateLimitBackoffHours = s.Config.RateLimitBackoffHours
		}
		notifSettings.WebhookURL = s.Config.WebhookURL
		notifSettings.WatchdogPingURL = s.Config.WatchdogPingURL
	}

	writeJSON(w, http.StatusOK, GlobalConfigResponse{
		Status:         "success",
		GlobalDefaults: globals,
		Daemon:         daemonSettings,
		Notifications:  notifSettings,
	})
}

func (s *Server) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	var req UpdateConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{
			Status: "error",
			Error:  "Malformed request payload: " + err.Error(),
		})
		return
	}

	if s.Config == nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{
			Status: "error",
			Error:  "Server configuration not initialized",
		})
		return
	}

	// Validate & apply Global Defaults
	if req.GlobalDefaults != nil {
		gd := *req.GlobalDefaults
		if gd.MaxDelete <= 0 || gd.MaxDelete > 100 {
			writeJSON(w, http.StatusBadRequest, models.ErrorResponse{
				Status: "error",
				Error:  "Invalid max_delete percentage: must be between 1 and 100",
			})
			return
		}
		if gd.ConflictResolve != "" {
			validStrategies := map[string]bool{"newer": true, "older": true, "larger": true, "smaller": true, "path1": true, "path2": true, "none": true}
			if !validStrategies[strings.ToLower(gd.ConflictResolve)] {
				writeJSON(w, http.StatusBadRequest, models.ErrorResponse{
					Status: "error",
					Error:  "Invalid conflict_resolve strategy: " + gd.ConflictResolve,
				})
				return
			}
		}
		if gd.CompareMode != "" {
			validModes := map[string]bool{
				"modtime":               true,
				"size,modtime":          true,
				"checksum":              true,
				"size,modtime,checksum": true,
			}
			if !validModes[strings.ToLower(gd.CompareMode)] {
				writeJSON(w, http.StatusBadRequest, models.ErrorResponse{
					Status: "error",
					Error:  "Invalid compare_mode: " + gd.CompareMode,
				})
				return
			}
		}
		s.Config.GlobalDefaults = gd
	}

	// Validate & apply Daemon tuning
	if req.Daemon != nil {
		d := req.Daemon
		if d.JobTimeoutSeconds > 0 {
			s.Config.JobTimeoutSeconds = d.JobTimeoutSeconds
		}
		if d.LockStaleTimeoutSeconds > 0 {
			s.Config.LockStaleTimeoutSeconds = d.LockStaleTimeoutSeconds
		}
		if d.JobPollInitialSleepSec > 0 {
			s.Config.JobPollInitialSleepSec = d.JobPollInitialSleepSec
		}
		if d.JobPollMaxSleepSec > 0 {
			s.Config.JobPollMaxSleepSec = d.JobPollMaxSleepSec
		}
		if d.RateLimitBackoffHours > 0 {
			s.Config.RateLimitBackoffHours = d.RateLimitBackoffHours
		}
	}

	// Apply Notifications
	if req.Notifications != nil {
		s.Config.WebhookURL = strings.TrimSpace(req.Notifications.WebhookURL)
		s.Config.WatchdogPingURL = strings.TrimSpace(req.Notifications.WatchdogPingURL)
	}

	// Return updated state
	s.handleGetConfig(w, r)
}
