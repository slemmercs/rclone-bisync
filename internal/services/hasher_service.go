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

var allowedHasherCommands = map[string]bool{
	"dump":         true,
	"fulldump":     true,
	"drop":         true,
	"import":       true,
	"stickyimport": true,
}

// HasherService coordinates Hasher overlay remote configuration and backend command operations.
type HasherService struct {
	Config    *config.Config
	Client    *client.RcloneClient
	Logger    *logger.Logger
	Profiles  *profiles.Service
	Telemetry *TelemetryService
}

// NewHasherService instantiates a new HasherService.
func NewHasherService(
	cfg *config.Config,
	c *client.RcloneClient,
	log *logger.Logger,
	profSvc *profiles.Service,
	telemetrySvc *TelemetryService,
) *HasherService {
	return &HasherService{
		Config:    cfg,
		Client:    c,
		Logger:    log,
		Profiles:  profSvc,
		Telemetry: telemetrySvc,
	}
}

// WrapRemoteInHasher creates a new Hasher virtual remote wrapping a target storage path.
func (s *HasherService) WrapRemoteInHasher(ctx context.Context, req models.HasherWrapRequest, actor string) (*models.CreateRemoteResult, error) {
	if s.Client == nil {
		return nil, fmt.Errorf("rclone client not configured")
	}

	cleanName := strings.TrimRight(strings.TrimSpace(req.Name), ":")
	if cleanName == "" {
		return nil, fmt.Errorf("hasher remote name cannot be empty")
	}
	if !validRemoteNameRegex.MatchString(cleanName) {
		return nil, fmt.Errorf("invalid hasher remote name %q: must contain only alphanumeric, period, dash, or underscore", cleanName)
	}

	baseRemote := strings.TrimSpace(req.Remote)
	if baseRemote == "" {
		return nil, fmt.Errorf("base remote to wrap cannot be empty")
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

	hashes := strings.TrimSpace(req.Hashes)
	if hashes == "" {
		hashes = "auto"
	}
	maxAge := strings.TrimSpace(req.MaxAge)
	if maxAge == "" {
		maxAge = "off"
	}

	params := map[string]interface{}{
		"remote":  baseRemote,
		"hashes":  hashes,
		"max_age": maxAge,
	}
	if req.AutoSize > 0 {
		params["auto_size"] = req.AutoSize
	}

	if err := s.Client.CreateRemote(ctx, cleanName, "hasher", params, nil); err != nil {
		return nil, fmt.Errorf("failed to create hasher remote %q: %w", cleanName, err)
	}

	if s.Telemetry != nil {
		s.Telemetry.FlushRemotesCache()
	}

	if s.Logger != nil {
		s.Logger.Info("[hasher] Hasher remote %q wrapping %q created by %s", cleanName, baseRemote, actor)
	}

	res := &models.CreateRemoteResult{
		RemoteName: cleanName,
		RemoteType: "hasher",
		Status:     "created",
		Success:    true,
		Message:    fmt.Sprintf("Hasher overlay %q wrapping %q created successfully", cleanName, baseRemote),
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

		profName := fmt.Sprintf("%s_hasher_sync", cleanName)
		newProf := &profiles.Profile{
			Name:       profName,
			Remote:     cleanName,
			RemotePath: "",
			LocalPath:  cleanLocal,
			Schedule:   strings.TrimSpace(req.Schedule),
		}

		createdProf, err := s.Profiles.Create(ctx, newProf)
		if err != nil {
			res.Message = fmt.Sprintf("Hasher %q created, but profile provisioning failed: %v", cleanName, err)
			return res, nil
		}

		res.ProfileID = createdProf.ID
		res.ProfileName = createdProf.Name
		res.Message = fmt.Sprintf("Hasher overlay %q and profile %q created successfully", cleanName, createdProf.Name)
	}

	return res, nil
}

// ExecuteHasherCommand dispatches supported Hasher maintenance commands (dump, fulldump, drop, import, stickyimport).
func (s *HasherService) ExecuteHasherCommand(ctx context.Context, req models.HasherCommandRequest) (*models.HasherCommandResponse, error) {
	if s.Client == nil {
		return nil, fmt.Errorf("rclone client not configured")
	}

	cleanRemote := strings.TrimRight(strings.TrimSpace(req.Remote), ":")
	if cleanRemote == "" {
		return nil, fmt.Errorf("target remote cannot be empty")
	}

	cmd := strings.ToLower(strings.TrimSpace(req.Command))
	if !allowedHasherCommands[cmd] {
		return nil, fmt.Errorf("unsupported hasher command %q (allowed: dump, fulldump, drop, import, stickyimport)", cmd)
	}

	fs := cleanRemote + ":"

	result, err := s.Client.BackendCommand(ctx, fs, cmd, req.Args, req.Opt)
	if err != nil {
		return &models.HasherCommandResponse{
			Status:  "error",
			Command: cmd,
			Remote:  cleanRemote,
			Error:   err.Error(),
		}, err
	}

	return &models.HasherCommandResponse{
		Status:  "success",
		Command: cmd,
		Remote:  cleanRemote,
		Result:  result,
	}, nil
}
