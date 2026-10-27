package review

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type mockRemoteClient struct {
	copyFileFunc   func(ctx context.Context, srcFs, srcRemote, dstFs, dstRemote string) (bool, error)
	deleteFileFunc func(ctx context.Context, fs, remote string) (bool, error)
}

func (m *mockRemoteClient) CopyFile(ctx context.Context, srcFs, srcRemote, dstFs, dstRemote string) (bool, error) {
	if m.copyFileFunc != nil {
		return m.copyFileFunc(ctx, srcFs, srcRemote, dstFs, dstRemote)
	}
	return true, nil
}

func (m *mockRemoteClient) DeleteFile(ctx context.Context, fs, remote string) (bool, error) {
	if m.deleteFileFunc != nil {
		return m.deleteFileFunc(ctx, fs, remote)
	}
	return true, nil
}

func TestBackup_CheckFreeDiskSpace(t *testing.T) {
	// Zero bytes should always succeed
	if err := CheckFreeDiskSpace(t.TempDir(), 0); err != nil {
		t.Fatalf("expected nil for 0 bytes required, got %v", err)
	}

	// Normal check on temp dir
	if err := CheckFreeDiskSpace(t.TempDir(), 1024); err != nil {
		t.Fatalf("expected sufficient space for 1KB, got %v", err)
	}

	// Unreasonable disk space (100 Petabytes) should fail
	hugeBytes := int64(100) * 1024 * 1024 * 1024 * 1024 * 1024
	if err := CheckFreeDiskSpace(t.TempDir(), hugeBytes); err == nil {
		t.Fatal("expected error for 100 Petabytes required")
	}
}

func TestBackup_ResolveSafePath(t *testing.T) {
	root := t.TempDir()

	// Valid path
	safe, err := ResolveSafePath(root, "docs/sub/report.pdf")
	if err != nil || safe == "" {
		t.Fatalf("expected valid safe path, got %v, err: %v", safe, err)
	}

	// Path traversal
	if _, err := ResolveSafePath(root, "../secret.txt"); err == nil {
		t.Fatal("expected traversal error for ../secret.txt")
	}
	if _, err := ResolveSafePath(root, "/etc/passwd"); err == nil {
		t.Fatal("expected error for absolute path traversal")
	}

	// Canary protection
	if _, err := ResolveSafePath(root, "RCLONE_TEST_file.txt"); err == nil {
		t.Fatal("expected canary safety violation error")
	}

	// Empty root
	if _, err := ResolveSafePath("", "file.txt"); err == nil {
		t.Fatal("expected error for empty root")
	}
}

func TestBackup_LocalAndRemoteBackup(t *testing.T) {
	tmpDir := t.TempDir()
	localRoot := filepath.Join(tmpDir, "local")
	backupDir := filepath.Join(tmpDir, "backups")
	_ = os.MkdirAll(localRoot, 0755)

	srcFile := filepath.Join(localRoot, "test.txt")
	_ = os.WriteFile(srcFile, []byte("hello backup"), 0644)

	// Local backup
	backupPath, err := LocalBackup(srcFile, backupDir, "sess-1", "test.txt")
	if err != nil {
		t.Fatalf("LocalBackup failed: %v", err)
	}
	if _, err := os.Stat(backupPath); err != nil {
		t.Fatalf("expected backup file %s to exist: %v", backupPath, err)
	}

	// Restore local backup
	restoredPath := filepath.Join(localRoot, "restored.txt")
	if err := RestoreLocalBackup(backupPath, restoredPath); err != nil {
		t.Fatalf("RestoreLocalBackup failed: %v", err)
	}
	data, _ := os.ReadFile(restoredPath)
	if string(data) != "hello backup" {
		t.Fatalf("restored data mismatch: %s", string(data))
	}

	// Non-existent source error
	if _, err := LocalBackup(filepath.Join(localRoot, "missing.txt"), backupDir, "sess-1", "missing.txt"); err == nil {
		t.Fatal("expected error backing up non-existent file")
	}

	// Missing backup restore error
	if err := RestoreLocalBackup(filepath.Join(backupDir, "no-backup.txt"), restoredPath); err == nil {
		t.Fatal("expected error restoring missing backup")
	}

	// Remote backup with mock client
	mock := &mockRemoteClient{
		copyFileFunc: func(ctx context.Context, srcFs, srcRemote, dstFs, dstRemote string) (bool, error) {
			if srcRemote != "remote.txt" {
				t.Fatalf("unexpected srcRemote: %s", srcRemote)
			}
			return true, nil
		},
	}

	remoteBackupPath, err := RemoteBackup(context.Background(), mock, "gdrive:", "remote.txt", "sess-1")
	if err != nil {
		t.Fatalf("RemoteBackup failed: %v", err)
	}
	expectedRemote := ".review_backups/sess-1/remote.txt"
	if remoteBackupPath != expectedRemote {
		t.Fatalf("expected %s, got %s", expectedRemote, remoteBackupPath)
	}

	// Nil remote client error
	if _, err := RemoteBackup(context.Background(), nil, "gdrive:", "remote.txt", "sess-1"); err == nil {
		t.Fatal("expected error with nil client")
	}
}

func TestBackup_PruneExpiredBackups(t *testing.T) {
	tmpDir := t.TempDir()
	backupsDir := filepath.Join(tmpDir, "backups")
	oldDir := filepath.Join(backupsDir, "sess-old")
	newDir := filepath.Join(backupsDir, "sess-new")
	_ = os.MkdirAll(oldDir, 0755)
	_ = os.MkdirAll(newDir, 0755)

	oldFile := filepath.Join(oldDir, "old.txt")
	_ = os.WriteFile(oldFile, []byte("old"), 0644)
	// Set old modtime (48 hours ago)
	past := time.Now().Add(-48 * time.Hour)
	_ = os.Chtimes(oldDir, past, past)

	pruned, err := PruneExpiredBackups(backupsDir, 24*time.Hour)
	if err != nil {
		t.Fatalf("PruneExpiredBackups failed: %v", err)
	}
	if pruned != 1 {
		t.Fatalf("expected 1 pruned backup dir, got %d", pruned)
	}
	if _, err := os.Stat(oldDir); !os.IsNotExist(err) {
		t.Fatalf("expected oldDir to be removed")
	}
	if _, err := os.Stat(newDir); err != nil {
		t.Fatalf("expected newDir to be retained")
	}
}

func TestBackup_RestoreLocalBackup_CopyFallback(t *testing.T) {
	tmpDir := t.TempDir()
	backupPath := filepath.Join(tmpDir, "backup.txt")
	_ = os.WriteFile(backupPath, []byte("content to restore"), 0644)

	targetPath := filepath.Join(tmpDir, "target.txt")
	if err := RestoreLocalBackup(backupPath, targetPath); err != nil {
		t.Fatalf("RestoreLocalBackup failed: %v", err)
	}

	data, err := os.ReadFile(targetPath)
	if err != nil || string(data) != "content to restore" {
		t.Fatalf("unexpected content: %s, err: %v", string(data), err)
	}
}

func TestBackup_PruneExpiredBackups_EdgeCases(t *testing.T) {
	// Empty backups dir or zero TTL
	pruned, err := PruneExpiredBackups("", 10*time.Hour)
	if err != nil || pruned != 0 {
		t.Fatalf("expected 0 for empty dir, got %d, %v", pruned, err)
	}
	pruned, err = PruneExpiredBackups("/tmp", 0)
	if err != nil || pruned != 0 {
		t.Fatalf("expected 0 for 0 TTL, got %d, %v", pruned, err)
	}
	pruned, err = PruneExpiredBackups("/nonexistent/path/for/backups", 10*time.Hour)
	if err != nil || pruned != 0 {
		t.Fatalf("expected 0 for missing path, got %d, %v", pruned, err)
	}
}
