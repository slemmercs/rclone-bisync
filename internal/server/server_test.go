package server

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/core/client"
	"github.com/slemmercs/rclone-bisync/internal/engine/review"
	"github.com/slemmercs/rclone-bisync/internal/models"
	"github.com/slemmercs/rclone-bisync/internal/remoteauth"
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
	var errResp models.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to decode ErrorResponse envelope: %v", err)
	}
	if errResp.Status != "error" || errResp.Msg == "" || errResp.Error == "" {
		t.Fatalf("expected uniform ErrorResponse with status, msg, and error, got: %+v", errResp)
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
		BaseDir:               tmpDir,
		LogFile:               filepath.Join(tmpDir, "bisync.log"),
		MetricsFile:           filepath.Join(tmpDir, "metrics.jsonl"),
		NotificationStateFile: filepath.Join(tmpDir, "notif.json"),
		RateLimitBackoffFile:  filepath.Join(tmpDir, "rate_limit.json"),
		PendingDeletionsFile:  filepath.Join(tmpDir, "pending.json"),
		LockFile:              filepath.Join(tmpDir, "lock.lck"),
		RCPass:                "supersecretpassword123",
		RCUser:                "myadmin",
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
		BaseDir:               tmpDir,
		LocalPath:             tmpDir,
		LogFile:               filepath.Join(tmpDir, "bisync.log"),
		MetricsFile:           filepath.Join(tmpDir, "metrics.jsonl"),
		NotificationStateFile: filepath.Join(tmpDir, "notif.json"),
		RateLimitBackoffFile:  filepath.Join(tmpDir, "rate_limit.json"),
		PendingDeletionsFile:  filepath.Join(tmpDir, "pending.json"),
		LockFile:              filepath.Join(tmpDir, "lock.lck"),
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

	// 10. GET /api/health -> 200 with DaemonHealthResponse
	req, _ = http.NewRequest("GET", "/api/health", nil)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for /api/health, got: %d", rec.Code)
	}
	var healthResp models.DaemonHealthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &healthResp); err != nil {
		t.Fatalf("failed to decode health response: %v", err)
	}
	if healthResp.Status != "offline" { // nil client reports offline
		t.Fatalf("expected offline status, got: %s", healthResp.Status)
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

	// 5. GET /api/logs/daemon with run_id=dedupe-... (correlated dedupe log from bisync.log)
	bisyncLog = filepath.Join(tmpDir, "bisync.log")
	dedupeRunID := "dedupe-987654321"
	dedupeLines := []string{
		fmt.Sprintf(`{"time":"%s","level":"info","type":"dedupe","msg":"[gdrive] Deduplication started","run_id":"%s","origin":"dedupe"}`, now.Format(time.RFC3339), dedupeRunID),
		fmt.Sprintf(`{"time":"%s","level":"notice","type":"dedupe","msg":"[gdrive] file.txt: 1 removed","run_id":"%s","origin":"dedupe"}`, now.Add(1*time.Second).Format(time.RFC3339), dedupeRunID),
	}
	_ = os.WriteFile(bisyncLog, []byte(strings.Join(dedupeLines, "\n")+"\n"), 0644)
	cfg.LogFile = bisyncLog

	req, _ = http.NewRequest("GET", "/api/logs/daemon?run_id="+dedupeRunID, nil)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for dedupe correlated logs, got: %d", rec.Code)
	}
	var dedupeResp models.DaemonLogsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &dedupeResp); err != nil {
		t.Fatalf("failed to decode dedupe logs response: %v", err)
	}
	if !dedupeResp.Correlated || dedupeResp.RunID != dedupeRunID || dedupeResp.Count != 2 {
		t.Fatalf("unexpected dedupe logs response: %+v", dedupeResp)
	}

	// 6. GET /api/logs/daemon with unknown run_id without window returns 0 lines (not unfiltered tail)
	req, _ = http.NewRequest("GET", "/api/logs/daemon?run_id=sync_unknown_nonexistent", nil)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got: %d", rec.Code)
	}
	var unkResp models.DaemonLogsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &unkResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if unkResp.Count != 0 {
		t.Fatalf("expected 0 lines for unknown run_id, got %d", unkResp.Count)
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
		BaseDir:               tmpDir,
		LocalPath:             tmpDir,
		LogFile:               filepath.Join(tmpDir, "missing_bisync.log"),
		NotificationStateFile: filepath.Join(tmpDir, "notif.json"),
		RateLimitBackoffFile:  filepath.Join(tmpDir, "rate_limit.json"),
		PendingDeletionsFile:  filepath.Join(tmpDir, "pending.json"),
		LockFile:              filepath.Join(tmpDir, "lock.lck"),
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

	// Origin validation checks (Finding M-2)
	allowedOrigins := []string{
		"",
		"http://localhost:8080",
		"http://127.0.0.1:8080",
		"http://[::1]:8080",
		"http://192.168.1.100:8080",
		"http://10.0.0.5:8080",
		"http://172.16.0.2:8080",
		"http://100.80.5.10:8080",
		"http://homelab.local:8080",
		"http://nas.lan:8080",
	}
	for _, origin := range allowedOrigins {
		if !isAllowedOrigin(origin) {
			t.Fatalf("expected origin '%s' to be allowed", origin)
		}
	}

	disallowedOrigins := []string{
		"http://[invalid-url]:999999",
		"http://evil.com",
		"http://8.8.8.8:8080",
		"http://localhost.evil.com",
		"http://192.168.1.100.attacker.com",
	}
	for _, origin := range disallowedOrigins {
		if isAllowedOrigin(origin) {
			t.Fatalf("expected origin '%s' to be disallowed", origin)
		}
	}
}

func TestAuthEndpoints(t *testing.T) {
	// Arrange
	mockRCServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/config/listremotes":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"remotes": ["gdrive"]}`))
		case "/config/get":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"type":"drive","client_id":"app-id.apps.googleusercontent.com","client_secret":"super_secret_value_123","token":"{\"access_token\":\"ya29.sample\"}"}`))
		case "/operations/about":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"total": 1000}`))
		case "/config/update":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		case "/config/oauthstatus":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status": "stopped"}`))
		case "/config/oauthstop":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		case "/core/command":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"error": false, "result": "Duplicate files consolidation complete."}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockRCServer.Close()

	rcClient := client.NewRcloneClient(mockRCServer.URL, "user", "pass")
	cfg := &config.Config{
		RCAPIURL: mockRCServer.URL,
	}

	mockRunner := func(ctx context.Context, name string, env []string, args ...string) (string, error) {
		return "Mock browser reconnect executed", nil
	}

	syncSvc := services.NewSyncService(cfg, rcClient)
	srv := NewServer(cfg, syncSvc, embed.FS{})
	srv.AuthService = remoteauth.NewRemoteAuthService(cfg, rcClient, mockRunner)

	// 1. GET /api/remotes/auth
	req, _ := http.NewRequest("GET", "/api/remotes/auth", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for remotes auth, got: %d", rec.Code)
	}
	var authResp models.AuthStatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &authResp); err != nil {
		t.Fatalf("failed to decode auth response: %v", err)
	}
	if len(authResp.Remotes) != 1 || authResp.Remotes[0].Name != "gdrive" {
		t.Fatalf("unexpected remotes in auth response: %+v", authResp)
	}

	// 2. POST /api/remotes/{name}/reconnect without CSRF header -> 403
	req, _ = http.NewRequest("POST", "/api/remotes/gdrive/reconnect", nil)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for reconnect missing CSRF header, got: %d", rec.Code)
	}

	// 3. POST /api/remotes/{name}/reconnect with CSRF header -> 200
	req, _ = http.NewRequest("POST", "/api/remotes/gdrive/reconnect", nil)
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for reconnect, got: %d", rec.Code)
	}
	var reconResp models.ReconnectResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &reconResp); err != nil {
		t.Fatalf("failed to decode reconnect response: %v", err)
	}
	if reconResp.Status != "success" || reconResp.Remote != "gdrive" {
		t.Fatalf("unexpected reconnect response: %+v", reconResp)
	}

	// 4. POST /api/remotes/{name}/credentials with secret -> ensure NO secret leak in response
	credBody := `{"client_id":"new-id.apps.googleusercontent.com","client_secret":"super_secret_value_123"}`
	req, _ = http.NewRequest("POST", "/api/remotes/gdrive/credentials", bytes.NewBuffer([]byte(credBody)))
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for credentials update, got: %d", rec.Code)
	}
	bodyStr := rec.Body.String()
	if bytes.Contains([]byte(bodyStr), []byte("super_secret_value_123")) {
		t.Fatalf("security violation: client_secret leaked in response: %s", bodyStr)
	}

	// 5. POST /api/remotes/{name}/credentials bad request
	req, _ = http.NewRequest("POST", "/api/remotes/gdrive/credentials", bytes.NewBuffer([]byte(`invalid-json`)))
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad credentials body, got: %d", rec.Code)
	}

	// 6. POST /api/remotes/{name}/validate
	req, _ = http.NewRequest("POST", "/api/remotes/gdrive/validate", nil)
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for remote validate, got: %d", rec.Code)
	}

	// 7. GET /api/remotes/auth/audit
	req, _ = http.NewRequest("GET", "/api/remotes/auth/audit?limit=25", nil)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for audit events, got: %d", rec.Code)
	}
	var auditResp models.AuditEventsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &auditResp); err != nil {
		t.Fatalf("failed to decode audit events: %v", err)
	}
	if auditResp.Count == 0 {
		t.Fatal("expected non-zero audit events after reconnect and credential update")
	}

	// 8. POST /api/remotes/auth/oauth-stop
	req, _ = http.NewRequest("POST", "/api/remotes/auth/oauth-stop", nil)
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for oauth-stop, got: %d", rec.Code)
	}

	// 9. POST /api/remotes/auth/check (active check with CSRF header)
	req, _ = http.NewRequest("POST", "/api/remotes/auth/check", nil)
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for auth check, got: %d", rec.Code)
	}
	var checkResp models.AuthStatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &checkResp); err != nil {
		t.Fatalf("failed to decode auth check response: %v", err)
	}
	if len(checkResp.Remotes) != 1 {
		t.Fatalf("expected 1 remote in auth check, got: %d", len(checkResp.Remotes))
	}
	// 9b. POST /api/remotes/auth/check with ?remote=gdrive
	req, _ = http.NewRequest("POST", "/api/remotes/auth/check?remote=gdrive", nil)
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for scoped remote auth check, got: %d", rec.Code)
	}
	var scopedCheckResp models.AuthStatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &scopedCheckResp); err != nil {
		t.Fatalf("failed to decode scoped auth check response: %v", err)
	}
	// 9c. POST /api/remotes/dedupe
	dedupePayload, _ := json.Marshal(models.DedupeRequest{
		Remote: "gdrive",
		Path:   "Google Photos",
		Mode:   "list",
	})
	req, _ = http.NewRequest("POST", "/api/remotes/dedupe", strings.NewReader(string(dedupePayload)))
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for dedupe, got: %d", rec.Code)
	}
	var dedupeResp models.DedupeResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &dedupeResp); err != nil {
		t.Fatalf("failed to decode dedupe response: %v", err)
	}
	if dedupeResp.Status != "success" {
		t.Fatalf("expected success status for dedupe, got: %s (err: %s)", dedupeResp.Status, dedupeResp.Error)
	}

	// Reconnect cancel
	req, _ = http.NewRequest("POST", "/api/remotes/gdrive/reconnect/cancel", nil)
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for reconnect cancel, got: %d", rec.Code)
	}

	// 10. Nil AuthService branches
	nilAuthSrv := NewServer(cfg, syncSvc, embed.FS{})
	nilAuthSrv.AuthService = nil

	req, _ = http.NewRequest("POST", "/api/remotes/auth/check", nil)
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	nilAuthSrv.ServeHTTP(rec, req)

	req, _ = http.NewRequest("GET", "/api/remotes/auth", nil)
	rec = httptest.NewRecorder()
	nilAuthSrv.ServeHTTP(rec, req)

	req, _ = http.NewRequest("POST", "/api/remotes/gdrive/reconnect", nil)
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	nilAuthSrv.ServeHTTP(rec, req)

	req, _ = http.NewRequest("POST", "/api/remotes/gdrive/credentials", bytes.NewBuffer([]byte(`{}`)))
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	nilAuthSrv.ServeHTTP(rec, req)

	req, _ = http.NewRequest("POST", "/api/remotes/gdrive/validate", nil)
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	nilAuthSrv.ServeHTTP(rec, req)

	req, _ = http.NewRequest("GET", "/api/remotes/auth/audit", nil)
	rec = httptest.NewRecorder()
	nilAuthSrv.ServeHTTP(rec, req)

	req, _ = http.NewRequest("POST", "/api/remotes/auth/oauth-stop", nil)
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	nilAuthSrv.ServeHTTP(rec, req)

	// 10. Update credentials validation failure (empty client_id)
	req, _ = http.NewRequest("POST", "/api/remotes/gdrive/credentials", bytes.NewBuffer([]byte(`{"client_id":""}`)))
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !bytes.Contains(rec.Body.Bytes(), []byte("cannot be empty")) {
		t.Fatalf("expected error on empty client_id: %s", rec.Body.String())
	}
}

func TestReviewEndpoints(t *testing.T) {
	tmpDir := t.TempDir()
	localDir := filepath.Join(tmpDir, "local")
	backupsDir := filepath.Join(tmpDir, "backups")
	_ = os.MkdirAll(localDir, 0755)
	_ = os.MkdirAll(backupsDir, 0755)

	mockRCServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer mockRCServer.Close()

	rcClient := client.NewRcloneClient(mockRCServer.URL, "user", "pass")
	cfg := &config.Config{
		LocalPath:            localDir,
		RemotePath:           "gdrive:data",
		BaseDir:              tmpDir,
		PendingDeletionsFile: filepath.Join(tmpDir, "pending.json"),
		LogFile:              filepath.Join(tmpDir, "bisync.log"),
		ActiveReviewFile:     filepath.Join(tmpDir, "active_review_session.json"),
		ReviewDir:            tmpDir,
		ReviewBackupsDir:     backupsDir,
		ReviewBackupTTLHours: 24.0,
	}

	syncSvc := services.NewSyncService(cfg, rcClient)
	srv := NewServer(cfg, syncSvc, embed.FS{})

	// 1. GET /api/review/session when no session exists
	req, _ := http.NewRequest("GET", "/api/review/session", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for review session, got: %d", rec.Code)
	}
	var sessResp models.ReviewSessionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &sessResp); err != nil {
		t.Fatalf("failed to decode review session response: %v", err)
	}
	if sessResp.Active {
		t.Fatal("expected active to be false initially")
	}

	// 2. CSRF policy enforcement on review endpoints
	reviewEndpoints := []string{
		"/api/review/action",
		"/api/review/execute",
		"/api/review/rollback",
		"/api/review/dismiss",
	}
	for _, ep := range reviewEndpoints {
		req, _ = http.NewRequest("POST", ep, bytes.NewBuffer([]byte(`{}`)))
		rec = httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403 on %s without CSRF header, got: %d", ep, rec.Code)
		}
	}

	// 3. Create pending deletions and verify auto-discovery via GET /api/review/session
	localFile := filepath.Join(localDir, "test_file.txt")
	_ = os.WriteFile(localFile, []byte("hello world"), 0644)
	_ = os.WriteFile(cfg.PendingDeletionsFile, []byte(`{"aborted_at":123456,"has_deletions":true,"files":["test_file.txt"],"error":"Too many deletes"}`), 0644)

	req, _ = http.NewRequest("GET", "/api/review/session", nil)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for review session, got: %d", rec.Code)
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &sessResp); err != nil {
		t.Fatalf("failed to decode review session: %v", err)
	}
	if !sessResp.Active || sessResp.Session == nil {
		t.Fatalf("expected active review session, got: %+v", sessResp)
	}
	if len(sessResp.Session.Deletions) == 0 {
		t.Fatalf("expected deletion candidates discovered, got %d", len(sessResp.Session.Deletions))
	}
	candID := sessResp.Session.Deletions[0].ID
	sessionID := sessResp.Session.ID

	// 4. POST /api/review/action validation errors
	req, _ = http.NewRequest("POST", "/api/review/action", bytes.NewBuffer([]byte(`invalid-json`)))
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad action body, got: %d", rec.Code)
	}

	req, _ = http.NewRequest("POST", "/api/review/action", bytes.NewBuffer([]byte(`{"candidate_id":"c1"}`)))
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing decision, got: %d", rec.Code)
	}

	req, _ = http.NewRequest("POST", "/api/review/action", bytes.NewBuffer([]byte(`{"decision":"accept-delete"}`)))
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing candidate ID, got: %d", rec.Code)
	}

	// 5. POST /api/review/action valid single candidate decision staging
	actionPayload, _ := json.Marshal(models.StageDecisionRequest{
		SessionID:   sessionID,
		CandidateID: candID,
		Decision:    models.DecisionAcceptDelete,
	})
	req, _ = http.NewRequest("POST", "/api/review/action", bytes.NewBuffer(actionPayload))
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for action staging, got: %d", rec.Code)
	}
	var actResp models.StageDecisionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &actResp); err != nil {
		t.Fatalf("failed to decode action response: %v", err)
	}
	if actResp.Status != "success" || actResp.Session == nil {
		t.Fatalf("expected successful staging, got: %+v", actResp)
	}

	// 5b. POST /api/review/action staging with 'restore-local' alias
	aliasPayload, _ := json.Marshal(models.StageDecisionRequest{
		SessionID:   sessionID,
		CandidateID: candID,
		Decision:    "restore-local",
	})
	req, _ = http.NewRequest("POST", "/api/review/action", bytes.NewBuffer(aliasPayload))
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for restore-local alias staging, got: %d", rec.Code)
	}
	var aliasResp models.StageDecisionResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &aliasResp)
	if aliasResp.Session.Deletions[0].Decision != models.DecisionRestoreItem {
		t.Fatalf("expected restore-local alias to normalize to restore-item, got %s", aliasResp.Session.Deletions[0].Decision)
	}

	// 5c. POST /api/review/action staging with 'skip'
	skipPayload, _ := json.Marshal(models.StageDecisionRequest{
		SessionID:   sessionID,
		CandidateID: candID,
		Decision:    "skip",
	})
	req, _ = http.NewRequest("POST", "/api/review/action", bytes.NewBuffer(skipPayload))
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for skip staging, got: %d", rec.Code)
	}
	var skipResp models.StageDecisionResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &skipResp)
	if skipResp.Session.Deletions[0].Decision != models.DecisionSkip {
		t.Fatalf("expected skip decision staged, got %s", skipResp.Session.Deletions[0].Decision)
	}
	// 6. POST /api/review/action batch candidate decision staging
	batchPayload, _ := json.Marshal(models.StageDecisionRequest{
		SessionID:    sessionID,
		CandidateIDs: []string{candID},
		Decision:     models.DecisionAcceptDelete,
	})
	req, _ = http.NewRequest("POST", "/api/review/action", bytes.NewBuffer(batchPayload))
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for batch staging, got: %d", rec.Code)
	}

	// 7. POST /api/review/execute
	execPayload, _ := json.Marshal(models.ExecuteReviewRequest{
		SessionID: sessionID,
	})
	req, _ = http.NewRequest("POST", "/api/review/execute", bytes.NewBuffer(execPayload))
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for execute, got: %d", rec.Code)
	}
	var execResp models.ExecuteReviewResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &execResp); err != nil {
		t.Fatalf("failed to decode execute response: %v", err)
	}
	if execResp.Status != "success" || execResp.Applied != 1 {
		t.Fatalf("expected successful execution with 1 applied action, got: %+v", execResp)
	}

	// 8. POST /api/review/rollback with invalid session ID -> error payload
	req, _ = http.NewRequest("POST", "/api/review/rollback", bytes.NewBuffer([]byte(`{"session_id":"invalid"}`)))
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with error payload for invalid rollback, got: %d", rec.Code)
	}
	var rollResp models.RollbackReviewResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &rollResp)
	if rollResp.Status != "error" {
		t.Fatalf("expected error status for invalid rollback session, got: %+v", rollResp)
	}

	// 9. POST /api/review/dismiss
	_, _ = syncSvc.Review.DiscoverSession(context.Background(), "CONFLICT", "Conflict testing")
	dismissPayload, _ := json.Marshal(models.DismissReviewRequest{})
	req, _ = http.NewRequest("POST", "/api/review/dismiss", bytes.NewBuffer(dismissPayload))
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for dismiss, got: %d", rec.Code)
	}
	var dismResp models.ActionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &dismResp); err != nil || dismResp.Status != "success" {
		t.Fatalf("expected successful dismiss response, got: %+v", dismResp)
	}

	// 9b. POST /api/review/action-batch
	batchSess, _ := syncSvc.Review.DiscoverSession(context.Background(), "CONFLICT", "Conflict testing")
	batchPayload, _ = json.Marshal(models.BatchReviewActionRequest{
		SessionID:      batchSess.ID,
		MinConfidence:  0.70,
		MaxAllowedRisk: "LOW",
	})
	req, _ = http.NewRequest("POST", "/api/review/action-batch", bytes.NewBuffer(batchPayload))
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for action-batch, got: %d", rec.Code)
	}
	var batchResp models.BatchReviewActionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &batchResp); err != nil || batchResp.Status != "success" {
		t.Fatalf("expected successful batch response, got: %+v", batchResp)
	}

	// 10. Nil ReviewService paths
	nilReviewSrv := NewServer(cfg, syncSvc, embed.FS{})
	nilReviewSrv.ReviewService = nil

	req, _ = http.NewRequest("GET", "/api/review/session", nil)
	rec = httptest.NewRecorder()
	nilReviewSrv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for nil review session, got: %d", rec.Code)
	}

	req, _ = http.NewRequest("POST", "/api/review/action", bytes.NewBuffer([]byte(`{"candidate_id":"c1","decision":"accept-delete"}`)))
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	nilReviewSrv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for nil review action, got: %d", rec.Code)
	}

	req, _ = http.NewRequest("POST", "/api/review/execute", bytes.NewBuffer([]byte(`{}`)))
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	nilReviewSrv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for nil review execute, got: %d", rec.Code)
	}

	req, _ = http.NewRequest("POST", "/api/review/rollback", bytes.NewBuffer([]byte(`{}`)))
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	nilReviewSrv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for nil review rollback, got: %d", rec.Code)
	}

	req, _ = http.NewRequest("POST", "/api/review/dismiss", bytes.NewBuffer([]byte(`{}`)))
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	nilReviewSrv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for nil review dismiss, got: %d", rec.Code)
	}

	req, _ = http.NewRequest("POST", "/api/review/action-batch", bytes.NewBuffer([]byte(`{}`)))
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	nilReviewSrv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for nil review action-batch, got: %d", rec.Code)
	}
}

func TestReviewEndpoints_ExtendedErrorsAndEdgeCases(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		BaseDir:              tmpDir,
		LocalPath:            filepath.Join(tmpDir, "local"),
		RemotePath:           "gdrive:data",
		ActiveReviewFile:     filepath.Join(tmpDir, "review_session.json"),
		PendingDeletionsFile: filepath.Join(tmpDir, "pending.json"),
		ReviewDir:            tmpDir,
		ReviewBackupsDir:     filepath.Join(tmpDir, "backups"),
	}
	_ = os.MkdirAll(cfg.LocalPath, 0755)

	syncSvc := services.NewSyncService(cfg, nil)
	srv := NewServer(cfg, syncSvc, embed.FS{})

	// 1. When no active session exists:
	// a. Execute with empty session_id -> "No active review session found"
	req, _ := http.NewRequest("POST", "/api/review/execute", bytes.NewBuffer([]byte(`{"session_id":""}`)))
	req.Header.Set("X-Dashboard-Request", "1")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	var execResp models.ExecuteReviewResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &execResp)
	if execResp.Status != "error" || !strings.Contains(execResp.Error, "No active review session found") {
		t.Fatalf("expected error on empty session execute without active session, got: %+v", execResp)
	}

	// b. Rollback with empty session_id -> "No active review session found"
	req, _ = http.NewRequest("POST", "/api/review/rollback", bytes.NewBuffer([]byte(`{"session_id":""}`)))
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	var rollResp models.RollbackReviewResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &rollResp)
	if rollResp.Status != "error" || !strings.Contains(rollResp.Error, "No active review session found") {
		t.Fatalf("expected error on empty session rollback without active session, got: %+v", rollResp)
	}

	// c. Action with empty session_id -> "No active review session found"
	req, _ = http.NewRequest("POST", "/api/review/action", bytes.NewBuffer([]byte(`{"candidate_id":"c1","decision":"skip"}`)))
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	var stageResp models.StageDecisionResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &stageResp)
	if stageResp.Status != "error" || !strings.Contains(stageResp.Error, "No active review session found") {
		t.Fatalf("expected error on empty session action without active session, got: %+v", stageResp)
	}

	// d. Dismiss with empty session_id -> "No active review session found"
	req, _ = http.NewRequest("POST", "/api/review/dismiss", bytes.NewBuffer([]byte(`{"session_id":""}`)))
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	var dismResp models.ActionResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &dismResp)
	if dismResp.Status != "error" || !strings.Contains(dismResp.Error, "No active review session found") {
		t.Fatalf("expected error on empty session dismiss without active session, got: %+v", dismResp)
	}

	// e. Action-batch with empty session_id -> "No active review session found"
	req, _ = http.NewRequest("POST", "/api/review/action-batch", bytes.NewBuffer([]byte(`{"session_id":""}`)))
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	var batchResp models.BatchReviewActionResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &batchResp)
	if batchResp.Status != "error" || !strings.Contains(batchResp.Error, "No active review session found") {
		t.Fatalf("expected error on empty session action-batch without active session, got: %+v", batchResp)
	}

	// 2. Execute on non-existent session ID -> ApplyReviewSession failure
	req, _ = http.NewRequest("POST", "/api/review/execute", bytes.NewBuffer([]byte(`{"session_id":"nonexistent_123"}`)))
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	var execFailResp models.ExecuteReviewResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &execFailResp)
	if execFailResp.Status != "error" || !strings.Contains(execFailResp.Error, "not found or inactive") {
		t.Fatalf("expected error on nonexistent session execute, got: %+v", execFailResp)
	}

	// 3. Staging and batch errors with active session
	sess, err := syncSvc.Review.DiscoverSession(context.Background(), "CONFLICT", "Conflict error")
	if err != nil || sess == nil {
		t.Fatalf("failed discovering session: %v", err)
	}

	// Staging on non-existent candidate ID
	req, _ = http.NewRequest("POST", "/api/review/action", bytes.NewBuffer([]byte(fmt.Sprintf(`{"session_id":%q,"candidate_id":"missing_cand","decision":"skip"}`, sess.ID))))
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	var stageErrResp models.StageDecisionResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &stageErrResp)
	if stageErrResp.Status != "error" {
		t.Fatalf("expected error staging missing candidate, got: %+v", stageErrResp)
	}

	// Action-batch with wrong session ID
	req, _ = http.NewRequest("POST", "/api/review/action-batch", bytes.NewBuffer([]byte(`{"session_id":"wrong_sess"}`)))
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	var batchErrResp models.BatchReviewActionResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &batchErrResp)
	if batchErrResp.Status != "error" {
		t.Fatalf("expected error on action-batch with wrong session id, got: %+v", batchErrResp)
	}

	// Action-batch with empty session ID falling back to active session
	req, _ = http.NewRequest("POST", "/api/review/action-batch", bytes.NewBuffer([]byte(`{"session_id":""}`)))
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	// Execute with empty session ID falling back to active session
	req, _ = http.NewRequest("POST", "/api/review/execute", bytes.NewBuffer([]byte(`{"session_id":""}`)))
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	// Dismiss with nonexistent session ID returning error
	req, _ = http.NewRequest("POST", "/api/review/dismiss", bytes.NewBuffer([]byte(`{"session_id":"nonexistent_sess_99"}`)))
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	var dismErrResp models.ActionResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &dismErrResp)
	if dismErrResp.Status != "error" {
		t.Fatalf("expected error dismissing nonexistent session, got: %+v", dismErrResp)
	}

	// Dismiss current session so active memory session is cleared
	_ = syncSvc.Review.DismissReviewSession(sess.ID)

	// 4. Successful rollback through HTTP with saved session containing actions
	rollSess := &models.ReviewSession{
		ID:     "roll-test-sess",
		Status: models.ReviewStatusApplying,
		Actions: []models.ResolutionAction{
			{
				ID:       "act-1",
				ItemPath: "restored.txt",
				Status:   "completed",
				Type:     models.DecisionAcceptDelete,
			},
		},
	}
	_ = review.SaveActiveSession(cfg.ActiveReviewFile, rollSess)

	// Rollback with empty session_id targeting active session
	req, _ = http.NewRequest("POST", "/api/review/rollback", bytes.NewBuffer([]byte(`{"session_id":""}`)))
	req.Header.Set("X-Dashboard-Request", "1")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	var rollSuccessResp models.RollbackReviewResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &rollSuccessResp)
	if rollSuccessResp.Status != "success" || rollSuccessResp.Restored != 1 {
		t.Fatalf("expected successful rollback with 1 restored, got: %+v", rollSuccessResp)
	}

}
func TestServer_RootHandler_TemplateAssembly(t *testing.T) {
	cfg := &config.Config{BaseDir: t.TempDir()}
	syncSvc := services.NewSyncService(cfg, nil)

	// 1. Valid template structure with partials
	mockFS := fstest.MapFS{
		"dashboard/frontend/index.html": &fstest.MapFile{
			Data: []byte(`<!DOCTYPE html><html><body>{{ template "view_telemetry" . }}{{ template "modal_review" . }}</body></html>`),
		},
		"dashboard/frontend/partials/view_telemetry.html": &fstest.MapFile{
			Data: []byte(`{{ define "view_telemetry" }}<div id="telemetry-view">Telemetry Content</div>{{ end }}`),
		},
		"dashboard/frontend/partials/modals/modal_review.html": &fstest.MapFile{
			Data: []byte(`{{ define "modal_review" }}<div id="review-modal">Review Modal Content</div>{{ end }}`),
		},
	}

	srv := NewServer(cfg, syncSvc, mockFS)

	req, _ := http.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for root index, got: %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `<div id="telemetry-view">Telemetry Content</div>`) {
		t.Fatalf("expected assembled view_telemetry, got: %s", body)
	}
	if !strings.Contains(body, `<div id="review-modal">Review Modal Content</div>`) {
		t.Fatalf("expected assembled modal_review, got: %s", body)
	}

	// 2. Fallback when templates fail to parse (e.g. invalid syntax in partial)
	brokenFS := fstest.MapFS{
		"dashboard/frontend/index.html": &fstest.MapFile{
			Data: []byte(`<!DOCTYPE html><html><body>Raw Fallback Content</body></html>`),
		},
		"dashboard/frontend/partials/broken.html": &fstest.MapFile{
			Data: []byte(`{{ define "broken" }} {{ end `), // Syntax error
		},
	}
	brokenSrv := NewServer(cfg, syncSvc, brokenFS)
	req, _ = http.NewRequest("GET", "/", nil)
	rec = httptest.NewRecorder()
	brokenSrv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for fallback index.html, got: %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Raw Fallback Content") {
		t.Fatalf("expected raw fallback content, got: %s", rec.Body.String())
	}
}

func TestServer_RootHandler_RealFileSystem(t *testing.T) {
	cfg := &config.Config{BaseDir: t.TempDir()}
	syncSvc := services.NewSyncService(cfg, nil)

	// Use real filesystem from repository root
	realFS := os.DirFS("../..")
	srv := NewServer(cfg, syncSvc, realFS)

	req, _ := http.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for real root index, got: %d", rec.Code)
	}

	body := rec.Body.String()

	// Verify all views are assembled
	expectedViewMarkers := []string{
		"Engine: ",
		"Total Cloud Space",
		"Dual-Pane Directory Explorer",
		"Audit Ledger",
		"Cryptographically Verified",
	}
	for _, marker := range expectedViewMarkers {
		if !strings.Contains(body, marker) {
			t.Errorf("missing expected view marker in assembled template: %q", marker)
		}
	}

	// Verify modals are assembled
	expectedModalMarkers := []string{
		"Confirm Deletion",
		"Sync Review & Resolution",
		"Pending Deletions Review",
		"Loading correlated daemon logs...",
		"OAuth App Configuration",
		"Authorizing in Browser...",
		"OAuth Authentication Audit Log",
	}
	for _, marker := range expectedModalMarkers {
		if !strings.Contains(body, marker) {
			t.Errorf("missing expected modal marker in assembled template: %q", marker)
		}
	}

	// Verify valid SVG path and object iteration in assembled templates
	if strings.Contains(body, "1 1 0 .696 10.75") {
		t.Error("found corrupted SVG arc flag syntax '1 1 0 .696' in assembled template")
	}
	if !strings.Contains(body, "1 1 0 0 1 0 .696 10.75") {
		t.Error("expected valid Lucide eye SVG arc path in assembled template")
	}
	if strings.Contains(body, `x-for="(val, key) in entry.details"`) {
		t.Error("found invalid x-for loop over non-iterable Object in modal_daemon")
	}
	if !strings.Contains(body, `Object.entries(cluster.entry.details`) {
		t.Error("expected Object.entries iteration for cluster.entry.details in modal_daemon")
	}

	// Verify Enterprise branding removed and dynamic version injected
	if strings.Contains(body, "Enterprise") {
		t.Error("expected 'Enterprise' branding to be removed from header")
	}
	if !strings.Contains(body, "v1.0") {
		t.Errorf("expected dynamic version 'v1.0' in header, got body: %s", body[:500])
	}

	// Verify global x-cloak CSS style rule and x-cloak presence on views and modals
	if !strings.Contains(body, "[x-cloak] { display: none !important; }") {
		t.Error("expected global [x-cloak] CSS rule in head")
	}
	if cloakCount := strings.Count(body, "x-cloak"); cloakCount < 10 {
		t.Errorf("expected at least 10 x-cloak attributes on views/modals, found %d", cloakCount)
	}

	// Verify keyboard escape listeners across modals
	if escCount := strings.Count(body, "@keydown.escape.window"); escCount < 7 {
		t.Errorf("expected at least 7 @keydown.escape.window listeners on modals, found %d", escCount)
	}

	// Verify no unsupported Alpine array destructuring in templates
	if strings.Contains(body, `x-for="[`) {
		t.Error("found unsupported Alpine x-for array destructuring syntax")
	}
}

func TestFormatMajorMinor(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"v1.0.0", "v1.0"},
		{"v1.0.0-beta.1", "v1.0"},
		{"1.2.3", "v1.2"},
		{"v2.4", "v2.4"},
		{"v2.4.0", "v2.4"},
		{"dev", "v1.0"},
		{"", "v1.0"},
	}

	for _, tt := range tests {
		got := FormatMajorMinor(tt.input)
		if got != tt.expected {
			t.Errorf("FormatMajorMinor(%q) = %q; expected %q", tt.input, got, tt.expected)
		}
	}
}

func TestHandleBisyncLogs_Paginated(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "bisync.log")
	cfg := &config.Config{
		BaseDir: tmpDir,
		LogFile: logFile,
	}

	var sb strings.Builder
	for i := 1; i <= 60; i++ {
		sb.WriteString(fmt.Sprintf(`{"time":"2026-10-27T10:00:00Z","level":"info","msg":"sync run %d"}`+"\n", i))
	}
	_ = os.WriteFile(logFile, []byte(sb.String()), 0600)

	srv := NewServer(cfg, nil, embed.FS{})

	req := httptest.NewRequest("GET", "/api/logs/bisync?page=1&limit=50", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	var resp models.PaginatedLogsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Total != 60 || resp.TotalPages != 2 || len(resp.Logs) != 50 || resp.Page != 1 {
		t.Fatalf("unexpected paginated response: %+v", resp)
	}
}
func TestServer_ResolveActor(t *testing.T) {
	srv := &Server{}

	// Scenario 1: Default fallback to USER
	req1 := httptest.NewRequest("GET", "/api/status", nil)
	if got := srv.resolveActor(req1); got != "USER" {
		t.Fatalf("expected USER, got: %s", got)
	}

	// Scenario 2: X-Forwarded-User header (e.g. Authentik, Cloudflare Access)
	req2 := httptest.NewRequest("GET", "/api/status", nil)
	req2.Header.Set("X-Forwarded-User", "admin@internal.domain")
	if got := srv.resolveActor(req2); got != "admin@internal.domain" {
		t.Fatalf("expected admin@internal.domain, got: %s", got)
	}

	// Scenario 3: Remote-User header (e.g. Authelia, Apache/Nginx basic auth)
	req3 := httptest.NewRequest("GET", "/api/status", nil)
	req3.Header.Set("Remote-User", "operator1")
	if got := srv.resolveActor(req3); got != "operator1" {
		t.Fatalf("expected operator1, got: %s", got)
	}

	// Scenario 4: X-Forwarded-Email header
	req4 := httptest.NewRequest("GET", "/api/status", nil)
	req4.Header.Set("X-Forwarded-Email", "user@company.com")
	if got := srv.resolveActor(req4); got != "user@company.com" {
		t.Fatalf("expected user@company.com, got: %s", got)
	}

	// Priority: X-Forwarded-User precedes Remote-User
	req5 := httptest.NewRequest("GET", "/api/status", nil)
	req5.Header.Set("X-Forwarded-User", "primary_user")
	req5.Header.Set("Remote-User", "secondary_user")
	if got := srv.resolveActor(req5); got != "primary_user" {
		t.Fatalf("expected primary_user, got: %s", got)
	}
}

func TestServer_DashboardAPIKey(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		BaseDir:         tmpDir,
		DashboardAPIKey: "secret-vault-api-key-42",
	}

	srv := NewServer(cfg, nil, embed.FS{})

	// 1. GET requests should not require API key
	getReq := httptest.NewRequest("GET", "/api/status", nil)
	getRec := httptest.NewRecorder()
	srv.ServeHTTP(getRec, getReq)
	if getRec.Code == http.StatusUnauthorized {
		t.Fatal("GET requests should not be blocked by API key requirement")
	}

	// 2. Mutating POST request without API key -> 401 Unauthorized
	postReq1 := httptest.NewRequest("POST", "/api/sync", strings.NewReader(`{"action":"sync"}`))
	postRec1 := httptest.NewRecorder()
	srv.ServeHTTP(postRec1, postReq1)
	if postRec1.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for missing API key, got %d", postRec1.Code)
	}

	// 3. Mutating POST request with invalid X-API-Key -> 401 Unauthorized
	postReq2 := httptest.NewRequest("POST", "/api/sync", strings.NewReader(`{"action":"sync"}`))
	postReq2.Header.Set("X-API-Key", "wrong-key")
	postRec2 := httptest.NewRecorder()
	srv.ServeHTTP(postRec2, postReq2)
	if postRec2.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for invalid API key, got %d", postRec2.Code)
	}

	// 4. Mutating POST request with valid X-API-Key -> passes auth (not 401)
	postReq3 := httptest.NewRequest("POST", "/api/sync", strings.NewReader(`{"action":"sync"}`))
	postReq3.Header.Set("X-API-Key", "secret-vault-api-key-42")
	postRec3 := httptest.NewRecorder()
	srv.ServeHTTP(postRec3, postReq3)
	if postRec3.Code == http.StatusUnauthorized {
		t.Fatal("expected valid X-API-Key to pass authorization")
	}

	// 5. Mutating POST request with valid Authorization: Bearer <key> -> passes auth
	postReq4 := httptest.NewRequest("POST", "/api/sync", strings.NewReader(`{"action":"sync"}`))
	postReq4.Header.Set("Authorization", "Bearer secret-vault-api-key-42")
	postRec4 := httptest.NewRecorder()
	srv.ServeHTTP(postRec4, postReq4)
	if postRec4.Code == http.StatusUnauthorized {
		t.Fatal("expected valid Bearer token to pass authorization")
	}
}

type logRecordCapturer struct {
	records []slog.Record
}

func (c *logRecordCapturer) Enabled(context.Context, slog.Level) bool { return true }
func (c *logRecordCapturer) Handle(_ context.Context, r slog.Record) error {
	c.records = append(c.records, r)
	return nil
}
func (c *logRecordCapturer) WithAttrs(attrs []slog.Attr) slog.Handler { return c }
func (c *logRecordCapturer) WithGroup(name string) slog.Handler       { return c }

func TestHTTPAccessLogLevel(t *testing.T) {
	capturer := &logRecordCapturer{}
	oldLogger := slog.Default()
	slog.SetDefault(slog.New(capturer))
	defer slog.SetDefault(oldLogger)

	cfg := &config.Config{BaseDir: t.TempDir()}
	srv := NewServer(cfg, nil, embed.FS{})

	// 1. Routine read-only endpoint -> Debug level
	req := httptest.NewRequest("GET", "/api/telemetry", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	// 2. Read-only API query -> Debug level
	req2 := httptest.NewRequest("GET", "/api/config", nil)
	rec2 := httptest.NewRecorder()
	srv.ServeHTTP(rec2, req2)

	// 3. Mutating successful request -> Debug level
	req3 := httptest.NewRequest("POST", "/api/review/dismiss", strings.NewReader(`{}`))
	req3.Header.Set("X-Dashboard-Request", "1")
	rec3 := httptest.NewRecorder()
	srv.ServeHTTP(rec3, req3)

	// 4. Failed request (404) -> Warn level
	req4 := httptest.NewRequest("GET", "/api/nonexistent_endpoint", nil)
	rec4 := httptest.NewRecorder()
	srv.ServeHTTP(rec4, req4)

	var telemetryRecord, configRecord, mutateRecord, errorRecord *slog.Record
	for i := range capturer.records {
		r := &capturer.records[i]
		if r.Message == "http_access" {
			r.Attrs(func(a slog.Attr) bool {
				if a.Key == "path" && a.Value.String() == "/api/telemetry" {
					telemetryRecord = r
				}
				if a.Key == "path" && a.Value.String() == "/api/config" {
					configRecord = r
				}
				if a.Key == "path" && a.Value.String() == "/api/review/dismiss" {
					mutateRecord = r
				}
				if a.Key == "path" && a.Value.String() == "/api/nonexistent_endpoint" {
					errorRecord = r
				}
				return true
			})
		}
	}

	if telemetryRecord == nil || telemetryRecord.Level != slog.LevelDebug {
		t.Fatalf("expected /api/telemetry to be logged at Debug, got %+v", telemetryRecord)
	}
	if configRecord == nil || configRecord.Level != slog.LevelDebug {
		t.Fatalf("expected /api/config to be logged at Debug, got %+v", configRecord)
	}
	if mutateRecord == nil || mutateRecord.Level != slog.LevelDebug {
		t.Fatalf("expected /api/review/dismiss to be logged at Debug, got %+v", mutateRecord)
	}
	if errorRecord == nil || errorRecord.Level != slog.LevelWarn {
		t.Fatalf("expected /api/nonexistent_endpoint to be logged at Warn, got %+v", errorRecord)
	}
}
