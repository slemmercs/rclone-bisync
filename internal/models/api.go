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
	MaxDelete          int    `json:"max_delete"`
	CanaryValidation   bool   `json:"canary_validation"`
	ConflictResolve    string `json:"conflict_resolve"`
	CompareMode        string `json:"compare_mode"`
	RemoveEmptyDirs    bool   `json:"remove_empty_dirs"`
	CreateEmptyDirs    bool   `json:"create_empty_dirs"`
	LockExplorerRemote bool   `json:"lock_explorer_remote"`
}

// ConfigDaemonSettings represents daemon scheduling, timeout, and backoff settings.
type ConfigDaemonSettings struct {
	JobPollInitialSleepSec  float64 `json:"job_poll_initial_sleep_sec"`
	JobPollMaxSleepSec      float64 `json:"job_poll_max_sleep_sec"`
	JobTimeoutSeconds       int     `json:"job_timeout_seconds"`
	LockStaleTimeoutSeconds int     `json:"lock_stale_timeout_seconds"`
	RateLimitBackoffHours   float64 `json:"rate_limit_backoff_hours"`
}

// ConfigNotificationSettings represents external alerting webhook endpoints.
type ConfigNotificationSettings struct {
	WebhookURL      string `json:"webhook_url"`
	WatchdogPingURL string `json:"watchdog_ping_url"`
}

// GlobalConfigResponse is the canonical response for GET/PUT /api/config.
type GlobalConfigResponse struct {
	Status         string                     `json:"status"`
	GlobalDefaults GlobalDefaults             `json:"global_defaults"`
	Daemon         ConfigDaemonSettings       `json:"daemon"`
	Notifications  ConfigNotificationSettings `json:"notifications"`
}

// UpdateConfigRequest represents the payload for PUT /api/config.
type UpdateConfigRequest struct {
	GlobalDefaults *GlobalDefaults             `json:"global_defaults,omitempty"`
	Daemon         *ConfigDaemonSettings       `json:"daemon,omitempty"`
	Notifications  *ConfigNotificationSettings `json:"notifications,omitempty"`
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
