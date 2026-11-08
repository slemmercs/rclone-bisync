package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/slemmercs/rclone-bisync/internal/core/client"
	"github.com/slemmercs/rclone-bisync/internal/models"
)

// MakeDir creates a new directory in local or remote storage, safeguarding canary rules and path jails.
func (e *ExplorerService) MakeDir(ctx context.Context, req models.MkdirRequest, localBase ...string) (models.MkdirResponse, error) {
	cleanPath := strings.Trim(strings.TrimSpace(req.Path), "/")
	if cleanPath == "" {
		return models.MkdirResponse{Status: "error", Error: "Directory name/path cannot be empty"}, nil
	}

	// Canary protection: directory cannot be named RCLONE_TEST
	if filepath.Base(cleanPath) == CanaryFilename {
		return models.MkdirResponse{Status: "error", Error: "Directory cannot be named after safety canary (RCLONE_TEST)"}, nil
	}

	remoteClean := strings.TrimSpace(req.Remote)
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
			return models.MkdirResponse{Status: "error", Error: "Path traversal or invalid target directory"}, nil
		}
		cleanBase := filepath.Clean(baseDir)
		basePrefix := cleanBase + string(filepath.Separator)
		if target == cleanBase || !strings.HasPrefix(target, basePrefix) {
			return models.MkdirResponse{Status: "error", Error: "Path traversal or invalid target directory"}, nil
		}

		if err := os.MkdirAll(target, 0755); err != nil {
			return models.MkdirResponse{Status: "error", Error: fmt.Sprintf("Failed to create local directory: %v", err)}, nil
		}
		return models.MkdirResponse{
			Status: "success",
			Msg:    fmt.Sprintf("Created local directory: %s", cleanPath),
			Path:   cleanPath,
		}, nil
	}

	// Remote target directory creation via Rclone RC
	if e.Client == nil {
		return models.MkdirResponse{Status: "error", Error: "Rclone client unavailable for remote directory creation"}, nil
	}

	fs := client.NormalizeFs(remoteClean)
	ok, err := e.Client.Mkdir(ctx, fs, cleanPath)
	if err != nil || !ok {
		errMsg := "Failed to create remote directory"
		if err != nil {
			errMsg = fmt.Sprintf("Failed to create remote directory: %v", err)
		}
		return models.MkdirResponse{Status: "error", Error: errMsg}, nil
	}

	return models.MkdirResponse{
		Status: "success",
		Msg:    fmt.Sprintf("Created remote directory: %s", cleanPath),
		Path:   cleanPath,
	}, nil
}
