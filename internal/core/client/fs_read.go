package client

import (
	"bytes"
	"context"
	"encoding/base64"
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

// CatStream opens a streaming reader using /operations/getfile with base64 decoding for inspecting or downloading remote files.
func (c *RcloneClient) CatStream(ctx context.Context, remote, path string, offset, count int64) (io.ReadCloser, int64, error) {
	fs := NormalizeFs(remote)
	cleanPath := strings.Trim(path, "/")

	payload := map[string]interface{}{
		"fs":     fs,
		"remote": cleanPath,
		"base64": true,
	}
	if offset > 0 {
		payload["offset"] = offset
	}
	if count > 0 {
		payload["count"] = count
	}

	resp, body, err := c.post(ctx, "operations/getfile", payload, 30*time.Second)
	if err != nil {
		return nil, 0, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("getfile failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var res struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, 0, fmt.Errorf("failed to parse getfile response: %w", err)
	}

	decoded, err := base64.StdEncoding.DecodeString(res.Result)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to decode base64 file data: %w", err)
	}

	return io.NopCloser(bytes.NewReader(decoded)), int64(len(decoded)), nil
}

// CatFile calls CatStream and reads up to count bytes directly into a byte buffer.
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
