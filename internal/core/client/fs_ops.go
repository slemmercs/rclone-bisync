package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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

// ListDirectory queries /operations/list to explore contents non-recursively.
func (c *RcloneClient) ListDirectory(ctx context.Context, remote, path string) DirectoryListing {
	fs := remote
	if !strings.HasSuffix(fs, ":") {
		fs += ":"
	}
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

// TouchFile calls /operations/touch to create/update timestamp on a file.
func (c *RcloneClient) TouchFile(ctx context.Context, remote, path string) (bool, error) {
	fs := remote
	if !strings.HasSuffix(fs, ":") {
		fs += ":"
	}
	cleanPath := strings.Trim(path, "/")

	payload := map[string]string{
		"fs":     fs,
		"remote": cleanPath,
	}

	resp, body, err := c.post(ctx, "operations/touch", payload, 10*time.Second)
	if err != nil {
		return false, err
	}
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("touch failed (HTTP %d): %s", resp.StatusCode, string(body))
	}
	return true, nil
}

// CopyFile calls /operations/copyfile to copy an individual file.
func (c *RcloneClient) CopyFile(ctx context.Context, srcFs, srcRemote, dstFs, dstRemote string) (bool, error) {
	payload := map[string]string{
		"srcFs":     srcFs,
		"srcRemote": srcRemote,
		"dstFs":     dstFs,
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

// CopyDir calls /sync/copy to copy an entire directory tree.
func (c *RcloneClient) CopyDir(ctx context.Context, srcFs, dstFs string) (bool, error) {
	payload := map[string]string{
		"srcFs": srcFs,
		"dstFs": dstFs,
	}
	resp, body, err := c.post(ctx, "sync/copy", payload, 60*time.Second)
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
	cleanFs := fs
	if !strings.HasSuffix(cleanFs, ":") {
		cleanFs += ":"
	}
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
	cleanFs := fs
	if !strings.HasSuffix(cleanFs, ":") {
		cleanFs += ":"
	}
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
