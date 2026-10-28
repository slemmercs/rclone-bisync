package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

// DirectoryListing contains directory contents and canary indicator.
type DirectoryListing struct {
	Remote    string                   `json:"remote"`
	Path      string                   `json:"path"`
	HasCanary bool                     `json:"has_canary"`
	Items     []map[string]interface{} `json:"items"`
	Error     string                   `json:"error,omitempty"`
}

// NormalizeFs formats a remote identifier or local path into a valid rclone fs string.
// Absolute or relative local paths are preserved without a trailing colon.
// Remote identifiers are given a trailing colon (e.g., "gdrive:").
// Remote paths with subdirectories (e.g., "onedrive:documents") preserve the remote colon without adding another trailing colon.
func NormalizeFs(remote string) string {
	clean := strings.TrimSpace(remote)
	if clean == "" {
		return ""
	}
	if strings.HasPrefix(clean, "/") || strings.HasPrefix(clean, ".") || strings.HasPrefix(clean, "~") {
		localClean := strings.TrimRight(strings.TrimLeft(remote, " \t\n\r"), "\t\n\r")
		return strings.TrimSuffix(localClean, ":")
	}
	// Windows drive letter check (e.g., C:\path or C:/path)
	if len(clean) >= 2 && clean[1] == ':' && ((clean[0] >= 'a' && clean[0] <= 'z') || (clean[0] >= 'A' && clean[0] <= 'Z')) {
		if len(clean) == 2 || clean[2] == '/' || clean[2] == '\\' {
			localClean := strings.TrimRight(strings.TrimLeft(remote, " \t\n\r"), "\t\n\r")
			return strings.TrimSuffix(localClean, ":")
		}
	}
	if strings.Contains(clean, ":") {
		return clean
	}
	return clean + ":"
}

// ListDirectory queries /operations/list to explore contents non-recursively.
func (c *RcloneClient) ListDirectory(ctx context.Context, remote, path string) DirectoryListing {
	fs := NormalizeFs(remote)
	cleanPath := strings.Trim(path, "/")

	payload := map[string]interface{}{
		"fs":     fs,
		"remote": cleanPath,
		"opt":    map[string]interface{}{"recurse": false},
	}

	listing := DirectoryListing{
		Remote: fs,
		Path:   cleanPath,
		Items:  make([]map[string]interface{}, 0),
	}

	resp, body, err := c.post(ctx, "operations/list", payload, 10*time.Second)
	if err != nil {
		listing.Error = err.Error()
		return listing
	}
	if resp.StatusCode != http.StatusOK {
		listing.Error = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body))
		return listing
	}

	var res struct {
		List []map[string]interface{} `json:"list"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		listing.Error = err.Error()
		return listing
	}

	listing.Items = res.List
	for _, it := range listing.Items {
		if cleanPath != "" {
			name, _ := it["Name"].(string)
			p, _ := it["Path"].(string)
			if p == "" {
				p = name
			}
			if !strings.HasPrefix(p, cleanPath) {
				it["Path"] = fmt.Sprintf("%s/%s", cleanPath, p)
			}
		}
		if name, ok := it["Name"].(string); ok && name == "RCLONE_TEST" {
			listing.HasCanary = true
		}
	}

	return listing
}

// TouchFile calls /core/command with "touch" to create/update timestamp on a file.
func (c *RcloneClient) TouchFile(ctx context.Context, remote, path string) (bool, error) {
	fs := NormalizeFs(remote)
	cleanPath := strings.Trim(path, "/")

	var target string
	if strings.HasSuffix(fs, ":") {
		target = fs + cleanPath
	} else {
		target = filepath.Join(fs, cleanPath)
	}

	payload := map[string]interface{}{
		"command": "touch",
		"arg":     []string{target},
	}

	resp, body, err := c.post(ctx, "core/command", payload, 15*time.Second)
	if err != nil {
		return false, err
	}
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("touch failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var cmdResp struct {
		Error  bool   `json:"error"`
		Result string `json:"result"`
	}
	if err := json.Unmarshal(body, &cmdResp); err == nil && cmdResp.Error {
		errMsg := strings.TrimSpace(cmdResp.Result)
		if errMsg == "" {
			errMsg = "unknown error from rclone touch"
		}
		return false, fmt.Errorf("touch failed: %s", errMsg)
	}

	return true, nil
}

// CopyFile calls /operations/copyfile to copy an individual file.
func (c *RcloneClient) CopyFile(ctx context.Context, srcFs, srcRemote, dstFs, dstRemote string) (bool, error) {
	payload := map[string]string{
		"srcFs":     NormalizeFs(srcFs),
		"srcRemote": srcRemote,
		"dstFs":     NormalizeFs(dstFs),
		"dstRemote": dstRemote,
	}
	resp, body, err := c.post(ctx, "operations/copyfile", payload, 30*time.Second)
	if err != nil {
		return false, err
	}
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("copyfile failed (HTTP %d): %s", resp.StatusCode, string(body))
	}
	return true, nil
}

// CopyDir calls /sync/copy to copy an entire directory tree with optional exclude filter rules.
func (c *RcloneClient) CopyDir(ctx context.Context, srcFs, dstFs string, excludeRules ...[]string) (bool, error) {
	payload := map[string]interface{}{
		"srcFs": NormalizeFs(srcFs),
		"dstFs": NormalizeFs(dstFs),
	}
	if len(excludeRules) > 0 && len(excludeRules[0]) > 0 {
		payload["_filter"] = map[string]interface{}{
			"ExcludeRule": excludeRules[0],
		}
	}
	resp, body, err := c.post(ctx, "sync/copy", payload, 120*time.Second)
	if err != nil {
		return false, err
	}
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("copy dir failed (HTTP %d): %s", resp.StatusCode, string(body))
	}
	return true, nil
}

// DeleteFile calls /operations/deletefile to delete a single file.
func (c *RcloneClient) DeleteFile(ctx context.Context, fs, remote string) (bool, error) {
	cleanFs := NormalizeFs(fs)
	payload := map[string]string{
		"fs":     cleanFs,
		"remote": strings.Trim(remote, "/"),
	}
	resp, body, err := c.post(ctx, "operations/deletefile", payload, 20*time.Second)
	if err != nil {
		return false, err
	}
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("deletefile failed (HTTP %d): %s", resp.StatusCode, string(body))
	}
	return true, nil
}

// PurgeDir calls /operations/purge to delete an entire directory.
func (c *RcloneClient) PurgeDir(ctx context.Context, fs, remote string) (bool, error) {
	cleanFs := NormalizeFs(fs)
	payload := map[string]string{
		"fs":     cleanFs,
		"remote": strings.Trim(remote, "/"),
	}
	resp, body, err := c.post(ctx, "operations/purge", payload, 30*time.Second)
	if err != nil {
		return false, err
	}
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("purge failed (HTTP %d): %s", resp.StatusCode, string(body))
	}
	return true, nil
}
