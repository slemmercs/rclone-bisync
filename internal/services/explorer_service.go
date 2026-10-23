package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/core/client"
	"github.com/slemmercs/rclone-bisync/internal/models"
)

// ExplorerService manages directory browsing, item copying, and deletion with strict path jail enforcement.
type ExplorerService struct {
	Config *config.Config
	Client *client.RcloneClient
	Canary *CanaryService
}

// NewExplorerService initializes a new ExplorerService.
func NewExplorerService(cfg *config.Config, rcClient *client.RcloneClient, canary *CanaryService) *ExplorerService {
	if canary == nil {
		canary = NewCanaryService(cfg, rcClient)
	}
	return &ExplorerService{
		Config: cfg,
		Client: rcClient,
		Canary: canary,
	}
}

// IsLocalPathJailed verifies that relPath resolved against rawBaseDir is strictly contained inside rawBaseDir.
// It resolves real paths on disk using filepath.EvalSymlinks to prevent symlink jail escapes.
// It returns baseAbs, targetAbs, and true if safe, or false if path traversal is detected.
func IsLocalPathJailed(rawBaseDir, relPath string) (string, string, bool) {
	if strings.HasPrefix(rawBaseDir, "~") {
		home, _ := os.UserHomeDir()
		rawBaseDir = filepath.Join(home, strings.TrimPrefix(rawBaseDir, "~"))
	}
	baseDir, err := filepath.Abs(rawBaseDir)
	if err != nil {
		return "", "", false
	}
	targetDir, err := filepath.Abs(filepath.Join(baseDir, relPath))
	if err != nil {
		return baseDir, "", false
	}
	// First: lexical check
	rel, err := filepath.Rel(baseDir, targetDir)
	if err != nil || strings.HasPrefix(rel, "..") {
		return baseDir, targetDir, false
	}

	// Second: symlink realpath confinement check
	realBase, err := filepath.EvalSymlinks(baseDir)
	if err == nil {
		fi, lstatErr := os.Lstat(targetDir)
		if lstatErr == nil && fi.Mode()&os.ModeSymlink != 0 {
			linkDest, err := os.Readlink(targetDir)
			if err == nil {
				if !filepath.IsAbs(linkDest) {
					linkDest = filepath.Join(filepath.Dir(targetDir), linkDest)
				}
				realDest, err := filepath.EvalSymlinks(linkDest)
				if err != nil {
					realDest, _ = filepath.Abs(linkDest)
				}
				realRel, err := filepath.Rel(realBase, realDest)
				if err != nil || strings.HasPrefix(realRel, "..") {
					return baseDir, targetDir, false
				}
			}
		}

		realTarget, err := filepath.EvalSymlinks(targetDir)
		if err == nil {
			realRel, err := filepath.Rel(realBase, realTarget)
			if err != nil || strings.HasPrefix(realRel, "..") {
				return baseDir, targetDir, false
			}
		} else if os.IsNotExist(err) {
			// Target does not exist yet (e.g. file to be created or copied)
			// Verify closest existing parent directory
			parentDir := filepath.Dir(targetDir)
			for {
				if realParent, pErr := filepath.EvalSymlinks(parentDir); pErr == nil {
					realRel, rErr := filepath.Rel(realBase, realParent)
					if rErr != nil || strings.HasPrefix(realRel, "..") {
						return baseDir, targetDir, false
					}
					break
				}
				nextParent := filepath.Dir(parentDir)
				if nextParent == parentDir {
					break
				}
				parentDir = nextParent
			}
		}
	}

	return baseDir, targetDir, true
}

func isLocalPathJailed(rawBaseDir, relPath string) (string, string, bool) {
	return IsLocalPathJailed(rawBaseDir, relPath)
}

// BrowsePath lists directory items and verifies canary presence, enforcing path jails for local FS.
func (e *ExplorerService) BrowsePath(ctx context.Context, remote, path string) client.DirectoryListing {
	cleanPath := strings.Trim(strings.TrimSpace(path), "/")
	remoteClean := strings.TrimSpace(remote)

	if strings.ToLower(remoteClean) == "local" || remoteClean == "" {
		rawBase := ""
		if e.Config != nil {
			rawBase = e.Config.LocalPath
		}
		if rawBase == "" {
			rawBase = "~/gdrive"
		}

		_, targetDir, ok := isLocalPathJailed(rawBase, cleanPath)
		if !ok {
			return client.DirectoryListing{
				Remote: "local",
				Path:   cleanPath,
				Error:  "Directory traversal forbidden",
			}
		}

		fi, err := os.Stat(targetDir)
		if os.IsNotExist(err) {
			return client.DirectoryListing{
				Remote: "local",
				Path:   cleanPath,
				Error:  fmt.Sprintf("Directory does not exist: %s", cleanPath),
			}
		}
		if err != nil || !fi.IsDir() {
			return client.DirectoryListing{
				Remote: "local",
				Path:   cleanPath,
				Error:  "Target is not a directory",
			}
		}

		entries, err := os.ReadDir(targetDir)
		if err != nil {
			return client.DirectoryListing{
				Remote: "local",
				Path:   cleanPath,
				Error:  err.Error(),
			}
		}

		items := make([]map[string]interface{}, 0, len(entries))
		hasCanary := false

		for _, entry := range entries {
			info, err := entry.Info()
			if err != nil {
				continue
			}

			isDir := entry.IsDir()
			var size int64 = -1
			if !isDir {
				size = info.Size()
			}

			itemRelPath := entry.Name()
			if cleanPath != "" {
				itemRelPath = fmt.Sprintf("%s/%s", cleanPath, entry.Name())
			}

			mimeType := "application/octet-stream"
			if isDir {
				mimeType = "inode/directory"
			}

			items = append(items, map[string]interface{}{
				"Path":     itemRelPath,
				"Name":     entry.Name(),
				"Size":     size,
				"MimeType": mimeType,
				"ModTime":  info.ModTime().UTC().Format(time.RFC3339),
				"IsDir":    isDir,
			})

			if entry.Name() == "RCLONE_TEST" {
				hasCanary = true
			}
		}

		// Sort: directories first, then alphabetical
		sort.Slice(items, func(i, j int) bool {
			dirI := items[i]["IsDir"].(bool)
			dirJ := items[j]["IsDir"].(bool)
			if dirI != dirJ {
				return dirI
			}
			return strings.ToLower(items[i]["Name"].(string)) < strings.ToLower(items[j]["Name"].(string))
		})

		return client.DirectoryListing{
			Remote:    "local",
			Path:      cleanPath,
			HasCanary: hasCanary,
			Items:     items,
		}
	}

	if e.Client == nil {
		return client.DirectoryListing{
			Remote: remoteClean,
			Path:   cleanPath,
			Error:  "client not configured",
		}
	}
	return e.Client.ListDirectory(ctx, remoteClean, cleanPath)
}

// CopyItem transfers a file or directory with path jail enforcement.
func (e *ExplorerService) CopyItem(ctx context.Context, srcRemote, srcPath, dstRemote, dstPath string, isDir bool) (models.CopyResponse, error) {
	cleanSrcPath := strings.Trim(strings.TrimSpace(srcPath), "/")
	cleanDstPath := strings.Trim(strings.TrimSpace(dstPath), "/")
	srcIsLocal := strings.ToLower(strings.TrimSpace(srcRemote)) == "local" || strings.TrimSpace(srcRemote) == ""
	dstIsLocal := strings.ToLower(strings.TrimSpace(dstRemote)) == "local" || strings.TrimSpace(dstRemote) == ""

	rawBase := ""
	if e.Config != nil {
		rawBase = e.Config.LocalPath
	}
	if rawBase == "" {
		rawBase = "~/gdrive"
	}

	var srcFs, srcFile, dstFs, dstFile string

	if srcIsLocal {
		_, targetSrc, ok := isLocalPathJailed(rawBase, cleanSrcPath)
		if !ok {
			return models.CopyResponse{Status: "error", Error: "Source path traversal forbidden"}, nil
		}

		if _, err := os.Stat(targetSrc); os.IsNotExist(err) {
			return models.CopyResponse{Status: "error", Error: fmt.Sprintf("Source does not exist: %s", cleanSrcPath)}, nil
		}

		if isDir {
			srcFs = targetSrc
			srcFile = ""
		} else {
			srcFs = filepath.Dir(targetSrc)
			srcFile = filepath.Base(targetSrc)
		}
	} else {
		remoteName := strings.TrimRight(srcRemote, ":") + ":"
		if isDir {
			if cleanSrcPath != "" {
				srcFs = fmt.Sprintf("%s%s", remoteName, cleanSrcPath)
			} else {
				srcFs = remoteName
			}
			srcFile = ""
		} else {
			parentDir := filepath.Dir(cleanSrcPath)
			if parentDir == "." || parentDir == "/" {
				srcFs = remoteName
			} else {
				srcFs = fmt.Sprintf("%s%s", remoteName, parentDir)
			}
			srcFile = filepath.Base(cleanSrcPath)
		}
	}

	if dstIsLocal {
		_, targetDst, ok := isLocalPathJailed(rawBase, cleanDstPath)
		if !ok {
			return models.CopyResponse{Status: "error", Error: "Destination path traversal forbidden"}, nil
		}

		if isDir {
			dstFs = targetDst
			dstFile = ""
		} else {
			fi, err := os.Stat(targetDst)
			if (err == nil && fi.IsDir()) || strings.HasSuffix(dstPath, "/") {
				dstFs = targetDst
				dstFile = srcFile
			} else {
				dstFs = filepath.Dir(targetDst)
				dstFile = filepath.Base(targetDst)
				if dstFile == "" || dstFile == "." {
					dstFile = srcFile
				}
			}
		}
	} else {
		remoteName := strings.TrimRight(dstRemote, ":") + ":"
		if isDir {
			if cleanDstPath != "" {
				dstFs = fmt.Sprintf("%s%s", remoteName, cleanDstPath)
			} else {
				dstFs = remoteName
			}
			dstFile = ""
		} else {
			dstFs = remoteName
			if cleanDstPath != "" {
				dstFs = fmt.Sprintf("%s%s", remoteName, cleanDstPath)
			}
			dstFile = srcFile
		}
	}

	if e.Client == nil {
		return models.CopyResponse{Status: "error", Error: "client not configured"}, nil
	}

	if isDir {
		ok, err := e.Client.CopyDir(ctx, srcFs, dstFs)
		if err != nil || !ok {
			return models.CopyResponse{Status: "error", Error: fmt.Sprintf("copy dir failed: %v", err)}, nil
		}
		return models.CopyResponse{Status: "success", Src: srcFs, Dst: dstFs}, nil
	}

	ok, err := e.Client.CopyFile(ctx, srcFs, srcFile, dstFs, dstFile)
	if err != nil || !ok {
		return models.CopyResponse{Status: "error", Error: fmt.Sprintf("copy file failed: %v", err)}, nil
	}
	return models.CopyResponse{Status: "success", Src: fmt.Sprintf("%s/%s", srcFs, srcFile), Dst: fmt.Sprintf("%s/%s", dstFs, dstFile)}, nil
}

// DeleteItem removes a file or directory, safeguarding RCLONE_TEST canary and enforcing path jail.
func (e *ExplorerService) DeleteItem(ctx context.Context, remote, path string, isDir bool) (map[string]interface{}, error) {
	cleanPath := strings.Trim(strings.TrimSpace(path), "/")
	if cleanPath == "" {
		return map[string]interface{}{"status": "error", "error": "Root directory cannot be deleted"}, nil
	}

	// Canary protection
	if filepath.Base(cleanPath) == "RCLONE_TEST" {
		return map[string]interface{}{"status": "error", "error": "Canary test file (RCLONE_TEST) cannot be deleted"}, nil
	}

	remoteClean := strings.TrimSpace(remote)
	isLocal := strings.ToLower(remoteClean) == "local" || remoteClean == ""

	if isLocal {
		rawBase := ""
		if e.Config != nil {
			rawBase = e.Config.LocalPath
		}
		if rawBase == "" {
			rawBase = "~/gdrive"
		}

		baseDir, target, ok := isLocalPathJailed(rawBase, cleanPath)
		if !ok || target == baseDir {
			return map[string]interface{}{"status": "error", "error": "Path traversal or root deletion forbidden"}, nil
		}

		if _, err := os.Stat(target); os.IsNotExist(err) {
			return map[string]interface{}{"status": "error", "error": fmt.Sprintf("Target does not exist: %s", cleanPath)}, nil
		}

		if isDir {
			// Recursive canary protection: refuse to delete directory containing RCLONE_TEST
			canaryBlocked := false
			if e.Canary != nil {
				canaryBlocked = e.Canary.CanaryInDir(ctx, remote, cleanPath)
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
				return map[string]interface{}{"status": "error", "error": "Directory contains canary test file (RCLONE_TEST) and cannot be deleted"}, nil
			}

			if err := os.RemoveAll(target); err != nil {
				return map[string]interface{}{"status": "error", "error": fmt.Sprintf("Failed to delete local directory: %v", err)}, nil
			}
		} else {
			if err := os.Remove(target); err != nil {
				return map[string]interface{}{"status": "error", "error": fmt.Sprintf("Failed to delete local file: %v", err)}, nil
			}
		}

		return map[string]interface{}{
			"status": "success",
			"msg":    fmt.Sprintf("Deleted %s", cleanPath),
			"remote": "local",
			"path":   cleanPath,
		}, nil
	}

	if e.Client == nil {
		return map[string]interface{}{"status": "error", "error": "client not configured"}, nil
	}

	remoteName := strings.TrimRight(remoteClean, ":") + ":"
	if isDir {
		if e.Canary != nil && e.Canary.CanaryInDir(ctx, remoteClean, cleanPath) {
			return map[string]interface{}{"status": "error", "error": "Directory contains canary test file (RCLONE_TEST) and cannot be deleted"}, nil
		}
		ok, err := e.Client.PurgeDir(ctx, remoteName, cleanPath)
		if err != nil || !ok {
			return map[string]interface{}{"status": "error", "error": fmt.Sprintf("Purge failed: %v", err)}, nil
		}
		return map[string]interface{}{"status": "success", "fs": remoteName, "remote": cleanPath}, nil
	}

	ok, err := e.Client.DeleteFile(ctx, remoteName, cleanPath)
	if err != nil || !ok {
		return map[string]interface{}{"status": "error", "error": fmt.Sprintf("Delete file failed: %v", err)}, nil
	}
	return map[string]interface{}{"status": "success", "fs": remoteName, "remote": cleanPath}, nil
}
