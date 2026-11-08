package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/slemmercs/rclone-bisync/internal/models"
)

// MoveItem moves a file or directory between paths/remotes with jail and canary safety.
func (e *ExplorerService) MoveItem(ctx context.Context, req models.MoveRequest, localBase ...string) (models.MoveResponse, error) {
	cleanSrcPath := strings.Trim(strings.TrimSpace(req.SrcPath), "/")
	cleanDstPath := strings.Trim(strings.TrimSpace(req.DstPath), "/")

	if req.SrcRemote == "" && req.Remote != "" {
		req.SrcRemote = req.Remote
	}
	if req.DstRemote == "" {
		if req.Remote != "" {
			req.DstRemote = req.Remote
		} else {
			req.DstRemote = req.SrcRemote
		}
	}

	if cleanSrcPath == "" {
		return models.MoveResponse{Status: "error", Error: "Root directory cannot be moved"}, nil
	}

	// Canary protection
	if filepath.Base(cleanSrcPath) == CanaryFilename || filepath.Base(cleanDstPath) == CanaryFilename {
		return models.MoveResponse{Status: "error", Error: "Canary test file (RCLONE_TEST) cannot be moved or overwritten"}, nil
	}

	srcIsLocal := strings.ToLower(strings.TrimSpace(req.SrcRemote)) == "local" || strings.TrimSpace(req.SrcRemote) == ""
	dstIsLocal := strings.ToLower(strings.TrimSpace(req.DstRemote)) == "local" || strings.TrimSpace(req.DstRemote) == ""

	rawBase := ""
	if len(localBase) > 0 && localBase[0] != "" {
		rawBase = localBase[0]
	} else if e.Config != nil {
		rawBase = e.Config.LocalPath
	}
	if rawBase == "" {
		rawBase = "~/gdrive"
	}

	if req.IsDir && e.Canary != nil && e.Canary.CanaryInDir(ctx, req.SrcRemote, cleanSrcPath, rawBase) {
		return models.MoveResponse{Status: "error", Error: "Directory contains canary test file (RCLONE_TEST) and cannot be moved"}, nil
	}

	var srcFs, srcFile string
	var targetSrc string

	if srcIsLocal {
		baseSrc, tSrc, ok := isLocalPathJailed(rawBase, cleanSrcPath)
		if !ok {
			return models.MoveResponse{Status: "error", Error: "Source path traversal forbidden"}, nil
		}
		cleanBaseSrc := filepath.Clean(baseSrc)
		srcPrefix := cleanBaseSrc + string(filepath.Separator)
		if tSrc != cleanBaseSrc && !strings.HasPrefix(tSrc, srcPrefix) {
			return models.MoveResponse{Status: "error", Error: "Source path traversal forbidden"}, nil
		}
		if _, err := os.Stat(tSrc); os.IsNotExist(err) {
			return models.MoveResponse{Status: "error", Error: fmt.Sprintf("Source does not exist: %s", cleanSrcPath)}, nil
		}
		targetSrc = tSrc
		if req.IsDir {
			srcFs = targetSrc
		} else {
			srcFs = filepath.Dir(targetSrc)
			srcFile = filepath.Base(targetSrc)
		}
	} else {
		remoteName := strings.TrimRight(req.SrcRemote, ":") + ":"
		if req.IsDir {
			srcFs = fmt.Sprintf("%s%s", remoteName, cleanSrcPath)
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
	return e.executeMove(ctx, req, rawBase, cleanSrcPath, cleanDstPath, srcIsLocal, dstIsLocal, srcFs, srcFile, targetSrc)
}
func (e *ExplorerService) executeMove(ctx context.Context, req models.MoveRequest, rawBase, cleanSrcPath, cleanDstPath string, srcIsLocal, dstIsLocal bool, srcFs, srcFile, targetSrc string) (models.MoveResponse, error) {
	var dstFs, dstFile, targetDst string
	if dstIsLocal {
		baseDst, tDst, ok := isLocalPathJailed(rawBase, cleanDstPath)
		if !ok {
			return models.MoveResponse{Status: "error", Error: "Destination path traversal forbidden"}, nil
		}
		cleanBaseDst := filepath.Clean(baseDst)
		dstPrefix := cleanBaseDst + string(filepath.Separator)
		if tDst != cleanBaseDst && !strings.HasPrefix(tDst, dstPrefix) {
			return models.MoveResponse{Status: "error", Error: "Destination path traversal forbidden"}, nil
		}
		targetDst = tDst
		if req.IsDir {
			dstFs = targetDst
		} else {
			fi, err := os.Stat(targetDst)
			if (err == nil && fi.IsDir()) || strings.HasSuffix(req.DstPath, "/") {
				targetDst = filepath.Join(targetDst, srcFile)
				dstFs = filepath.Dir(targetDst)
				dstFile = filepath.Base(targetDst)
			} else {
				dstFs = filepath.Dir(targetDst)
				dstFile = filepath.Base(targetDst)
				if dstFile == "" || dstFile == "." {
					dstFile = srcFile
					targetDst = filepath.Join(targetDst, srcFile)
				}
			}
		}
	} else {
		remoteName := strings.TrimRight(req.DstRemote, ":") + ":"
		if req.IsDir {
			dstFs = fmt.Sprintf("%s%s", remoteName, cleanDstPath)
		} else {
			dstFs = remoteName
			if cleanDstPath != "" {
				dstFs = fmt.Sprintf("%s%s", remoteName, cleanDstPath)
			}
			dstFile = srcFile
		}
	}

	if srcIsLocal && dstIsLocal {
		if err := os.MkdirAll(filepath.Dir(targetDst), 0750); err != nil {
			return models.MoveResponse{Status: "error", Error: fmt.Sprintf("Failed to prepare destination: %v", err)}, nil
		}
		if err := os.Rename(targetSrc, targetDst); err != nil {
			if e.Client != nil {
				if req.IsDir {
					if err := e.Client.MoveDir(ctx, srcFs, dstFs); err != nil {
						return models.MoveResponse{Status: "error", Error: fmt.Sprintf("Move directory failed: %v", err)}, nil
					}
				} else {
					if err := e.Client.MoveFile(ctx, srcFs, srcFile, dstFs, dstFile); err != nil {
						return models.MoveResponse{Status: "error", Error: fmt.Sprintf("Move file failed: %v", err)}, nil
					}
				}
			} else {
				return models.MoveResponse{Status: "error", Error: fmt.Sprintf("Failed to move: %v", err)}, nil
			}
		}
		return models.MoveResponse{
			Status: "success",
			Msg:    fmt.Sprintf("Moved %s to %s", cleanSrcPath, cleanDstPath),
			Src:    cleanSrcPath,
			Dst:    cleanDstPath,
		}, nil
	}

	if e.Client == nil {
		return models.MoveResponse{Status: "error", Error: "client not configured"}, nil
	}
	if req.IsDir {
		if err := e.Client.MoveDir(ctx, srcFs, dstFs); err != nil {
			return models.MoveResponse{Status: "error", Error: fmt.Sprintf("Move directory failed: %v", err)}, nil
		}
		return models.MoveResponse{Status: "success", Msg: fmt.Sprintf("Moved directory from %s to %s", cleanSrcPath, cleanDstPath), Src: srcFs, Dst: dstFs}, nil
	}
	if err := e.Client.MoveFile(ctx, srcFs, srcFile, dstFs, dstFile); err != nil {
		return models.MoveResponse{Status: "error", Error: fmt.Sprintf("Move file failed: %v", err)}, nil
	}
	return models.MoveResponse{Status: "success", Msg: fmt.Sprintf("Moved %s to %s", srcFile, dstFile), Src: fmt.Sprintf("%s/%s", srcFs, srcFile), Dst: fmt.Sprintf("%s/%s", dstFs, dstFile)}, nil
}
