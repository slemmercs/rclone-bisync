package server

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/models"
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

func TestServerAPIEndpoints(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "server-api-test-*")
	if err != nil {
		t.Fatalf("temp dir error: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		BaseDir:                tmpDir,
		LocalPath:              tmpDir,
		LogFile:                filepath.Join(tmpDir, "bisync.log"),
		MetricsFile:            filepath.Join(tmpDir, "metrics.jsonl"),
		NotificationStateFile:  filepath.Join(tmpDir, "notif.json"),
		RateLimitBackoffFile:   filepath.Join(tmpDir, "rate_limit.json"),
		PendingDeletionsFile:   filepath.Join(tmpDir, "pending.json"),
		LockFile:               filepath.Join(tmpDir, "lock.lck"),
	}
	svc := services.NewSyncService(cfg, nil)
	srv := NewServer(cfg, svc, embed.FS{})

	// 1. Untrusted Referer -> 403
	req, _ := http.NewRequest("POST", "/api/action", bytes.NewBuffer([]byte(`{"action":"test"}`)))
	req.Header.Set("X-Dashboard-Request", "1")
	req.Header.Set("Referer", "http://malicious-site.com/dashboard")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for untrusted Referer, got: %d", rec.Code)
	}

	// 2. Action with invalid JSON body -> 400
	req, _ = http.NewRequest("POST", "/api/action", bytes.NewBuffer([]byte(`not-json`)))
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid action JSON, got: %d", rec.Code)
	}

	// 3. Action with valid body -> 200
	req, _ = http.NewRequest("POST", "/api/action", bytes.NewBuffer([]byte(`{"action":"unknown"}`)))
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for action, got: %d", rec.Code)
	}

	// 4. Pending Deletions -> 200
	req, _ = http.NewRequest("GET", "/api/deletions/pending", nil)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for pending deletions, got: %d", rec.Code)
	}

	// 5. Remotes -> 200
	req, _ = http.NewRequest("GET", "/api/remotes", nil)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for remotes, got: %d", rec.Code)
	}

	// 6. Provision Canary -> 200
	req, _ = http.NewRequest("POST", "/api/browse/canary", bytes.NewBuffer([]byte(`{"remote":"local","path":""}`)))
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for canary provision, got: %d", rec.Code)
	}

	// 7. Copy Item with invalid body -> 400
	req, _ = http.NewRequest("POST", "/api/browse/copy", bytes.NewBuffer([]byte(`bad-body`)))
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad copy body, got: %d", rec.Code)
	}

	// 8. Delete Item with invalid body -> 400
	req, _ = http.NewRequest("POST", "/api/browse/delete", bytes.NewBuffer([]byte(`bad-body`)))
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad delete body, got: %d", rec.Code)
	}

	// 9. Delete Canary -> 200 with safety error
	req, _ = http.NewRequest("POST", "/api/browse/delete", bytes.NewBuffer([]byte(`{"remote":"local","path":"RCLONE_TEST","is_dir":false}`)))
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for delete request, got: %d", rec.Code)
	}
}

func TestLogDownloadEndpoints(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "log-download-*")
	if err != nil {
		t.Fatalf("temp dir error: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	bisyncLog := filepath.Join(tmpDir, "bisync.log")
	_ = os.WriteFile(bisyncLog, []byte("bisync log content"), 0644)
	daemonLog := filepath.Join(tmpDir, "rclone-daemon.log")
	_ = os.WriteFile(daemonLog, []byte("daemon log content"), 0644)

	cfg := &config.Config{
		BaseDir: tmpDir,
		LogFile: bisyncLog,
	}
	svc := services.NewSyncService(cfg, nil)
	srv := NewServer(cfg, svc, embed.FS{})

	// 1. GET /api/logs/bisync/raw
	req, _ := http.NewRequest("GET", "/api/logs/bisync/raw", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for bisync raw log, got: %d", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("bisync log content")) {
		t.Fatal("expected bisync log content in response")
	}

	// 2. GET /api/logs/daemon/raw
	req, _ = http.NewRequest("GET", "/api/logs/daemon/raw", nil)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for daemon raw log, got: %d", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("daemon log content")) {
		t.Fatal("expected daemon log content in response")
	}

	// 3. GET /api/logs/daemon (tail without window)
	req, _ = http.NewRequest("GET", "/api/logs/daemon?limit=10", nil)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for daemon logs query, got: %d", rec.Code)
	}
	var daemonResp models.DaemonLogsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &daemonResp); err != nil {
		t.Fatalf("failed to parse daemon logs response: %v", err)
	}
	if daemonResp.Count == 0 || daemonResp.Correlated {
		t.Fatalf("expected tail response: %+v", daemonResp)
	}

	// 4. GET /api/logs/daemon (correlated with time window)
	// Write JSON log lines to daemon log
	now := time.Now()
	jsonDaemonLog := fmt.Sprintf(`{"level":"info","msg":"file1 synced","time":"%s"}`+"\n"+`{"level":"info","msg":"file2 synced","time":"%s"}`+"\n",
		now.Format(time.RFC3339Nano), now.Add(2*time.Second).Format(time.RFC3339Nano))
	_ = os.WriteFile(daemonLog, []byte(jsonDaemonLog), 0644)

	windowURL := fmt.Sprintf("/api/logs/daemon?run_id=sync_123&job_id=99&start_time=%s&end_time=%s&limit=50",
		url.QueryEscape(now.Add(-1*time.Second).Format(time.RFC3339)),
		url.QueryEscape(now.Add(3*time.Second).Format(time.RFC3339)))
	req, _ = http.NewRequest("GET", windowURL, nil)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for correlated daemon logs, got: %d", rec.Code)
	}
	var corrResp models.DaemonLogsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &corrResp); err != nil {
		t.Fatalf("failed to parse correlated response: %v", err)
	}
	if !corrResp.Correlated || corrResp.RunID != "sync_123" || corrResp.JobID != 99 || corrResp.Count != 2 {
		t.Fatalf("unexpected correlated response: %+v", corrResp)
	}
}



func TestServerAdditionalBranches(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "server-branches-*")
	if err != nil {
		t.Fatalf("temp dir error: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a dummy file for copy and delete
	sampleFile := filepath.Join(tmpDir, "sample.txt")
	_ = os.WriteFile(sampleFile, []byte("data"), 0644)

	cfg := &config.Config{
		BaseDir:                tmpDir,
		LocalPath:              tmpDir,
		LogFile:                filepath.Join(tmpDir, "missing_bisync.log"),
		NotificationStateFile:  filepath.Join(tmpDir, "notif.json"),
		RateLimitBackoffFile:   filepath.Join(tmpDir, "rate_limit.json"),
		PendingDeletionsFile:   filepath.Join(tmpDir, "pending.json"),
		LockFile:               filepath.Join(tmpDir, "lock.lck"),
	}
	svc := services.NewSyncService(cfg, nil)
	srv := NewServer(cfg, svc, embed.FS{})

	// 1. Missing raw logs -> returns 200 with error JSON
	req, _ := http.NewRequest("GET", "/api/logs/bisync/raw", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !bytes.Contains(rec.Body.Bytes(), []byte("Log file not found")) {
		t.Fatalf("expected error JSON for missing bisync log, got code %d, body: %s", rec.Code, rec.Body.String())
	}

	req, _ = http.NewRequest("GET", "/api/logs/daemon/raw", nil)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !bytes.Contains(rec.Body.Bytes(), []byte("Daemon log file not found")) {
		t.Fatalf("expected error JSON for missing daemon log, got code %d, body: %s", rec.Code, rec.Body.String())
	}

	// 2. Telemetry with custom ?limit=75
	req, _ = http.NewRequest("GET", "/api/telemetry?limit=75", nil)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for telemetry with limit, got: %d", rec.Code)
	}

	// 3. Valid CopyItem POST -> 200
	copyBody := `{"src_remote":"local","src_path":"sample.txt","dst_remote":"local","dst_path":"sample_copy.txt","is_dir":false}`
	req, _ = http.NewRequest("POST", "/api/browse/copy", bytes.NewBuffer([]byte(copyBody)))
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid copy, got: %d", rec.Code)
	}

	// 4. Valid DeleteItem POST -> 200
	delBody := `{"remote":"local","path":"sample.txt","is_dir":false}`
	req, _ = http.NewRequest("POST", "/api/browse/delete", bytes.NewBuffer([]byte(delBody)))
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid delete, got: %d", rec.Code)
	}

	// 5. Origin checker with unparseable URL
	// 6. Favicon and 404 paths
	req, _ = http.NewRequest("GET", "/favicon.ico", nil)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	req, _ = http.NewRequest("GET", "/unknown-subpath", nil)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	// 7. Canary provision error path
	req, _ = http.NewRequest("POST", "/api/browse/canary", bytes.NewBuffer([]byte(`{"remote":"local","path":"../../../etc"}`)))
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 response for canary error: %d", rec.Code)
	}

	if isAllowedOrigin("http://[invalid-url]:999999") {
		t.Fatal("expected isAllowedOrigin to return false on invalid url")
	}
}
