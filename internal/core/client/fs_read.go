package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// StatItem calls /operations/stat to retrieve metadata about a single file or directory on a remote.
func (c *RcloneClient) StatItem(ctx context.Context, remote, path string) (map[string]interface{}, error) {
	fs := NormalizeFs(remote)
	cleanPath := strings.Trim(path, "/")

	payload := map[string]interface{}{
		"fs":     fs,
		"remote": cleanPath,
	}

	resp, body, err := c.post(ctx, "operations/stat", payload, 10*time.Second)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("stat failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var res struct {
		Item map[string]interface{} `json:"item"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("failed to parse stat response: %w", err)
	}
	if res.Item == nil {
		return nil, fmt.Errorf("item not found")
	}
	return res.Item, nil
}

// CatStream opens a streaming reader from /operations/cat for downloading or inspecting remote files.
func (c *RcloneClient) CatStream(ctx context.Context, remote, path string, offset, count int64) (io.ReadCloser, int64, error) {
	fs := NormalizeFs(remote)
	cleanPath := strings.Trim(path, "/")

	payload := map[string]interface{}{
		"fs":     fs,
		"remote": cleanPath,
	}
	if offset > 0 {
		payload["offset"] = offset
	}
	if count > 0 {
		payload["count"] = count
	}

	fullURL := fmt.Sprintf("%s/operations/cat", c.APIURL)
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Username != "" && c.Password != "" {
		req.SetBasicAuth(c.Username, c.Password)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return nil, 0, fmt.Errorf("cat failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	return resp.Body, resp.ContentLength, nil
}

// CatFile calls /operations/cat and reads up to count bytes directly into a byte buffer.
func (c *RcloneClient) CatFile(ctx context.Context, remote, path string, offset, count int64) ([]byte, error) {
	stream, _, err := c.CatStream(ctx, remote, path, offset, count)
	if err != nil {
		return nil, err
	}
	defer stream.Close()

	if count > 0 {
		return io.ReadAll(io.LimitReader(stream, count))
	}
	return io.ReadAll(stream)
}
