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

func TestCanaryService_SafeguardCanary(t *testing.T) {
	svc := NewCanaryService(&config.Config{}, nil)

	// Protected canary path
	if err := svc.SafeguardCanary("RCLONE_TEST"); err == nil {
		t.Fatal("expected error safeguarding RCLONE_TEST, got nil")
	}
	if err := svc.SafeguardCanary("subfolder/RCLONE_TEST"); err == nil {
		t.Fatal("expected error safeguarding subfolder/RCLONE_TEST, got nil")
	}

	// Normal files permitted
	if err := svc.SafeguardCanary("subfolder/normal_file.txt"); err != nil {
		t.Fatalf("expected nil for normal file, got: %v", err)
	}
}

func TestCanaryService_LocalCanaryLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subfolder")
	_ = os.MkdirAll(subDir, 0755)

	cfg := &config.Config{
		LocalPath: tmpDir,
	}
	svc := NewCanaryService(cfg, nil)
	ctx := context.Background()

	// 1. Canary does not exist initially
	if exists := svc.CanaryExists(ctx, "local", "subfolder"); exists {
		t.Fatal("expected CanaryExists to be false initially")
	}

	// 2. Traversal attempt blocked
	res, err := svc.CreateCanary(ctx, "local", "../outside")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res["status"] != "error" {
		t.Fatalf("expected traversal error, got: %v", res)
	}

	// 3. Nonexistent directory returns error
	res, err = svc.CreateCanary(ctx, "local", "missing_dir")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res["status"] != "error" {
		t.Fatalf("expected missing dir error, got: %v", res)
	}

	// 4. Create canary successfully
	res, err = svc.CreateCanary(ctx, "local", "subfolder")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res["status"] != "success" {
		t.Fatalf("expected success, got: %v", res)
	}

	// 5. CanaryExists now returns true
	if exists := svc.CanaryExists(ctx, "local", "subfolder"); !exists {
		t.Fatal("expected CanaryExists to be true after creation")
	}

	// 6. CanaryExists on root local
	if exists := svc.CanaryExists(ctx, "", ""); exists {
		t.Fatal("expected false for root when missing")
	}
}

func TestCanaryService_RemoteCanaryLifecycle(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/core/command":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": false, "result": ""})
		case "/operations/list":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"list": []map[string]interface{}{
					{"Name": "RCLONE_TEST", "IsDir": false, "Size": 0},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	rcClient := client.NewRcloneClient(srv.URL, "user", "pass")
	svc := NewCanaryService(&config.Config{}, rcClient)
	ctx := context.Background()

	// 1. Create canary on remote
	res, err := svc.CreateCanary(ctx, "gdrive", "documents")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res["status"] != "success" {
		t.Fatalf("expected remote creation success, got: %v", res)
	}

	// 2. CanaryExists on remote
	if exists := svc.CanaryExists(ctx, "gdrive", "documents"); !exists {
		t.Fatal("expected CanaryExists true on remote")
	}

	// 3. Client nil branch
	svcNilClient := NewCanaryService(&config.Config{}, nil)
	if exists := svcNilClient.CanaryExists(ctx, "gdrive", "documents"); exists {
		t.Fatal("expected false when rcClient is nil")
	}
	resNil, _ := svcNilClient.CreateCanary(ctx, "gdrive", "documents")
	if resNil["status"] != "error" {
		t.Fatalf("expected error when rcClient is nil, got: %v", resNil)
	}
}

func TestCanaryService_CanaryInDir_LocalAndRemote(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "folderA", "nested")
	_ = os.MkdirAll(subDir, 0755)

	cfg := &config.Config{
		LocalPath: tmpDir,
	}
	svc := NewCanaryService(cfg, nil)
	ctx := context.Background()

	// 1. CanaryInDir returns false when empty
	if hasCanary := svc.CanaryInDir(ctx, "local", "folderA"); hasCanary {
		t.Fatal("expected CanaryInDir to be false when no canary exists")
	}

	// 2. Traversal path returns false
	if hasCanary := svc.CanaryInDir(ctx, "local", "../outside"); hasCanary {
		t.Fatal("expected CanaryInDir to return false on path traversal attempt")
	}

	// 3. Create canary in nested subdirectory
	canaryPath := filepath.Join(subDir, CanaryFilename)
	_ = os.WriteFile(canaryPath, []byte("canary"), 0644)

	// CanaryInDir on folderA should find the nested canary
	if hasCanary := svc.CanaryInDir(ctx, "local", "folderA"); !hasCanary {
		t.Fatal("expected CanaryInDir to find nested canary in folderA")
	}

	// CanaryInDir on root local should also find it
	if hasCanary := svc.CanaryInDir(ctx, "", ""); !hasCanary {
		t.Fatal("expected CanaryInDir to find canary from root local")
	}

	// 4. Remote CanaryInDir with mock server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/operations/list" {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"list": []map[string]interface{}{
					{"Name": "RCLONE_TEST", "IsDir": false, "Size": 10},
				},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	rcClient := client.NewRcloneClient(srv.URL, "", "")
	svcRemote := NewCanaryService(cfg, rcClient)

	if hasCanary := svcRemote.CanaryInDir(ctx, "remoteDrive", "folderB"); !hasCanary {
		t.Fatal("expected CanaryInDir to return true for remote listing with canary")
	}

	// Remote with nil client
	svcRemoteNil := NewCanaryService(cfg, nil)
	if hasCanary := svcRemoteNil.CanaryInDir(ctx, "remoteDrive", "folderB"); hasCanary {
		t.Fatal("expected CanaryInDir to return false for nil remote client")
	}
}

func TestCanaryService_DirectoryTrapAndRemoteErrors(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "trapDir")
	_ = os.MkdirAll(subDir, 0755)

	// Create directory named RCLONE_TEST instead of a file
	dirCanary := filepath.Join(subDir, CanaryFilename)
	_ = os.MkdirAll(dirCanary, 0755)

	cfg := &config.Config{
		LocalPath: tmpDir,
	}
	svc := NewCanaryService(cfg, nil)
	ctx := context.Background()

	// 1. CanaryExists returns false because it is a directory, not a file
	if exists := svc.CanaryExists(ctx, "local", "trapDir"); exists {
		t.Fatal("expected CanaryExists to return false when RCLONE_TEST is a directory")
	}

	// 2. CanaryExists traversal returns false
	if exists := svc.CanaryExists(ctx, "local", "../outside"); exists {
		t.Fatal("expected CanaryExists to return false for path traversal")
	}

	// 3. Remote creation failure (server returns 500)
	srvFail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer srvFail.Close()

	rcFailClient := client.NewRcloneClient(srvFail.URL, "", "")
	svcFail := NewCanaryService(cfg, rcFailClient)

	res, err := svcFail.CreateCanary(ctx, "remoteDrive", "docs")
	if err != nil {
		t.Fatalf("unexpected error return: %v", err)
	}
	if res["status"] != "error" {
		t.Fatalf("expected error status on remote server failure, got: %v", res)
	}

	// 4. Remote root canary creation (cleanPath == "")
	srvSuccess := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/core/command" {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": false, "result": ""})
			return
		}
		http.NotFound(w, r)
	}))
	defer srvSuccess.Close()

	rcSuccessClient := client.NewRcloneClient(srvSuccess.URL, "", "")
	svcSuccess := NewCanaryService(cfg, rcSuccessClient)

	resRoot, err := svcSuccess.CreateCanary(ctx, "remoteDrive", "")
	if err != nil {
		t.Fatalf("unexpected error return: %v", err)
	}
	if resRoot["status"] != "success" {
		t.Fatalf("expected success creating root remote canary, got: %v", resRoot)
	}
}

func TestCanaryService_TildeAndFallbackAndPermissions(t *testing.T) {
	ctx := context.Background()

	// 1. Tilde path expansion
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		tmpUnderHome := filepath.Join(home, ".test_rclone_canary_tmp")
		_ = os.MkdirAll(tmpUnderHome, 0755)
		defer os.RemoveAll(tmpUnderHome)

		cfgTilde := &config.Config{
			LocalPath: "~/.test_rclone_canary_tmp",
		}
		svcTilde := NewCanaryService(cfgTilde, nil)

		// Create canary with tilde path
		res, err := svcTilde.CreateCanary(ctx, "local", "")
		if err != nil || res["status"] != "success" {
			t.Fatalf("expected success with tilde path, got: %v, err: %v", res, err)
		}

		// CanaryExists with tilde path
		if exists := svcTilde.CanaryExists(ctx, "local", ""); !exists {
			t.Fatal("expected CanaryExists true with tilde path")
		}
	}

	// 2. Empty LocalPath fallback
	cfgEmpty := &config.Config{
		LocalPath: "",
	}
	svcEmpty := NewCanaryService(cfgEmpty, nil)
	_ = svcEmpty.CanaryExists(ctx, "local", "some_dir")
	_ = svcEmpty.CanaryInDir(ctx, "local", "some_dir")

	// 3. Permission denied on file open
	tmpDir := t.TempDir()
	roDir := filepath.Join(tmpDir, "readonly")
	_ = os.MkdirAll(roDir, 0555) // read & execute only, no write
	defer os.Chmod(roDir, 0755)

	cfgRO := &config.Config{
		LocalPath: tmpDir,
	}
	svcRO := NewCanaryService(cfgRO, nil)
	resRO, _ := svcRO.CreateCanary(ctx, "local", "readonly")
	if resRO["status"] != "error" {
		t.Fatalf("expected error on readonly dir, got: %v", resRO)
	}
}
