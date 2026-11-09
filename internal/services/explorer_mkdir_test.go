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
	"github.com/slemmercs/rclone-bisync/internal/models"
)

func TestExplorerService_MakeDir_Local(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	cfg := &config.Config{LocalPath: tmpDir}
	svc := NewExplorerService(cfg, nil, nil)
	ctx := context.Background()

	// Act: Create nested directory
	req := models.MkdirRequest{
		Remote: "local",
		Path:   "projects/new-app",
	}
	resp, err := svc.MakeDir(ctx, req, tmpDir)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "success" {
		t.Fatalf("expected status success, got %s: %s", resp.Status, resp.Error)
	}

	createdPath := filepath.Join(tmpDir, "projects", "new-app")
	fi, statErr := os.Stat(createdPath)
	if statErr != nil || !fi.IsDir() {
		t.Fatalf("expected directory to exist at %s, err: %v", createdPath, statErr)
	}
}

func TestExplorerService_MakeDir_EmptyPath(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	cfg := &config.Config{LocalPath: tmpDir}
	svc := NewExplorerService(cfg, nil, nil)
	ctx := context.Background()

	// Act
	req := models.MkdirRequest{
		Remote: "local",
		Path:   "   ",
	}
	resp, err := svc.MakeDir(ctx, req, tmpDir)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "error" || resp.Error == "" {
		t.Fatalf("expected error response for empty path, got %+v", resp)
	}
}

func TestExplorerService_MakeDir_CanaryRejection(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	cfg := &config.Config{LocalPath: tmpDir}
	svc := NewExplorerService(cfg, nil, nil)
	ctx := context.Background()

	// Act
	req := models.MkdirRequest{
		Remote: "local",
		Path:   "sub/RCLONE_TEST",
	}
	resp, err := svc.MakeDir(ctx, req, tmpDir)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "error" || resp.Error == "" {
		t.Fatalf("expected error response for canary name, got %+v", resp)
	}
}

func TestExplorerService_MakeDir_PathTraversal(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	cfg := &config.Config{LocalPath: tmpDir}
	svc := NewExplorerService(cfg, nil, nil)
	ctx := context.Background()

	// Act: Attempt jail escape
	req := models.MkdirRequest{
		Remote: "local",
		Path:   "../escaped_folder",
	}
	resp, err := svc.MakeDir(ctx, req, tmpDir)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "error" || resp.Error == "" {
		t.Fatalf("expected error response for jail escape, got %+v", resp)
	}
}

func TestExplorerService_MakeDir_Remote(t *testing.T) {
	// Arrange
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/operations/mkdir" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	rcClient := client.NewRcloneClient(ts.URL, "", "")
	svc := NewExplorerService(&config.Config{}, rcClient, nil)
	ctx := context.Background()

	// Act
	req := models.MkdirRequest{
		Remote: "myremote:",
		Path:   "remote/new_folder",
	}
	resp, err := svc.MakeDir(ctx, req)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "success" {
		t.Fatalf("expected status success, got %s: %s", resp.Status, resp.Error)
	}
}
