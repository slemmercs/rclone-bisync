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
