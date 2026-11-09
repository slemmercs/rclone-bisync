package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/models"
)

// MoveFile moves a single file between remote or local filesystems via rclone RC operations/movefile.
func (c *RcloneClient) MoveFile(ctx context.Context, srcFs, srcRemote, dstFs, dstRemote string) error {
	payload := map[string]interface{}{
		"srcFs":     NormalizeFs(srcFs),
		"srcRemote": strings.Trim(strings.TrimSpace(srcRemote), "/"),
		"dstFs":     NormalizeFs(dstFs),
		"dstRemote": strings.Trim(strings.TrimSpace(dstRemote), "/"),
	}

	resp, body, err := c.post(ctx, "operations/movefile", payload, 60*time.Second)
	if err != nil {
		return fmt.Errorf("movefile request failed: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("movefile failed (HTTP %d): %s", resp.StatusCode, string(body))
	}
	return nil
}

// MoveDir moves a directory from srcFs to dstFs via rclone RC operations/movedir.
func (c *RcloneClient) MoveDir(ctx context.Context, srcFs, dstFs string) error {
	payload := map[string]interface{}{
		"srcFs": NormalizeFs(srcFs),
		"dstFs": NormalizeFs(dstFs),
	}

	resp, body, err := c.post(ctx, "operations/movedir", payload, 120*time.Second)
	if err != nil {
		return fmt.Errorf("movedir request failed: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("movedir failed (HTTP %d): %s", resp.StatusCode, string(body))
	}
	return nil
}

// Purge recursively purges a directory and all its contents via rclone RC operations/purge.
func (c *RcloneClient) Purge(ctx context.Context, fs, remote string) error {
	payload := map[string]interface{}{
		"fs":     NormalizeFs(fs),
		"remote": strings.Trim(strings.TrimSpace(remote), "/"),
	}

	resp, body, err := c.post(ctx, "operations/purge", payload, 60*time.Second)
	if err != nil {
		return fmt.Errorf("purge request failed: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("purge failed (HTTP %d): %s", resp.StatusCode, string(body))
	}
	return nil
}

// PublicLink retrieves or removes a public share link for an item via rclone RC operations/publiclink.
func (c *RcloneClient) PublicLink(ctx context.Context, fs, remote string, unlink bool) (*models.PublicLinkResponse, error) {
	payload := map[string]interface{}{
		"fs":     NormalizeFs(fs),
		"remote": strings.Trim(strings.TrimSpace(remote), "/"),
		"unlink": unlink,
	}

	resp, body, err := c.post(ctx, "operations/publiclink", payload, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("publiclink request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		bodyStr := string(body)
		if strings.Contains(strings.ToLower(bodyStr), "not supported") || strings.Contains(strings.ToLower(bodyStr), "doesn't support") {
			return &models.PublicLinkResponse{
				Status:    "error",
				Supported: false,
				Error:     "Remote provider does not support public sharing links",
			}, nil
		}
		return nil, fmt.Errorf("publiclink failed (HTTP %d): %s", resp.StatusCode, bodyStr)
	}

	var res struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("failed to parse publiclink response: %w", err)
	}

	return &models.PublicLinkResponse{
		Status:    "success",
		URL:       res.URL,
		Supported: true,
	}, nil
}
