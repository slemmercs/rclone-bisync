package server

import (
	"bytes"
	"embed"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/services"
)

func TestCSRFProtectionMiddleware(t *testing.T) {
	cfg := &config.Config{}
	svc := services.NewSyncService(cfg, nil)
	srv := NewServer(cfg, svc, embed.FS{})

	// 1. POST without X-Dashboard-Request header -> 403
	req, _ := http.NewRequest("POST", "/api/action", bytes.NewBuffer([]byte(`{"action":"start_sync"}`)))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for missing header, got %d", rec.Code)
	}

	// 2. POST with untrusted Origin header -> 403
	req, _ = http.NewRequest("POST", "/api/action", bytes.NewBuffer([]byte(`{"action":"start_sync"}`)))
	req.Header.Set("X-Dashboard-Request", "1")
	req.Header.Set("Origin", "https://evil.com")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for untrusted origin, got %d", rec.Code)
	}

	// 3. POST with localhost Origin and X-Dashboard-Request header -> processed
	req, _ = http.NewRequest("POST", "/api/browse", bytes.NewBuffer([]byte(`{"remote":"local","path":""}`)))
	req.Header.Set("X-Dashboard-Request", "1")
	req.Header.Set("Origin", "http://localhost:8080")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for valid request, got %d", rec.Code)
	}
}

func TestTelemetryRedaction(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "telemetry-test-*")
	if err != nil {
		t.Fatalf("temp dir error: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		BaseDir:                tmpDir,
		LogFile:                filepath.Join(tmpDir, "bisync.log"),
		MetricsFile:            filepath.Join(tmpDir, "metrics.jsonl"),
		NotificationStateFile:  filepath.Join(tmpDir, "notif.json"),
		RateLimitBackoffFile:   filepath.Join(tmpDir, "rate_limit.json"),
		PendingDeletionsFile:   filepath.Join(tmpDir, "pending.json"),
		LockFile:               filepath.Join(tmpDir, "lock.lck"),
		RCPass:                 "supersecretpassword123",
		RCUser:                 "myadmin",
	}
	svc := services.NewSyncService(cfg, nil)
	srv := NewServer(cfg, svc, embed.FS{})

	req, _ := http.NewRequest("GET", "/api/telemetry", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for telemetry, got %d", rec.Code)
	}

	body := rec.Body.String()
	if bytes.Contains([]byte(body), []byte("supersecretpassword123")) {
		t.Fatalf("telemetry response leaked secret password: %s", body)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		t.Fatalf("failed to parse telemetry json: %v", err)
	}
	cfgMap, ok := payload["config"].(map[string]interface{})
	if !ok {
		t.Fatal("config map missing from telemetry payload")
	}
	if cfgMap["RCPASS"] != "********" {
		t.Fatalf("expected RCPASS to be masked, got: %v", cfgMap["RCPASS"])
	}
}

func TestStaticAndIndexEmbedding(t *testing.T) {
	cfg := &config.Config{}
	svc := services.NewSyncService(cfg, nil)
	// Pass embed.FS or test root
	srv := NewServer(cfg, svc, embed.FS{})

	req, _ := http.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	// With empty embed.FS, returns 404 or index.html not found
	if rec.Code != http.StatusNotFound && rec.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", rec.Code)
	}
}
