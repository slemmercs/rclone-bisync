package services

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/core/client"
	"github.com/slemmercs/rclone-bisync/internal/core/logger"
	"github.com/slemmercs/rclone-bisync/internal/models"
	"github.com/slemmercs/rclone-bisync/internal/profiles"
)

var (
	validRemoteNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,63}$`)

	// PopularProviders highlights high-frequency cloud backends in the UI wizard and MCP.
	PopularProviders = map[string]bool{
		"drive":     true,
		"onedrive":  true,
		"dropbox":   true,
		"box":       true,
		"s3":        true,
		"b2":        true,
		"azureblob": true,
		"gcs":       true,
		"sftp":      true,
		"webdav":    true,
		"smb":       true,
		"alias":     true,
	}
)

// CategorizeProvider returns the standardized category for a given Rclone backend type.
func CategorizeProvider(pType string) models.ProviderCategory {
	switch strings.ToLower(strings.TrimSpace(pType)) {
	case "drive", "onedrive", "dropbox", "box", "pcloud", "mega", "zoho", "yandex",
		"mailru", "hidrive", "opendrive", "sugasync", "uptobox", "pikpak", "seafile", "jottacloud", "koofr", "fichier":
		return models.CategoryCloud
	case "s3", "b2", "azureblob", "gcs", "google cloud storage", "swift", "storj",
		"wasabi", "minio", "digitalocean", "oracle", "cos", "sia", "filebase", "qingstor":
		return models.CategoryObject
	case "sftp", "ftp", "webdav", "smb", "http", "hdfs":
		return models.CategoryProtocol
	case "alias", "chunker", "crypt", "union", "combine", "compress":
		return models.CategoryVirtual
	default:
		return models.CategoryOther
	}
}

// RemoteWizardService coordinates provider catalog discovery, remote creation, and sync profile bridges.
type RemoteWizardService struct {
	Config    *config.Config
	Client    *client.RcloneClient
	Logger    *logger.Logger
	Profiles  *profiles.Service
	Telemetry *TelemetryService

	mu           sync.RWMutex
	catalogCache *models.ProviderCatalogResponse
	catalogTTL   time.Time
}

// NewRemoteWizardService instantiates a new RemoteWizardService.
func NewRemoteWizardService(
	cfg *config.Config,
	c *client.RcloneClient,
	log *logger.Logger,
	profSvc *profiles.Service,
	telemetrySvc *TelemetryService,
) *RemoteWizardService {
	return &RemoteWizardService{
		Config:    cfg,
		Client:    c,
		Logger:    log,
		Profiles:  profSvc,
		Telemetry: telemetrySvc,
	}
}

// GetProvidersCatalog queries and categorizes all Rclone backend providers (with 5m caching).
func (s *RemoteWizardService) GetProvidersCatalog(ctx context.Context, forceRefresh bool) (*models.ProviderCatalogResponse, error) {
	if s.Client == nil {
		return nil, fmt.Errorf("rclone client not configured")
	}

	if !forceRefresh {
		s.mu.RLock()
		if s.catalogCache != nil && time.Now().Before(s.catalogTTL) {
			cached := s.catalogCache
			s.mu.RUnlock()
			return cached, nil
		}
		s.mu.RUnlock()
	}

	rawProviders, err := s.Client.GetProviders(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch rclone providers: %w", err)
	}

	items := make([]models.ProviderCatalogItem, 0, len(rawProviders))
	for _, p := range rawProviders {
		if p.Hide {
			continue
		}
		cat := CategorizeProvider(p.Name)
		supportsOAuth := models.IsOAuthProvider(p.Name) || models.IsOAuthProvider(p.Prefix)
		items = append(items, models.ProviderCatalogItem{
			Name:          p.Name,
			Description:   p.Description,
			Prefix:        p.Prefix,
			Category:      cat,
			Popular:       PopularProviders[p.Name],
			SupportsOAuth: supportsOAuth,
			Options:       p.Options,
		})
	}

	categories := []models.ProviderCategory{
		models.CategoryCloud,
		models.CategoryObject,
		models.CategoryProtocol,
		models.CategoryVirtual,
		models.CategoryOther,
	}

	resp := &models.ProviderCatalogResponse{
		Providers:  items,
		Categories: categories,
	}

	s.mu.Lock()
	s.catalogCache = resp
	s.catalogTTL = time.Now().Add(5 * time.Minute)
	s.mu.Unlock()

	return resp, nil
}

// CreateRemote validates configuration, calls /config/create, flushes caches, and optionally creates a sync profile.
func (s *RemoteWizardService) CreateRemote(ctx context.Context, req models.CreateRemoteBridgeRequest, actor string) (*models.CreateRemoteResult, error) {
	if s.Client == nil {
		return nil, fmt.Errorf("rclone client not configured")
	}

	cleanName := strings.TrimRight(strings.TrimSpace(req.Name), ":")
	if cleanName == "" {
		return nil, fmt.Errorf("remote name cannot be empty")
	}
	if !validRemoteNameRegex.MatchString(cleanName) {
		return nil, fmt.Errorf("invalid remote name %q: must contain only alphanumeric, period, dash, or underscore", cleanName)
	}
	cleanType := strings.TrimSpace(req.Type)
	if cleanType == "" {
		return nil, fmt.Errorf("remote type cannot be empty")
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

	// Issue Rclone RC config/create
	if err := s.Client.CreateRemote(ctx, cleanName, cleanType, req.Parameters, req.Opt); err != nil {
		return nil, fmt.Errorf("failed to create remote %q: %w", cleanName, err)
	}

	if s.Telemetry != nil {
		s.Telemetry.FlushRemotesCache()
	}

	// Probe OAuth status in case this provider initiated an interactive OAuth flow
	needsAuth := false
	authURL := ""
	status := "created"
	if s.Client != nil {
		if oStatus, oURL, err := s.Client.OAuthStatus(ctx); err == nil && (oStatus == "running" || oURL != "") {
			needsAuth = true
			authURL = oURL
			status = "oauth_pending"
		}
	}

	if s.Logger != nil {
		s.Logger.Info("[remotes] Remote %q (type: %s) created by %s (status: %s)", cleanName, cleanType, actor, status)
	}

	res := &models.CreateRemoteResult{
		RemoteName: cleanName,
		RemoteType: cleanType,
		NeedsAuth:  needsAuth,
		AuthURL:    authURL,
		Status:     status,
		Success:    true,
		Message:    fmt.Sprintf("Remote %q created successfully", cleanName),
	}
	if needsAuth && authURL != "" {
		res.Message = fmt.Sprintf("Remote %q created. Browser authorization required at: %s", cleanName, authURL)
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

		profName := strings.TrimSpace(req.ProfileName)
		if profName == "" {
			profName = cleanName
		}

		remoteTarget := cleanName + ":"
		sub := strings.Trim(req.RemoteSubpath, "/\\ ")
		if sub != "" {
			remoteTarget += sub
		}

		newProf := &profiles.Profile{
			Name:       profName,
			Remote:     cleanName,
			RemotePath: strings.Trim(req.RemoteSubpath, "/\\ "),
			LocalPath:  cleanLocal,
			Schedule:   strings.TrimSpace(req.Schedule),
		}

		createdProf, err := s.Profiles.Create(ctx, newProf)
		if err != nil {
			res.Message = fmt.Sprintf("Remote %q created, but profile provisioning failed: %v", cleanName, err)
			return res, nil
		}

		res.ProfileID = createdProf.ID
		res.ProfileName = createdProf.Name
		res.Message = fmt.Sprintf("Remote %q and profile %q created successfully", cleanName, createdProf.Name)
	}

	return res, nil
}

// DeleteRemote removes a remote via /config/delete, guarding against deletion if active profiles depend on it.
func (s *RemoteWizardService) DeleteRemote(ctx context.Context, name string, force bool, actor string) error {
	if s.Client == nil {
		return fmt.Errorf("rclone client not configured")
	}

	cleanName := strings.TrimRight(strings.TrimSpace(name), ":")
	if cleanName == "" {
		return fmt.Errorf("remote name cannot be empty")
	}

	if s.Profiles != nil {
		profs := s.Profiles.List(false)
		var dependent []string
		for _, p := range profs {
			pRemote := strings.TrimRight(p.Remote, ":")
			if pRemote == cleanName {
				dependent = append(dependent, fmt.Sprintf("%s (%s)", p.ID, p.Name))
			}
		}
		if len(dependent) > 0 && !force {
			return fmt.Errorf("cannot delete remote %q: referenced by active profile(s): %s (use force=true to override)", cleanName, strings.Join(dependent, ", "))
		}
	}

	if err := s.Client.DeleteRemote(ctx, cleanName); err != nil {
		return fmt.Errorf("failed to delete remote %q: %w", cleanName, err)
	}

	if s.Telemetry != nil {
		s.Telemetry.FlushRemotesCache()
	}

	if s.Logger != nil {
		s.Logger.Info("[remotes] Remote %q deleted by %s (force: %t)", cleanName, actor, force)
	}

	return nil
}
