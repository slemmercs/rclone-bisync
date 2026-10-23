package server

import (
	"bytes"
	"context"
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
	"github.com/slemmercs/rclone-bisync/internal/core/client"
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

}
