package services

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/core/client"
)

// CanaryFilename is the standardized name of the safety canary file.
const CanaryFilename = "RCLONE_TEST"

// CanaryService handles canary verification, creation, and deletion protection.
type CanaryService struct {
	Config *config.Config
	Client *client.RcloneClient
}

// NewCanaryService initializes a new CanaryService.
func NewCanaryService(cfg *config.Config, rcClient *client.RcloneClient) *CanaryService {
	return &CanaryService{
		Config: cfg,
		Client: rcClient,
	}
}

// SafeguardCanary checks if the target path is a protected canary file.
func (c *CanaryService) SafeguardCanary(cleanPath string) error {
	if filepath.Base(cleanPath) == CanaryFilename {
		return errors.New("Canary test file (RCLONE_TEST) cannot be deleted")
	}
	return nil
}

// CanaryInDir checks whether a directory (or any of its subdirectories) contains RCLONE_TEST.
func (c *CanaryService) CanaryInDir(ctx context.Context, remote, path string) bool {
	cleanPath := strings.Trim(strings.TrimSpace(path), "/")
	remoteClean := strings.TrimSpace(remote)

	if strings.ToLower(remoteClean) == "local" || remoteClean == "" {
		rawBase := ""
		if c.Config != nil {
			rawBase = c.Config.LocalPath
		}
		if rawBase == "" {
			rawBase = "~/gdrive"
		}
		_, targetDir, ok := IsLocalPathJailed(rawBase, cleanPath)
		if !ok {
			return false
		}
		found := false
		_ = filepath.Walk(targetDir, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if !info.IsDir() && info.Name() == CanaryFilename {
				found = true
				return filepath.SkipAll
			}
			return nil
		})
		return found
	}

	if c.Client == nil {
		return false
	}

	listing := c.Client.ListDirectory(ctx, remoteClean, cleanPath)
	return listing.HasCanary
}

// CanaryExists checks if the canary file exists locally or remotely at the target path.
func (c *CanaryService) CanaryExists(ctx context.Context, remote, path string) bool {
	cleanPath := strings.Trim(strings.TrimSpace(path), "/")
	remoteClean := strings.TrimSpace(remote)

	if strings.ToLower(remoteClean) == "local" || remoteClean == "" {
		rawBase := c.Config.LocalPath
		if rawBase == "" {
			rawBase = "~/gdrive"
		}
		if strings.HasPrefix(rawBase, "~") {
			home, _ := os.UserHomeDir()
			rawBase = filepath.Join(home, strings.TrimPrefix(rawBase, "~"))
		}
		baseDir, err := filepath.Abs(rawBase)
		if err != nil {
			return false
		}

		targetDir, err := filepath.Abs(filepath.Join(baseDir, cleanPath))
		if err != nil {
			return false
		}

		rel, err := filepath.Rel(baseDir, targetDir)
		if err != nil || strings.HasPrefix(rel, "..") {
			return false
		}

		canaryFile := filepath.Join(targetDir, CanaryFilename)
		fi, err := os.Stat(canaryFile)
		return err == nil && !fi.IsDir()
	}

	if c.Client == nil {
		return false
	}

	listing := c.Client.ListDirectory(ctx, remoteClean, cleanPath)
	return listing.HasCanary
}

// CreateCanary provisions an RCLONE_TEST canary file on target local or remote filesystem.
func (c *CanaryService) CreateCanary(ctx context.Context, remote, path string) (map[string]interface{}, error) {
	cleanPath := strings.Trim(strings.TrimSpace(path), "/")
	remoteClean := strings.TrimSpace(remote)

	if strings.ToLower(remoteClean) == "local" || remoteClean == "" {
		rawBase := c.Config.LocalPath
		if strings.HasPrefix(rawBase, "~") {
			home, _ := os.UserHomeDir()
			rawBase = filepath.Join(home, strings.TrimPrefix(rawBase, "~"))
		}
		baseDir, _ := filepath.Abs(rawBase)
		targetDir, _ := filepath.Abs(filepath.Join(baseDir, cleanPath))

		rel, err := filepath.Rel(baseDir, targetDir)
		if err != nil || strings.HasPrefix(rel, "..") {
			return map[string]interface{}{"status": "error", "error": "Directory traversal forbidden", "remote": "local", "path": cleanPath}, nil
		}

		if _, err := os.Stat(targetDir); os.IsNotExist(err) {
			return map[string]interface{}{"status": "error", "error": fmt.Sprintf("Target directory does not exist: %s", cleanPath), "remote": "local", "path": cleanPath}, nil
		}

		canaryFile := filepath.Join(targetDir, CanaryFilename)
		f, err := os.OpenFile(canaryFile, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return map[string]interface{}{"status": "error", "error": fmt.Sprintf("Failed to create canary file: %v", err), "remote": "local", "path": cleanPath}, nil
		}
		_ = f.Close()
		now := time.Now()
		_ = os.Chtimes(canaryFile, now, now)

		return map[string]interface{}{
			"status": "success",
			"msg":    fmt.Sprintf("Canary created successfully at %s/RCLONE_TEST", cleanPath),
			"remote": "local",
			"path":   cleanPath,
		}, nil
	}

	canaryRemotePath := CanaryFilename
	if cleanPath != "" {
		canaryRemotePath = fmt.Sprintf("%s/%s", cleanPath, CanaryFilename)
	}

	if c.Client == nil {
		return map[string]interface{}{"status": "error", "error": "Rclone client unavailable", "remote": remoteClean, "path": cleanPath}, nil
	}

	ok, err := c.Client.TouchFile(ctx, remoteClean, canaryRemotePath)
	if err != nil || !ok {
		errStr := "Failed to create remote canary"
		if err != nil {
			errStr = err.Error()
		}
		return map[string]interface{}{"status": "error", "error": errStr, "remote": remoteClean, "path": cleanPath}, nil
	}

	return map[string]interface{}{
		"status": "success",
		"msg":    fmt.Sprintf("Canary created successfully at %s:%s", remoteClean, canaryRemotePath),
		"remote": remoteClean,
		"path":   cleanPath,
	}, nil
}
