package server

import (
	"embed"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/app"
	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/core/client"
	"github.com/slemmercs/rclone-bisync/internal/models"
)

func TestHandleDiagnostics_OfflineDaemon(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "bisync.log")
	_ = os.WriteFile(logPath, []byte("log data line\n"), 0644)

	cfg := &config.Config{
		BaseDir: tmpDir,
		LogFile: logPath,
	}

	srv := NewServer(cfg, nil, embed.FS{})
	srv.Version = "v1.2.3"

	req, _ := http.NewRequest(http.MethodGet, "/api/diagnostics", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	var resp models.DiagnosticsResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}

	if resp.Status != "error" {
		t.Errorf("expected overall status 'error' when daemon offline, got '%s'", resp.Status)
	}

	if resp.Summary["fail"] < 1 {
		t.Errorf("expected at least 1 failure in summary, got %+v", resp.Summary)
	}

	// Verify System Info
	if resp.SystemInfo.Version != "v1.2.3" {
		t.Errorf("expected version v1.2.3, got '%s'", resp.SystemInfo.Version)
	}
	if resp.SystemInfo.PID != os.Getpid() {
		t.Errorf("expected PID %d, got %d", os.Getpid(), resp.SystemInfo.PID)
	}
	if resp.SystemInfo.OS != runtime.GOOS || resp.SystemInfo.Arch != runtime.GOARCH {
		t.Errorf("unexpected OS/Arch: %s/%s", resp.SystemInfo.OS, resp.SystemInfo.Arch)
	}
	if _, err := time.Parse(time.RFC3339, resp.Timestamp); err != nil {
		t.Errorf("expected valid RFC3339 timestamp, got '%s': %v", resp.Timestamp, err)
	}

	// Verify Check Categories and IDs
	checkMap := make(map[string]models.DiagnosticCheck)
	for _, c := range resp.Checks {
		checkMap[c.ID] = c
	}

	daemonCheck, exists := checkMap["daemon_connectivity"]
	if !exists {
		t.Fatal("missing daemon_connectivity check")
	}
	if daemonCheck.Status != models.DiagnosticStatusFail {
		t.Errorf("expected daemon fail status, got %s", daemonCheck.Status)
	}

	baseCheck, exists := checkMap["storage_base_dir"]
	if !exists {
		t.Fatal("missing storage_base_dir check")
	}
	if baseCheck.Status != models.DiagnosticStatusPass {
		t.Errorf("expected storage pass status, got %s", baseCheck.Status)
	}

	canaryCheck, exists := checkMap["safety_canary_guard"]
	if !exists {
		t.Fatal("missing safety_canary_guard check")
	}
	if canaryCheck.Status != models.DiagnosticStatusPass {
		t.Errorf("expected canary pass status, got %s", canaryCheck.Status)
	}

	sysCheck, exists := checkMap["system_runtime"]
	if !exists {
		t.Fatal("missing system_runtime check")
	}
	if sysCheck.Status != models.DiagnosticStatusPass {
		t.Errorf("expected system runtime pass status, got %s", sysCheck.Status)
	}
}

func TestHandleDiagnostics_HealthyDaemonAndRemotes(t *testing.T) {
	tmpDir := t.TempDir()

	mockRC := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/core/stats":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"bytes": 0, "speed": 0})
		case "/config/listremotes":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"remotes": []string{"test-drive:"}})
		case "/core/version":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"version": "v1.66.0"})
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	defer mockRC.Close()

	rcClient := client.NewRcloneClient(mockRC.URL, "", "")
	cfg := &config.Config{
		BaseDir: tmpDir,
		LogFile: filepath.Join(tmpDir, "missing.log"),
	}

	container, err := app.NewContainer(cfg, app.WithClient(rcClient))
	if err != nil {
		t.Fatalf("failed to create container: %v", err)
	}

	srv := NewServerWithContainer(container, embed.FS{})

	req, _ := http.NewRequest(http.MethodGet, "/api/diagnostics", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	var resp models.DiagnosticsResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	checkMap := make(map[string]models.DiagnosticCheck)
	for _, c := range resp.Checks {
		checkMap[c.ID] = c
	}

	if daemonCheck := checkMap["daemon_connectivity"]; daemonCheck.Status != models.DiagnosticStatusPass {
		t.Errorf("expected healthy daemon pass status, got %s", daemonCheck.Status)
	}

	if logCheck := checkMap["storage_log_file"]; logCheck.Status != models.DiagnosticStatusPass {
		t.Errorf("expected missing log file to pass with creation notice, got %s", logCheck.Status)
	}
}

func TestHandleDiagnostics_ElevatedLatencyWarning(t *testing.T) {
	tmpDir := t.TempDir()

	mockRC := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/core/stats" {
			time.Sleep(520 * time.Millisecond)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"bytes": 0})
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer mockRC.Close()

	rcClient := client.NewRcloneClient(mockRC.URL, "", "")
	cfg := &config.Config{
		BaseDir: tmpDir,
	}

	container, err := app.NewContainer(cfg, app.WithClient(rcClient))
	if err != nil {
		t.Fatalf("failed to create container: %v", err)
	}

	srv := NewServerWithContainer(container, embed.FS{})

	req, _ := http.NewRequest(http.MethodGet, "/api/diagnostics", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	var resp models.DiagnosticsResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	var daemonCheck *models.DiagnosticCheck
	for _, c := range resp.Checks {
		if c.ID == "daemon_connectivity" {
			daemonCheck = &c
			break
		}
	}

	if daemonCheck == nil {
		t.Fatal("missing daemon_connectivity check")
	}
	if daemonCheck.Status != models.DiagnosticStatusWarning {
		t.Errorf("expected warning status for elevated latency, got %s", daemonCheck.Status)
	}
}
