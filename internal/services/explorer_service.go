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

// BrowseOptions defines optional flags and filtering when exploring directories.
type BrowseOptions struct {
	LocalBase       string
	SkipGdocs       bool
	ExcludePatterns []string
}

// BrowsePath lists directory items and verifies canary presence, enforcing path jails for local FS.
func (e *ExplorerService) BrowsePath(ctx context.Context, remote, path string, localBase ...string) client.DirectoryListing {
	opts := BrowseOptions{}
	if len(localBase) > 0 {
		opts.LocalBase = localBase[0]
	}
	return e.BrowsePathWithOptions(ctx, remote, path, opts)
}

// BrowsePathWithOptions lists directory items and verifies canary presence with configurable options.
func (e *ExplorerService) BrowsePathWithOptions(ctx context.Context, remote, path string, opts BrowseOptions) client.DirectoryListing {
	cleanPath := strings.Trim(path, "/")
	if strings.TrimSpace(cleanPath) == "" {
		cleanPath = ""
	}
	remoteClean := strings.TrimSpace(remote)

	if strings.ToLower(remoteClean) == "local" || remoteClean == "" {
		rawBase := opts.LocalBase
		if rawBase == "" && e.Config != nil {
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
		cleanBase := filepath.Clean(baseDir)
		basePrefix := cleanBase + string(filepath.Separator)
		if targetDir != cleanBase && !strings.HasPrefix(targetDir, basePrefix) {
			return client.DirectoryListing{
				Remote: "local",
				Path:   cleanPath,
				Error:  "Directory traversal forbidden",
			}
		}

		fi, err := os.Stat(targetDir)
		if os.IsNotExist(err) && cleanPath != "" {
			// Auto-recovery: check if parent directory has a matching folder (e.g. with trailing whitespace)
			parentRel := filepath.Dir(cleanPath)
			if parentRel == "." {
				parentRel = ""
			}
			_, parentAbs, parentOk := isLocalPathJailed(rawBase, parentRel)
			if parentOk {
				if parentEntries, pErr := os.ReadDir(parentAbs); pErr == nil {
					targetBaseName := filepath.Base(cleanPath)
					trimmedTarget := strings.TrimSpace(targetBaseName)
					for _, pe := range parentEntries {
						if pe.IsDir() && (strings.TrimSpace(pe.Name()) == trimmedTarget || strings.EqualFold(strings.TrimSpace(pe.Name()), trimmedTarget)) {
							if parentRel != "" {
								cleanPath = parentRel + "/" + pe.Name()
							} else {
								cleanPath = pe.Name()
							}
							_, targetDir, ok = isLocalPathJailed(rawBase, cleanPath)
							if ok {
								fi, err = os.Stat(targetDir)
							}
							break
						}
					}
				}
			}
		}
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
			if entry.Name() == ".review_backups" {
				continue
			}
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

			mimeType := "inode/directory"
			if !isDir {
				mimeType = DetectMimeType(entry.Name(), nil)
			}

			isSymlink := (entry.Type() & os.ModeSymlink) != 0

			itemMap := map[string]interface{}{
				"Path":     itemRelPath,
				"Name":     entry.Name(),
				"Size":     size,
				"MimeType": mimeType,
				"ModTime":  info.ModTime().UTC().Format(time.RFC3339),
				"IsDir":    isDir,
			}
			if isSymlink {
				itemMap["IsShortcut"] = true
			}
			items = append(items, itemMap)

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
	listing := e.Client.ListDirectory(ctx, remoteClean, cleanPath)
	if listing.Error != "" && cleanPath != "" && strings.Contains(strings.ToLower(listing.Error), "not found") {
		// Attempt transparent auto-resolution by inspecting the parent directory
		parentPath := ""
		targetName := cleanPath
		if lastSlash := strings.LastIndex(cleanPath, "/"); lastSlash != -1 {
			parentPath = cleanPath[:lastSlash]
			targetName = cleanPath[lastSlash+1:]
		}

		parentListing := e.Client.ListDirectory(ctx, remoteClean, parentPath)
		if parentListing.Error == "" && len(parentListing.Items) > 0 {
			var matchedItem map[string]interface{}
			trimmedTarget := strings.TrimSpace(targetName)
			for _, item := range parentListing.Items {
				isDir, _ := item["IsDir"].(bool)
				mime, _ := item["MimeType"].(string)
				id, _ := item["ID"].(string)
				isShortcut := mime == "application/vnd.google-apps.shortcut" || strings.Contains(id, "\t")
				if !isDir && !isShortcut {
					continue
				}
				name, _ := item["Name"].(string)
				if name == targetName || strings.TrimSpace(name) == trimmedTarget || strings.EqualFold(strings.TrimSpace(name), trimmedTarget) {
					matchedItem = item
					break
				}
			}

			if matchedItem != nil {
				actualPath, _ := matchedItem["Path"].(string)
				if actualPath == "" {
					actualName, _ := matchedItem["Name"].(string)
					if parentPath != "" {
						actualPath = parentPath + "/" + actualName
					} else {
						actualPath = actualName
					}
				}
				recovered := e.Client.ListDirectory(ctx, remoteClean, actualPath)
				if recovered.Error == "" {
					recovered.Path = actualPath
					listing = recovered
				}
			}
		}
	}

	if len(listing.Items) > 0 {
		filtered := make([]map[string]interface{}, 0, len(listing.Items))
		for _, item := range listing.Items {
			name, _ := item["Name"].(string)
			pathStr, _ := item["Path"].(string)
			if name == ".review_backups" || strings.HasPrefix(pathStr, ".review_backups") {
				continue
			}
			mime, _ := item["MimeType"].(string)
			id, _ := item["ID"].(string)

			if opts.SkipGdocs {
				isDir, _ := item["IsDir"].(bool)
				if !isDir {
					var size int64 = 0
					switch v := item["Size"].(type) {
					case int64:
						size = v
					case float64:
						size = int64(v)
					case int:
						size = int64(v)
					}
					if size < 0 || strings.HasPrefix(mime, "application/vnd.google-apps.") {
						continue
					}
				}
			}

			if mime == "application/vnd.google-apps.shortcut" || strings.Contains(id, "\t") {
				item["IsShortcut"] = true
			}
			filtered = append(filtered, item)
		}
		listing.Items = filtered
	}
	return listing
}
