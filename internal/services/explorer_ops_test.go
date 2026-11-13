package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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
	var receivedPayload map[string]interface{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewDecoder(r.Body).Decode(&receivedPayload)
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
	if receivedPayload["srcFs"] != "remote:folder" {
		t.Fatalf("expected srcFs to be 'remote:folder', got: %v", receivedPayload["srcFs"])
	}
	if receivedPayload["srcRemote"] != "file.txt" {
		t.Fatalf("expected srcRemote to be 'file.txt', got: %v", receivedPayload["srcRemote"])
	}
	if receivedPayload["dstFs"] != "remote:archive" {
		t.Fatalf("expected dstFs to be 'remote:archive', got: %v", receivedPayload["dstFs"])
	}
	if receivedPayload["dstRemote"] != "file.txt" {
		t.Fatalf("expected dstRemote to be 'file.txt', got: %v", receivedPayload["dstRemote"])
	}
}

func TestExplorerOps_MoveItem_RemoteRenameInSameDir(t *testing.T) {
	// Arrange
	var receivedPayload map[string]interface{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewDecoder(r.Body).Decode(&receivedPayload)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer ts.Close()

	rcClient := client.NewRcloneClient(ts.URL, "", "")
	expSvc := NewExplorerService(&config.Config{}, rcClient, nil)

	// Act: rename test1/archive_info.json to test1/archive_info_v2.json
	resp, err := expSvc.MoveItem(context.Background(), models.MoveRequest{
		SrcRemote: "testremote:",
		SrcPath:   "test1/archive_info.json",
		DstRemote: "testremote:",
		DstPath:   "test1/archive_info_v2.json",
	})

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "success" {
		t.Fatalf("expected success, got %s: %s", resp.Status, resp.Error)
	}
	if receivedPayload["srcFs"] != "testremote:test1" {
		t.Fatalf("expected srcFs to be 'testremote:test1', got: %v", receivedPayload["srcFs"])
	}
	if receivedPayload["srcRemote"] != "archive_info.json" {
		t.Fatalf("expected srcRemote to be 'archive_info.json', got: %v", receivedPayload["srcRemote"])
	}
	if receivedPayload["dstFs"] != "testremote:test1" {
		t.Fatalf("expected dstFs to be 'testremote:test1', got: %v", receivedPayload["dstFs"])
	}
	if receivedPayload["dstRemote"] != "archive_info_v2.json" {
		t.Fatalf("expected dstRemote to be 'archive_info_v2.json', got: %v", receivedPayload["dstRemote"])
	}
}

func TestExplorerOps_MoveItem_IdenticalBlocked(t *testing.T) {
	expSvc := NewExplorerService(&config.Config{}, nil, nil)

	resp, err := expSvc.MoveItem(context.Background(), models.MoveRequest{
		SrcRemote: "remote:",
		SrcPath:   "folder/same.txt",
		DstRemote: "remote:",
		DstPath:   "folder/same.txt",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "error" || resp.Error != "Source and destination paths are identical" {
		t.Fatalf("expected identical error, got: %+v", resp)
	}
}
func TestExplorerOps_MoveItem_PathTraversalBlocked(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{LocalPath: tmpDir}
	expSvc := NewExplorerService(cfg, nil, nil)

	testCases := []struct {
		name    string
		src     string
		dst     string
		wantErr string
	}{
		{
			name:    "dot dot in src",
			src:     "../escape.txt",
			dst:     "dest.txt",
			wantErr: "Path traversal forbidden",
		},
		{
			name:    "dot dot in dst",
			src:     "source.txt",
			dst:     "../escape.txt",
			wantErr: "Path traversal forbidden",
		},
		{
			name:    "nested traversal in dst",
			src:     "source.txt",
			dst:     "sub/../../escape.txt",
			wantErr: "Path traversal forbidden",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := expSvc.MoveItem(context.Background(), models.MoveRequest{
				SrcRemote: "local",
				SrcPath:   tc.src,
				DstRemote: "local",
				DstPath:   tc.dst,
			}, tmpDir)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.Status != "error" || !strings.Contains(resp.Error, tc.wantErr) {
				t.Fatalf("expected error containing %q, got: %+v", tc.wantErr, resp)
			}
		})
	}
}

func TestExplorerOps_MoveItem_LocalSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{LocalPath: tmpDir}
	expSvc := NewExplorerService(cfg, nil, nil)

	srcFile := filepath.Join(tmpDir, "source.txt")
	_ = os.WriteFile(srcFile, []byte("payload"), 0600)

	resp, err := expSvc.MoveItem(context.Background(), models.MoveRequest{
		SrcRemote: "local",
		SrcPath:   "source.txt",
		DstRemote: "local",
		DstPath:   "sub/dest.txt",
	}, tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "success" {
		t.Fatalf("expected success, got: %+v", resp)
	}

	dstFile := filepath.Join(tmpDir, "sub", "dest.txt")
	if _, err := os.Stat(dstFile); os.IsNotExist(err) {
		t.Fatalf("expected moved file at %s", dstFile)
	}
	if _, err := os.Stat(srcFile); !os.IsNotExist(err) {
		t.Fatalf("expected source file to be removed at %s", srcFile)
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

func TestExplorerOps_CopyDir_LocalToRemotePreservesContainer(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subfolder")
	_ = os.MkdirAll(subDir, 0755)
	_ = os.WriteFile(filepath.Join(subDir, "nested.txt"), []byte("data"), 0644)

	var lastSrcFs, lastDstFs string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/sync/copy" {
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if src, ok := body["srcFs"].(string); ok {
				lastSrcFs = src
			}
			if dst, ok := body["dstFs"].(string); ok {
				lastDstFs = dst
			}
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	rcClient := client.NewRcloneClient(ts.URL, "", "")
	expSvc := NewExplorerService(&config.Config{LocalPath: tmpDir}, rcClient, nil)

	// Act: copy "subfolder" into remote root ""
	resp, err := expSvc.CopyItem(context.Background(), "local", "subfolder", "testremote:", "", true, tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "success" {
		t.Fatalf("expected success, got: %+v", resp)
	}
	if lastDstFs != "testremote:subfolder" {
		t.Fatalf("expected dstFs to be 'testremote:subfolder', got: %s", lastDstFs)
	}
	if lastSrcFs != subDir {
		t.Fatalf("expected srcFs to be '%s', got: %s", subDir, lastSrcFs)
	}
}

func TestExplorerOps_CopyDir_RemoteToLocalPreservesContainer(t *testing.T) {
	tmpDir := t.TempDir()

	var lastSrcFs, lastDstFs string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/sync/copy" {
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if src, ok := body["srcFs"].(string); ok {
				lastSrcFs = src
			}
			if dst, ok := body["dstFs"].(string); ok {
				lastDstFs = dst
			}
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	rcClient := client.NewRcloneClient(ts.URL, "", "")
	expSvc := NewExplorerService(&config.Config{LocalPath: tmpDir}, rcClient, nil)

	// Act: copy "backup_docs" from testremote: to local root ""
	resp, err := expSvc.CopyItem(context.Background(), "testremote:", "backup_docs", "local", "", true, tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "success" {
		t.Fatalf("expected success, got: %+v", resp)
	}
	expectedDst := filepath.Join(tmpDir, "backup_docs")
	if lastDstFs != expectedDst {
		t.Fatalf("expected dstFs to be '%s', got: %s", expectedDst, lastDstFs)
	}
	if lastSrcFs != "testremote:backup_docs" {
		t.Fatalf("expected srcFs to be 'testremote:backup_docs', got: %s", lastSrcFs)
	}
}

func TestExplorerOps_CopyDir_IdenticalBlocked(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "folderA")
	_ = os.MkdirAll(subDir, 0755)

	expSvc := NewExplorerService(&config.Config{LocalPath: tmpDir}, nil, nil)

	// Act: copy "folderA" to "folderA" on local
	resp, err := expSvc.CopyItem(context.Background(), "local", "folderA", "local", "folderA", true, tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "error" || resp.Error != "Source and destination paths are identical" {
		t.Fatalf("expected identical error, got: %+v", resp)
	}
}

func TestExplorerOps_UploadFile(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{LocalPath: tmpDir}
	expSvc := NewExplorerService(cfg, nil, nil)

	// 1. Invalid filename
	resp, _ := expSvc.UploadFile(context.Background(), "local", "", "../escape.txt", strings.NewReader("data"), 4, tmpDir)
	if resp.Status != "error" || !strings.Contains(resp.Error, "Invalid filename") {
		t.Fatalf("expected invalid filename error, got: %+v", resp)
	}

	// 2. Canary file blocked
	resp, _ = expSvc.UploadFile(context.Background(), "local", "", "RCLONE_TEST", strings.NewReader("data"), 4, tmpDir)
	if resp.Status != "error" || !strings.Contains(resp.Error, "Canary test file") {
		t.Fatalf("expected canary upload blocked, got: %+v", resp)
	}

	// 3. Directory traversal in target dir
	resp, _ = expSvc.UploadFile(context.Background(), "local", "../escape_dir", "file.txt", strings.NewReader("data"), 4, tmpDir)
	if resp.Status != "error" || !strings.Contains(resp.Error, "Directory traversal forbidden") {
		t.Fatalf("expected traversal forbidden, got: %+v", resp)
	}

	// 4. Non-existent directory
	resp, _ = expSvc.UploadFile(context.Background(), "local", "missing_subdir", "file.txt", strings.NewReader("data"), 4, tmpDir)
	if resp.Status != "error" || !strings.Contains(resp.Error, "Target directory does not exist") {
		t.Fatalf("expected non-existent dir error, got: %+v", resp)
	}

	// 5. Successful local upload to root
	resp, err := expSvc.UploadFile(context.Background(), "local", "", "uploaded.txt", strings.NewReader("hello world"), 11, tmpDir)
	if err != nil || resp.Status != "success" {
		t.Fatalf("expected local upload success, got: %+v, err: %v", resp, err)
	}
	content, _ := os.ReadFile(filepath.Join(tmpDir, "uploaded.txt"))
	if string(content) != "hello world" {
		t.Fatalf("expected file content 'hello world', got %q", string(content))
	}

	// 6. Successful local upload to existing subfolder
	subDir := filepath.Join(tmpDir, "subfolder")
	_ = os.Mkdir(subDir, 0755)
	resp, err = expSvc.UploadFile(context.Background(), "local", "subfolder", "nested.txt", strings.NewReader("nested payload"), 14, tmpDir)
	if err != nil || resp.Status != "success" {
		t.Fatalf("expected local upload to subfolder success, got: %+v, err: %v", resp, err)
	}
	nestedContent, _ := os.ReadFile(filepath.Join(subDir, "nested.txt"))
	if string(nestedContent) != "nested payload" {
		t.Fatalf("expected nested content, got %q", string(nestedContent))
	}

	// 7. Remote upload with nil client
	resp, _ = expSvc.UploadFile(context.Background(), "myremote:", "", "remote.txt", strings.NewReader("data"), 4)
	if resp.Status != "error" || !strings.Contains(resp.Error, "client not configured") {
		t.Fatalf("expected client not configured error, got: %+v", resp)
	}

	// 8. Remote upload with mocked client
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer ts.Close()

	rcClient := client.NewRcloneClient(ts.URL, "", "")
	expRemote := NewExplorerService(cfg, rcClient, nil)
	respRemote, err := expRemote.UploadFile(context.Background(), "myremote:", "remote_sub", "cloud.txt", strings.NewReader("cloud data"), 10)
	if err != nil || respRemote.Status != "success" {
		t.Fatalf("expected remote upload success, got: %+v, err: %v", respRemote, err)
	}
}
