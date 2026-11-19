package models

type ActionRequest struct {
	Action    string `json:"action"`
	ProfileID string `json:"profile_id,omitempty"`
}

type ActionResponse struct {
	Status string `json:"status"`
	Msg    string `json:"msg,omitempty"`
	Error  string `json:"error,omitempty"`
}

type RemotesResponse struct {
	Remotes      []map[string]interface{} `json:"remotes"`
	Count        int                      `json:"count"`
	ActiveRemote interface{}              `json:"active_remote"`
	Summary      map[string]interface{}   `json:"summary"`
	Error        string                   `json:"error,omitempty"`
}

type DaemonHealthResponse struct {
	Status         string  `json:"status"`
	ResponseTimeMs float64 `json:"response_time_ms"`
	LastCheck      float64 `json:"last_check"`
	Error          string  `json:"error,omitempty"`
}

type UpdateCredentialsResponse struct {
	Status     string            `json:"status"`
	Remote     string            `json:"remote,omitempty"`
	Msg        string            `json:"msg,omitempty"`
	Validation *ValidateResponse `json:"validation,omitempty"`
	Error      string            `json:"error,omitempty"`
}

type CancelReconnectResponse struct {
	Status    string `json:"status"`
	Remote    string `json:"remote"`
	Cancelled bool   `json:"cancelled"`
	Msg       string `json:"msg"`
}

type OAuthStopResponse struct {
	Status string `json:"status"`
	Msg    string `json:"msg"`
	Error  string `json:"error,omitempty"`
}

type ErrorResponse struct {
	Status string `json:"status"`
	Msg    string `json:"msg,omitempty"`
	Error  string `json:"error,omitempty"`
}

// NewErrorResponse creates a uniform ErrorResponse envelope ensuring both Status, Msg, and Error fields are set.
func NewErrorResponse(msg string) ErrorResponse {
	return ErrorResponse{
		Status: "error",
		Msg:    msg,
		Error:  msg,
	}
}

type TransferStats struct {
	Bytes        int64                    `json:"bytes"`
	TotalBytes   int64                    `json:"total_bytes"`
	Speed        float64                  `json:"speed"`
	SpeedStr     string                   `json:"speed_str"`
	ETA          *int                     `json:"eta,omitempty"`
	Percentage   float64                  `json:"percentage"`
	Transferring []map[string]interface{} `json:"transferring"`
}

type TelemetryPayload struct {
	DaemonHealth      interface{}              `json:"daemon_health"`
	NotificationState map[string]interface{}   `json:"notification_state"`
	Metrics           []map[string]interface{} `json:"metrics"`
	BisyncLogs        []string                 `json:"bisync_logs"`
	DaemonLogs        []string                 `json:"daemon_logs"`
	Config            map[string]interface{}   `json:"config"`
	IsSyncing         bool                     `json:"is_syncing"`
	ExecutionStatus   string                   `json:"execution_status"`
	TransferStats     interface{}              `json:"transfer_stats"`
	HasReview         bool                     `json:"has_review"`
	ReviewPending     bool                     `json:"review_pending"`
	MCPStatus         *MCPStatus               `json:"mcp_status,omitempty"`
}

// TelemetryResponse is the canonical API response for /api/telemetry.
type TelemetryResponse = TelemetryPayload

// GlobalDefaults represents baseline operational settings applied across all profiles.
type GlobalDefaults struct {
	MaxDelete          int    `json:"max_delete"`                    // Safety deletion threshold percentage (0–100%). Aborts sync if pending deletions exceed this ratio.
	CanaryValidation   bool   `json:"canary_validation"`             // Verifies presence of RCLONE_TEST canary token files before proceeding.
	ConflictResolve    string `json:"conflict_resolve"`              // Heuristic resolution mode: path1, path2, newer, older, larger, smaller.
	CompareMode        string `json:"compare_mode"`                  // Divergence detection: size,modtime, size,modtime,checksum, checksum, modtime.
	RemoveEmptyDirs    bool   `json:"remove_empty_dirs"`             // Cleans empty directories on target after file deletions.
	CreateEmptyDirs    bool   `json:"create_empty_dirs"`             // Replicates empty directory trees between replicas.
	LockExplorerRemote bool   `json:"lock_explorer_remote"`          // Restricts browser file explorer to profile root directory paths.
	CustomFlags        string `json:"custom_flags,omitempty"`        // Raw CLI flags appended to Rclone bisync subprocess executions.
	DefaultLocalPath   string `json:"default_local_path,omitempty"`  // Suggested default base directory when creating local replica endpoints.
	DefaultRemotePath  string `json:"default_remote_path,omitempty"` // Suggested default remote bucket or folder for new profiles.
}

// ConfigDaemonSettings represents daemon scheduling, timeout, and backoff settings.
type ConfigDaemonSettings struct {
	JobPollInitialSleepSec     float64 `json:"job_poll_initial_sleep_sec"`
	JobPollMaxSleepSec         float64 `json:"job_poll_max_sleep_sec"`
	JobPollBackoffMultiplier   float64 `json:"job_poll_backoff_multiplier,omitempty"`
	JobTimeoutSeconds          int     `json:"job_timeout_seconds"`
	LockStaleTimeoutSeconds    int     `json:"lock_stale_timeout_seconds"`
	RateLimitBackoffHours      float64 `json:"rate_limit_backoff_hours"`
	RateLimitRetryAfterBackoff bool    `json:"rate_limit_retry_after_backoff,omitempty"`
	CheckStaleLocks            bool    `json:"check_stale_locks,omitempty"`
}

// ConfigLoggingSettings represents log file size and backup retention.
type ConfigLoggingSettings struct {
	MaxLogSizeMB   int `json:"max_log_size_mb"`
	LogBackupCount int `json:"log_backup_count"`
}

// ConfigCleanupSettings represents post-sync file cleanup globs.
type ConfigCleanupSettings struct {
	FilesToCleanup string `json:"files_to_cleanup"`
}

// ConfigTelemetrySettings represents telemetry and metrics collection settings.
type ConfigTelemetrySettings struct {
	EnableMetrics     bool `json:"enable_metrics"`
	MetricsMaxEntries int  `json:"metrics_max_entries"`
	MetricsMaxBackups int  `json:"metrics_max_backups"`
}

// ConfigFeatureSettings represents subsystem feature flags and debug toggles.
type ConfigFeatureSettings struct {
	DebugMode          bool `json:"debug_mode"`
	MCPReadOnly        bool `json:"mcp_read_only"`
	StaleLockDetection bool `json:"stale_lock_detection"`
	RateLimitBackoff   bool `json:"rate_limit_backoff"`
	DaemonHealthCheck  bool `json:"daemon_health_check"`
	MetricsCollection  bool `json:"metrics_collection"`
	SmartNotifications bool `json:"smart_notifications"`
	WebhookAlerts      bool `json:"webhook_alerts"`
}

// ConfigNotificationSettings represents external alerting webhook endpoints and throttling.
type ConfigNotificationSettings struct {
	AppName            string  `json:"app_name,omitempty"`
	SuccessIntervalMin float64 `json:"success_interval_min,omitempty"`
	ErrorRepeatHours   float64 `json:"error_repeat_hours,omitempty"`
	WebhookURL         string  `json:"webhook_url"`
	WebhookTimeoutSec  int     `json:"webhook_timeout_sec,omitempty"`
	WatchdogPingURL    string  `json:"watchdog_ping_url"`
	WatchdogTimeoutSec int     `json:"watchdog_timeout_sec,omitempty"`
}

// GlobalConfigResponse is the canonical response for GET/PUT /api/config.
type GlobalConfigResponse struct {
	Status         string                     `json:"status"`
	GlobalDefaults GlobalDefaults             `json:"global_defaults"`
	Daemon         ConfigDaemonSettings       `json:"daemon"`
	Logging        ConfigLoggingSettings      `json:"logging"`
	Cleanup        ConfigCleanupSettings      `json:"cleanup"`
	Telemetry      ConfigTelemetrySettings    `json:"telemetry"`
	Notifications  ConfigNotificationSettings `json:"notifications"`
	Features       ConfigFeatureSettings      `json:"features"`
}

// UpdateConfigRequest represents the payload for PUT /api/config.
type UpdateConfigRequest struct {
	GlobalDefaults *GlobalDefaults             `json:"global_defaults,omitempty"`
	Daemon         *ConfigDaemonSettings       `json:"daemon,omitempty"`
	Logging        *ConfigLoggingSettings      `json:"logging,omitempty"`
	Cleanup        *ConfigCleanupSettings      `json:"cleanup,omitempty"`
	Telemetry      *ConfigTelemetrySettings    `json:"telemetry,omitempty"`
	Notifications  *ConfigNotificationSettings `json:"notifications,omitempty"`
	Features       *ConfigFeatureSettings      `json:"features,omitempty"`
}

// MCPStatus represents the operational status of the Model Context Protocol server.
type MCPStatus struct {
	Status    string `json:"status"`              // "active" or "offline"
	Transport string `json:"transport,omitempty"` // "stdio" or "sse"
	ReadOnly  bool   `json:"read_only"`           // true if mutations are disallowed
	Host      string `json:"host,omitempty"`      // host address if sse transport
	Port      int    `json:"port,omitempty"`      // port if sse transport
	PID       int    `json:"pid,omitempty"`       // process ID
	StartedAt string `json:"started_at,omitempty"`
}

// SyncFileOp represents an individual file operation detected during sync execution.
type SyncFileOp struct {
	Path   string `json:"path"`
	Action string `json:"action"`
	Msg    string `json:"msg"`
}

type BisyncLogEntry struct {
	Time        string  `json:"time"`
	Level       string  `json:"level"`
	Type        string  `json:"type,omitempty"` // "sync" (or omitted)
	ProfileID   string  `json:"profile_id,omitempty"`
	ProfileName string  `json:"profile_name,omitempty"`
	Msg         string  `json:"msg"`
	RunID       string  `json:"run_id"`
	JobID       int     `json:"job_id,omitempty"`
	Origin      string  `json:"origin,omitempty"`
	Status      string  `json:"status,omitempty"`     // SUCCESS, FAULT, ABORTED
	ErrorType   string  `json:"error_type,omitempty"` // TOO_MANY_DELETES, RATE_LIMIT, etc.
	DurationSec float64 `json:"duration_sec,omitempty"`
	FilesSynced int     `json:"files_synced,omitempty"`
	StartTime   string  `json:"start_time,omitempty"` // RFC3339
	EndTime     string  `json:"end_time,omitempty"`   // RFC3339
	Object      string  `json:"object,omitempty"`
	Action      string  `json:"action,omitempty"`
	SeqID       int64   `json:"seq_id,omitempty"`
	Actor       string  `json:"actor,omitempty"`
}
