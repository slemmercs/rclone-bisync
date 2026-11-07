package models

import "encoding/json"

type ActionRequest struct {
	Action    string `json:"action"`
	ProfileID string `json:"profile_id,omitempty"`
}

type ActionResponse struct {
	Status string `json:"status"`
	Msg    string `json:"msg,omitempty"`
	Error  string `json:"error,omitempty"`
}

type BrowseRequest struct {
	Remote    string `json:"remote"`
	Path      string `json:"path"`
	ProfileID string `json:"profile_id,omitempty"`
}

type BrowseItem struct {
	Path     string `json:"Path,omitempty"`
	Name     string `json:"Name"`
	Size     int64  `json:"Size"`
	Mtime    string `json:"Mtime,omitempty"`
	ModTime  string `json:"ModTime,omitempty"`
	MimeType string `json:"MimeType,omitempty"`
	IsDir    bool   `json:"IsDir"`
}

type BrowseResponse struct {
	Remote    string                   `json:"remote"`
	Path      string                   `json:"path"`
	Items     []map[string]interface{} `json:"items"`
	HasCanary bool                     `json:"has_canary"`
	Error     string                   `json:"error,omitempty"`
}

type CanaryResponse struct {
	Status string `json:"status"`
	Remote string `json:"remote,omitempty"`
	Path   string `json:"path,omitempty"`
	Msg    string `json:"msg,omitempty"`
	Error  string `json:"error,omitempty"`
}

type CopyRequest struct {
	SrcRemote string `json:"src_remote"`
	SrcPath   string `json:"src_path"`
	DstRemote string `json:"dst_remote"`
	DstPath   string `json:"dst_path"`
	IsDir     bool   `json:"is_dir"`
	ProfileID string `json:"profile_id,omitempty"`
}

type CopyResponse struct {
	Status string `json:"status"`
	Msg    string `json:"msg,omitempty"`
	Src    string `json:"src,omitempty"`
	Dst    string `json:"dst,omitempty"`
	Error  string `json:"error,omitempty"`
}

// UploadResponse represents the result of a file upload operation.
type UploadResponse struct {
	Status string `json:"status"`
	Msg    string `json:"msg,omitempty"`
	Remote string `json:"remote,omitempty"`
	Path   string `json:"path,omitempty"`
	Size   int64  `json:"size,omitempty"`
	Error  string `json:"error,omitempty"`
}

type DeleteRequest struct {
	Remote    string `json:"remote"`
	Path      string `json:"path"`
	IsDir     bool   `json:"is_dir"`
	ProfileID string `json:"profile_id,omitempty"`
}

type DeleteResponse struct {
	Status string `json:"status"`
	Msg    string `json:"msg,omitempty"`
	Remote string `json:"remote,omitempty"`
	Path   string `json:"path,omitempty"`
	Fs     string `json:"fs,omitempty"`
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

type PendingDeletionsResponse struct {
	HasDeletions bool          `json:"has_deletions"`
	TotalCount   int           `json:"total_count"`
	Source       string        `json:"source,omitempty"`
	TargetAction string        `json:"target_action,omitempty"`
	Message      string        `json:"message,omitempty"`
	Files        []interface{} `json:"files"`
	Error        string        `json:"error,omitempty"`
	AbortedAt    *float64      `json:"aborted_at,omitempty"`
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
	Status      string  `json:"status"`               // SUCCESS, FAULT, ABORTED
	ErrorType   string  `json:"error_type,omitempty"` // TOO_MANY_DELETES, RATE_LIMIT, etc.
	DurationSec float64 `json:"duration_sec"`
	FilesSynced int     `json:"files_synced"`
	StartTime   string  `json:"start_time"` // RFC3339
	EndTime     string  `json:"end_time"`   // RFC3339
}

type PaginatedLogsResponse struct {
	Page       int      `json:"page"`
	Limit      int      `json:"limit"`
	Total      int      `json:"total"`
	TotalPages int      `json:"total_pages"`
	Logs       []string `json:"logs"`
}

// AuditLogRecord represents a cryptographically verified milestone logged in the unified bisync.log stream.

type DedupeRequest struct {
	Remote string `json:"remote"`
	Path   string `json:"path"`
	Mode   string `json:"mode"` // "list", "oldest", "newest"
	ByHash bool   `json:"by_hash"`
}

type DedupeResponse struct {
	Status string `json:"status"`
	RunID  string `json:"run_id,omitempty"`
	Remote string `json:"remote"`
	Path   string `json:"path"`
	Mode   string `json:"mode"`
	ByHash bool   `json:"by_hash,omitempty"`
	Output string `json:"output"`
	Error  string `json:"error,omitempty"`
}

type AuditLogRecord struct {
	Time      string          `json:"time"`
	Level     string          `json:"level"`
	Type      string          `json:"type"` // "audit"
	Msg       string          `json:"msg"`
	Object    string          `json:"object,omitempty"`
	SeqID     int64           `json:"seq_id"`
	ProfileID string          `json:"profile_id"`
	Action    string          `json:"action"`
	Actor     string          `json:"actor"`
	Hash      string          `json:"hash"`
	PrevHash  string          `json:"prev_hash"`
	Details   json.RawMessage `json:"details,omitempty"`
}

type DaemonLogsQuery struct {
	RunID     string `json:"run_id"`
	JobID     int    `json:"job_id"`
	StartTime string `json:"start_time"` // RFC3339
	EndTime   string `json:"end_time"`   // RFC3339
	Limit     int    `json:"limit"`
}

type DaemonLogsResponse struct {
	RunID      string   `json:"run_id,omitempty"`
	JobID      int      `json:"job_id,omitempty"`
	Count      int      `json:"count"`
	Logs       []string `json:"logs"`
	Correlated bool     `json:"correlated"`
	Error      string   `json:"error,omitempty"`
}

// DiagnosticCheckStatus represents the outcome of a diagnostic evaluation.
type DiagnosticCheckStatus string

const (
	DiagnosticStatusPass    DiagnosticCheckStatus = "pass"
	DiagnosticStatusWarning DiagnosticCheckStatus = "warning"
	DiagnosticStatusFail    DiagnosticCheckStatus = "fail"
)

// DiagnosticCheck represents an individual operational health evaluation.
type DiagnosticCheck struct {
	ID          string                `json:"id"`
	Category    string                `json:"category"` // "daemon", "storage", "remotes", "safety", "system"
	Name        string                `json:"name"`
	Status      DiagnosticCheckStatus `json:"status"`
	Message     string                `json:"message"`
	Detail      string                `json:"detail,omitempty"`
	ActionTitle string                `json:"action_title,omitempty"`
	ActionURL   string                `json:"action_url,omitempty"`
}

// DiagnosticSystemInfo summarizes host environment and runtime metadata.
type DiagnosticSystemInfo struct {
	Version     string `json:"version"`
	GoVersion   string `json:"go_version"`
	OS          string `json:"os"`
	Arch        string `json:"arch"`
	PID         int    `json:"pid"`
	UptimeSec   int64  `json:"uptime_sec"`
	BaseDir     string `json:"base_dir"`
	LogFilePath string `json:"log_file_path"`
	IsSystemd   bool   `json:"is_systemd"`
}

// DiagnosticsResponse represents the response payload for GET /api/diagnostics.
type DiagnosticsResponse struct {
	Status     string               `json:"status"`  // "ok", "warning", "error"
	Summary    map[string]int       `json:"summary"` // counts of pass, warning, fail
	Checks     []DiagnosticCheck    `json:"checks"`
	SystemInfo DiagnosticSystemInfo `json:"system_info"`
	Timestamp  string               `json:"timestamp"`
}
