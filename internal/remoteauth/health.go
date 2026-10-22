package remoteauth

import (
	"context"
	"strings"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/models"
	"github.com/slemmercs/rclone-bisync/internal/services"
)

// OAuthCapableProviders lists cloud providers that use OAuth 2.0 authorization flows.
var OAuthCapableProviders = map[string]bool{
	"drive":         true,
	"onedrive":      true,
	"dropbox":       true,
	"box":           true,
	"pcloud":        true,
	"hubic":         true,
	"yandex":        true,
	"mailru":        true,
	"google photos": true,
	"zoho":          true,
}

// IsOAuthProvider returns true if the remote provider type uses OAuth authentication.
func IsOAuthProvider(providerType string) bool {
	return OAuthCapableProviders[strings.ToLower(strings.TrimSpace(providerType))]
}

// GetProviderLabel returns a human-friendly provider name.
func GetProviderLabel(providerType string) string {
	cleanType := strings.ToLower(strings.TrimSpace(providerType))
	if lbl, ok := services.ProviderLabels[cleanType]; ok {
		return lbl
	}
	if cleanType == "" {
		return "Unknown"
	}
	return strings.ToUpper(cleanType[:1]) + cleanType[1:]
}

// GetRemoteAuthHealth inspects the configuration, token state, and connectivity of a single remote.
func (s *RemoteAuthService) GetRemoteAuthHealth(ctx context.Context, remoteName string) (models.RemoteAuthInfo, error) {
	cleanName := strings.TrimRight(remoteName, ":")
	activeTarget := ""
	if s.Config != nil {
		activeTarget = strings.TrimRight(s.Config.RemotePath, ":")
	}
	isActive := (cleanName == activeTarget)

	cfg, err := s.Client.GetRemoteAuthConfig(ctx, cleanName)
	if err != nil {
		return models.RemoteAuthInfo{
			Name:           cleanName,
			Type:           "unknown",
			TypeLabel:      "Unknown",
			IsOAuth:        false,
			Status:         models.AuthStatusError,
			StatusMessage:  "Unable to retrieve remote configuration: " + err.Error(),
			IsActiveTarget: isActive,
			LastChecked:    time.Now().UTC().Format(time.RFC3339),
		}, nil
	}

	pType, _ := cfg["type"].(string)
	clientID, _ := cfg["client_id"].(string)
	hasSecret, _ := cfg["has_client_secret"].(bool)
	hasToken, _ := cfg["has_token"].(bool)
	tokenExpiry, _ := cfg["token_expiry"].(string)

	isOAuth := IsOAuthProvider(pType)
	typeLabel := GetProviderLabel(pType)

	if !isOAuth {
		// Non-OAuth provider (e.g. local, s3, sftp)
		return models.RemoteAuthInfo{
			Name:            cleanName,
			Type:            pType,
			TypeLabel:       typeLabel,
			IsOAuth:         false,
			Status:          models.AuthStatusConnected,
			StatusMessage:   "Provider does not use OAuth",
			ClientID:        clientID,
			HasClientSecret: hasSecret,
			HasToken:        false,
			IsActiveTarget:  isActive,
			LastChecked:     time.Now().UTC().Format(time.RFC3339),
		}, nil
	}

	// OAuth Provider Assessment
	migrationRequired := false
	if pType == "drive" && strings.TrimSpace(clientID) == "" {
		migrationRequired = true
	}

	if !hasToken {
		return models.RemoteAuthInfo{
			Name:            cleanName,
			Type:            pType,
			TypeLabel:       typeLabel,
			IsOAuth:         true,
			Status:          models.AuthStatusAuthorizationRequired,
			StatusMessage:   "Remote requires OAuth authorization.",
			ClientID:        clientID,
			HasClientSecret: hasSecret,
			HasToken:        false,
			TokenExpiry:     tokenExpiry,
			IsActiveTarget:  isActive,
			LastChecked:     time.Now().UTC().Format(time.RFC3339),
		}, nil
	}

	// Check connectivity / token validity via passive probe (do not generate audit log records)
	valResp, _ := s.probeRemote(ctx, cleanName)
	var finalStatus string
	var finalMsg string

	if valResp.Valid {
		if migrationRequired {
			finalStatus = models.AuthStatusMigrationRequired
			finalMsg = "Google Drive remote uses shared Rclone Client ID. Migration recommended."
		} else {
			finalStatus = models.AuthStatusConnected
			finalMsg = "Remote authentication healthy."
		}
	} else {
		finalStatus = valResp.Status
		finalMsg = valResp.Msg
	}

	return models.RemoteAuthInfo{
		Name:            cleanName,
		Type:            pType,
		TypeLabel:       typeLabel,
		IsOAuth:         true,
		Status:          finalStatus,
		StatusMessage:   finalMsg,
		ClientID:        clientID,
		HasClientSecret: hasSecret,
		HasToken:        hasToken,
		TokenExpiry:     tokenExpiry,
		IsActiveTarget:  isActive,
		LastChecked:     time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// GetAllRemotesAuthStatus queries all configured remotes and calculates the aggregate authentication health.
func (s *RemoteAuthService) GetAllRemotesAuthStatus(ctx context.Context) (models.AuthStatusResponse, error) {
	remotes, err := s.Client.ListRemotes(ctx)
	if err != nil {
		return models.AuthStatusResponse{}, err
	}

	infos := make([]models.RemoteAuthInfo, 0, len(remotes))
	summary := models.AuthHealthSummary{
		TotalRemotes: len(remotes),
	}

	for _, r := range remotes {
		info, err := s.GetRemoteAuthHealth(ctx, r)
		if err != nil {
			info = models.RemoteAuthInfo{
				Name:          strings.TrimRight(r, ":"),
				Status:        models.AuthStatusError,
				StatusMessage: err.Error(),
				LastChecked:   time.Now().UTC().Format(time.RFC3339),
			}
		}

		if info.IsOAuth {
			summary.OAuthRemotes++
		}

		switch info.Status {
		case models.AuthStatusConnected:
			summary.Connected++
		case models.AuthStatusMigrationRequired:
			summary.MigrationRequired++
		case models.AuthStatusAuthorizationRequired:
			summary.AuthorizationRequired++
		case models.AuthStatusConfigurationRequired:
			summary.ConfigurationRequired++
		case models.AuthStatusError:
			summary.Errors++
		}

		infos = append(infos, info)
	}

	return models.AuthStatusResponse{
		Summary: summary,
		Remotes: infos,
	}, nil
}

// CheckAllRemotesAuth explicitly validates authentication for all configured remotes,
// recording audit entries for each OAuth-capable remote, and calculates aggregate authentication health.
// This is invoked when the user explicitly clicks "Check Auth & Quotas".
func (s *RemoteAuthService) CheckAllRemotesAuth(ctx context.Context) (models.AuthStatusResponse, error) {
	remotes, err := s.Client.ListRemotes(ctx)
	if err != nil {
		return models.AuthStatusResponse{}, err
	}

	infos := make([]models.RemoteAuthInfo, 0, len(remotes))
	summary := models.AuthHealthSummary{
		TotalRemotes: len(remotes),
	}

	for _, r := range remotes {
		cleanName := strings.TrimRight(r, ":")
		cfg, _ := s.Client.GetRemoteAuthConfig(ctx, cleanName)
		pType, _ := cfg["type"].(string)

		// Explicit user check: validate connectivity and log to audit trail for OAuth remotes
		if IsOAuthProvider(pType) {
			_, _ = s.ValidateRemote(ctx, cleanName)
		}

		info, err := s.GetRemoteAuthHealth(ctx, cleanName)
		if err != nil {
			info = models.RemoteAuthInfo{
				Name:          cleanName,
				Status:        models.AuthStatusError,
				StatusMessage: err.Error(),
				LastChecked:   time.Now().UTC().Format(time.RFC3339),
			}
		}

		if info.IsOAuth {
			summary.OAuthRemotes++
		}

		switch info.Status {
		case models.AuthStatusConnected:
			summary.Connected++
		case models.AuthStatusMigrationRequired:
			summary.MigrationRequired++
		case models.AuthStatusAuthorizationRequired:
			summary.AuthorizationRequired++
		case models.AuthStatusConfigurationRequired:
			summary.ConfigurationRequired++
		case models.AuthStatusError:
			summary.Errors++
		}

		infos = append(infos, info)
	}

	return models.AuthStatusResponse{
		Summary: summary,
		Remotes: infos,
	}, nil
}
