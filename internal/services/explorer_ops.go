package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/slemmercs/rclone-bisync/internal/models"
)

// CopyItem transfers a file or directory with path jail enforcement.
func (e *ExplorerService) CopyItem(ctx context.Context, srcRemote, srcPath, dstRemote, dstPath string, isDir bool, localBase ...string) (models.CopyResponse, error) {
	base := ""
	if len(localBase) > 0 {
		base = localBase[0]
	}
	return e.CopyItemWithOptions(ctx, srcRemote, srcPath, dstRemote, dstPath, isDir, base, nil)
}

// CopyItemWithOptions transfers a file or directory with path jail enforcement and optional exclude filter rules.
func (e *ExplorerService) CopyItemWithOptions(ctx context.Context, srcRemote, srcPath, dstRemote, dstPath string, isDir bool, localBase string, excludeRules []string) (models.CopyResponse, error) {
	cleanSrcPath := strings.Trim(strings.TrimSpace(srcPath), "/")
	cleanDstPath := strings.Trim(strings.TrimSpace(dstPath), "/")
	srcIsLocal := strings.ToLower(strings.TrimSpace(srcRemote)) == "local" || strings.TrimSpace(srcRemote) == ""
	dstIsLocal := strings.ToLower(strings.TrimSpace(dstRemote)) == "local" || strings.TrimSpace(dstRemote) == ""

	rawBase := localBase
	if rawBase == "" && e.Config != nil {
		rawBase = e.Config.LocalPath
	}
	if rawBase == "" {
		rawBase = "~/gdrive"
	}

	var srcFs, srcFile, dstFs, dstFile string

	if srcIsLocal {
		baseSrc, targetSrc, ok := isLocalPathJailed(rawBase, cleanSrcPath)
		if !ok {
			return models.CopyResponse{Status: "error", Error: "Source path traversal forbidden"}, nil
		}
		cleanBaseSrc := filepath.Clean(baseSrc)
		srcPrefix := cleanBaseSrc + string(filepath.Separator)
		if targetSrc != cleanBaseSrc && !strings.HasPrefix(targetSrc, srcPrefix) {
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
		baseDst, targetDst, ok := isLocalPathJailed(rawBase, cleanDstPath)
		if !ok {
			return models.CopyResponse{Status: "error", Error: "Destination path traversal forbidden"}, nil
		}
		cleanBaseDst := filepath.Clean(baseDst)
		dstPrefix := cleanBaseDst + string(filepath.Separator)
		if targetDst != cleanBaseDst && !strings.HasPrefix(targetDst, dstPrefix) {
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
		ok, err := e.Client.CopyDir(ctx, srcFs, dstFs, excludeRules)
		if err != nil || !ok {
			return models.CopyResponse{Status: "error", Error: fmt.Sprintf("copy dir failed: %v", err)}, nil
		}
		// DeleteItem removed from here
		return models.CopyResponse{Status: "success", Src: srcFs, Dst: dstFs}, nil
	}

	ok, err := e.Client.CopyFile(ctx, srcFs, srcFile, dstFs, dstFile)
	if err != nil || !ok {
		return models.CopyResponse{Status: "error", Error: fmt.Sprintf("copy file failed: %v", err)}, nil
	}
	return models.CopyResponse{Status: "success", Src: fmt.Sprintf("%s/%s", srcFs, srcFile), Dst: fmt.Sprintf("%s/%s", dstFs, dstFile)}, nil
}

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
