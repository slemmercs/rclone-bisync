package server

import (
	"bytes"
	"embed"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/slemmercs/rclone-bisync/internal/config"
)

func TestConfigEndpoints_GetAndUpdate(t *testing.T) {
	cfg := &config.Config{
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
	if putResp.GlobalDefaults.MaxDelete != 25 || putResp.GlobalDefaults.ConflictResolve != "larger" {
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
