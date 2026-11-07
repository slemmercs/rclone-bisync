package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/models"
)

// GetConfigPaths queries /config/paths to obtain runtime configuration, cache, and temp directory paths.
func (c *RcloneClient) GetConfigPaths(ctx context.Context) (*models.ConfigPathsResponse, error) {
	resp, body, err := c.post(ctx, "config/paths", nil, 5*time.Second)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("config/paths returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var res models.ConfigPathsResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("failed to decode config/paths response: %w", err)
	}
	return &res, nil
}
