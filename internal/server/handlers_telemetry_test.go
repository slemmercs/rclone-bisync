package server

import (
	"context"
	"embed"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/models"
	"github.com/slemmercs/rclone-bisync/internal/profiles"
	"github.com/slemmercs/rclone-bisync/internal/services"
)

func TestHandleTelemetry_ProfileSyncIntegration(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		BaseDir:               tmpDir,
		LogFile:               filepath.Join(tmpDir, "bisync.log"),
		MetricsFile:           filepath.Join(tmpDir, "metrics.jsonl"),
		NotificationStateFile: filepath.Join(tmpDir, "notif.json"),
		RateLimitBackoffFile:  filepath.Join(tmpDir, "rate_limit.json"),
		PendingDeletionsFile:  filepath.Join(tmpDir, "pending.json"),
		LockFile:              filepath.Join(tmpDir, "lock.lck"),
	}

	store, err := profiles.NewStore(
		filepath.Join(tmpDir, "profiles.json"),
		filepath.Join(tmpDir, "profiles_history.json"),
	)
	if err != nil {
		t.Fatalf("failed to create profile store: %v", err)
	}

	p := &profiles.Profile{
		ID:        "prof-active",
		Name:      "Active Profile",
		Remote:    "gdrive",
		LocalPath: tmpDir,
		Enabled:   true,
		Schedule:  "manual",
	}
	if err := store.Save(p, "Init"); err != nil {
		t.Fatalf("failed to save profile: %v", err)
	}

	started := make(chan struct{})
	release := make(chan struct{})

	executor := func(ctx context.Context, prof *profiles.Profile, opts profiles.SyncOptions) error {
		if prof.ID == "prof-active" {
			close(started)
			<-release
		}
		return nil
	}

	sched := profiles.NewScheduler(store, executor)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := sched.Start(ctx); err != nil {
		t.Fatalf("failed to start scheduler: %v", err)
	}
	defer sched.Stop()

	profSvc := profiles.NewService(store, nil, sched)
	syncSvc := services.NewSyncService(cfg, nil)
	srv := NewServer(cfg, syncSvc, embed.FS{})
	srv.ProfileService = profSvc

	// 1. Initial State: Idle
	req, _ := http.NewRequest("GET", "/api/telemetry?limit=10", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse json response: %v", err)
	}

	if isSync, ok := payload["is_syncing"].(bool); !ok || isSync {
		t.Fatalf("expected is_syncing = false initially, got %v", payload["is_syncing"])
	}
	if execStatus, ok := payload["execution_status"].(string); !ok || execStatus != "IDLE" {
		t.Fatalf("expected execution_status = IDLE initially, got %v", payload["execution_status"])
	}

	// 2. Trigger profile sync in background
	syncDone := make(chan error, 1)
	go func() {
		syncDone <- sched.TriggerSync(context.Background(), "prof-active", false)
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for profile sync to start")
	}

	// 3. Poll Telemetry while profile is syncing
	recSync := httptest.NewRecorder()
	srv.ServeHTTP(recSync, req)

	if recSync.Code != http.StatusOK {
		t.Fatalf("expected 200 OK while syncing, got %d", recSync.Code)
	}

	var syncPayload map[string]interface{}
	if err := json.Unmarshal(recSync.Body.Bytes(), &syncPayload); err != nil {
		t.Fatalf("failed to parse json response: %v", err)
	}

	if isSync, ok := syncPayload["is_syncing"].(bool); !ok || !isSync {
		t.Errorf("expected is_syncing = true during profile sync, got %v", syncPayload["is_syncing"])
	}
	if execStatus, ok := syncPayload["execution_status"].(string); !ok || execStatus != "SYNCING" {
		t.Errorf("expected execution_status = SYNCING during profile sync, got %v", syncPayload["execution_status"])
	}

	// 4. Release and complete sync
	close(release)
	select {
	case <-syncDone:
	case <-time.After(3 * time.Second):
		t.Fatalf("timeout waiting for profile sync to complete")
	}

	var postPayload map[string]interface{}
	deadline := time.Now().Add(2 * time.Second)
	for {
		recPost := httptest.NewRecorder()
		srv.ServeHTTP(recPost, req)

		if recPost.Code != http.StatusOK {
			t.Fatalf("expected 200 OK after completion, got %d", recPost.Code)
		}

		if err := json.Unmarshal(recPost.Body.Bytes(), &postPayload); err != nil {
			t.Fatalf("failed to parse json response: %v", err)
		}

		isSync, _ := postPayload["is_syncing"].(bool)
		execStatus, _ := postPayload["execution_status"].(string)
		if !isSync && execStatus == "IDLE" {
			break
		}
		if time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if isSync, ok := postPayload["is_syncing"].(bool); !ok || isSync {
		t.Errorf("expected is_syncing = false after completion, got %v", postPayload["is_syncing"])
	}
	if execStatus, ok := postPayload["execution_status"].(string); !ok || execStatus != "IDLE" {
		t.Errorf("expected execution_status = IDLE after completion, got %v", postPayload["execution_status"])
	}
}

func TestMCPStatusEndpoint(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		BaseDir:       tmpDir,
		MCPStatusFile: filepath.Join(tmpDir, "mcp_status.json"),
	}

	telSvc := services.NewTelemetryService(cfg, nil)
	srv := NewServer(cfg, nil, embed.FS{})
	srv.TelemetryService = telSvc

	// 1. Initial State: Offline
	req1 := httptest.NewRequest("GET", "/api/mcp/status", nil)
	rec1 := httptest.NewRecorder()
	srv.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec1.Code)
	}
	var resp1 models.MCPStatus
	if err := json.Unmarshal(rec1.Body.Bytes(), &resp1); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp1.Status != "offline" {
		t.Errorf("expected offline, got %s", resp1.Status)
	}

	// 2. Active MCP server with current PID
	activeStatus := models.MCPStatus{
		Status:    "active",
		Transport: "sse",
		ReadOnly:  false,
		Host:      "127.0.0.1",
		Port:      8081,
		PID:       os.Getpid(),
		StartedAt: time.Now().Format(time.RFC3339),
	}
	data, _ := json.Marshal(activeStatus)
	if err := os.WriteFile(cfg.MCPStatusFile, data, 0644); err != nil {
		t.Fatalf("failed to write mcp_status.json: %v", err)
	}

	req2 := httptest.NewRequest("GET", "/api/mcp/status", nil)
	rec2 := httptest.NewRecorder()
	srv.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec2.Code)
	}
	var resp2 models.MCPStatus
	if err := json.Unmarshal(rec2.Body.Bytes(), &resp2); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp2.Status != "active" || resp2.Transport != "sse" || resp2.Port != 8081 {
		t.Errorf("unexpected active status response: %+v", resp2)
	}

	// 3. /api/telemetry should include mcp_status
	req3 := httptest.NewRequest("GET", "/api/telemetry?limit=0", nil)
	rec3 := httptest.NewRecorder()
	srv.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec3.Code)
	}
	var telResp map[string]interface{}
	if err := json.Unmarshal(rec3.Body.Bytes(), &telResp); err != nil {
		t.Fatalf("failed to decode telemetry: %v", err)
	}
	mcpMap, ok := telResp["mcp_status"].(map[string]interface{})
	if !ok || mcpMap["status"] != "active" {
		t.Errorf("expected active mcp_status in telemetry payload, got: %v", telResp["mcp_status"])
	}

	// 4. Stale PID should be detected and marked offline
	staleStatus := models.MCPStatus{
		Status: "active",
		PID:    99999999,
	}
	staleData, _ := json.Marshal(staleStatus)
	_ = os.WriteFile(cfg.MCPStatusFile, staleData, 0644)

	req4 := httptest.NewRequest("GET", "/api/mcp/status", nil)
	rec4 := httptest.NewRecorder()
	srv.ServeHTTP(rec4, req4)
	var resp4 models.MCPStatus
	_ = json.Unmarshal(rec4.Body.Bytes(), &resp4)
	if resp4.Status != "offline" {
		t.Errorf("expected stale PID to resolve to offline, got %s", resp4.Status)
	}
}
