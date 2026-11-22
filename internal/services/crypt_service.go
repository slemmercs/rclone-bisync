package services

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/core/client"
	"github.com/slemmercs/rclone-bisync/internal/core/logger"
	"github.com/slemmercs/rclone-bisync/internal/models"
	"github.com/slemmercs/rclone-bisync/internal/profiles"
)

// CryptService coordinates Crypt encryption overlay creation and profile bridging.
type CryptService struct {
	Config    *config.Config
	Client    *client.RcloneClient
	Logger    *logger.Logger
	Profiles  *profiles.Service
	Telemetry *TelemetryService
}

// NewCryptService instantiates a new CryptService.
func NewCryptService(
	cfg *config.Config,
	c *client.RcloneClient,
	log *logger.Logger,
	profSvc *profiles.Service,
	telemetrySvc *TelemetryService,
) *CryptService {
	return &CryptService{
		Config:    cfg,
		Client:    c,
		Logger:    log,
		Profiles:  profSvc,
		Telemetry: telemetrySvc,
	}
}

// WrapRemoteInCrypt creates a new Crypt virtual remote with obscured passphrases wrapping a storage target.
func (s *CryptService) WrapRemoteInCrypt(ctx context.Context, req models.CryptWrapRequest, actor string) (*models.CryptWrapResponse, error) {
	if s.Client == nil {
		return nil, fmt.Errorf("rclone client not configured")
	}

	cleanName := strings.TrimRight(strings.TrimSpace(req.Name), ":")
	if cleanName == "" {
		return nil, fmt.Errorf("crypt remote name cannot be empty")
	}
	if !validRemoteNameRegex.MatchString(cleanName) {
		return nil, fmt.Errorf("invalid crypt remote name %q: must contain only alphanumeric, period, dash, or underscore", cleanName)
	}

	baseRemote := strings.TrimSpace(req.Remote)
	if baseRemote == "" {
		return nil, fmt.Errorf("base remote to wrap cannot be empty")
	}

	rawPassword := strings.TrimSpace(req.Password)
	if rawPassword == "" {
		return nil, fmt.Errorf("encryption password cannot be empty")
	}

	// Guard against existing remote collision
	existing, err := s.Client.ListRemotes(ctx)
	if err == nil {
		for _, r := range existing {
			if strings.TrimRight(r, ":") == cleanName {
				return nil, fmt.Errorf("remote %q already exists", cleanName)
			}
		}
	}

	// Obscure password
	obsPassword, err := s.Client.Obscure(ctx, rawPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to obscure primary password: %w", err)
	}

	params := map[string]interface{}{
		"remote":   baseRemote,
		"password": obsPassword,
	}

	if strings.TrimSpace(req.Password2) != "" {
		obsPassword2, err := s.Client.Obscure(ctx, strings.TrimSpace(req.Password2))
		if err != nil {
			return nil, fmt.Errorf("failed to obscure salt password: %w", err)
		}
		params["password2"] = obsPassword2
	}

	fnEnc := strings.ToLower(strings.TrimSpace(req.FilenameEncryption))
	if fnEnc == "" {
		fnEnc = "standard"
	}
	switch fnEnc {
	case "standard", "obfuscate", "off":
		params["filename_encryption"] = fnEnc
	default:
		return nil, fmt.Errorf("invalid filename_encryption %q (allowed: standard, obfuscate, off)", fnEnc)
	}

	if fnEnc == "off" {
		params["directory_name_encryption"] = "false"
	} else if req.DirectoryNameEncryption {
		params["directory_name_encryption"] = "true"
	} else {
		params["directory_name_encryption"] = "false"
	}

	if err := s.Client.CreateRemote(ctx, cleanName, "crypt", params, nil); err != nil {
		return nil, fmt.Errorf("failed to create crypt remote %q: %w", cleanName, err)
	}

	if s.Telemetry != nil {
		s.Telemetry.FlushRemotesCache()
	}

	if s.Logger != nil {
		s.Logger.Info("[crypt] Crypt remote %q wrapping %q created by %s", cleanName, baseRemote, actor)
	}

	res := &models.CryptWrapResponse{
		RemoteName: cleanName,
		BaseRemote: baseRemote,
		Success:    true,
		Message:    fmt.Sprintf("Crypt overlay %q wrapping %q created successfully", cleanName, baseRemote),
	}

	if req.CreateProfile {
		localPath := strings.TrimSpace(req.LocalPath)
		if localPath == "" {
			res.Message += " (profile skipped: local_path required)"
			return res, nil
		}

		cleanLocal := filepath.Clean(localPath)
		if strings.Contains(cleanLocal, "..") {
			res.Message += " (profile skipped: invalid path traversal in local_path)"
			return res, nil
		}

		if s.Profiles == nil {
			res.Message += " (profile skipped: profiles service not initialized)"
			return res, nil
		}

		profName := fmt.Sprintf("%s_crypt_sync", cleanName)
		newProf := &profiles.Profile{
			Name:       profName,
			Remote:     cleanName,
			RemotePath: "",
			LocalPath:  cleanLocal,
			Schedule:   strings.TrimSpace(req.Schedule),
		}

		createdProf, err := s.Profiles.Create(ctx, newProf)
		if err != nil {
			res.Message = fmt.Sprintf("Crypt %q created, but profile provisioning failed: %v", cleanName, err)
			return res, nil
		}

		res.ProfileID = createdProf.ID
		res.ProfileName = createdProf.Name
		res.Message = fmt.Sprintf("Crypt overlay %q and profile %q created successfully", cleanName, createdProf.Name)
	}

	return res, nil
}
