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
)

func TestConfigEndpoints_GetAndUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		EnvFilePath: filepath.Join(tmpDir, "bisync.env"),
		GlobalDefaults: config.GlobalDefaults{
			MaxDelete:        40,
			CanaryValidation: true,
			ConflictResolve:  "newer",
			CompareMode:      "modtime",
			RemoveEmptyDirs:  true,
			CreateEmptyDirs:  true,
		},
		JobPollInitialSleepSec:  3,
		JobPollMaxSleepSec:      15,
		JobTimeoutSeconds:       1200,
		LockStaleTimeoutSeconds: 2400,
		RateLimitBackoffHours:   2,
		WebhookURL:              "https://example.com/webhook",
		WatchdogPingURL:         "https://example.com/watchdog",
	}

	srv := NewServer(cfg, nil, embed.FS{})

	// 1. GET /api/config
	reqGet, _ := http.NewRequest("GET", "/api/config", nil)
	recGet := httptest.NewRecorder()
	srv.ServeHTTP(recGet, reqGet)

	if recGet.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for GET /api/config, got %d", recGet.Code)
	}

	var getResp GlobalConfigResponse
	if err := json.NewDecoder(recGet.Body).Decode(&getResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if getResp.GlobalDefaults.MaxDelete != 40 || getResp.Daemon.JobTimeoutSeconds != 1200 {
		t.Fatalf("unexpected GET config values: %+v", getResp)
	}
	if getResp.Notifications.WebhookURL != "********" && getResp.Notifications.WebhookURL != "https://example.com/webhook" {
		t.Fatalf("unexpected webhook url: %s", getResp.Notifications.WebhookURL)
	}

	// 2. PUT /api/config with valid update
	newMaxDel := 25
	updatePayload := UpdateConfigRequest{
		GlobalDefaults: &config.GlobalDefaults{
			MaxDelete:        newMaxDel,
			CanaryValidation: false,
			ConflictResolve:  "larger",
			CompareMode:      "checksum",
			CustomFlags:      "--tpslimit 10 --checkers 8",
		},
		Daemon: &ConfigDaemonSettings{
			JobTimeoutSeconds: 3600,
		},
		Notifications: &ConfigNotificationSettings{
			WebhookURL: "https://example.com/new-webhook",
		},
	}
	body, _ := json.Marshal(updatePayload)
	reqPut, _ := http.NewRequest("PUT", "/api/config", bytes.NewBuffer(body))
	reqPut.Header.Set("X-Dashboard-Request", "1")
	reqPut.Header.Set("Origin", "http://localhost:5572")
	recPut := httptest.NewRecorder()
	srv.ServeHTTP(recPut, reqPut)

	if recPut.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for PUT /api/config, got %d: %s", recPut.Code, recPut.Body.String())
	}

	var putResp GlobalConfigResponse
	if err := json.NewDecoder(recPut.Body).Decode(&putResp); err != nil {
		t.Fatalf("failed to decode put response: %v", err)
	}
	if putResp.GlobalDefaults.MaxDelete != 25 || putResp.GlobalDefaults.ConflictResolve != "larger" || putResp.GlobalDefaults.CustomFlags != "--tpslimit 10 --checkers 8" {
		t.Fatalf("expected updated global defaults, got %+v", putResp.GlobalDefaults)
	}
	if putResp.Daemon.JobTimeoutSeconds != 3600 {
		t.Fatalf("expected updated job timeout 3600, got %d", putResp.Daemon.JobTimeoutSeconds)
	}
	if putResp.Notifications.WebhookURL != "********" && putResp.Notifications.WebhookURL != "https://example.com/new-webhook" {
		t.Fatalf("expected updated webhook url, got %s", putResp.Notifications.WebhookURL)
	}

	// 3. PUT /api/config with invalid MaxDelete (150 > 100) -> 400 Bad Request
	invalidPayload := UpdateConfigRequest{
		GlobalDefaults: &config.GlobalDefaults{
			MaxDelete: 150,
		},
	}
	badBody, _ := json.Marshal(invalidPayload)
	reqBad, _ := http.NewRequest("PUT", "/api/config", bytes.NewBuffer(badBody))
	reqBad.Header.Set("X-Dashboard-Request", "1")
	reqBad.Header.Set("Origin", "http://localhost:5572")
	recBad := httptest.NewRecorder()
	srv.ServeHTTP(recBad, reqBad)

	if recBad.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for max_delete > 100, got %d", recBad.Code)
	}

	// 4. PUT /api/config with CompareMode size,modtime,checksum -> 200 OK
	cmp3WayPayload := UpdateConfigRequest{
		GlobalDefaults: &config.GlobalDefaults{
			MaxDelete:   30,
			CompareMode: "size,modtime,checksum",
		},
	}
	cmp3Body, _ := json.Marshal(cmp3WayPayload)
	reqCmp3, _ := http.NewRequest("PUT", "/api/config", bytes.NewBuffer(cmp3Body))
	reqCmp3.Header.Set("X-Dashboard-Request", "1")
	reqCmp3.Header.Set("Origin", "http://localhost:5572")
	recCmp3 := httptest.NewRecorder()
	srv.ServeHTTP(recCmp3, reqCmp3)

	if recCmp3.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for compare_mode 'size,modtime,checksum', got %d: %s", recCmp3.Code, recCmp3.Body.String())
	}
}

func TestConfigEndpoints_BidirectionalEnvPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, "bisync.env")
	initialContent := `# Initial Bisync Environment
LOCAL_PATH=~/initial_local
REMOTE_PATH=initial_remote:
JOB_TIMEOUT_SECONDS=1800
MAX_LOG_SIZE_MB=5
DEBUG_MODE=false
`
	if err := os.WriteFile(envPath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("failed to write initial bisync.env: %v", err)
	}

	cfg := &config.Config{
		EnvFilePath: envPath,
		LocalPath:   "~/initial_local",
		RemotePath:  "initial_remote:",
		GlobalDefaults: config.GlobalDefaults{
			MaxDelete:        50,
			CanaryValidation: true,
			ConflictResolve:  "newer",
			CompareMode:      "size,modtime",
		},
		JobTimeoutSeconds: 1800,
		MaxLogSizeMB:      5,
		DebugMode:         false,
	}

	srv := NewServer(cfg, nil, embed.FS{})

	// 1. PUT update spanning all sections
	updatePayload := UpdateConfigRequest{
		GlobalDefaults: &config.GlobalDefaults{
			MaxDelete:          35,
			CanaryValidation:   true,
			ConflictResolve:    "larger",
			CompareMode:        "checksum",
			DefaultLocalPath:   "~/updated_local",
			DefaultRemotePath:  "updated_remote:",
			CustomFlags:        "--tpslimit 12",
			LockExplorerRemote: true,
		},
		Daemon: &ConfigDaemonSettings{
			JobTimeoutSeconds:       2400,
			LockStaleTimeoutSeconds: 7200,
			RateLimitBackoffHours:   2.5,
		},
		Logging: &ConfigLoggingSettings{
			MaxLogSizeMB:   10,
			LogBackupCount: 8,
		},
		Cleanup: &ConfigCleanupSettings{
			FilesToCleanup: "*.tmp, *.conflict",
		},
		Telemetry: &ConfigTelemetrySettings{
			EnableMetrics:     true,
			MetricsMaxEntries: 5000,
			MetricsMaxBackups: 10,
		},
		Notifications: &ConfigNotificationSettings{
			AppName:            "Bisync Fleet Manager",
			SuccessIntervalMin: 45,
			ErrorRepeatHours:   12,
			WebhookURL:         "https://example.com/alerts",
		},
		Features: &ConfigFeatureSettings{
			DebugMode:          true,
			MCPReadOnly:        true,
			StaleLockDetection: true,
		},
	}

	body, _ := json.Marshal(updatePayload)
	reqPut, _ := http.NewRequest("PUT", "/api/config", bytes.NewBuffer(body))
	reqPut.Header.Set("X-Dashboard-Request", "1")
	reqPut.Header.Set("Origin", "http://localhost:5572")
	recPut := httptest.NewRecorder()
	srv.ServeHTTP(recPut, reqPut)

	if recPut.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from PUT /api/config, got %d: %s", recPut.Code, recPut.Body.String())
	}

	var resp GlobalConfigResponse
	if err := json.NewDecoder(recPut.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.GlobalDefaults.MaxDelete != 35 || resp.GlobalDefaults.DefaultLocalPath != "~/updated_local" {
		t.Fatalf("unexpected response defaults: %+v", resp.GlobalDefaults)
	}
	if resp.Logging.MaxLogSizeMB != 10 || resp.Cleanup.FilesToCleanup != "*.tmp, *.conflict" {
		t.Fatalf("unexpected logging/cleanup response: %+v, %+v", resp.Logging, resp.Cleanup)
	}
	if !resp.Features.DebugMode || !resp.Features.MCPReadOnly {
		t.Fatalf("unexpected features response: %+v", resp.Features)
	}

	// 2. Verify disk persistence: parse bisync.env directly
	parsed, err := config.ParseEnvFile(envPath)
	if err != nil {
		t.Fatalf("failed to parse updated bisync.env: %v", err)
	}

	if parsed["LOCAL_PATH"] != "~/updated_local" {
		t.Errorf("expected LOCAL_PATH=~/updated_local, got %q", parsed["LOCAL_PATH"])
	}
	if parsed["REMOTE_PATH"] != "updated_remote:" {
		t.Errorf("expected REMOTE_PATH=updated_remote:, got %q", parsed["REMOTE_PATH"])
	}
	if parsed["GLOBAL_MAX_DELETE"] != "35" {
		t.Errorf("expected GLOBAL_MAX_DELETE=35, got %q", parsed["GLOBAL_MAX_DELETE"])
	}
	if parsed["GLOBAL_CONFLICT_RESOLVE"] != "larger" {
		t.Errorf("expected GLOBAL_CONFLICT_RESOLVE=larger, got %q", parsed["GLOBAL_CONFLICT_RESOLVE"])
	}
	if parsed["JOB_TIMEOUT_SECONDS"] != "2400" {
		t.Errorf("expected JOB_TIMEOUT_SECONDS=2400, got %q", parsed["JOB_TIMEOUT_SECONDS"])
	}
	if parsed["MAX_LOG_SIZE_MB"] != "10" {
		t.Errorf("expected MAX_LOG_SIZE_MB=10, got %q", parsed["MAX_LOG_SIZE_MB"])
	}
	if parsed["FILES_TO_CLEANUP"] != "*.tmp, *.conflict" {
		t.Errorf("expected FILES_TO_CLEANUP=*.tmp, *.conflict, got %q", parsed["FILES_TO_CLEANUP"])
	}
	if parsed["ENABLE_METRICS"] != "true" {
		t.Errorf("expected ENABLE_METRICS=true, got %q", parsed["ENABLE_METRICS"])
	}
	if parsed["NOTIFICATION_APP_NAME"] != "Bisync Fleet Manager" {
		t.Errorf("expected NOTIFICATION_APP_NAME='Bisync Fleet Manager', got %q", parsed["NOTIFICATION_APP_NAME"])
	}
	if parsed["DEBUG_MODE"] != "true" {
		t.Errorf("expected DEBUG_MODE=true, got %q", parsed["DEBUG_MODE"])
	}
	if parsed["MCP_READ_ONLY"] != "true" {
		t.Errorf("expected MCP_READ_ONLY=true, got %q", parsed["MCP_READ_ONLY"])
	}
}

