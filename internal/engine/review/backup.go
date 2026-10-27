package review

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

// RemoteClient defines the minimal interface required for remote cloud file operations.
type RemoteClient interface {
	CopyFile(ctx context.Context, srcFs, srcRemote, dstFs, dstRemote string) (bool, error)
	DeleteFile(ctx context.Context, fs, remote string) (bool, error)
}

const (
	// MinFreeDiskSafetyBuffer is the required reserve disk space (10MB).
	MinFreeDiskSafetyBuffer = 10 * 1024 * 1024
)

// CheckFreeDiskSpace verifies that the filesystem hosting dir has sufficient free space.
func CheckFreeDiskSpace(dir string, requiredBytes int64) error {
	if requiredBytes <= 0 {
		return nil
	}

	targetDir := dir
	for targetDir != "" && targetDir != "/" && targetDir != "." {
		if fi, err := os.Stat(targetDir); err == nil && fi.IsDir() {
			break
		}
		targetDir = filepath.Dir(targetDir)
	}
	if targetDir == "" || targetDir == "." {
		targetDir = "/"
	}

	var stat unix.Statfs_t
	if err := unix.Statfs(targetDir, &stat); err != nil {
		return fmt.Errorf("failed to check filesystem disk space: %w", err)
	}

	if stat.Bsize < 0 {
		return fmt.Errorf("invalid disk block size reported by filesystem")
	}
	if requiredBytes < 0 {
		requiredBytes = 0
	}
	availableBytes := stat.Bavail * uint64(stat.Bsize)
	needed := uint64(requiredBytes) + MinFreeDiskSafetyBuffer
	if availableBytes < needed {
		return fmt.Errorf("insufficient disk space: need %d bytes (incl 10MB safety margin), available %d bytes", needed, availableBytes)
	}

	return nil
}

// ResolveSafePath confines relPath strictly within root, blocking traversal and canary files.
func ResolveSafePath(root, relPath string) (string, error) {
	if root == "" {
		return "", fmt.Errorf("root path is empty")
	}
	cleanRel := filepath.Clean(relPath)
	if strings.HasPrefix(cleanRel, "..") || filepath.IsAbs(cleanRel) {
		return "", fmt.Errorf("path traversal rejected: %q", relPath)
	}

	if strings.Contains(strings.ToUpper(cleanRel), "RCLONE_TEST") {
		return "", fmt.Errorf("safety violation: forbidden modification of canary item %q", relPath)
	}

	full := filepath.Join(root, cleanRel)
	rel, err := filepath.Rel(root, full)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path traversal rejected: %q", relPath)
	}

	return full, nil
}

// LocalBackup creates a zero-byte hardlink or fallbacks to a copy under backupDir.
func LocalBackup(srcPath, backupDir, sessionID, relPath string) (string, error) {
	fi, err := os.Stat(srcPath)
	if err != nil {
		return "", fmt.Errorf("cannot backup non-existent file %s: %w", srcPath, err)
	}

	sessionBackupDir := filepath.Join(backupDir, sessionID)
	destPath := filepath.Join(sessionBackupDir, filepath.Clean(relPath))
	destDir := filepath.Dir(destPath)

	if err := os.MkdirAll(destDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create backup directory %s: %w", destDir, err)
	}

	_ = os.Remove(destPath)

	// Attempt 1: Zero-byte POSIX Hardlink
	if err := os.Link(srcPath, destPath); err == nil {
		return destPath, nil
	}

	// Attempt 2: Fallback copy with disk space safety check
	if err := CheckFreeDiskSpace(destDir, fi.Size()); err != nil {
		return "", fmt.Errorf("backup copy aborted: %w", err)
	}

	// #nosec G304 - srcPath is strictly confined via ResolveSafePath
	srcFile, err := os.Open(filepath.Clean(srcPath))
	if err != nil {
		return "", fmt.Errorf("failed to open source for backup: %w", err)
	}
	defer srcFile.Close()

	// #nosec G304 - destPath is strictly confined via ResolveSafePath
	destFile, err := os.OpenFile(filepath.Clean(destPath), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return "", fmt.Errorf("failed to create backup file: %w", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, srcFile); err != nil {
		return "", fmt.Errorf("failed to copy data for backup: %w", err)
	}

	return destPath, nil
}

// FormatRemoteFs formats a remote identifier or local path into a valid rclone fs string.
// Local absolute or relative paths are preserved without a trailing colon.
// Remote identifiers are given a trailing colon (e.g., "gdrive:").
func FormatRemoteFs(remote string) string {
	clean := strings.TrimSpace(remote)
	if strings.HasPrefix(clean, "/") || strings.HasPrefix(clean, ".") {
		return strings.TrimSuffix(clean, ":")
	}
	if !strings.HasSuffix(clean, ":") {
		return clean + ":"
	}
	return clean
}

// RemoteBackup initiates a cloud server-side copy for backup with zero local bandwidth.
func RemoteBackup(ctx context.Context, client RemoteClient, remoteName, remotePath, sessionID string) (string, error) {
	if client == nil {
		return "", fmt.Errorf("remote client is nil")
	}

	remoteFs := FormatRemoteFs(remoteName)
	backupRemotePath := fmt.Sprintf(".review_backups/%s/%s", sessionID, filepath.Clean(remotePath))

	ok, err := client.CopyFile(ctx, remoteFs, remotePath, remoteFs, backupRemotePath)
	if err != nil || !ok {
		return "", fmt.Errorf("server-side remote backup failed: %w", err)
	}

	return backupRemotePath, nil
}

// RestoreLocalBackup restores a local file from its backup copy.
func RestoreLocalBackup(backupPath, targetPath string) error {
	fi, err := os.Stat(backupPath)
	if err != nil {
		return fmt.Errorf("backup file missing: %w", err)
	}

	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0700); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	_ = os.Remove(targetPath)

	// Attempt hardlink first
	if err := os.Link(backupPath, targetPath); err == nil {
		return nil
	}

	// Fallback to copy
	// #nosec G304 - backupPath is strictly confined via ResolveSafePath
	src, err := os.Open(filepath.Clean(backupPath))
	if err != nil {
		return fmt.Errorf("failed to read backup: %w", err)
	}
	defer src.Close()

	// #nosec G304 - targetPath is strictly confined via ResolveSafePath
	dst, err := os.OpenFile(filepath.Clean(targetPath), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create restored target: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("failed to copy restore bytes: %w", err)
	}

	// Preserve mod time
	_ = os.Chtimes(targetPath, fi.ModTime(), fi.ModTime())
	return nil
}

// PruneExpiredBackups removes review backups older than ttl.
func PruneExpiredBackups(backupsDir string, ttl time.Duration) (int, error) {
	if backupsDir == "" || ttl <= 0 {
		return 0, nil
	}

	entries, err := os.ReadDir(backupsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	cutoff := time.Now().Add(-ttl)
	pruned := 0

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			target := filepath.Join(backupsDir, entry.Name())
			if err := os.RemoveAll(target); err == nil {
				pruned++
			}
		}
	}

	return pruned, nil
}
