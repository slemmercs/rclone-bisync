package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/slemmercs/rclone-bisync/internal/models"
)

// PurgeDir purges an entire directory tree, safeguarding canary files and enforcing path jails.
func (e *ExplorerService) PurgeDir(ctx context.Context, req models.PurgeRequest, localBase ...string) (models.PurgeResponse, error) {
	cleanPath := strings.Trim(strings.TrimSpace(req.Path), "/")
	if cleanPath == "" {
		return models.PurgeResponse{Status: "error", Error: "Root directory cannot be purged"}, nil
	}

	// Canary protection
	if filepath.Base(cleanPath) == CanaryFilename {
		return models.PurgeResponse{Status: "error", Error: "Canary test file (RCLONE_TEST) cannot be purged"}, nil
	}

	remoteClean := strings.TrimSpace(req.Remote)
	isLocal := strings.ToLower(remoteClean) == "local" || remoteClean == ""

	rawBase := ""
	if len(localBase) > 0 && localBase[0] != "" {
		rawBase = localBase[0]
	} else if e.Config != nil {
		rawBase = e.Config.LocalPath
	}
	if rawBase == "" {
		rawBase = "~/gdrive"
	}

	if e.Canary != nil && e.Canary.CanaryInDir(ctx, req.Remote, cleanPath, rawBase) {
		return models.PurgeResponse{Status: "error", Error: "Directory contains canary test file (RCLONE_TEST) and cannot be purged"}, nil
	}

	if isLocal {
		baseDir, target, ok := isLocalPathJailed(rawBase, cleanPath)
		if !ok || target == baseDir {
			return models.PurgeResponse{Status: "error", Error: "Path traversal or root purge forbidden"}, nil
		}
		cleanBase := filepath.Clean(baseDir)
		basePrefix := cleanBase + string(filepath.Separator)
		if target == cleanBase || !strings.HasPrefix(target, basePrefix) {
			return models.PurgeResponse{Status: "error", Error: "Path traversal or root purge forbidden"}, nil
		}

		if _, err := os.Stat(target); os.IsNotExist(err) {
			return models.PurgeResponse{Status: "error", Error: fmt.Sprintf("Target does not exist: %s", cleanPath)}, nil
		}

		if err := os.RemoveAll(target); err != nil {
			return models.PurgeResponse{Status: "error", Error: fmt.Sprintf("Failed to purge local directory: %v", err)}, nil
		}

		return models.PurgeResponse{
			Status: "success",
			Msg:    fmt.Sprintf("Purged %s", cleanPath),
			Path:   cleanPath,
		}, nil
	}

	if e.Client == nil {
		return models.PurgeResponse{Status: "error", Error: "client not configured"}, nil
	}

	remoteName := strings.TrimRight(remoteClean, ":") + ":"
	if err := e.Client.Purge(ctx, remoteName, cleanPath); err != nil {
		return models.PurgeResponse{Status: "error", Error: fmt.Sprintf("Purge failed: %v", err)}, nil
	}

	return models.PurgeResponse{
		Status: "success",
		Msg:    fmt.Sprintf("Purged %s", cleanPath),
		Path:   cleanPath,
	}, nil
}

// PurgeDirectory is an alias for PurgeDir.
func (e *ExplorerService) PurgeDirectory(ctx context.Context, req models.PurgeRequest, localBase ...string) (models.PurgeResponse, error) {
	return e.PurgeDir(ctx, req, localBase...)
}

// GetPublicLink resolves a public sharing link for a cloud storage item.
func (e *ExplorerService) GetPublicLink(ctx context.Context, req models.PublicLinkRequest) (models.PublicLinkResponse, error) {
	cleanPath := strings.Trim(strings.TrimSpace(req.Path), "/")
	remoteClean := strings.TrimSpace(req.Remote)
	isLocal := strings.ToLower(remoteClean) == "local" || remoteClean == ""

	if isLocal {
		return models.PublicLinkResponse{
			Status:    "error",
			Supported: false,
			Error:     "Public sharing links are not supported for local storage",
		}, nil
	}

	if e.Client == nil {
		return models.PublicLinkResponse{
			Status:    "error",
			Supported: false,
			Error:     "client not configured",
		}, nil
	}

	remoteName := strings.TrimRight(remoteClean, ":") + ":"
	resp, err := e.Client.PublicLink(ctx, remoteName, cleanPath, req.Unlink)
	if err != nil {
		return models.PublicLinkResponse{
			Status:    "error",
			Supported: false,
			Error:     err.Error(),
		}, nil
	}

	return *resp, nil
}
