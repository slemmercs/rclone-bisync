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

func TestExplorerService_IsLocalPathJailed(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "jail-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	subDir := filepath.Join(tmpDir, "allowed")
	_ = os.MkdirAll(subDir, 0755)

	// Valid path inside jail
	base, target, ok := IsLocalPathJailed(tmpDir, "allowed")
	if !ok || base == "" || target != subDir {
		t.Fatalf("expected valid path within jail, got ok=%v, base=%s, target=%s", ok, base, target)
	}

	// Alias isLocalPathJailed test
	_, _, okAlias := isLocalPathJailed(tmpDir, "allowed")
	if !okAlias {
		t.Fatal("expected isLocalPathJailed alias to succeed")
	}

	// Traversal outside jail
	_, _, okEscape := IsLocalPathJailed(tmpDir, "../../../etc/passwd")
	if okEscape {
		t.Fatal("expected traversal attempt to be rejected")
	}

	// Traversal using dot-dot within path
	_, _, okSubEscape := IsLocalPathJailed(tmpDir, "allowed/../../..")
	if okSubEscape {
		t.Fatal("expected relative escape to be rejected")
	}
}

func TestExplorerService_BrowsePath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "browse-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create subfolder, canary, and dummy file
	subDir := filepath.Join(tmpDir, "docs")
	_ = os.MkdirAll(subDir, 0755)
	_ = os.WriteFile(filepath.Join(subDir, "RCLONE_TEST"), []byte("canary"), 0644)
	_ = os.WriteFile(filepath.Join(subDir, "notes.txt"), []byte("hello"), 0644)

	cfg := &config.Config{LocalPath: tmpDir}
	expSvc := NewExplorerService(cfg, nil, nil)

	// 1. Valid local directory browsing
	res := expSvc.BrowsePath(context.Background(), "local", "docs")
	if res.Error != "" {
		t.Fatalf("unexpected browse error: %s", res.Error)
	}
	if !res.HasCanary {
		t.Fatal("expected canary to be detected")
	}
	if len(res.Items) != 2 {
		t.Fatalf("expected 2 items, got: %d", len(res.Items))
	}

	// 2. Traversal attempt forbidden
	resTraversal := expSvc.BrowsePath(context.Background(), "local", "../../etc")
	if resTraversal.Error != "Directory traversal forbidden" {
		t.Fatalf("expected 'Directory traversal forbidden', got: %s", resTraversal.Error)
	}

	// 3. Non-existent directory
	resMissing := expSvc.BrowsePath(context.Background(), "local", "missing_dir")
	if resMissing.Error == "" {
		t.Fatal("expected error browsing missing directory")
	}

	// 4. File is not a directory
	resFile := expSvc.BrowsePath(context.Background(), "local", "docs/notes.txt")
	if resFile.Error != "Target is not a directory" {
		t.Fatalf("expected 'Target is not a directory', got: %s", resFile.Error)
	}

	// 5. Remote browsing with nil client
	resNilClient := expSvc.BrowsePath(context.Background(), "remote:", "sub")
	if resNilClient.Error != "client not configured" {
		t.Fatalf("expected 'client not configured', got: %s", resNilClient.Error)
	}

	// 6. Remote browsing with mock client
	mockSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"list":[{"Path":"file.txt","Name":"file.txt","Size":123,"IsDir":false}]}`))
	}))
	defer mockSrv.Close()

	rcClient := client.NewRcloneClient(mockSrv.URL, "user", "pass")
	expWithClient := NewExplorerService(cfg, rcClient, nil)

	resRemote := expWithClient.BrowsePath(context.Background(), "gdrive", "folder")
	if resRemote.Error != "" {
		t.Fatalf("unexpected remote browse error: %s", resRemote.Error)
	}
	if len(resRemote.Items) != 1 {
		t.Fatalf("expected 1 remote item, got: %d", len(resRemote.Items))
	}
}
func TestExplorerService_CopyItem(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "copy-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	_ = os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content1"), 0644)
	subDir := filepath.Join(tmpDir, "folder")
	_ = os.MkdirAll(subDir, 0755)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/operations/copyfile":
			w.WriteHeader(http.StatusOK)
		case "/sync/copy":
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	rcClient := client.NewRcloneClient(srv.URL, "user", "pass")
	cfg := &config.Config{LocalPath: tmpDir}
	expSvc := NewExplorerService(cfg, rcClient, nil)

	// 1. Source traversal forbidden
	resTrav, _ := expSvc.CopyItem(context.Background(), "local", "../../../etc/passwd", "local", "out.txt", false)
	if resTrav["status"] != "error" || resTrav["error"] != "Source path traversal forbidden" {
		t.Fatalf("expected source traversal forbidden, got: %v", resTrav)
	}

	// 2. Destination traversal forbidden
	resDstTrav, _ := expSvc.CopyItem(context.Background(), "local", "file1.txt", "local", "../out.txt", false)
	if resDstTrav["status"] != "error" || resDstTrav["error"] != "Destination path traversal forbidden" {
		t.Fatalf("expected destination traversal forbidden, got: %v", resDstTrav)
	}

	// 3. Source missing
	resMissing, _ := expSvc.CopyItem(context.Background(), "local", "non_existent.txt", "local", "out.txt", false)
	if resMissing["status"] != "error" {
		t.Fatalf("expected error for missing source, got: %v", resMissing)
	}

	// 4. Local-to-local file copy
	resOk, err := expSvc.CopyItem(context.Background(), "local", "file1.txt", "local", "file1_copy.txt", false)
	if err != nil || resOk["status"] != "success" {
		t.Fatalf("expected copy success, got: %v, err: %v", resOk, err)
	}

	// 5. Local-to-remote directory copy
	resDir, err := expSvc.CopyItem(context.Background(), "local", "folder", "gdrive", "sub/folder", true)
	if err != nil || resDir["status"] != "success" {
		t.Fatalf("expected copydir success: %v, err: %v", resDir, err)
	}

	// 6. Nil client error check
	expNil := NewExplorerService(cfg, nil, nil)
	resNil, _ := expNil.CopyItem(context.Background(), "gdrive", "file.txt", "local", "file.txt", false)
	if resNil["status"] != "error" || resNil["error"] != "client not configured" {
		t.Fatalf("expected client not configured, got: %v", resNil)
	}
}

func TestExplorerService_DeleteItem(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "del-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	_ = os.WriteFile(filepath.Join(tmpDir, "del.txt"), []byte("delete me"), 0644)
	subDir := filepath.Join(tmpDir, "deldir")
	_ = os.MkdirAll(subDir, 0755)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/operations/deletefile":
			w.WriteHeader(http.StatusOK)
		case "/operations/purge":
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	rcClient := client.NewRcloneClient(srv.URL, "user", "pass")
	cfg := &config.Config{LocalPath: tmpDir}
	expSvc := NewExplorerService(cfg, rcClient, nil)

	// 1. Root or empty path deletion forbidden
	resRoot, _ := expSvc.DeleteItem(context.Background(), "local", "", false)
	if resRoot["status"] != "error" || resRoot["error"] != "Root directory cannot be deleted" {
		t.Fatalf("expected root deletion error, got: %v", resRoot)
	}

	// 2. Canary file protected
	resCanary, _ := expSvc.DeleteItem(context.Background(), "local", "RCLONE_TEST", false)
	if resCanary["status"] != "error" || resCanary["error"] != "Canary test file (RCLONE_TEST) cannot be deleted" {
		t.Fatalf("expected canary deletion error, got: %v", resCanary)
	}

	// 3. Traversal or root deletion forbidden
	resTrav, _ := expSvc.DeleteItem(context.Background(), "local", "../../", false)
	if resTrav["status"] != "error" {
		t.Fatalf("expected traversal forbidden, got: %v", resTrav)
	}

	// 4. Missing target
	resMissing, _ := expSvc.DeleteItem(context.Background(), "local", "not_there.txt", false)
	if resMissing["status"] != "error" {
		t.Fatalf("expected missing target error, got: %v", resMissing)
	}

	// 5. Successful local file deletion
	resFile, err := expSvc.DeleteItem(context.Background(), "local", "del.txt", false)
	if err != nil || resFile["status"] != "success" {
		t.Fatalf("expected file delete success, got: %v, err: %v", resFile, err)
	}

	// 6. Successful local directory deletion
	resDir, err := expSvc.DeleteItem(context.Background(), "local", "deldir", true)
	if err != nil || resDir["status"] != "success" {
		t.Fatalf("expected dir delete success, got: %v, err: %v", resDir, err)
	}

	// 7. Remote file deletion
	resRemFile, err := expSvc.DeleteItem(context.Background(), "gdrive", "remote.txt", false)
	if err != nil || resRemFile["status"] != "success" {
		t.Fatalf("expected remote file delete success, got: %v, err: %v", resRemFile, err)
	}

	// 8. Remote dir purge
	resRemPurge, err := expSvc.DeleteItem(context.Background(), "gdrive", "remotedir", true)
	if err != nil || resRemPurge["status"] != "success" {
		t.Fatalf("expected remote dir purge success, got: %v, err: %v", resRemPurge, err)
	}

	// 9. Nil client error handling
	expNil := NewExplorerService(cfg, nil, nil)
	resNil, _ := expNil.DeleteItem(context.Background(), "gdrive", "test.txt", false)
	if resNil["status"] != "error" || resNil["error"] != "client not configured" {
		t.Fatalf("expected client not configured, got: %v", resNil)
	}
}

