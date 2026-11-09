package services

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/slemmercs/rclone-bisync/internal/models"
)

// DeleteItem removes a file or directory, safeguarding RCLONE_TEST canary and enforcing path jail.
func (e *ExplorerService) DeleteItem(ctx context.Context, remote, path string, isDir bool, localBase ...string) (models.DeleteResponse, error) {
	cleanPath := strings.Trim(strings.TrimSpace(path), "/")
	if cleanPath == "" {
		return models.DeleteResponse{Status: "error", Error: "Root directory cannot be deleted"}, nil
	}

	// Canary protection
	if filepath.Base(cleanPath) == "RCLONE_TEST" {
		return models.DeleteResponse{Status: "error", Error: "Canary test file (RCLONE_TEST) cannot be deleted"}, nil
	}

	remoteClean := strings.TrimSpace(remote)
	isLocal := strings.ToLower(remoteClean) == "local" || remoteClean == ""

	if isLocal {
		rawBase := ""
		if len(localBase) > 0 && localBase[0] != "" {
			rawBase = localBase[0]
		} else if e.Config != nil {
			rawBase = e.Config.LocalPath
		}
		if rawBase == "" {
			rawBase = "~/gdrive"
		}

		baseDir, target, ok := isLocalPathJailed(rawBase, cleanPath)
		if !ok || target == baseDir {
			return models.DeleteResponse{Status: "error", Error: "Path traversal or root deletion forbidden"}, nil
		}
		cleanBase := filepath.Clean(baseDir)
		basePrefix := cleanBase + string(filepath.Separator)
		if target == cleanBase || !strings.HasPrefix(target, basePrefix) {
			return models.DeleteResponse{Status: "error", Error: "Path traversal or root deletion forbidden"}, nil
		}

		if _, err := os.Stat(target); os.IsNotExist(err) {
			return models.DeleteResponse{Status: "error", Error: fmt.Sprintf("Target does not exist: %s", cleanPath)}, nil
		}

		if isDir {
			// Recursive canary protection: refuse to delete directory containing RCLONE_TEST
			canaryBlocked := false
			if e.Canary != nil {
				canaryBlocked = e.Canary.CanaryInDir(ctx, remote, cleanPath, rawBase)
			} else {
				_ = filepath.Walk(target, func(p string, info os.FileInfo, err error) error {
					if err != nil {
						return nil
					}
					if !info.IsDir() && info.Name() == CanaryFilename {
						canaryBlocked = true
						return filepath.SkipAll
					}
					return nil
				})
			}
			if canaryBlocked {
				return models.DeleteResponse{Status: "error", Error: "Directory contains canary test file (RCLONE_TEST) and cannot be deleted"}, nil
			}

			if err := os.RemoveAll(target); err != nil {
				return models.DeleteResponse{Status: "error", Error: fmt.Sprintf("Failed to delete local directory: %v", err)}, nil
			}
		} else {
			if err := os.Remove(target); err != nil {
				return models.DeleteResponse{Status: "error", Error: fmt.Sprintf("Failed to delete local file: %v", err)}, nil
			}
		}

		return models.DeleteResponse{
			Status: "success",
			Msg:    fmt.Sprintf("Deleted %s", cleanPath),
			Remote: "local",
			Path:   cleanPath,
		}, nil
	}

	if e.Client == nil {
		return models.DeleteResponse{Status: "error", Error: "client not configured"}, nil
	}

	remoteName := strings.TrimRight(remoteClean, ":") + ":"
	if isDir {
		if e.Canary != nil && e.Canary.CanaryInDir(ctx, remoteClean, cleanPath) {
			return models.DeleteResponse{Status: "error", Error: "Directory contains canary test file (RCLONE_TEST) and cannot be deleted"}, nil
		}
		ok, err := e.Client.PurgeDir(ctx, remoteName, cleanPath)
		if err != nil || !ok {
			return models.DeleteResponse{Status: "error", Error: fmt.Sprintf("Purge failed: %v", err)}, nil
		}
		return models.DeleteResponse{Status: "success", Fs: remoteName, Remote: cleanPath, Path: cleanPath}, nil
	}

	ok, err := e.Client.DeleteFile(ctx, remoteName, cleanPath)
	if err != nil || !ok {
		return models.DeleteResponse{Status: "error", Error: fmt.Sprintf("Delete file failed: %v", err)}, nil
	}
	return models.DeleteResponse{Status: "success", Fs: remoteName, Remote: cleanPath, Path: cleanPath}, nil
}

// UploadFile saves a streamed file to the specified remote directory, enforcing path jails and canary protection.
func (e *ExplorerService) UploadFile(ctx context.Context, remote, dirPath, fileName string, reader io.Reader, size int64, localBase ...string) (models.UploadResponse, error) {
	cleanDirPath := strings.Trim(strings.TrimSpace(dirPath), "/")
	cleanFileName := filepath.Base(strings.TrimSpace(fileName))

	if cleanFileName == "" || cleanFileName == "." || cleanFileName == "/" || strings.Contains(fileName, "..") || strings.Contains(fileName, "/") || strings.Contains(fileName, "\\") {
		return models.UploadResponse{Status: "error", Error: "Invalid filename or path traversal detected"}, nil
	}

	if cleanFileName == CanaryFilename {
		return models.UploadResponse{Status: "error", Error: "Canary test file (RCLONE_TEST) cannot be uploaded or overwritten"}, nil
	}

	remoteClean := strings.TrimSpace(remote)
	isLocal := strings.ToLower(remoteClean) == "local" || remoteClean == ""

	if isLocal {
		rawBase := ""
		if len(localBase) > 0 && localBase[0] != "" {
			rawBase = localBase[0]
		} else if e.Config != nil {
			rawBase = e.Config.LocalPath
		}
		if rawBase == "" {
			rawBase = "~/gdrive"
		}

		baseDir, targetDir, ok := isLocalPathJailed(rawBase, cleanDirPath)
		if !ok {
			return models.UploadResponse{Status: "error", Error: "Directory traversal forbidden"}, nil
		}
		cleanBase := filepath.Clean(baseDir)
		basePrefix := cleanBase + string(filepath.Separator)
		if targetDir != cleanBase && !strings.HasPrefix(targetDir, basePrefix) {
			return models.UploadResponse{Status: "error", Error: "Directory traversal forbidden"}, nil
		}

		fi, err := os.Stat(targetDir)
		if err != nil || !fi.IsDir() {
			return models.UploadResponse{Status: "error", Error: fmt.Sprintf("Target directory does not exist: %s", cleanDirPath)}, nil
		}

		destPath := filepath.Join(targetDir, cleanFileName)
		tmpFile, err := os.CreateTemp(targetDir, fmt.Sprintf(".upload-%d-*.tmp", os.Getpid()))
		if err != nil {
			return models.UploadResponse{Status: "error", Error: fmt.Sprintf("Failed to create temporary file: %v", err)}, nil
		}
		tmpPath := tmpFile.Name()
		defer func() {
			_ = tmpFile.Close()
			_ = os.Remove(tmpPath)
		}()

		written, err := io.Copy(tmpFile, reader)
		if err != nil {
			return models.UploadResponse{Status: "error", Error: fmt.Sprintf("Failed to write uploaded content: %v", err)}, nil
		}
		_ = tmpFile.Close()

		if err := os.Chmod(tmpPath, 0600); err != nil {
			return models.UploadResponse{Status: "error", Error: fmt.Sprintf("Failed to set file permissions: %v", err)}, nil
		}

		if err := os.Rename(tmpPath, destPath); err != nil {
			return models.UploadResponse{Status: "error", Error: fmt.Sprintf("Failed to commit uploaded file: %v", err)}, nil
		}

		relPath := cleanFileName
		if cleanDirPath != "" {
			relPath = fmt.Sprintf("%s/%s", cleanDirPath, cleanFileName)
		}

		return models.UploadResponse{
			Status: "success",
			Msg:    fmt.Sprintf("Uploaded %s", cleanFileName),
			Remote: "local",
			Path:   relPath,
			Size:   written,
		}, nil
	}

	// Remote destination
	if e.Client == nil {
		return models.UploadResponse{Status: "error", Error: "client not configured"}, nil
	}

	tmpDir, err := os.MkdirTemp("", "rclone-upload-*")
	if err != nil {
		return models.UploadResponse{Status: "error", Error: fmt.Sprintf("Failed to create temporary staging directory: %v", err)}, nil
	}
	defer os.RemoveAll(tmpDir)

	tmpFilePath := filepath.Join(tmpDir, cleanFileName)
	// #nosec G304 - staged file created in isolated temporary directory
	tmpFile, err := os.OpenFile(tmpFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return models.UploadResponse{Status: "error", Error: fmt.Sprintf("Failed to create staged upload file: %v", err)}, nil
	}
	written, err := io.Copy(tmpFile, reader)
	_ = tmpFile.Close()
	if err != nil {
		return models.UploadResponse{Status: "error", Error: fmt.Sprintf("Failed to write staged content: %v", err)}, nil
	}

	remoteName := strings.TrimRight(remoteClean, ":") + ":"
	var dstFs string
	if cleanDirPath != "" {
		dstFs = fmt.Sprintf("%s%s", remoteName, cleanDirPath)
	} else {
		dstFs = remoteName
	}

	ok, err := e.Client.CopyFile(ctx, tmpDir, cleanFileName, dstFs, cleanFileName)
	if err != nil || !ok {
		return models.UploadResponse{Status: "error", Error: fmt.Sprintf("Remote upload failed: %v", err)}, nil
	}

	relPath := cleanFileName
	if cleanDirPath != "" {
		relPath = fmt.Sprintf("%s/%s", cleanDirPath, cleanFileName)
	}

	return models.UploadResponse{
		Status: "success",
		Msg:    fmt.Sprintf("Uploaded %s", cleanFileName),
		Remote: remoteClean,
		Path:   relPath,
		Size:   written,
	}, nil
}
