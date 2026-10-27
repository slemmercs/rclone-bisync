package client

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// RcloneClient communicates with the Rclone Remote Control (RC) HTTP API.
type RcloneClient struct {
	APIURL     string
	Username   string
	Password   string
	HTTPClient *http.Client
	MaxRetries int
	BaseDelay  time.Duration
}

// NewRcloneClient constructs a new client with sensible connection pooling and retry defaults.
func NewRcloneClient(apiURL, username, password string) *RcloneClient {
	cleanURL := strings.TrimRight(apiURL, "/")
	tr := &http.Transport{
		MaxIdleConns:        50,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
	}
	return &RcloneClient{
		APIURL:     cleanURL,
		Username:   username,
		Password:   password,
		MaxRetries: 3,
		BaseDelay:  150 * time.Millisecond,
		HTTPClient: &http.Client{
			Transport: tr,
			Timeout:   60 * time.Second,
		},
	}
}

func (c *RcloneClient) backoffWithJitter(attempt int, baseDelay time.Duration) time.Duration {
	multiplier := 1 << attempt
	if multiplier > 32 {
		multiplier = 32
	}
	backoff := baseDelay * time.Duration(multiplier)
	jitterMax := int64(backoff / 4)
	if jitterMax > 0 {
		var b [8]byte
		_, _ = rand.Read(b[:])
		n := int64(binary.LittleEndian.Uint64(b[:]) & 0x7fffffffffffffff)
		jitter := time.Duration(n % jitterMax)
		backoff += jitter
	}
	return backoff
}

func (c *RcloneClient) post(ctx context.Context, endpoint string, payload interface{}, timeout time.Duration) (*http.Response, []byte, error) {
	fullURL := fmt.Sprintf("%s/%s", c.APIURL, strings.TrimLeft(endpoint, "/"))

	var payloadBytes []byte
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		payloadBytes = data
	} else {
		// Rclone RC endpoints strictly require a valid JSON body (at least `{}`)
		payloadBytes = []byte("{}")
	}

	maxAttempts := c.MaxRetries
	if maxAttempts <= 0 {
		maxAttempts = 1
	} else {
		maxAttempts = maxAttempts + 1
	}

	baseDelay := c.BaseDelay
	if baseDelay <= 0 {
		baseDelay = 100 * time.Millisecond
	}

	var lastResp *http.Response
	var lastBytes []byte
	var lastErr error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if ctx.Err() != nil {
			return nil, nil, ctx.Err()
		}

		reqCtx := ctx
		var cancel context.CancelFunc
		if timeout > 0 {
			reqCtx, cancel = context.WithTimeout(ctx, timeout)
		}

		req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, fullURL, bytes.NewReader(payloadBytes))
		if err != nil {
			if cancel != nil {
				cancel()
			}
			return nil, nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		if c.Username != "" && c.Password != "" {
			req.SetBasicAuth(c.Username, c.Password)
		}

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			if cancel != nil {
				cancel()
			}
			lastErr = err
			if ctx.Err() != nil {
				return nil, nil, ctx.Err()
			}
			if attempt < maxAttempts-1 {
				time.Sleep(c.backoffWithJitter(attempt, baseDelay))
				continue
			}
			return nil, nil, err
		}

		respBytes, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if cancel != nil {
			cancel()
		}

		if readErr != nil {
			lastErr = fmt.Errorf("failed to read response: %w", readErr)
			if attempt < maxAttempts-1 {
				time.Sleep(c.backoffWithJitter(attempt, baseDelay))
				continue
			}
			return resp, nil, lastErr
		}

		// Check for retryable HTTP status codes: 429 Too Many Requests or 503 Service Unavailable
		if (resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable) && attempt < maxAttempts-1 {
			lastResp = resp
			lastBytes = respBytes
			sleepDur := c.backoffWithJitter(attempt, baseDelay)
			if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
				if seconds, parseErr := strconv.Atoi(retryAfter); parseErr == nil && seconds > 0 {
					sleepDur = time.Duration(seconds) * time.Second
				}
			}

			select {
			case <-ctx.Done():
				return lastResp, lastBytes, ctx.Err()
			case <-time.After(sleepDur):
			}
			continue
		}

		return resp, respBytes, nil
	}

	return lastResp, lastBytes, lastErr
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
