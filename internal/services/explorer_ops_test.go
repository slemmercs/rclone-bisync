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

func TestExplorerOps_MoveItem_LocalToLocal(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	cfg := &config.Config{LocalPath: tmpDir}
	expSvc := NewExplorerService(cfg, nil, nil)

	srcFile := filepath.Join(tmpDir, "source.txt")
	_ = os.WriteFile(srcFile, []byte("content"), 0644)

	// Act
	resp, err := expSvc.MoveItem(context.Background(), models.MoveRequest{
		SrcRemote: "local",
		SrcPath:   "source.txt",
		DstRemote: "local",
		DstPath:   "target.txt",
	}, tmpDir)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "success" {
		t.Fatalf("expected status success, got %s: %s", resp.Status, resp.Error)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "target.txt")); os.IsNotExist(err) {
		t.Fatal("expected target file to exist after move")
	}
	if _, err := os.Stat(srcFile); !os.IsNotExist(err) {
		t.Fatal("expected source file to not exist after move")
	}
}

func TestExplorerOps_MoveItem_CanaryBlocked(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	cfg := &config.Config{LocalPath: tmpDir}
	expSvc := NewExplorerService(cfg, nil, nil)

	// 1. Cannot move canary file
	resp, _ := expSvc.MoveItem(context.Background(), models.MoveRequest{
		SrcRemote: "local",
		SrcPath:   "RCLONE_TEST",
		DstRemote: "local",
		DstPath:   "moved_canary",
	}, tmpDir)
	if resp.Status != "error" || resp.Error == "" {
		t.Fatalf("expected error moving canary, got %+v", resp)
	}

	// 2. Cannot move dir containing canary
	canaryDir := filepath.Join(tmpDir, "canary_folder")
	_ = os.MkdirAll(canaryDir, 0755)
	_ = os.WriteFile(filepath.Join(canaryDir, "RCLONE_TEST"), []byte("canary"), 0644)

	respDir, _ := expSvc.MoveItem(context.Background(), models.MoveRequest{
		SrcRemote: "local",
		SrcPath:   "canary_folder",
		DstRemote: "local",
		DstPath:   "target_folder",
		IsDir:     true,
	}, tmpDir)
	if respDir.Status != "error" || respDir.Error == "" {
		t.Fatalf("expected error moving folder with canary, got %+v", respDir)
	}
}

func TestExplorerOps_MoveItem_JailEscapeBlocked(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	cfg := &config.Config{LocalPath: tmpDir}
	expSvc := NewExplorerService(cfg, nil, nil)

	// Act
	resp, _ := expSvc.MoveItem(context.Background(), models.MoveRequest{
		SrcRemote: "local",
		SrcPath:   "../secret.txt",
		DstRemote: "local",
		DstPath:   "stolen.txt",
	}, tmpDir)

	// Assert
	if resp.Status != "error" {
		t.Fatalf("expected jail escape error, got status: %s", resp.Status)
	}
}

func TestExplorerOps_MoveItem_RemoteSuccess(t *testing.T) {
	// Arrange
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer ts.Close()

	rcClient := client.NewRcloneClient(ts.URL, "", "")
	expSvc := NewExplorerService(&config.Config{}, rcClient, nil)

	// Act
	resp, err := expSvc.MoveItem(context.Background(), models.MoveRequest{
		SrcRemote: "remote:",
		SrcPath:   "folder/file.txt",
		DstRemote: "remote:",
		DstPath:   "archive/file.txt",
	})

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "success" {
		t.Fatalf("expected success, got %s: %s", resp.Status, resp.Error)
	}
}

func TestExplorerOps_PurgeDir_LocalSuccess(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	cfg := &config.Config{LocalPath: tmpDir}
	expSvc := NewExplorerService(cfg, nil, nil)

	subDir := filepath.Join(tmpDir, "purge_me")
	_ = os.MkdirAll(subDir, 0755)
	_ = os.WriteFile(filepath.Join(subDir, "file.txt"), []byte("data"), 0644)

	// Act
	resp, err := expSvc.PurgeDir(context.Background(), models.PurgeRequest{
		Remote: "local",
		Path:   "purge_me",
	}, tmpDir)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "success" {
		t.Fatalf("expected success, got %s: %s", resp.Status, resp.Error)
	}
	if _, err := os.Stat(subDir); !os.IsNotExist(err) {
		t.Fatal("expected directory to be deleted")
	}
}

func TestExplorerOps_PurgeDir_CanaryBlocked(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	cfg := &config.Config{LocalPath: tmpDir}
	expSvc := NewExplorerService(cfg, nil, nil)

	// 1. Root purge forbidden
	respRoot, _ := expSvc.PurgeDir(context.Background(), models.PurgeRequest{
		Remote: "local",
		Path:   "",
	}, tmpDir)
	if respRoot.Status != "error" {
		t.Fatal("expected root purge to be rejected")
	}

	// 2. Directory containing canary cannot be purged
	canaryDir := filepath.Join(tmpDir, "protected_dir")
	_ = os.MkdirAll(canaryDir, 0755)
	_ = os.WriteFile(filepath.Join(canaryDir, "RCLONE_TEST"), []byte("canary"), 0644)

	respCanary, _ := expSvc.PurgeDir(context.Background(), models.PurgeRequest{
		Remote: "local",
		Path:   "protected_dir",
	}, tmpDir)
	if respCanary.Status != "error" {
		t.Fatal("expected canary directory purge to be rejected")
	}
}

func TestExplorerOps_PurgeDir_RemoteSuccess(t *testing.T) {
	// Arrange
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer ts.Close()

	rcClient := client.NewRcloneClient(ts.URL, "", "")
	expSvc := NewExplorerService(&config.Config{}, rcClient, nil)

	// Act
	resp, err := expSvc.PurgeDirectory(context.Background(), models.PurgeRequest{
		Remote: "remote:",
		Path:   "trash",
	})

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "success" {
		t.Fatalf("expected success, got %s: %s", resp.Status, resp.Error)
	}
}

func TestExplorerOps_GetPublicLink(t *testing.T) {
	// 1. Local unsupported
	expSvc := NewExplorerService(&config.Config{}, nil, nil)
	respLocal, err := expSvc.GetPublicLink(context.Background(), models.PublicLinkRequest{
		Remote: "local",
		Path:   "doc.pdf",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if respLocal.Supported || respLocal.Status != "error" {
		t.Fatalf("expected local to be unsupported, got %+v", respLocal)
	}

	// 2. Remote success
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"url": "https://drive.google.com/link"}`))
	}))
	defer ts.Close()

	rcClient := client.NewRcloneClient(ts.URL, "", "")
	expRemote := NewExplorerService(&config.Config{}, rcClient, nil)

	respRemote, err := expRemote.GetPublicLink(context.Background(), models.PublicLinkRequest{
		Remote: "remote:",
		Path:   "doc.pdf",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !respRemote.Supported || respRemote.URL != "https://drive.google.com/link" {
		t.Fatalf("expected remote public link success, got %+v", respRemote)
	}
}

