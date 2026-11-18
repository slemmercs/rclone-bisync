package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/models"
)

// Config type aliases pointing to models for backward compatibility.
type ConfigDaemonSettings = models.ConfigDaemonSettings
type ConfigLoggingSettings = models.ConfigLoggingSettings
type ConfigCleanupSettings = models.ConfigCleanupSettings
type ConfigTelemetrySettings = models.ConfigTelemetrySettings
type ConfigFeatureSettings = models.ConfigFeatureSettings
type ConfigNotificationSettings = models.ConfigNotificationSettings
type GlobalConfigResponse = models.GlobalConfigResponse
type UpdateConfigRequest = models.UpdateConfigRequest

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	globals := s.getGlobalDefaults()

	daemonSettings := ConfigDaemonSettings{
		JobPollInitialSleepSec:     2,
		JobPollMaxSleepSec:         10,
		JobPollBackoffMultiplier:   1.5,
		JobTimeoutSeconds:          1800,
		LockStaleTimeoutSeconds:    3600,
		RateLimitBackoffHours:      1,
		RateLimitRetryAfterBackoff: true,
		CheckStaleLocks:            true,
	}

	loggingSettings := ConfigLoggingSettings{
		MaxLogSizeMB:   5,
		LogBackupCount: 5,
	}

	cleanupSettings := ConfigCleanupSettings{}

	telemetrySettings := ConfigTelemetrySettings{
		EnableMetrics:     true,
		MetricsMaxEntries: 1000,
		MetricsMaxBackups: 5,
	}

	notifSettings := ConfigNotificationSettings{
		AppName:            "Rclone Bisync",
		SuccessIntervalMin: 30,
		ErrorRepeatHours:   6,
		WebhookTimeoutSec:  5,
		WatchdogTimeoutSec: 5,
	}

	featureSettings := ConfigFeatureSettings{
		DebugMode:          false,
		MCPReadOnly:        false,
		StaleLockDetection: true,
		RateLimitBackoff:   true,
		DaemonHealthCheck:  true,
		MetricsCollection:  true,
		SmartNotifications: true,
		WebhookAlerts:      false,
	}

	if s.Config != nil {
		globals.DefaultLocalPath = s.Config.LocalPath
		globals.DefaultRemotePath = s.Config.RemotePath
		if globals.CustomFlags == "" && len(s.Config.RcloneCommonArgs) > 0 {
			globals.CustomFlags = strings.Join(s.Config.RcloneCommonArgs, " ")
		}

		if s.Config.JobPollInitialSleepSec > 0 {
			daemonSettings.JobPollInitialSleepSec = s.Config.JobPollInitialSleepSec
		}
		if s.Config.JobPollMaxSleepSec > 0 {
			daemonSettings.JobPollMaxSleepSec = s.Config.JobPollMaxSleepSec
		}
		if s.Config.JobPollBackoffMultiplier > 0 {
			daemonSettings.JobPollBackoffMultiplier = s.Config.JobPollBackoffMultiplier
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
		daemonSettings.RateLimitRetryAfterBackoff = s.Config.RateLimitRetryAfterBackoff
		daemonSettings.CheckStaleLocks = s.Config.CheckStaleLocks

		if s.Config.MaxLogSizeMB > 0 {
			loggingSettings.MaxLogSizeMB = s.Config.MaxLogSizeMB
		}
		if s.Config.LogBackupCount > 0 {
			loggingSettings.LogBackupCount = s.Config.LogBackupCount
		}

		cleanupSettings.FilesToCleanup = strings.Join(s.Config.FilesToCleanup, ", ")

		telemetrySettings.EnableMetrics = s.Config.EnableMetrics
		if s.Config.MetricsMaxEntries > 0 {
			telemetrySettings.MetricsMaxEntries = s.Config.MetricsMaxEntries
		}
		if s.Config.MetricsMaxBackups > 0 {
			telemetrySettings.MetricsMaxBackups = s.Config.MetricsMaxBackups
		}

		if s.Config.NotificationAppName != "" {
			notifSettings.AppName = s.Config.NotificationAppName
		}
		if s.Config.SuccessNotificationIntervalMin > 0 {
			notifSettings.SuccessIntervalMin = s.Config.SuccessNotificationIntervalMin
		}
		if s.Config.ErrorNotificationRepeatHours > 0 {
			notifSettings.ErrorRepeatHours = s.Config.ErrorNotificationRepeatHours
		}
		notifSettings.WebhookURL = s.Config.WebhookURL
		if s.Config.WebhookTimeoutSec > 0 {
			notifSettings.WebhookTimeoutSec = s.Config.WebhookTimeoutSec
		}
		notifSettings.WatchdogPingURL = s.Config.WatchdogPingURL
		if s.Config.WatchdogTimeoutSec > 0 {
			notifSettings.WatchdogTimeoutSec = s.Config.WatchdogTimeoutSec
		}

		featureSettings.DebugMode = s.Config.DebugMode
		featureSettings.MCPReadOnly = s.Config.MCPReadOnly
		if s.Config.Features != nil {
			if v, ok := s.Config.Features["stale_lock_detection"]; ok {
				featureSettings.StaleLockDetection = v
			}
			if v, ok := s.Config.Features["rate_limit_backoff"]; ok {
				featureSettings.RateLimitBackoff = v
			}
			if v, ok := s.Config.Features["daemon_health_check"]; ok {
				featureSettings.DaemonHealthCheck = v
			}
			if v, ok := s.Config.Features["metrics_collection"]; ok {
				featureSettings.MetricsCollection = v
			}
			if v, ok := s.Config.Features["smart_notifications"]; ok {
				featureSettings.SmartNotifications = v
			}
			if v, ok := s.Config.Features["webhook_alerts"]; ok {
				featureSettings.WebhookAlerts = v
			}
		}
	}

	writeJSON(w, http.StatusOK, GlobalConfigResponse{
		Status:         "success",
		GlobalDefaults: globals,
		Daemon:         daemonSettings,
		Logging:        loggingSettings,
		Cleanup:        cleanupSettings,
		Telemetry:      telemetrySettings,
		Notifications:  notifSettings,
		Features:       featureSettings,
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

	envUpdates := make(map[string]string)

	if req.GlobalDefaults != nil {
		if err := s.applyGlobalDefaultsUpdates(req.GlobalDefaults, envUpdates); err != nil {
			writeJSON(w, http.StatusBadRequest, models.ErrorResponse{
				Status: "error",
				Error:  err.Error(),
			})
			return
		}
	}

	s.applySubsystemConfigUpdates(req, envUpdates)

	envPath := s.Config.EnvFilePath
	if envPath == "" {
		envPath = "bisync.env"
	}
	if len(envUpdates) > 0 {
		if err := config.UpdateEnvKeys(envPath, envUpdates); err != nil {
			writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{
				Status: "error",
				Error:  fmt.Sprintf("Failed to persist configuration to %s: %v", envPath, err),
			})
			return
		}
	}

	s.handleGetConfig(w, r)
}

func (s *Server) applyGlobalDefaultsUpdates(gd *models.GlobalDefaults, envUpdates map[string]string) error {
	if gd.MaxDelete <= 0 || gd.MaxDelete > 100 {
		return fmt.Errorf("invalid max_delete percentage: must be between 1 and 100")
	}
	if gd.ConflictResolve != "" {
		validStrategies := map[string]bool{"newer": true, "older": true, "larger": true, "smaller": true, "path1": true, "path2": true, "none": true}
		if !validStrategies[strings.ToLower(gd.ConflictResolve)] {
			return fmt.Errorf("invalid conflict_resolve strategy: %s", gd.ConflictResolve)
		}
		envUpdates["GLOBAL_CONFLICT_RESOLVE"] = gd.ConflictResolve
	}
	if gd.CompareMode != "" {
		validModes := map[string]bool{
			"modtime":               true,
			"size,modtime":          true,
			"checksum":              true,
			"size,modtime,checksum": true,
		}
		if !validModes[strings.ToLower(gd.CompareMode)] {
			return fmt.Errorf("invalid compare_mode: %s", gd.CompareMode)
		}
		envUpdates["GLOBAL_COMPARE_MODE"] = gd.CompareMode
	}

	envUpdates["GLOBAL_MAX_DELETE"] = strconv.Itoa(gd.MaxDelete)
	envUpdates["GLOBAL_CANARY_VALIDATION"] = strconv.FormatBool(gd.CanaryValidation)
	envUpdates["GLOBAL_REMOVE_EMPTY_DIRS"] = strconv.FormatBool(gd.RemoveEmptyDirs)
	envUpdates["GLOBAL_CREATE_EMPTY_DIRS"] = strconv.FormatBool(gd.CreateEmptyDirs)
	envUpdates["GLOBAL_LOCK_EXPLORER_REMOTE"] = strconv.FormatBool(gd.LockExplorerRemote)

	if gd.CustomFlags != "" {
		s.Config.RcloneCommonArgs = strings.Fields(gd.CustomFlags)
		envUpdates["RCLONE_COMMON_ARGS"] = gd.CustomFlags
	}
	if gd.DefaultLocalPath != "" {
		s.Config.LocalPath = gd.DefaultLocalPath
		envUpdates["LOCAL_PATH"] = gd.DefaultLocalPath
	}
	if gd.DefaultRemotePath != "" {
		s.Config.RemotePath = gd.DefaultRemotePath
		envUpdates["REMOTE_PATH"] = gd.DefaultRemotePath
	}
	s.Config.GlobalDefaults = *gd
	return nil
}

func (s *Server) applySubsystemConfigUpdates(req UpdateConfigRequest, envUpdates map[string]string) {
	if req.Daemon != nil {
		d := req.Daemon
		if d.JobTimeoutSeconds > 0 {
			s.Config.JobTimeoutSeconds = d.JobTimeoutSeconds
			envUpdates["JOB_TIMEOUT_SECONDS"] = strconv.Itoa(d.JobTimeoutSeconds)
		}
		if d.LockStaleTimeoutSeconds > 0 {
			s.Config.LockStaleTimeoutSeconds = d.LockStaleTimeoutSeconds
			envUpdates["LOCK_STALE_TIMEOUT_SECONDS"] = strconv.Itoa(d.LockStaleTimeoutSeconds)
		}
		if d.JobPollInitialSleepSec > 0 {
			s.Config.JobPollInitialSleepSec = d.JobPollInitialSleepSec
			envUpdates["JOB_POLL_INITIAL_SLEEP"] = strconv.FormatFloat(d.JobPollInitialSleepSec, 'f', -1, 64)
		}
		if d.JobPollMaxSleepSec > 0 {
			s.Config.JobPollMaxSleepSec = d.JobPollMaxSleepSec
			envUpdates["JOB_POLL_MAX_SLEEP"] = strconv.FormatFloat(d.JobPollMaxSleepSec, 'f', -1, 64)
		}
		if d.JobPollBackoffMultiplier > 0 {
			s.Config.JobPollBackoffMultiplier = d.JobPollBackoffMultiplier
			envUpdates["JOB_POLL_BACKOFF_MULTIPLIER"] = strconv.FormatFloat(d.JobPollBackoffMultiplier, 'f', -1, 64)
		}
		if d.RateLimitBackoffHours > 0 {
			s.Config.RateLimitBackoffHours = d.RateLimitBackoffHours
			envUpdates["RATE_LIMIT_BACKOFF_HOURS"] = strconv.FormatFloat(d.RateLimitBackoffHours, 'f', -1, 64)
		}
		s.Config.CheckStaleLocks = d.CheckStaleLocks
		envUpdates["CHECK_STALE_LOCKS"] = strconv.FormatBool(d.CheckStaleLocks)
		s.Config.RateLimitRetryAfterBackoff = d.RateLimitRetryAfterBackoff
		envUpdates["RATE_LIMIT_RETRY_AFTER_BACKOFF"] = strconv.FormatBool(d.RateLimitRetryAfterBackoff)
	}

	if req.Logging != nil {
		if req.Logging.MaxLogSizeMB > 0 {
			s.Config.MaxLogSizeMB = req.Logging.MaxLogSizeMB
			envUpdates["MAX_LOG_SIZE_MB"] = strconv.Itoa(req.Logging.MaxLogSizeMB)
		}
		if req.Logging.LogBackupCount > 0 {
			s.Config.LogBackupCount = req.Logging.LogBackupCount
			envUpdates["LOG_BACKUP_COUNT"] = strconv.Itoa(req.Logging.LogBackupCount)
		}
	}

	if req.Cleanup != nil {
		raw := strings.TrimSpace(req.Cleanup.FilesToCleanup)
		var cleaned []string
		if raw != "" {
			for _, item := range strings.Split(raw, ",") {
				if tr := strings.TrimSpace(item); tr != "" {
					cleaned = append(cleaned, tr)
				}
			}
		}
		s.Config.FilesToCleanup = cleaned
		envUpdates["FILES_TO_CLEANUP"] = strings.Join(cleaned, ", ")
	}

	if req.Telemetry != nil {
		s.Config.EnableMetrics = req.Telemetry.EnableMetrics
		envUpdates["ENABLE_METRICS"] = strconv.FormatBool(req.Telemetry.EnableMetrics)
		if req.Telemetry.MetricsMaxEntries > 0 {
			s.Config.MetricsMaxEntries = req.Telemetry.MetricsMaxEntries
			envUpdates["METRICS_MAX_ENTRIES"] = strconv.Itoa(req.Telemetry.MetricsMaxEntries)
		}
		if req.Telemetry.MetricsMaxBackups > 0 {
			s.Config.MetricsMaxBackups = req.Telemetry.MetricsMaxBackups
			envUpdates["METRICS_MAX_BACKUPS"] = strconv.Itoa(req.Telemetry.MetricsMaxBackups)
		}
	}

	if req.Notifications != nil {
		n := req.Notifications
		if n.AppName != "" {
			s.Config.NotificationAppName = n.AppName
			envUpdates["NOTIFICATION_APP_NAME"] = n.AppName
		}
		if n.SuccessIntervalMin > 0 {
			s.Config.SuccessNotificationIntervalMin = n.SuccessIntervalMin
			envUpdates["SUCCESS_NOTIFICATION_INTERVAL_MIN"] = strconv.FormatFloat(n.SuccessIntervalMin, 'f', -1, 64)
		}
		if n.ErrorRepeatHours > 0 {
			s.Config.ErrorNotificationRepeatHours = n.ErrorRepeatHours
			envUpdates["ERROR_NOTIFICATION_REPEAT_HOURS"] = strconv.FormatFloat(n.ErrorRepeatHours, 'f', -1, 64)
		}
		if n.WebhookURL != "" && n.WebhookURL != "********" {
			s.Config.WebhookURL = strings.TrimSpace(n.WebhookURL)
			envUpdates["WEBHOOK_URL"] = s.Config.WebhookURL
		}
		if n.WatchdogPingURL != "" {
			s.Config.WatchdogPingURL = strings.TrimSpace(n.WatchdogPingURL)
			envUpdates["WATCHDOG_PING_URL"] = s.Config.WatchdogPingURL
		}
		if n.WebhookTimeoutSec > 0 {
			s.Config.WebhookTimeoutSec = n.WebhookTimeoutSec
			envUpdates["WEBHOOK_TIMEOUT"] = strconv.Itoa(n.WebhookTimeoutSec)
		}
		if n.WatchdogTimeoutSec > 0 {
			s.Config.WatchdogTimeoutSec = n.WatchdogTimeoutSec
			envUpdates["WATCHDOG_TIMEOUT"] = strconv.Itoa(n.WatchdogTimeoutSec)
		}
	}

	if req.Features != nil {
		f := req.Features
		s.Config.DebugMode = f.DebugMode
		envUpdates["DEBUG_MODE"] = strconv.FormatBool(f.DebugMode)
		s.Config.MCPReadOnly = f.MCPReadOnly
		envUpdates["MCP_READ_ONLY"] = strconv.FormatBool(f.MCPReadOnly)

		if s.Config.Features == nil {
			s.Config.Features = make(map[string]bool)
		}
		s.Config.Features["stale_lock_detection"] = f.StaleLockDetection
		envUpdates["FEATURE_STALE_LOCK_DETECTION"] = strconv.FormatBool(f.StaleLockDetection)
		s.Config.Features["rate_limit_backoff"] = f.RateLimitBackoff
		envUpdates["FEATURE_RATE_LIMIT_BACKOFF"] = strconv.FormatBool(f.RateLimitBackoff)
		s.Config.Features["daemon_health_check"] = f.DaemonHealthCheck
		envUpdates["FEATURE_DAEMON_HEALTH_CHECK"] = strconv.FormatBool(f.DaemonHealthCheck)
		s.Config.Features["metrics_collection"] = f.MetricsCollection
		envUpdates["FEATURE_METRICS_COLLECTION"] = strconv.FormatBool(f.MetricsCollection)
		s.Config.Features["smart_notifications"] = f.SmartNotifications
		envUpdates["FEATURE_SMART_NOTIFICATIONS"] = strconv.FormatBool(f.SmartNotifications)
		s.Config.Features["webhook_alerts"] = f.WebhookAlerts
		envUpdates["FEATURE_WEBHOOK_ALERTS"] = strconv.FormatBool(f.WebhookAlerts)
	}
}
