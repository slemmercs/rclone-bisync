package models

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
