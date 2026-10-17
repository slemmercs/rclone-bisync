package services

import (
"context"
"net/http"
"net/http/httptest"
"os"
"path/filepath"
"testing"

"github.com/slemmercs/rclone-bisync/internal/config"
"github.com/slemmercs/rclone-bisync/internal/core/client"
)

func TestBrowsePathLocalJailAndCanary(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "jail-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create subfolder and canary file
	subDir := filepath.Join(tmpDir, "docs")
	_ = os.MkdirAll(subDir, 0755)
	canaryPath := filepath.Join(subDir, "RCLONE_TEST")
	_ = os.WriteFile(canaryPath, []byte("ok"), 0644)
	regularPath := filepath.Join(subDir, "report.pdf")
	_ = os.WriteFile(regularPath, []byte("pdf-data"), 0644)

	cfg := &config.Config{
		LocalPath: tmpDir,
	}
	svc := NewSyncService(cfg, nil)

	// 1. Browse normal path
	listing := svc.BrowsePath(context.Background(), "local", "docs")
	if listing.Error != "" {
		t.Fatalf("unexpected browse error: %s", listing.Error)
	}
	if !listing.HasCanary {
		t.Fatal("expected canary to be detected")
	}
	if len(listing.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(listing.Items))
	}

	// 2. Traversal attempt forbidden
	traversal := svc.BrowsePath(context.Background(), "local", "../../../etc")
	if traversal.Error != "Directory traversal forbidden" {
		t.Fatalf("expected Directory traversal forbidden, got: %s", traversal.Error)
	}
}

func TestCanaryCreationLocal(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "canary-create-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		LocalPath: tmpDir,
	}
	svc := NewSyncService(cfg, nil)

	res, err := svc.CreateCanary(context.Background(), "local", "")
	if err != nil || res["status"] != "success" {
		t.Fatalf("expected success creating canary, got: %v, err: %v", res, err)
	}

	// Check file exists
	if _, err := os.Stat(filepath.Join(tmpDir, "RCLONE_TEST")); os.IsNotExist(err) {
		t.Fatal("expected RCLONE_TEST to exist on disk")
	}
}

func TestDeleteCanaryProtection(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "canary-protect-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		LocalPath: tmpDir,
	}
	svc := NewSyncService(cfg, nil)

	res, _ := svc.DeleteItem(context.Background(), "local", "RCLONE_TEST", false)
	if res["status"] != "error" || res["error"] != "Canary test file (RCLONE_TEST) cannot be deleted" {
		t.Fatalf("expected canary deletion rejection, got: %v", res)
	}
}

func TestTailFileChronological(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tail-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logFile := filepath.Join(tmpDir, "test.log")
	content := "line 1\nline 2\nline 3\nline 4\nline 5\n"
	_ = os.WriteFile(logFile, []byte(content), 0644)

	lines := TailFile(logFile, 3)
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %v", len(lines), lines)
	}
	if lines[0] != "line 3" || lines[1] != "line 4" || lines[2] != "line 5" {
		t.Fatalf("expected lines 3, 4, 5 in order, got: %v", lines)
	}
}

func TestGetDaemonHealthLive(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
if r.URL.Path == "/core/stats" {
w.WriteHeader(http.StatusOK)
_, _ = w.Write([]byte(`{"bytes": 100}`))
return
}
w.WriteHeader(http.StatusNotFound)
}))
	defer server.Close()

	rcClient := client.NewRcloneClient(server.URL, "user", "pass")
	cfg := &config.Config{
		RCAPIURL: server.URL,
	}
	svc := NewSyncService(cfg, rcClient)

	health := svc.GetDaemonHealth(context.Background())
	if health["status"] != "healthy" {
		t.Fatalf("expected healthy daemon, got: %v", health["status"])
	}
}
