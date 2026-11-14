package models

import "encoding/json"

// PaginatedLogsResponse represents a paged slice of log entries.
type PaginatedLogsResponse struct {
	Page       int      `json:"page"`
	Limit      int      `json:"limit"`
	Total      int      `json:"total"`
	TotalPages int      `json:"total_pages"`
	Logs       []string `json:"logs"`
}

// DedupeRequest represents payload for triggering deduplication.
type DedupeRequest struct {
	Remote     string `json:"remote"`
	Path       string `json:"path"`
	Mode       string `json:"mode"` // "list", "oldest", "newest"
	ByHash     bool   `json:"by_hash"`
	DedupeBoth bool   `json:"dedupe_both"`
	ProfileID  string `json:"profile_id,omitempty"`
}

// DedupeResponse represents the outcome of deduplication.
type DedupeResponse struct {
	Status      string `json:"status"`
	RunID       string `json:"run_id,omitempty"`
	Remote      string `json:"remote"`
	Path        string `json:"path"`
	Mode        string `json:"mode"`
	ByHash      bool   `json:"by_hash,omitempty"`
	Output      string `json:"output"`
	LocalPath   string `json:"local_path,omitempty"`
	LocalOutput string `json:"local_output,omitempty"`
	Collisions  int    `json:"collisions"`
	Deleted     int    `json:"deleted"`
	Error       string `json:"error,omitempty"`
}

// AuditLogRecord represents a cryptographically verified milestone logged in the unified bisync.log stream.
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

// DaemonLogsQuery represents parameters for inspecting correlated daemon logs.
type DaemonLogsQuery struct {
	RunID     string `json:"run_id"`
	JobID     int    `json:"job_id"`
	StartTime string `json:"start_time"` // RFC3339
	EndTime   string `json:"end_time"`   // RFC3339
	Limit     int    `json:"limit"`
}

// DaemonLogsResponse holds correlated daemon log lines scoped to a sync execution.
type DaemonLogsResponse struct {
	RunID      string   `json:"run_id,omitempty"`
	JobID      int      `json:"job_id,omitempty"`
	Count      int      `json:"count"`
	Logs       []string `json:"logs"`
	Correlated bool     `json:"correlated"`
	Error      string   `json:"error,omitempty"`
}
