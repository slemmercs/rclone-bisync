package server

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
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
	err := svc.UpdateState("prof_sync_test", profiles.OperationalState{
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
