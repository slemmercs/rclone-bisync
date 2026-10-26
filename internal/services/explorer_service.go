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

// BrowsePath lists directory items and verifies canary presence, enforcing path jails for local FS.
func (e *ExplorerService) BrowsePath(ctx context.Context, remote, path string, localBase ...string) client.DirectoryListing {
	cleanPath := strings.Trim(strings.TrimSpace(path), "/")
	remoteClean := strings.TrimSpace(remote)

	if strings.ToLower(remoteClean) == "local" || remoteClean == "" {
		rawBase := ""
		if len(localBase) > 0 && localBase[0] != "" {
			rawBase = localBase[0]
		} else if e.Config != nil {
			rawBase = e.Config.LocalPath
		}
		if rawBase == "" {
			rawBase = "~/gdrive"
		}

		baseDir, targetDir, ok := isLocalPathJailed(rawBase, cleanPath)
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

		rootCanary := false
		if fi, err := os.Stat(filepath.Join(baseDir, "RCLONE_TEST")); err == nil && !fi.IsDir() {
			rootCanary = true
		}

		return client.DirectoryListing{
			Remote:    "local",
			Path:      cleanPath,
			HasCanary: rootCanary || hasCanary,
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
