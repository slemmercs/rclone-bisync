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
	"github.com/slemmercs/rclone-bisync/internal/core/lock"
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

func TestTailJSONLog(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tailjson-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logPath := filepath.Join(tmpDir, "test.log")
	line1 := `{"time":"2026-10-17T10:00:00Z","level":"info","msg":"plain message"}`
	line2 := `{"time":"2026-10-17T10:01:00Z","level":"notice","msg":"{\"inner\":\"data\"}"}`
	line3 := `not-a-json-line`
	_ = os.WriteFile(logPath, []byte(line1+"\n"+line2+"\n"+line3+"\n"), 0644)

	entries := TailJSONLog(logPath, 2)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
}

func TestParsePendingDeletions(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pending-log-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	daemonLog := filepath.Join(tmpDir, "rclone-daemon.log")
	logContent := `{"msg": "bisync: Safety abort: too many deletes (2)"}
{"msg": "File was deleted - doc1.pdf"}
{"msg": "File was deleted - doc2.txt"}
`
	_ = os.WriteFile(daemonLog, []byte(logContent), 0644)

	cfg := &config.Config{
		BaseDir:              tmpDir,
		PendingDeletionsFile: filepath.Join(tmpDir, "pending.json"),
	}
	svc := NewSyncService(cfg, nil)

	data := svc.ParsePendingDeletionsFromLog()
	if !data.HasDeletions {
		t.Fatal("expected pending deletions to be detected")
	}
	if len(data.Files) != 2 {
		t.Fatalf("expected 2 deleted files, got: %v", data.Files)
	}

	// Test GetPendingDeletions fallback to log
	deletions := svc.GetPendingDeletions()
	if !deletions.HasDeletions || len(deletions.Files) != 2 {
		t.Fatalf("expected GetPendingDeletions to return parsed files: %v", deletions)
	}
}

func TestGetRemotesSummary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/config/listremotes":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"remotes": ["gdrive"]}`))
		case "/config/get":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"type": "drive", "scope": "drive"}`))
		case "/operations/about":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"total": 1000000, "used": 250000, "free": 750000}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	rcClient := client.NewRcloneClient(server.URL, "user", "pass")
	cfg := &config.Config{
		LocalPath: "/tmp/local",
	}
	svc := NewSyncService(cfg, rcClient)

	res, err := svc.GetRemotesSummary(context.Background())
	if err != nil {
		t.Fatalf("GetRemotesSummary failed: %v", err)
	}
	remotesList, ok := res["remotes"].([]map[string]interface{})
	if !ok || len(remotesList) != 1 {
		t.Fatalf("expected 1 remote, got %v in %v", len(remotesList), res)
	}
	if remotesList[0]["name"] != "gdrive" || remotesList[0]["type"] != "drive" {
		t.Fatalf("unexpected remote details: %+v", remotesList[0])
	}
}

func TestCopyItem(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/operations/copyfile" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	tmpDir, err := os.MkdirTemp("", "copy-item-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create source file
	srcPath := filepath.Join(tmpDir, "file1.txt")
	_ = os.WriteFile(srcPath, []byte("hello world"), 0644)

	cfg := &config.Config{
		LocalPath: tmpDir,
	}
	rcClient := client.NewRcloneClient(srv.URL, "user", "pass")
	svc := NewSyncService(cfg, rcClient)

	// 1. Path traversal copy attempt forbidden
	res, err := svc.CopyItem(context.Background(), "local", "../../../etc/passwd", "local", "out.txt", false)
	if err != nil || res["status"] != "error" {
		t.Fatalf("expected path traversal rejection, got: %v", res)
	}

	// 2. Missing source file
	resMissing, _ := svc.CopyItem(context.Background(), "local", "non_existent.txt", "local", "out.txt", false)
	if resMissing["status"] != "error" {
		t.Fatalf("expected error for missing source file, got: %v", resMissing)
	}

	// 3. Successful local file copy
	resOk, err := svc.CopyItem(context.Background(), "local", "file1.txt", "local", "file1_copy.txt", false)
	if err != nil || resOk["status"] != "success" {
		t.Fatalf("expected successful copy, got: %v, err: %v", resOk, err)
	}
}

func TestDeleteItemFull(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "delete-item-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create file and dir
	filePath := filepath.Join(tmpDir, "to_delete.txt")
	_ = os.WriteFile(filePath, []byte("content"), 0644)
	dirPath := filepath.Join(tmpDir, "subdir")
	_ = os.MkdirAll(dirPath, 0755)

	cfg := &config.Config{
		LocalPath: tmpDir,
	}
	svc := NewSyncService(cfg, nil)

	// 1. Delete root or empty path
	resRoot, _ := svc.DeleteItem(context.Background(), "local", "", false)
	if resRoot["status"] != "error" {
		t.Fatalf("expected error deleting root, got: %v", resRoot)
	}

	// 2. Delete non-existent file
	resMissing, _ := svc.DeleteItem(context.Background(), "local", "missing.txt", false)
	if resMissing["status"] != "error" {
		t.Fatalf("expected error deleting missing target, got: %v", resMissing)
	}

	// 3. Delete existing local file
	resFile, err := svc.DeleteItem(context.Background(), "local", "to_delete.txt", false)
	if err != nil || resFile["status"] != "success" {
		t.Fatalf("expected success deleting file, got: %v, err: %v", resFile, err)
	}

	// 4. Delete existing local directory
	resDir, err := svc.DeleteItem(context.Background(), "local", "subdir", true)
	if err != nil || resDir["status"] != "success" {
		t.Fatalf("expected success deleting dir, got: %v, err: %v", resDir, err)
	}
}

func TestHandleActionAndSyncing(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "action-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	lockFile := filepath.Join(tmpDir, "test.lock")
	rateLimitFile := filepath.Join(tmpDir, "rate_limit.flag")
	pendingFile := filepath.Join(tmpDir, "pending.json")

	_ = os.WriteFile(rateLimitFile, []byte(`{"flag":true}`), 0644)
	_ = os.WriteFile(pendingFile, []byte(`{"has_deletions":true}`), 0644)

	cfg := &config.Config{
		LockFile:             lockFile,
		RateLimitBackoffFile: rateLimitFile,
		PendingDeletionsFile: pendingFile,
	}
	svc := NewSyncService(cfg, nil)

	// IsSyncing when lock missing vs present
	if svc.IsSyncing() {
		t.Fatal("expected IsSyncing to be false when lock file missing")
	}

	lm := lock.NewLockManager(lockFile, 3600, false)
	acquired, err := lm.Acquire()
	if err != nil || !acquired {
		t.Fatalf("failed to acquire lock: %v", err)
	}
	defer lm.Release()

	if !svc.IsSyncing() {
		t.Fatal("expected IsSyncing to be true when lock file actively held")
	}

	// HandleAction unknown action
	resUnknown, _ := svc.HandleAction(context.Background(), "invalid_action")
	if resUnknown["status"] != "error" {
		t.Fatalf("expected error for unknown action, got: %v", resUnknown)
	}

	// HandleAction clear_rate_limit
	resClear, _ := svc.HandleAction(context.Background(), "clear_rate_limit")
	if resClear["status"] != "success" {
		t.Fatalf("expected success clearing rate limit, got: %v", resClear)
	}

	// HandleAction dismiss_deletions (handled by pending deletions clearing or remove_lock)
	resLock, _ := svc.HandleAction(context.Background(), "remove_lock")
	// Since isSyncing is true (lockFile exists), removing lock should report error or conflict
	if resLock["status"] != "error" {
		t.Fatalf("expected error removing active lock, got: %v", resLock)
	}
}

func TestSyncServiceAdditionalBranches(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/operations/touch":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		case "/sync/copy":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		case "/operations/copyfile":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		case "/operations/deletefile":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		case "/operations/purge":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		case "/job/list":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"jobids": [99]}`))
		case "/job/stop":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		case "/operations/list":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"list": [{"Name": "test.txt", "IsDir": false, "Size": 100}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	tmpDir, err := os.MkdirTemp("", "additional-branches-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		LocalPath: tmpDir,
		LockFile:  filepath.Join(tmpDir, "test.lock"),
		BaseDir:   tmpDir,
	}
	rcClient := client.NewRcloneClient(srv.URL, "user", "pass")
	svc := NewSyncService(cfg, rcClient)
	ctx := context.Background()

	// 1. Create remote canary
	resCanary, err := svc.CreateCanary(ctx, "gdrive", "sub")
	if err != nil || resCanary["status"] != "success" {
		t.Fatalf("expected remote canary creation to succeed: %v, %v", resCanary, err)
	}

	// 2. Remote CopyDir and remote CopyFile
	_ = os.MkdirAll(filepath.Join(tmpDir, "folder"), 0755)
	resDir, err := svc.CopyItem(ctx, "local", "folder", "gdrive", "sub/folder", true)
	if err != nil || resDir["status"] != "success" {
		t.Fatalf("expected CopyDir success: %v", resDir)
	}
	resRemoteFile, err := svc.CopyItem(ctx, "gdrive", "file.pdf", "local", "file.pdf", false)
	if err != nil || resRemoteFile["status"] != "success" {
		t.Fatalf("expected remote-to-local CopyFile success: %v", resRemoteFile)
	}

	// 3. Remote DeleteItem (file & purge dir)
	resDelFile, err := svc.DeleteItem(ctx, "gdrive", "old.pdf", false)
	if err != nil || resDelFile["status"] != "success" {
		t.Fatalf("expected remote DeleteItem success: %v", resDelFile)
	}
	resPurge, err := svc.DeleteItem(ctx, "gdrive", "old_dir", true)
	if err != nil || resPurge["status"] != "success" {
		t.Fatalf("expected remote Purge success: %v", resPurge)
	}

	// 4. HandleAction guards when actively syncing
	lm := lock.NewLockManager(cfg.LockFile, 3600, false)
	_, _ = lm.Acquire()

	resStart, _ := svc.HandleAction(ctx, "start_sync")
	if resStart["status"] != "error" {
		t.Fatalf("expected error starting sync while locked: %v", resStart)
	}
	resForceResync, _ := svc.HandleAction(ctx, "force_resync")
	if resForceResync["status"] != "error" {
		t.Fatalf("expected error force_resync while locked: %v", resForceResync)
	}
	resForceSync, _ := svc.HandleAction(ctx, "force_sync")
	if resForceSync["status"] != "error" {
		t.Fatalf("expected error force_sync while locked: %v", resForceSync)
	}
	resDryRun, _ := svc.HandleAction(ctx, "dry_run")
	if resDryRun["status"] != "error" {
		t.Fatalf("expected error dry_run while locked: %v", resDryRun)
	}

	_ = lm.Release()

	// 5. HandleAction remove_lock when stale lock file exists
	_ = os.WriteFile(cfg.LockFile, []byte("stale"), 0644)
	resRemLockSuccess, _ := svc.HandleAction(ctx, "remove_lock")
	if resRemLockSuccess["status"] != "success" {
		t.Fatalf("expected success removing stale lock file: %v", resRemLockSuccess)
	}

	// 6. HandleAction remove_lock when lock file missing
	resRemLockMissing, _ := svc.HandleAction(ctx, "remove_lock")
	if resRemLockMissing["status"] != "info" {
		t.Fatalf("expected info on missing lock: %v", resRemLockMissing)
	}

	// 7. HandleAction stop_sync
	resStop, _ := svc.HandleAction(ctx, "stop_sync")
	if resStop["status"] != "success" {
		t.Fatalf("expected success on stop_sync: %v", resStop)
	}

	// 8. Remote BrowsePath
	listing := svc.BrowsePath(ctx, "gdrive", "sub")
	if listing.Error != "" {
		t.Fatalf("expected remote browse success: %s", listing.Error)
	}

	// 9. BrowsePath missing directory
	resMissingBrowse := svc.BrowsePath(ctx, "local", "does_not_exist")
	if resMissingBrowse.Error == "" {
		t.Fatal("expected error browsing missing local dir")
	}

	// 10. CreateCanary traversal forbidden
	resBadCanary, _ := svc.CreateCanary(ctx, "local", "../../../etc")
	if resBadCanary["status"] != "error" {
		t.Fatalf("expected error on canary traversal: %v", resBadCanary)
	}

	// 11. Local canary touch/creation
	_ = os.WriteFile(filepath.Join(tmpDir, "RCLONE_TEST"), []byte("ok"), 0644)
	resCanaryExists, _ := svc.CreateCanary(ctx, "local", "")
	if resCanaryExists["status"] != "success" {
		t.Fatalf("expected canary creation success, got: %v", resCanaryExists)
	}

	// 12. Copy local directory to local
	resLocalDirCopy, err := svc.CopyItem(ctx, "local", "folder", "local", "folder2", true)
	if err != nil || resLocalDirCopy["status"] != "success" {
		t.Fatalf("expected local dir copy success, got: %v", resLocalDirCopy)
	}

	// 13. ParsePendingDeletionsFromLog when daemon log is missing
	svcNoLog := NewSyncService(&config.Config{BaseDir: filepath.Join(tmpDir, "empty")}, nil)
	missingLogData := svcNoLog.ParsePendingDeletionsFromLog()
	if missingLogData.HasDeletions {
		t.Fatal("expected HasDeletions false when log is missing")
	}

	// 14. runBisyncSystemd execution path
	_ = svc.runBisyncSystemd([]string{"--dry-run"}, "test-unit", "test")
}



