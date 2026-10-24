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
	fs := NormalizeFs(remote)
	resp, body, err := c.post(ctx, "operations/about", map[string]string{"fs": fs}, 5*time.Second)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil
	}

	var res map[string]interface{}
	_ = json.Unmarshal(body, &res)
	return res
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

// GetProviders queries /config/providers and returns available storage backends and their schemas.
func (c *RcloneClient) GetProviders(ctx context.Context) ([]models.Provider, error) {
	resp, body, err := c.post(ctx, "config/providers", nil, 10*time.Second)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("config/providers returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var res models.ProvidersResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("failed to unmarshal providers response: %w", err)
	}
	return res.Providers, nil
}

// CreateRemote creates a new remote configuration via /config/create.
func (c *RcloneClient) CreateRemote(ctx context.Context, name, remoteType string, parameters map[string]interface{}, opt map[string]interface{}) error {
	cleanName := strings.TrimRight(name, ":")
	if cleanName == "" {
		return fmt.Errorf("remote name cannot be empty")
	}
	if remoteType == "" {
		return fmt.Errorf("remote type cannot be empty")
	}

	payload := map[string]interface{}{
		"name":       cleanName,
		"type":       remoteType,
		"parameters": parameters,
	}
	if len(opt) > 0 {
		payload["opt"] = opt
	}

	resp, body, err := c.post(ctx, "config/create", payload, 15*time.Second)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("config/create failed (HTTP %d): %s", resp.StatusCode, string(body))
	}
	return nil
}

// DeleteRemote deletes a remote configuration via /config/delete.
func (c *RcloneClient) DeleteRemote(ctx context.Context, name string) error {
	cleanName := strings.TrimRight(name, ":")
	if cleanName == "" {
		return fmt.Errorf("remote name cannot be empty")
	}

	payload := map[string]interface{}{
		"name": cleanName,
	}

	resp, body, err := c.post(ctx, "config/delete", payload, 10*time.Second)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("config/delete failed (HTTP %d): %s", resp.StatusCode, string(body))
	}
	return nil
}
