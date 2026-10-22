package remoteauth

import (
	"context"
	"fmt"
	"strings"

	"github.com/slemmercs/rclone-bisync/internal/models"
)

// isAuthError checks whether an error message indicates an expired, revoked, or invalid OAuth token.
func isAuthError(msg string) bool {
	lower := strings.ToLower(msg)
	patterns := []string{
		"invalid_grant",
		"token expired",
		"token is expired",
		"unauthorized",
		"authentication error",
		"authorization error",
		"oauth",
		"401 unauthorized",
		"403 forbidden",
		"access_denied",
		"token revoked",
		"refresh token has expired",
		"could not refresh token",
	}
	for _, p := range patterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

// probeRemote performs active accessibility and authentication validation without generating audit log events.
// This is used for background health checks and dashboard status polling.
func (s *RemoteAuthService) probeRemote(ctx context.Context, remoteName string) (models.ValidateResponse, error) {
	cleanName := strings.TrimRight(remoteName, ":")

	// 1. First probe via operations/about
	about := s.Client.GetRemoteAbout(ctx, cleanName)
	if about != nil {
		return models.ValidateResponse{
			Status: models.AuthStatusConnected,
			Remote: cleanName,
			Valid:  true,
			Msg:    "Remote accessible and authentication valid.",
		}, nil
	}

	// 2. Fallback probe via operations/list root
	listing := s.Client.ListDirectory(ctx, cleanName, "")
	if listing.Error == "" {
		return models.ValidateResponse{
			Status: models.AuthStatusConnected,
			Remote: cleanName,
			Valid:  true,
			Msg:    "Remote accessible and authentication valid.",
		}, nil
	}

	errMsg := listing.Error
	if isAuthError(errMsg) {
		return models.ValidateResponse{
			Status: models.AuthStatusAuthorizationRequired,
			Remote: cleanName,
			Valid:  false,
			Msg:    "Authentication expired or revoked. Please reconnect the remote.",
			Error:  errMsg,
		}, nil
	}

	return models.ValidateResponse{
		Status: models.AuthStatusError,
		Remote: cleanName,
		Valid:  false,
		Msg:    fmt.Sprintf("Failed to connect to remote: %s", errMsg),
		Error:  errMsg,
	}, nil
}

// ValidateRemote performs active accessibility and authentication validation against the cloud provider
// and logs the result to the OAuth audit trail. This is invoked by direct user actions.
func (s *RemoteAuthService) ValidateRemote(ctx context.Context, remoteName string) (models.ValidateResponse, error) {
	cleanName := strings.TrimRight(remoteName, ":")
	resp, err := s.probeRemote(ctx, cleanName)
	if err != nil {
		if s.AuditLogger != nil {
			s.AuditLogger.LogEvent("oauth_validation_failed", cleanName, "error", "Remote connection error: "+err.Error())
		}
		return resp, err
	}

	if s.AuditLogger != nil {
		if resp.Valid {
			s.AuditLogger.LogEvent("oauth_validation_success", cleanName, "success", resp.Msg)
		} else if resp.Status == models.AuthStatusAuthorizationRequired {
			s.AuditLogger.LogEvent("oauth_validation_failed", cleanName, "failed", "OAuth authorization error: "+resp.Error)
		} else {
			s.AuditLogger.LogEvent("oauth_validation_failed", cleanName, "error", "Remote connection error: "+resp.Error)
		}
	}

	return resp, nil
}
