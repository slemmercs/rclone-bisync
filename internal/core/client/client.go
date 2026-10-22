package client

import (
"bytes"
"context"
"encoding/json"
"fmt"
"io"
"net/http"
"net/url"
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

// TriggerBisync calls /sync/bisync?_async=true and returns the job ID integer.
func (c *RcloneClient) TriggerBisync(ctx context.Context, params map[string]interface{}, group string) (int, error) {
	endpoint := "sync/bisync?_async=true"
	if group != "" {
		endpoint += fmt.Sprintf("&_group=%s", url.QueryEscape(group))
	}

	resp, body, err := c.post(ctx, endpoint, params, 30*time.Second)
	if err != nil {
		return 0, err
	}

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("trigger bisync failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var res struct {
		JobID int `json:"jobid"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return 0, fmt.Errorf("failed to parse jobid: %w", err)
	}

	if res.JobID == 0 {
		return 0, fmt.Errorf("rclone RC returned 0 or missing jobid: %s", string(body))
	}

	return res.JobID, nil
}

// GetJobStatus queries /job/status for a specific job.
func (c *RcloneClient) GetJobStatus(ctx context.Context, jobID int) (map[string]interface{}, error) {
	resp, body, err := c.post(ctx, "job/status", map[string]interface{}{"jobid": jobID}, 10*time.Second)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("query job status failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var res map[string]interface{}
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, err
	}
	return res, nil
}

// StopJob stops a running job via /job/stop.
func (c *RcloneClient) StopJob(ctx context.Context, jobID int) bool {
	resp, _, err := c.post(ctx, "job/stop", map[string]interface{}{"jobid": jobID}, 10*time.Second)
	if err != nil {
		return false
	}
	return resp.StatusCode == http.StatusOK
}

// ListJobs queries /job/list and returns slice of running job IDs.
func (c *RcloneClient) ListJobs(ctx context.Context) []int {
	resp, body, err := c.post(ctx, "job/list", nil, 5*time.Second)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil
	}

	var res struct {
		JobIDs []int `json:"jobids"`
	}
	_ = json.Unmarshal(body, &res)
	return res.JobIDs
}

// ListRemotes queries /config/listremotes to list configured cloud providers.
func (c *RcloneClient) ListRemotes(ctx context.Context) ([]string, error) {
	resp, body, err := c.post(ctx, "config/listremotes", nil, 5*time.Second)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list remotes: HTTP %d", resp.StatusCode)
	}

	var res struct {
		Remotes []string `json:"remotes"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, err
	}
	return res.Remotes, nil
}

// GetRemoteConfig queries /config/get and filters safe metadata.
func (c *RcloneClient) GetRemoteConfig(ctx context.Context, remoteName string) map[string]interface{} {
	cleanName := strings.TrimRight(remoteName, ":")
	resp, body, err := c.post(ctx, "config/get", map[string]string{"name": cleanName}, 5*time.Second)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil
	}

	safeKeys := map[string]bool{
		"type":       true,
		"scope":      true,
		"endpoint":   true,
		"region":     true,
		"env_auth":   true,
		"team_drive": true,
	}

	safe := make(map[string]interface{})
	for k, v := range raw {
		if safeKeys[k] {
			safe[k] = v
		}
	}
	return safe
}

// GetRemoteAbout queries /operations/about for quota details.
func (c *RcloneClient) GetRemoteAbout(ctx context.Context, remote string) map[string]interface{} {
	fs := remote
	if !strings.HasSuffix(fs, ":") {
		fs += ":"
	}
	resp, body, err := c.post(ctx, "operations/about", map[string]string{"fs": fs}, 5*time.Second)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil
	}

	var res map[string]interface{}
	_ = json.Unmarshal(body, &res)
	return res
}

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

// GetRemoteAuthConfig queries /config/get and extracts sanitized auth metadata without leaking secrets.
func (c *RcloneClient) GetRemoteAuthConfig(ctx context.Context, remoteName string) (map[string]interface{}, error) {
	cleanName := strings.TrimRight(remoteName, ":")
	resp, body, err := c.post(ctx, "config/get", map[string]string{"name": cleanName}, 5*time.Second)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("config/get returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	result := make(map[string]interface{})
	if t, ok := raw["type"].(string); ok {
		result["type"] = t
	}
	if cid, ok := raw["client_id"].(string); ok {
		result["client_id"] = cid
	}
	if scope, ok := raw["scope"].(string); ok {
		result["scope"] = scope
	}

	// Determine if client_secret is configured without returning the secret itself
	hasSecret := false
	if secret, ok := raw["client_secret"].(string); ok && strings.TrimSpace(secret) != "" {
		hasSecret = true
	}
	result["has_client_secret"] = hasSecret

	// Determine if token is present and extract expiry safely
	hasToken := false
	tokenExpiry := ""
	if tokenRaw, ok := raw["token"]; ok && tokenRaw != nil {
		switch t := tokenRaw.(type) {
		case string:
			if strings.TrimSpace(t) != "" {
				var tokenMap map[string]interface{}
				if err := json.Unmarshal([]byte(t), &tokenMap); err == nil {
					if _, hasAccess := tokenMap["access_token"]; hasAccess {
						hasToken = true
					}
					if _, hasRefresh := tokenMap["refresh_token"]; hasRefresh {
						hasToken = true
					}
					if exp, ok := tokenMap["expiry"].(string); ok {
						tokenExpiry = exp
					}
				} else {
					hasToken = true
				}
			}
		case map[string]interface{}:
			if _, hasAccess := t["access_token"]; hasAccess {
				hasToken = true
			}
			if _, hasRefresh := t["refresh_token"]; hasRefresh {
				hasToken = true
			}
			if exp, ok := t["expiry"].(string); ok {
				tokenExpiry = exp
			}
		}
	}
	result["has_token"] = hasToken
	if tokenExpiry != "" {
		result["token_expiry"] = tokenExpiry
	}

	return result, nil
}

// UpdateRemoteConfig updates configuration parameters on a remote via /config/update.
func (c *RcloneClient) UpdateRemoteConfig(ctx context.Context, remoteName string, parameters map[string]string) error {
	cleanName := strings.TrimRight(remoteName, ":")
	payload := map[string]interface{}{
		"name":       cleanName,
		"parameters": parameters,
	}
	resp, body, err := c.post(ctx, "config/update", payload, 10*time.Second)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("config/update failed (HTTP %d): %s", resp.StatusCode, string(body))
	}
	return nil
}

// OAuthStatus queries /config/oauthstatus to check if the internal OAuth web server is running.
func (c *RcloneClient) OAuthStatus(ctx context.Context) (status string, authURL string, err error) {
	resp, body, err := c.post(ctx, "config/oauthstatus", nil, 5*time.Second)
	if err != nil {
		return "", "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("config/oauthstatus returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var res struct {
		Status  string `json:"status"`
		AuthURL string `json:"authUrl"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return "", "", err
	}
	return res.Status, res.AuthURL, nil
}

// OAuthStop stops any running OAuth authentication web server in rclone.
func (c *RcloneClient) OAuthStop(ctx context.Context) error {
	resp, body, err := c.post(ctx, "config/oauthstop", nil, 5*time.Second)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("config/oauthstop returned HTTP %d: %s", resp.StatusCode, string(body))
	}
	return nil
}
