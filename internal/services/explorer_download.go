package services

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// DownloadStream returns an open io.ReadCloser, filename, and file size for direct downloading.
func (e *ExplorerService) DownloadStream(ctx context.Context, remote, path string, localBase ...string) (io.ReadCloser, string, int64, error) {
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

		_, targetAbs, safe := IsLocalPathJailed(rawBase, cleanPath)
		if !safe {
			return nil, "", 0, fmt.Errorf("access denied: path traversal detected")
		}

		fi, err := os.Stat(targetAbs)
		if err != nil {
			return nil, "", 0, fmt.Errorf("file not found: %w", err)
		}
		if fi.IsDir() {
			return nil, "", 0, fmt.Errorf("cannot download directory")
		}

		// #nosec G304 - targetAbs is strictly verified by IsLocalPathJailed
		f, err := os.Open(targetAbs)
		if err != nil {
			return nil, "", 0, fmt.Errorf("failed to open file: %w", err)
		}
		return f, filepath.Base(targetAbs), fi.Size(), nil
	}

	if e.Client == nil {
		return nil, "", 0, fmt.Errorf("rclone client is not configured")
	}

	item, err := e.Client.StatItem(ctx, remoteClean, cleanPath)
	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to stat remote item: %w", err)
	}

	if isDir, _ := item["IsDir"].(bool); isDir {
		return nil, "", 0, fmt.Errorf("cannot download directory")
	}

	var size int64
	if s, ok := item["Size"].(float64); ok {
		size = int64(s)
	}

	name, _ := item["Name"].(string)
	if name == "" {
		name = filepath.Base(cleanPath)
	}

	stream, _, err := e.Client.CatStream(ctx, remoteClean, cleanPath, 0, 0)
	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to open remote stream: %w", err)
	}

	return stream, name, size, nil
}
