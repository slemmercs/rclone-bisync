package models

import "encoding/json"

type ActionRequest struct {
	Action string `json:"action"`
}

type ActionResponse struct {
	Status string `json:"status"`
	Msg    string `json:"msg,omitempty"`
	Error  string `json:"error,omitempty"`
}

type BrowseRequest struct {
	Remote string `json:"remote"`
	Path   string `json:"path"`
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
}

type CopyResponse struct {
	Status string `json:"status"`
	Msg    string `json:"msg,omitempty"`
	Src    string `json:"src,omitempty"`
	Dst    string `json:"dst,omitempty"`
	Error  string `json:"error,omitempty"`
}

type DeleteRequest struct {
	Remote string `json:"remote"`
	Path   string `json:"path"`
	IsDir  bool   `json:"is_dir"`
}

type DeleteResponse struct {
	Status string `json:"status"`
	Msg    string `json:"msg,omitempty"`
	Remote string `json:"remote,omitempty"`
	Path   string `json:"path,omitempty"`
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
	TransferStats     *TransferStats           `json:"transfer_stats"`
}

type BisyncLogEntry struct {
	Time        string  `json:"time"`
	Level       string  `json:"level"`
	Type        string  `json:"type,omitempty"` // "sync" (or omitted)
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

// AuditLogRecord represents a cryptographically verified milestone logged in the unified bisync.log stream.
type AuditLogRecord struct {
	Time      string          `json:"time"`
	Level     string          `json:"level"`
	Type      string          `json:"type"` // "audit"
	Msg       string          `json:"msg"`
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
