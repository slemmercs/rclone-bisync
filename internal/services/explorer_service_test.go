package services

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

	// Symlink pointing outside jail (H-2)
	outsideDir, err := os.MkdirTemp("", "outside-jail-*")
	if err != nil {
		t.Fatalf("failed to create outside dir: %v", err)
	}
	defer os.RemoveAll(outsideDir)
	_ = os.WriteFile(filepath.Join(outsideDir, "secret.txt"), []byte("secret"), 0644)

	outsideSymlink := filepath.Join(tmpDir, "symlink_outside")
	if err := os.Symlink(outsideDir, outsideSymlink); err == nil {
		_, _, okOutside := IsLocalPathJailed(tmpDir, "symlink_outside")
		if okOutside {
			t.Fatal("expected symlink pointing outside jail to be rejected")
		}

		_, _, okFileOutside := IsLocalPathJailed(tmpDir, "symlink_outside/secret.txt")
		if okFileOutside {
			t.Fatal("expected file inside outside symlink to be rejected")
		}
	}

	// Symlink pointing inside jail should be permitted
	insideSymlink := filepath.Join(tmpDir, "symlink_inside")
	if err := os.Symlink(subDir, insideSymlink); err == nil {
		_, _, okInside := IsLocalPathJailed(tmpDir, "symlink_inside")
		if !okInside {
			t.Fatal("expected symlink pointing inside jail to be permitted")
		}
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

	// 1b. Root canary inherited by subfolder without canary
	rootCanaryDir := filepath.Join(tmpDir, "subwithoutcanary")
	_ = os.MkdirAll(rootCanaryDir, 0755)
	_ = os.WriteFile(filepath.Join(tmpDir, "RCLONE_TEST"), []byte("root-canary"), 0644)
	resSubWithout := expSvc.BrowsePath(context.Background(), "local", "subwithoutcanary")
	if !resSubWithout.HasCanary {
		t.Fatal("expected subfolder to inherit root canary status")
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
	if resTrav.Status != "error" || resTrav.Error != "Source path traversal forbidden" {
		t.Fatalf("expected source traversal forbidden, got: %v", resTrav)
	}

	// 2. Destination traversal forbidden
	resDstTrav, _ := expSvc.CopyItem(context.Background(), "local", "file1.txt", "local", "../out.txt", false)
	if resDstTrav.Status != "error" || resDstTrav.Error != "Destination path traversal forbidden" {
		t.Fatalf("expected destination traversal forbidden, got: %v", resDstTrav)
	}

	// 3. Source missing
	resMissing, _ := expSvc.CopyItem(context.Background(), "local", "non_existent.txt", "local", "out.txt", false)
	if resMissing.Status != "error" {
		t.Fatalf("expected error for missing source, got: %v", resMissing)
	}

	// 4. Local-to-local file copy
	resOk, err := expSvc.CopyItem(context.Background(), "local", "file1.txt", "local", "file1_copy.txt", false)
	if err != nil || resOk.Status != "success" {
		t.Fatalf("expected copy success, got: %v, err: %v", resOk, err)
	}

	// 5. Local-to-remote directory copy
	resDir, err := expSvc.CopyItem(context.Background(), "local", "folder", "gdrive", "sub/folder", true)
	if err != nil || resDir.Status != "success" {
		t.Fatalf("expected copydir success: %v, err: %v", resDir, err)
	}

	// 6. Nil client error check
	expNil := NewExplorerService(cfg, nil, nil)
	resNil, _ := expNil.CopyItem(context.Background(), "gdrive", "file.txt", "local", "file.txt", false)
	if resNil.Status != "error" || resNil.Error != "client not configured" {
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
		case "/operations/list":
			w.Header().Set("Content-Type", "application/json")
			reqBody, _ := io.ReadAll(r.Body)
			if strings.Contains(string(reqBody), "canary_remote_dir") {
				_, _ = w.Write([]byte(`{"list":[{"Name":"RCLONE_TEST","IsDir":false}]}`))
			} else {
				_, _ = w.Write([]byte(`{"list":[]}`))
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	rcClient := client.NewRcloneClient(srv.URL, "user", "pass")
	cfg := &config.Config{LocalPath: tmpDir}
	canarySvc := NewCanaryService(cfg, rcClient)
	expSvc := NewExplorerService(cfg, rcClient, canarySvc)

	// 1. Root or empty path deletion forbidden
	resRoot, _ := expSvc.DeleteItem(context.Background(), "local", "", false)
	if resRoot.Status != "error" || resRoot.Error != "Root directory cannot be deleted" {
		t.Fatalf("expected root deletion error, got: %v", resRoot)
	}

	// 2. Canary file protected
	resCanary, _ := expSvc.DeleteItem(context.Background(), "local", "RCLONE_TEST", false)
	if resCanary.Status != "error" || resCanary.Error != "Canary test file (RCLONE_TEST) cannot be deleted" {
		t.Fatalf("expected canary deletion error, got: %v", resCanary)
	}

	// 2b. Directory containing canary file protected (H-3 local)
	canaryDir := filepath.Join(tmpDir, "canary_dir")
	_ = os.MkdirAll(canaryDir, 0755)
	_ = os.WriteFile(filepath.Join(canaryDir, "RCLONE_TEST"), []byte("canary"), 0644)
	resCanaryDir, _ := expSvc.DeleteItem(context.Background(), "local", "canary_dir", true)
	if resCanaryDir.Status != "error" || resCanaryDir.Error != "Directory contains canary test file (RCLONE_TEST) and cannot be deleted" {
		t.Fatalf("expected canary directory deletion error, got: %v", resCanaryDir)
	}
	if _, err := os.Stat(canaryDir); err != nil {
		t.Fatalf("canary directory should not have been removed: %v", err)
	}

	// 2c. Remote directory containing canary file protected (H-3 remote)
	resRemoteCanaryDir, _ := expSvc.DeleteItem(context.Background(), "gdrive", "canary_remote_dir", true)
	if resRemoteCanaryDir.Status != "error" || resRemoteCanaryDir.Error != "Directory contains canary test file (RCLONE_TEST) and cannot be deleted" {
		t.Fatalf("expected remote canary directory deletion error, got: %v", resRemoteCanaryDir)
	}

	// 3. Traversal or root deletion forbidden
	resTrav, _ := expSvc.DeleteItem(context.Background(), "local", "../../", false)
	if resTrav.Status != "error" {
		t.Fatalf("expected traversal forbidden, got: %v", resTrav)
	}

	// 4. Missing target
	resMissing, _ := expSvc.DeleteItem(context.Background(), "local", "not_there.txt", false)
	if resMissing.Status != "error" {
		t.Fatalf("expected missing target error, got: %v", resMissing)
	}

	// 5. Successful local file deletion
	resFile, err := expSvc.DeleteItem(context.Background(), "local", "del.txt", false)
	if err != nil || resFile.Status != "success" {
		t.Fatalf("expected file delete success, got: %v, err: %v", resFile, err)
	}

	// 6. Successful local directory deletion
	resDir, err := expSvc.DeleteItem(context.Background(), "local", "deldir", true)
	if err != nil || resDir.Status != "success" {
		t.Fatalf("expected dir delete success, got: %v, err: %v", resDir, err)
	}

	// 7. Remote file deletion
	resRemFile, err := expSvc.DeleteItem(context.Background(), "gdrive", "remote.txt", false)
	if err != nil || resRemFile.Status != "success" {
		t.Fatalf("expected remote file delete success, got: %v, err: %v", resRemFile, err)
	}

	// 8. Remote dir purge
	resRemPurge, err := expSvc.DeleteItem(context.Background(), "gdrive", "remotedir", true)
	if err != nil || resRemPurge.Status != "success" {
		t.Fatalf("expected remote dir purge success, got: %v, err: %v", resRemPurge, err)
	}

	// 9. Nil client error handling
	expNil := NewExplorerService(cfg, nil, nil)
	resNil, _ := expNil.DeleteItem(context.Background(), "gdrive", "test.txt", false)
	if resNil.Status != "error" || resNil.Error != "client not configured" {
		t.Fatalf("expected client not configured, got: %v", resNil)
	}
}

func TestExplorerService_IsLocalPathJailed_Extended(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "allowed")
	_ = os.MkdirAll(subDir, 0755)

	// 1. Base dir with tilde expansion
	baseHome, targetHome, okHome := IsLocalPathJailed("~/test-folder", "sub/file.txt")
	if !okHome || !strings.Contains(baseHome, "test-folder") || !strings.Contains(targetHome, "sub/file.txt") {
		t.Fatalf("expected tilde expansion to resolve jail safely, got: %v, %s, %s", okHome, baseHome, targetHome)
	}

	// 2. Relative symlink pointing inside jail
	relSymlink := filepath.Join(tmpDir, "rel_link")
	_ = os.Symlink("allowed", relSymlink)
	_, _, okRel := IsLocalPathJailed(tmpDir, "rel_link")
	if !okRel {
		t.Fatal("expected relative symlink inside jail to be permitted")
	}

	// 3. Non-existent target traversing missing parent directories
	_, _, okMissingNested := IsLocalPathJailed(tmpDir, "does/not/exist/yet/file.txt")
	if !okMissingNested {
		t.Fatal("expected non-existent nested target inside jail to be permitted")
	}

	// 4. Non-existent target whose parent resolves outside jail via symlink
	outsideDir := t.TempDir()
	outsideSymlink := filepath.Join(tmpDir, "outside_sym")
	_ = os.Symlink(outsideDir, outsideSymlink)
	_, _, okOutNested := IsLocalPathJailed(tmpDir, "outside_sym/does/not/exist/file.txt")
	if okOutNested {
		t.Fatal("expected non-existent file inside escaping symlink to be rejected")
	}
}

func TestExplorerService_BrowsePath_Extended(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "docs")
	_ = os.MkdirAll(subDir, 0755)

	nestedFolder := filepath.Join(subDir, "nested_folder")
	_ = os.MkdirAll(nestedFolder, 0755)
	_ = os.WriteFile(filepath.Join(subDir, "alpha.txt"), []byte("a"), 0644)
	_ = os.WriteFile(filepath.Join(subDir, "beta.txt"), []byte("b"), 0644)

	cfg := &config.Config{LocalPath: tmpDir}
	expSvc := NewExplorerService(cfg, nil, nil)

	// 1. Listing directory with subdirectory and multiple files (tests sorting & isDir mime)
	res := expSvc.BrowsePath(context.Background(), "local", "docs")
	if res.Error != "" {
		t.Fatalf("unexpected browse error: %s", res.Error)
	}
	if len(res.Items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(res.Items))
	}
	// First item should be the directory due to sort
	if res.Items[0]["IsDir"] != true || res.Items[0]["MimeType"] != "inode/directory" {
		t.Fatalf("expected directory first with inode/directory mime, got: %v", res.Items[0])
	}

	// 2. Unreadable directory (ReadDir error)
	unreadable := filepath.Join(tmpDir, "unreadable")
	_ = os.MkdirAll(unreadable, 0000)
	defer os.Chmod(unreadable, 0755)

	resUnreadable := expSvc.BrowsePath(context.Background(), "local", "unreadable")
	if resUnreadable.Error == "" {
		t.Fatal("expected error browsing unreadable directory")
	}

	// 3. Fallback when cfg or LocalPath is empty
	expEmptyCfg := NewExplorerService(nil, nil, nil)
	// Browse non-existent path in fallback ~/gdrive
	resFallback := expEmptyCfg.BrowsePath(context.Background(), "local", "non_existent_dir_xyz_123")
	if !strings.Contains(resFallback.Error, "Directory does not exist") {
		t.Fatalf("expected missing dir in fallback ~/gdrive, got: %s", resFallback.Error)
	}
}

func TestExplorerService_CopyItem_Extended(t *testing.T) {
	tmpDir := t.TempDir()
	localFile := filepath.Join(tmpDir, "source.txt")
	_ = os.WriteFile(localFile, []byte("content"), 0644)
	subDir := filepath.Join(tmpDir, "existing_dir")
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

	// 1. Remote source to local target (Directory)
	resDir, err := expSvc.CopyItem(context.Background(), "gdrive", "remoteDir", "local", "existing_dir", true)
	if err != nil || resDir.Status != "success" {
		t.Fatalf("expected remote to local dir copy success, got: %v, err: %v", resDir, err)
	}

	// 1b. Remote root dir to local target
	resRootDir, err := expSvc.CopyItem(context.Background(), "gdrive", "", "local", "existing_dir", true)
	if err != nil || resRootDir.Status != "success" {
		t.Fatalf("expected remote root dir copy success, got: %v, err: %v", resRootDir, err)
	}

	// 2. Remote source file to local target ending with /
	resSlash, err := expSvc.CopyItem(context.Background(), "gdrive", "remote/file.txt", "local", "existing_dir/", false)
	if err != nil || resSlash.Status != "success" {
		t.Fatalf("expected remote file to local slash dir success, got: %v, err: %v", resSlash, err)
	}

	// 3. Remote root file to local directory
	resRootFile, err := expSvc.CopyItem(context.Background(), "gdrive", "file.txt", "local", "existing_dir", false)
	if err != nil || resRootFile.Status != "success" {
		t.Fatalf("expected remote root file to local dir success, got: %v, err: %v", resRootFile, err)
	}

	// 4. Remote file to local target "."
	resDot, err := expSvc.CopyItem(context.Background(), "gdrive", "sub/file.txt", "local", ".", false)
	if err != nil || resDot.Status != "success" {
		t.Fatalf("expected remote file to local '.' success, got: %v, err: %v", resDot, err)
	}

	// 5. Local source to remote target with empty dstPath
	resRemEmpty, err := expSvc.CopyItem(context.Background(), "local", "source.txt", "gdrive", "", false)
	if err != nil || resRemEmpty.Status != "success" {
		t.Fatalf("expected local to remote root file copy success, got: %v, err: %v", resRemEmpty, err)
	}

	// 5b. Local directory to remote root
	resRemDirEmpty, err := expSvc.CopyItem(context.Background(), "local", "existing_dir", "gdrive", "", true)
	if err != nil || resRemDirEmpty.Status != "success" {
		t.Fatalf("expected local dir to remote root copy success, got: %v, err: %v", resRemDirEmpty, err)
	}

	// 6. Failing server testing CopyDir and CopyFile errors
	failSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"failed"}`))
	}))
	defer failSrv.Close()

	failClient := client.NewRcloneClient(failSrv.URL, "user", "pass")
	failExp := NewExplorerService(cfg, failClient, nil)

	resFailDir, _ := failExp.CopyItem(context.Background(), "local", "existing_dir", "gdrive", "sub", true)
	if resFailDir.Status != "error" || !strings.Contains(resFailDir.Error, "copy dir failed") {
		t.Fatalf("expected copy dir failure, got: %v", resFailDir)
	}

	resFailFile, _ := failExp.CopyItem(context.Background(), "local", "source.txt", "gdrive", "sub/file.txt", false)
	// 7. Nil config fallback in CopyItem
	expNilCfg := NewExplorerService(nil, rcClient, nil)
	_, _ = expNilCfg.CopyItem(context.Background(), "local", "invalid_nonexistent_src_12345", "local", "out.txt", false)

	if resFailFile.Status != "error" || !strings.Contains(resFailFile.Error, "copy file failed") {
		t.Fatalf("expected copy file failure, got: %v", resFailFile)
	}
}

func TestExplorerService_DeleteItem_Extended(t *testing.T) {
	tmpDir := t.TempDir()
	canaryDir := filepath.Join(tmpDir, "with_canary")
	_ = os.MkdirAll(canaryDir, 0755)
	_ = os.WriteFile(filepath.Join(canaryDir, "RCLONE_TEST"), []byte("canary"), 0644)

	cleanDir := filepath.Join(tmpDir, "without_canary")
	_ = os.MkdirAll(cleanDir, 0755)
	_ = os.WriteFile(filepath.Join(cleanDir, "somefile.txt"), []byte("data"), 0644)

	cfg := &config.Config{LocalPath: tmpDir}
	// ExplorerService with Canary == nil to exercise fallback walk logic
	expNoCanary := &ExplorerService{
		Config: cfg,
		Client: nil,
		Canary: nil,
	}

	// 1. Refuse deleting local directory containing canary when Canary is nil
	resCanaryWalk, _ := expNoCanary.DeleteItem(context.Background(), "local", "with_canary", true)
	if resCanaryWalk.Status != "error" || !strings.Contains(resCanaryWalk.Error, "contains canary test file") {
		t.Fatalf("expected canary protection walk error, got: %v", resCanaryWalk)
	}

	// 2. Allow deleting local directory without canary when Canary is nil
	resCleanWalk, err := expNoCanary.DeleteItem(context.Background(), "local", "without_canary", true)
	if err != nil || resCleanWalk.Status != "success" {
		t.Fatalf("expected clean dir deletion success, got: %v, err: %v", resCleanWalk, err)
	}

	// 3. Error deleting non-empty directory as a file (os.Remove fails on non-empty directory)
	nonEmptyDir := filepath.Join(tmpDir, "non_empty")
	_ = os.MkdirAll(nonEmptyDir, 0755)
	_ = os.WriteFile(filepath.Join(nonEmptyDir, "item.txt"), []byte("item"), 0644)
	resFileErr, _ := expNoCanary.DeleteItem(context.Background(), "local", "non_empty", false)
	if resFileErr.Status != "error" || !strings.Contains(resFileErr.Error, "Failed to delete local file") {
		t.Fatalf("expected file remove failure on directory, got: %v", resFileErr)
	}

	// 4. Remote deletion errors with failing server
	failSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"failed"}`))
	}))
	defer failSrv.Close()

	failClient := client.NewRcloneClient(failSrv.URL, "user", "pass")
	failExp := NewExplorerService(cfg, failClient, nil)

	// Purge failure
	resPurgeErr, _ := failExp.DeleteItem(context.Background(), "gdrive", "remote_folder", true)
	if resPurgeErr.Status != "error" || !strings.Contains(resPurgeErr.Error, "Purge failed") {
		t.Fatalf("expected purge failure, got: %v", resPurgeErr)
	}
	// 5. Nil config fallback in DeleteItem
	expNilCfg := NewExplorerService(nil, nil, nil)
	_, _ = expNilCfg.DeleteItem(context.Background(), "local", "invalid_nonexistent_src_12345", false)

	// 6. os.RemoveAll failure on directory in read-only parent
	roParent := filepath.Join(tmpDir, "ro_parent")
	_ = os.MkdirAll(roParent, 0755)
	childDir := filepath.Join(roParent, "child")
	_ = os.MkdirAll(childDir, 0755)
	_ = os.Chmod(roParent, 0555)
	defer os.Chmod(roParent, 0755)

	expRO := NewExplorerService(&config.Config{LocalPath: roParent}, nil, nil)
	resROErr, _ := expRO.DeleteItem(context.Background(), "local", "child", true)
	if resROErr.Status != "error" || !strings.Contains(resROErr.Error, "Failed to delete local directory") {
		t.Fatalf("expected remove directory failure on read-only parent, got: %v", resROErr)
	}

	// Delete file failure
	resDelErr, _ := failExp.DeleteItem(context.Background(), "gdrive", "remote_file.txt", false)
	if resDelErr.Status != "error" || !strings.Contains(resDelErr.Error, "Delete file failed") {
		t.Fatalf("expected delete file failure, got: %v", resDelErr)
	}
}
