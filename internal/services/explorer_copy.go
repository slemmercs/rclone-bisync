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

	canonSrc := strings.TrimSuffix(strings.TrimSpace(srcRemote), ":")
	canonDst := strings.TrimSuffix(strings.TrimSpace(dstRemote), ":")
	if (srcIsLocal && dstIsLocal) || strings.EqualFold(canonSrc, canonDst) {
		if cleanSrcPath == cleanDstPath {
			return models.CopyResponse{Status: "error", Error: "Source and destination paths are identical"}, nil
		}
	}

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
		targetDstPath := cleanDstPath
		if isDir && cleanSrcPath != "" {
			srcDirName := filepath.Base(cleanSrcPath)
			if filepath.Base(cleanDstPath) != srcDirName {
				targetDstPath = filepath.Join(cleanDstPath, srcDirName)
			}
		}

		baseDst, targetDst, ok := isLocalPathJailed(rawBase, targetDstPath)
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
			targetDst := cleanDstPath
			if cleanSrcPath != "" {
				srcDirName := filepath.Base(cleanSrcPath)
				if filepath.Base(cleanDstPath) != srcDirName {
					if cleanDstPath != "" {
						targetDst = fmt.Sprintf("%s/%s", cleanDstPath, srcDirName)
					} else {
						targetDst = srcDirName
					}
				}
			}
			if targetDst != "" {
				dstFs = fmt.Sprintf("%s%s", remoteName, targetDst)
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

	if (srcIsLocal && dstIsLocal) || strings.EqualFold(canonSrc, canonDst) {
		if isDir {
			if srcFs == dstFs {
				return models.CopyResponse{Status: "error", Error: "Source and destination paths are identical"}, nil
			}
		} else {
			if fmt.Sprintf("%s/%s", strings.TrimRight(srcFs, "/"), srcFile) == fmt.Sprintf("%s/%s", strings.TrimRight(dstFs, "/"), dstFile) {
				return models.CopyResponse{Status: "error", Error: "Source and destination paths are identical"}, nil
			}
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
		return models.CopyResponse{Status: "success", Src: srcFs, Dst: dstFs}, nil
	}

	ok, err := e.Client.CopyFile(ctx, srcFs, srcFile, dstFs, dstFile)
	if err != nil || !ok {
		return models.CopyResponse{Status: "error", Error: fmt.Sprintf("copy file failed: %v", err)}, nil
	}
	return models.CopyResponse{Status: "success", Src: fmt.Sprintf("%s/%s", srcFs, srcFile), Dst: fmt.Sprintf("%s/%s", dstFs, dstFile)}, nil
}
