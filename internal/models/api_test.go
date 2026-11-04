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

