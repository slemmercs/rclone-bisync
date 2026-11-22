package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// BackendCommand issues a command to a backend running on a remote via rclone RC backend/command.
// Examples include hasher commands (dump, drop, import, stickyimport), s3 commands, etc.
func (c *RcloneClient) BackendCommand(ctx context.Context, fs, command string, args []string, opt map[string]string) (interface{}, error) {
	if c == nil {
		return nil, fmt.Errorf("rclone client not configured")
	}

	cleanFs := strings.TrimSpace(fs)
	if cleanFs == "" {
		return nil, fmt.Errorf("fs cannot be empty")
	}
	cleanCommand := strings.TrimSpace(command)
	if cleanCommand == "" {
		return nil, fmt.Errorf("command cannot be empty")
	}

	normalizedFs := NormalizeFs(cleanFs)

	payload := map[string]interface{}{
		"fs":      normalizedFs,
		"command": cleanCommand,
	}
	if len(args) > 0 {
		payload["arg"] = args
	}
	if len(opt) > 0 {
		payload["opt"] = opt
	}

	// Backend commands such as large database dumps or import verification may take up to 60 seconds
	resp, body, err := c.post(ctx, "backend/command", payload, 60*time.Second)
	if err != nil {
		return nil, fmt.Errorf("backend/command request failed: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("backend/command failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("failed to decode backend/command response: %w", err)
	}

	if res, exists := raw["result"]; exists {
		return res, nil
	}

	return raw, nil
}
