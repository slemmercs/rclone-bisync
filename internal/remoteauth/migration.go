package remoteauth

import (
	"context"
	"fmt"
	"strings"
)

// UpdateCredentials saves custom OAuth client ID and client secret to the remote config and logs the event.
func (s *RemoteAuthService) UpdateCredentials(ctx context.Context, remoteName, clientID, clientSecret string) error {
	cleanName := strings.TrimRight(remoteName, ":")
	if cleanName == "" {
		return fmt.Errorf("remote name cannot be empty")
	}

	cleanClientID := strings.TrimSpace(clientID)
	cleanClientSecret := strings.TrimSpace(clientSecret)

	if cleanClientID == "" {
		return fmt.Errorf("client_id cannot be empty")
	}

	params := map[string]string{
		"client_id": cleanClientID,
	}
	if cleanClientSecret != "" {
		params["client_secret"] = cleanClientSecret
	}

	err := s.Client.UpdateRemoteConfig(ctx, cleanName, params)
	if err != nil {
		s.AuditLogger.LogEvent("oauth_credentials_update_failed", cleanName, "failed", err.Error())
		return fmt.Errorf("failed to update credentials for remote %s: %w", cleanName, err)
	}

	// Record audit event - NEVER log client_secret!
	s.AuditLogger.LogEvent("oauth_credentials_updated", cleanName, "success", "OAuth credentials successfully updated")

	return nil
}
