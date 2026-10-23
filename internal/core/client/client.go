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

// RcloneClient communicates with the Rclone Remote Control (RC) HTTP API.
type RcloneClient struct {
	APIURL     string
	Username   string
	Password   string
	HTTPClient *http.Client
}

// NewRcloneClient constructs a new client with sensible connection pooling defaults.
func NewRcloneClient(apiURL, username, password string) *RcloneClient {
	cleanURL := strings.TrimRight(apiURL, "/")
	tr := &http.Transport{
		MaxIdleConns:        50,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
	}
	return &RcloneClient{
		APIURL:   cleanURL,
		Username: username,
		Password: password,
		HTTPClient: &http.Client{
			Transport: tr,
			Timeout:   60 * time.Second,
		},
	}
}

func (c *RcloneClient) post(ctx context.Context, endpoint string, payload interface{}, timeout time.Duration) (*http.Response, []byte, error) {
	fullURL := fmt.Sprintf("%s/%s", c.APIURL, strings.TrimLeft(endpoint, "/"))

	var bodyReader io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	} else {
		// Rclone RC endpoints strictly require a valid JSON body (at least `{}`)
		bodyReader = bytes.NewReader([]byte("{}"))
	}

	reqCtx := ctx
	var cancel context.CancelFunc
	if timeout > 0 {
		reqCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, fullURL, bodyReader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.Username != "" && c.Password != "" {
		req.SetBasicAuth(c.Username, c.Password)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, fmt.Errorf("failed to read response: %w", err)
	}

	return resp, respBytes, nil
}

// CheckHealth queries /core/stats. Returns: (isHealthy, responseTimeMs, errorMessage).
func (c *RcloneClient) CheckHealth(ctx context.Context, retries int, retryDelay time.Duration) (bool, float64, error) {
	if retries <= 0 {
		retries = 1
	}

	var lastErr error
	for attempt := 0; attempt < retries; attempt++ {
		start := time.Now()
		resp, _, err := c.post(ctx, "/core/stats", nil, 5*time.Second)
		if err == nil {
			latency := float64(time.Since(start).Microseconds()) / 1000.0
			if resp.StatusCode == http.StatusOK {
				return true, latency, nil
			}
			lastErr = fmt.Errorf("daemon returned HTTP status %d", resp.StatusCode)
		} else {
			lastErr = fmt.Errorf("cannot connect to daemon at %s: %w", c.APIURL, err)
		}

		if attempt < retries-1 {
			time.Sleep(retryDelay)
		}
	}

	return false, 0.0, lastErr
}

func (c *RcloneClient) GetCoreStats(ctx context.Context) ([]byte, error) {
	resp, body, err := c.post(ctx, "/core/stats", nil, 2*time.Second)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
	return body, nil
}
