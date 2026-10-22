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
		case "/operations/touch":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{})
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
