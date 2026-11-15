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

// RcloneGetFileMaxBytes defines the hardcoded ceiling in Rclone's /operations/getfile RC endpoint (2,097,152 bytes = 2MB).
const RcloneGetFileMaxBytes int64 = 2 * 1024 * 1024

// CatStream opens a streaming reader using /operations/getfile with base64 decoding for inspecting or downloading remote files.
// An optional maxSize parameter specifies the limit, strictly clamped to Rclone's built-in 2MB limit on /operations/getfile.
func (c *RcloneClient) CatStream(ctx context.Context, remote, path string, offset, count int64, maxSize ...int64) (io.ReadCloser, int64, error) {
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
		if count > RcloneGetFileMaxBytes {
			payload["count"] = RcloneGetFileMaxBytes
		} else {
			payload["count"] = count
		}
	}
	var maxLimit int64
	if len(maxSize) > 0 && maxSize[0] > 0 {
		maxLimit = maxSize[0]
	} else if count > 0 {
		maxLimit = count
	}
	if maxLimit > RcloneGetFileMaxBytes {
		maxLimit = RcloneGetFileMaxBytes
	}
	if maxLimit > 0 {
		payload["maxSize"] = maxLimit
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
func (c *RcloneClient) CatFile(ctx context.Context, remote, path string, offset, count int64, maxSize ...int64) ([]byte, error) {
	stream, _, err := c.CatStream(ctx, remote, path, offset, count, maxSize...)
	if err != nil {
		return nil, err
	}
	defer stream.Close()

	if count > 0 {
		return io.ReadAll(io.LimitReader(stream, count))
	}
	return io.ReadAll(stream)
}

// DownloadStream opens a streaming reader for a file. If direct HTTP download via --rc-serve is available,
// it streams directly from Rclone's HTTP endpoint; otherwise it falls back to CatStream with explicit maxSize.
func (c *RcloneClient) DownloadStream(ctx context.Context, remote, path string, size int64) (io.ReadCloser, int64, error) {
	fs := NormalizeFs(remote)
	cleanPath := strings.Trim(path, "/")

	// Attempt direct streaming if RC server serves files via --rc-serve
	if c.APIURL != "" {
		serveURL := fmt.Sprintf("%s/[%s]/%s", c.APIURL, fs, cleanPath)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, serveURL, nil)
		if err == nil {
			if c.Username != "" || c.Password != "" {
				req.SetBasicAuth(c.Username, c.Password)
			}
			tr := http.DefaultTransport
			if c.HTTPClient != nil && c.HTTPClient.Transport != nil {
				tr = c.HTTPClient.Transport
			}
			streamClient := &http.Client{
				Transport: tr,
				Timeout:   0,
			}
			resp, err := streamClient.Do(req)
			if err == nil && resp.StatusCode == http.StatusOK {
				return resp.Body, resp.ContentLength, nil
			}
			if resp != nil {
				_ = resp.Body.Close()
			}
		}
	}

	// Fallback to /operations/getfile with explicit maxSize
	return c.CatStream(ctx, remote, path, 0, 0, size)
}
