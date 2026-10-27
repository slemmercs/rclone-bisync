package server

import (
	"context"
	"embed"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/config"
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
	go func() {
		_ = sched.TriggerSync(context.Background(), "prof-active", false)
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
	time.Sleep(50 * time.Millisecond)

	recPost := httptest.NewRecorder()
	srv.ServeHTTP(recPost, req)

	var postPayload map[string]interface{}
	if err := json.Unmarshal(recPost.Body.Bytes(), &postPayload); err != nil {
		t.Fatalf("failed to parse json response: %v", err)
	}

	if isSync, ok := postPayload["is_syncing"].(bool); !ok || isSync {
		t.Errorf("expected is_syncing = false after completion, got %v", postPayload["is_syncing"])
	}
	if execStatus, ok := postPayload["execution_status"].(string); !ok || execStatus != "IDLE" {
		t.Errorf("expected execution_status = IDLE after completion, got %v", postPayload["execution_status"])
	}
}
