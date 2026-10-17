package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/core/client"
)

func TestGetRuntimeConfigAndRedaction(t *testing.T) {
	cfg := &config.Config{
		BaseDir:    "/tmp/basedir",
		LocalPath:  "/tmp/local",
		RemotePath: "gdrive:",
		RCPass:     "supersecretpassword",
		RCUser:     "rclone",
		WebhookURL: "https://discord.com/api/webhooks/123/xyz",
	}

	syncSvc := NewSyncService(cfg, nil)
	runtimeCfg := syncSvc.GetRuntimeConfig()

	// Verify snake_case keys exist
	if runtimeCfg["BASE_DIR"] != "/tmp/basedir" {
		t.Fatalf("expected BASE_DIR to be /tmp/basedir, got: %v", runtimeCfg["BASE_DIR"])
	}
	if runtimeCfg["LOCAL_PATH"] != "/tmp/local" {
		t.Fatalf("expected LOCAL_PATH to be /tmp/local, got: %v", runtimeCfg["LOCAL_PATH"])
	}
	if runtimeCfg["REMOTE_PATH"] != "gdrive:" {
		t.Fatalf("expected REMOTE_PATH to be gdrive:, got: %v", runtimeCfg["REMOTE_PATH"])
	}

	// Verify sensitive keys are redacted
	if runtimeCfg["RC_PASS"] != "********" {
		t.Fatalf("expected RC_PASS to be masked, got: %v", runtimeCfg["RC_PASS"])
	}
	if runtimeCfg["RCPASS"] != "********" {
		t.Fatalf("expected RCPASS alias to be masked, got: %v", runtimeCfg["RCPASS"])
	}
	if runtimeCfg["WEBHOOK_URL"] != "********" {
		t.Fatalf("expected WEBHOOK_URL to be masked, got: %v", runtimeCfg["WEBHOOK_URL"])
	}

	// Verify non-sensitive keys are unmasked
	if runtimeCfg["RC_USER"] != "rclone" {
		t.Fatalf("expected RC_USER to be unmasked, got: %v", runtimeCfg["RC_USER"])
	}

	// Helper tests
	if toUpperSnakeCase("LocalPath") != "LOCAL_PATH" {
		t.Fatalf("expected LOCAL_PATH, got: %s", toUpperSnakeCase("LocalPath"))
	}
	if toUpperSnakeCase("BaseDir") != "BASE_DIR" {
		t.Fatalf("expected BASE_DIR, got: %s", toUpperSnakeCase("BaseDir"))
	}
	if toUpperSnakeCase("DaemonStartupTimeoutSec") != "DAEMON_STARTUP_TIMEOUT_SEC" {
		t.Fatalf("expected DAEMON_STARTUP_TIMEOUT_SEC, got: %s", toUpperSnakeCase("DaemonStartupTimeoutSec"))
	}

	if !isKeySensitive("USER_PASSWORD") || !isKeySensitive("API_SECRET") || !isKeySensitive("AUTH_TOKEN") {
		t.Fatal("expected sensitive pattern detection to match credentials")
	}
	if isKeySensitive("LOCAL_PATH") || isKeySensitive("BASE_DIR") {
		t.Fatal("unexpected sensitive match for path keys")
	}
}

func TestGetTransferStats(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/core/stats" {
			w.WriteHeader(http.StatusOK)
			eta := 45
			data := map[string]interface{}{
				"bytes":        int64(10485760),
				"totalBytes":   int64(20971520),
				"speed":        2097152.0,
				"eta":          &eta,
				"transferring": []map[string]interface{}{{"name": "file.pdf"}},
			}
			_ = json.NewEncoder(w).Encode(data)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	rClient := client.NewRcloneClient(srv.URL, "user", "pass")
	syncSvc := NewSyncService(&config.Config{}, rClient)

	stats := syncSvc.GetTransferStats(context.Background())
	if stats == nil {
		t.Fatal("expected non-nil transfer stats")
	}
	if stats.Bytes != 10485760 || stats.Percentage != 50.0 {
		t.Fatalf("unexpected stats values: bytes=%d pct=%.1f", stats.Bytes, stats.Percentage)
	}
	if len(stats.Transferring) != 1 {
		t.Fatalf("expected 1 transferring item, got: %d", len(stats.Transferring))
	}

	syncSvcBad := NewSyncService(&config.Config{}, client.NewRcloneClient("http://127.0.0.1:1", "", ""))
	if badStats := syncSvcBad.GetTransferStats(context.Background()); badStats != nil {
		t.Fatalf("expected nil stats on connection error, got: %v", badStats)
	}
}


func TestGetCachedMetrics(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "metrics-cache-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	metricsFile := filepath.Join(tmpDir, "metrics.jsonl")
	cfg := &config.Config{MetricsFile: metricsFile}
	syncSvc := NewSyncService(cfg, nil)

	// Missing file returns empty slice
	empty := syncSvc.GetCachedMetrics(10)
	if len(empty) != 0 {
		t.Fatalf("expected 0 entries for missing file, got: %d", len(empty))
	}

	// Populate file
	line1 := `{"timestamp": 100.0, "success": true, "duration_seconds": 1.2}`
	line2 := `{"timestamp": 200.0, "success": false, "error_type": "RATE_LIMIT"}`
	_ = os.WriteFile(metricsFile, []byte(line1+"\n"+line2+"\n"), 0644)

	// First fetch parses from disk
	entries := syncSvc.GetCachedMetrics(10)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries parsed, got: %d", len(entries))
	}

	// Second fetch hits memory cache
	cached := syncSvc.GetCachedMetrics(10)
	if len(cached) != 2 {
		t.Fatalf("expected 2 entries from cache, got: %d", len(cached))
	}
}

func TestGetTelemetryPayload(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "telemetry-payload-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	notifFile := filepath.Join(tmpDir, "notification_state.json")
	_ = os.WriteFile(notifFile, []byte(`{"last_success_timestamp": 50.0, "last_error_type": "NONE"}`), 0644)

	metricsFile := filepath.Join(tmpDir, "metrics.jsonl")
	_ = os.WriteFile(metricsFile, []byte(`{"timestamp": 150.0, "success": true}`+"\n"), 0644)

	bisyncLog := filepath.Join(tmpDir, "bisync.log")
	_ = os.WriteFile(bisyncLog, []byte(`{"time":"2026-10-17T20:00:00Z","level":"info","msg":"sync started"}`+"\n"), 0644)

	cfg := &config.Config{
		BaseDir:               tmpDir,
		NotificationStateFile: notifFile,
		MetricsFile:           metricsFile,
		LogFile:               bisyncLog,
		LockFile:              filepath.Join(tmpDir, "bisync.lock"),
	}

	syncSvc := NewSyncService(cfg, nil)
	payload := syncSvc.GetTelemetryPayload(context.Background(), 10)

	if payload == nil {
		t.Fatal("expected non-nil telemetry payload")
	}

	notifState, ok := payload["notification_state"].(map[string]interface{})
	if !ok {
		t.Fatal("expected notification_state map in payload")
	}
	if notifState["last_success_timestamp"] != 150.0 {
		t.Fatalf("expected updated last_success_timestamp 150.0, got: %v", notifState["last_success_timestamp"])
	}

	logs, ok := payload["bisync_logs"].([]string)
	if !ok || len(logs) == 0 {
		t.Fatalf("expected parsed bisync_logs in payload: %v", payload["bisync_logs"])
	}
}
