package server

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/profiles"
)

func TestHandler_ProfileSync(t *testing.T) {
	s, svc := setupTestServerWithProfiles(t)

	// Create test profile (starts with State.LastSync == nil)
	testDir := filepath.Join(s.Config.BaseDir, "sync_test_dir")
	p := &profiles.Profile{
		ID:        "prof_sync_test",
		Name:      "Sync Test Profile",
		Remote:    "gdrive",
		LocalPath: testDir,
		Schedule:  "manual",
		Enabled:   true,
	}
	if _, err := svc.Create(context.Background(), p); err != nil {
		t.Fatalf("failed to create test profile: %v", err)
	}

	// 1. Initial sync trigger without resync -> should require resync
	reqBody := map[string]any{
		"force":   false,
		"resync":  false,
		"dry_run": false,
	}
	w := doJSONReq(s, http.MethodPost, "/api/profiles/prof_sync_test/sync", reqBody, true)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}
	if resp["requires_resync"] != true || resp["status"] != "requires_resync" {
		t.Fatalf("expected requires_resync: true and status: requires_resync, got %+v", resp)
	}

	// Verify that the intercept notice was logged to bisync.log
	logData, err := os.ReadFile(s.Config.LogFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}
	if !strings.Contains(string(logData), "REQUIRES_RESYNC") || !strings.Contains(string(logData), "prof_sync_test") {
		t.Fatalf("expected REQUIRES_RESYNC log entry in bisync.log, got: %s", string(logData))
	}

	// 2. Initial sync trigger with resync: true -> should succeed
	reqBodyResync := map[string]any{
		"force":   false,
		"resync":  true,
		"dry_run": false,
	}
	w = doJSONReq(s, http.MethodPost, "/api/profiles/prof_sync_test/sync", reqBodyResync, true)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK with resync: true, got %d: %s", w.Code, w.Body.String())
	}
	var respResync map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &respResync); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}
	if respResync["status"] != "success" {
		t.Fatalf("expected status: success with resync: true, got %+v", respResync)
	}

	// 3. Mark profile as having synced (LastSync set)
	now := time.Now()
	err = svc.UpdateState("prof_sync_test", profiles.OperationalState{
		LastSync: &now,
		Status:   profiles.StatusHealthy,
	})
	if err != nil {
		t.Fatalf("failed to update state: %v", err)
	}

	// Subsequent sync without resync -> should now succeed directly
	w = doJSONReq(s, http.MethodPost, "/api/profiles/prof_sync_test/sync", reqBody, true)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK after initialized, got %d: %s", w.Code, w.Body.String())
	}
	var respSubsequent map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &respSubsequent); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}
	if respSubsequent["status"] != "success" {
		t.Fatalf("expected status: success after initialization, got %+v", respSubsequent)
	}

	// 4. Trigger sync on nonexistent profile -> should return 404
	w = doJSONReq(s, http.MethodPost, "/api/profiles/nonexistent_prof/sync", reqBody, true)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found for missing profile, got %d", w.Code)
	}
}

func TestHandleAction_RequiresResync(t *testing.T) {
	s, svc := setupTestServerWithProfiles(t)

	// Create test profile (starts with State.LastSync == nil)
	testDir := filepath.Join(s.Config.BaseDir, "sync_test_dir_action")
	p := &profiles.Profile{
		ID:        "prof_action_test",
		Name:      "Action Sync Test Profile",
		Remote:    "gdrive",
		LocalPath: testDir,
		Schedule:  "manual",
		Enabled:   true,
	}
	if _, err := svc.Create(context.Background(), p); err != nil {
		t.Fatalf("failed to create test profile: %v", err)
	}

	// 1. Post start_sync via /api/action without resync -> should return status: requires_resync
	reqBody := map[string]any{
		"action":     "start_sync",
		"profile_id": "prof_action_test",
	}
	w := doJSONReq(s, http.MethodPost, "/api/action", reqBody, true)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}
	if resp["requires_resync"] != true || resp["status"] != "requires_resync" {
		t.Fatalf("expected requires_resync: true and status: requires_resync, got %+v", resp)
	}

	// Verify that the intercept notice was logged to bisync.log
	logDataAction, err := os.ReadFile(s.Config.LogFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}
	if !strings.Contains(string(logDataAction), "REQUIRES_RESYNC") || !strings.Contains(string(logDataAction), "prof_action_test") {
		t.Fatalf("expected REQUIRES_RESYNC log entry in bisync.log, got: %s", string(logDataAction))
	}

	// 2. Post force_resync via /api/action -> should succeed
	reqBodyResync := map[string]any{
		"action":     "force_resync",
		"profile_id": "prof_action_test",
	}
	w = doJSONReq(s, http.MethodPost, "/api/action", reqBodyResync, true)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK with force_resync, got %d: %s", w.Code, w.Body.String())
	}
	var respResync map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &respResync); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}
	if respResync["status"] != "success" {
		t.Fatalf("expected status: success with force_resync, got %+v", respResync)
	}

	// 3. Mark profile as initialized
	now := time.Now()
	err = svc.UpdateState("prof_action_test", profiles.OperationalState{
		LastSync: &now,
		Status:   profiles.StatusHealthy,
	})
	if err != nil {
		t.Fatalf("failed to update state: %v", err)
	}

	// 4. Subsequent start_sync via /api/action -> should succeed directly
	w = doJSONReq(s, http.MethodPost, "/api/action", reqBody, true)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK after initialized, got %d: %s", w.Code, w.Body.String())
	}
	var respSubsequent map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &respSubsequent); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}
	if respSubsequent["status"] != "success" {
		t.Fatalf("expected status: success after initialization, got %+v", respSubsequent)
	}
}
