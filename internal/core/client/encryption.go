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

// UnlockConfig sends POST /config/unlock to unlock the encrypted configuration file in memory.
func (c *RcloneClient) UnlockConfig(ctx context.Context, password string) error {
	payload := map[string]string{"configPassword": password}
	_, _, err := c.post(ctx, "config/unlock", payload, 5*time.Second)
	return err
}

// GetPID sends POST /core/pid to retrieve the process ID of the running rclone daemon.
func (c *RcloneClient) GetPID(ctx context.Context) (int, error) {
	_, body, err := c.post(ctx, "core/pid", nil, 2*time.Second)
	if err != nil {
		return 0, err
	}
	var res struct {
		PID int `json:"pid"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return 0, err
	}
	return res.PID, nil
}
