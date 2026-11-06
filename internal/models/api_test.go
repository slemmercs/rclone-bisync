package models

import (
	"encoding/json"
	"testing"
)

func TestNewErrorResponse(t *testing.T) {
	errResp := NewErrorResponse("Something went wrong")

	if errResp.Status != "error" {
		t.Errorf("expected status 'error', got %s", errResp.Status)
	}
	if errResp.Msg != "Something went wrong" {
		t.Errorf("expected msg 'Something went wrong', got %s", errResp.Msg)
	}
	if errResp.Error != "Something went wrong" {
		t.Errorf("expected error 'Something went wrong', got %s", errResp.Error)
	}

	data, err := json.Marshal(errResp)
	if err != nil {
		t.Fatalf("failed to marshal ErrorResponse: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if parsed["status"] != "error" || parsed["msg"] != "Something went wrong" || parsed["error"] != "Something went wrong" {
		t.Fatalf("unexpected JSON payload: %+v", parsed)
	}
}

func TestTelemetryAndConfigSchemas(t *testing.T) {
	// 1. TelemetryResponse contract
	tel := TelemetryResponse{
		IsSyncing:       false,
		ExecutionStatus: "IDLE",
		HasReview:       true,
		ReviewPending:   true,
		MCPStatus: &MCPStatus{
			Status:    "active",
			Transport: "sse",
			Port:      5573,
		},
	}
	telBytes, err := json.Marshal(tel)
	if err != nil {
		t.Fatalf("failed to marshal TelemetryResponse: %v", err)
	}
	var telMap map[string]interface{}
	if err := json.Unmarshal(telBytes, &telMap); err != nil {
		t.Fatalf("failed to unmarshal TelemetryResponse: %v", err)
	}
	if telMap["execution_status"] != "IDLE" || telMap["is_syncing"] != false || telMap["has_review"] != true {
		t.Fatalf("unexpected TelemetryResponse JSON: %+v", telMap)
	}

	// 2. GlobalConfigResponse contract
	cfgResp := GlobalConfigResponse{
		Status: "success",
		GlobalDefaults: GlobalDefaults{
			MaxDelete:       20,
			ConflictResolve: "newer",
		},
		Daemon: ConfigDaemonSettings{
			JobTimeoutSeconds: 1800,
		},
		Notifications: ConfigNotificationSettings{
			WebhookURL: "https://example.com/webhook",
		},
	}
	cfgBytes, err := json.Marshal(cfgResp)
	if err != nil {
		t.Fatalf("failed to marshal GlobalConfigResponse: %v", err)
	}
	var cfgMap map[string]interface{}
	if err := json.Unmarshal(cfgBytes, &cfgMap); err != nil {
		t.Fatalf("failed to unmarshal GlobalConfigResponse: %v", err)
	}
	if cfgMap["status"] != "success" {
		t.Fatalf("unexpected GlobalConfigResponse JSON: %+v", cfgMap)
	}
}

func TestDiagnosticsResponseSchema(t *testing.T) {
	resp := DiagnosticsResponse{
		Status: "warning",
		Summary: map[string]int{
			"pass":    4,
			"warning": 1,
			"fail":    0,
		},
		Checks: []DiagnosticCheck{
			{
				ID:          "daemon_connectivity",
				Category:    "daemon",
				Name:        "Rclone RC Daemon",
				Status:      DiagnosticStatusPass,
				Message:     "Daemon is active and responsive",
				Detail:      "Latency: 1.2ms",
				ActionTitle: "View Logs",
				ActionURL:   "/api/logs/daemon",
			},
			{
				ID:          "remotes_oauth",
				Category:    "remotes",
				Name:        "OAuth Token Health",
				Status:      DiagnosticStatusWarning,
				Message:     "Remote 'gdrive' token expires in 2 hours",
				Detail:      "Auto-refresh active",
				ActionTitle: "Reconnect",
				ActionURL:   "/api/remotes/gdrive/reconnect",
			},
		},
		SystemInfo: DiagnosticSystemInfo{
			Version:     "v1.0.0-beta.28",
			GoVersion:   "go1.22.0",
			OS:          "linux",
			Arch:        "amd64",
			PID:         12345,
			UptimeSec:   3600,
			BaseDir:     "/home/user/.config/rclone-bisync",
			LogFilePath: "/home/user/.config/rclone-bisync/bisync.log",
			IsSystemd:   true,
		},
		Timestamp: "2026-11-06T20:00:00Z",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal DiagnosticsResponse: %v", err)
	}

	var parsed DiagnosticsResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal DiagnosticsResponse: %v", err)
	}

	if parsed.Status != "warning" {
		t.Errorf("expected status 'warning', got %s", parsed.Status)
	}
	if len(parsed.Checks) != 2 {
		t.Errorf("expected 2 checks, got %d", len(parsed.Checks))
	}
	if parsed.Checks[0].Status != DiagnosticStatusPass {
		t.Errorf("expected DiagnosticStatusPass, got %s", parsed.Checks[0].Status)
	}
	if parsed.Checks[1].Status != DiagnosticStatusWarning {
		t.Errorf("expected DiagnosticStatusWarning, got %s", parsed.Checks[1].Status)
	}
	if parsed.SystemInfo.PID != 12345 {
		t.Errorf("expected PID 12345, got %d", parsed.SystemInfo.PID)
	}
}
