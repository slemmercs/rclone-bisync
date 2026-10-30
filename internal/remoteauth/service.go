package remoteauth

import (
	"context"
	"sync"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/core/client"
	"github.com/slemmercs/rclone-bisync/internal/models"
)

// RemoteAuthService coordinates authentication health monitoring, reconnect flows,
// credential migration, connectivity validation, and audit logging.
type RemoteAuthService struct {
	Config           *config.Config
	Client           *client.RcloneClient
	AuditLogger      *AuditLogger
	Runner           CommandRunner
	activeReconnects map[string]context.CancelFunc
	cachedStatus     models.AuthStatusResponse
	cachedStatusTime time.Time
	mu               sync.Mutex
}

// NewRemoteAuthService constructs a configured RemoteAuthService.
func NewRemoteAuthService(cfg *config.Config, rcloneClient *client.RcloneClient, runner CommandRunner) *RemoteAuthService {
	if rcloneClient == nil && cfg != nil {
		user := ""
		pass := ""
		if cfg.RCUser != "" {
			user = cfg.RCUser
			pass = cfg.RCPass
		}
		rcloneClient = client.NewRcloneClient(cfg.RCAPIURL, user, pass)
	}

	if runner == nil {
		runner = DefaultCommandRunner
	}

	return &RemoteAuthService{
		Config:           cfg,
		Client:           rcloneClient,
		AuditLogger:      NewAuditLogger(DefaultMaxAuditEvents),
		Runner:           runner,
		activeReconnects: make(map[string]context.CancelFunc),
	}
}

// GetAuthStatus queries the complete authentication health state across all configured remotes (default cached).
func (s *RemoteAuthService) GetAuthStatus(ctx context.Context) (models.AuthStatusResponse, error) {
	return s.GetAuthStatusCached(ctx, false)
}

// GetAuthStatusCached queries authentication health state with 60s TTL caching.
func (s *RemoteAuthService) GetAuthStatusCached(ctx context.Context, forceRefresh bool) (models.AuthStatusResponse, error) {
	if !forceRefresh {
		s.mu.Lock()
		if !s.cachedStatusTime.IsZero() && time.Since(s.cachedStatusTime) < 60*time.Second {
			cached := s.cachedStatus
			s.mu.Unlock()
			return cached, nil
		}
		s.mu.Unlock()
	}

	res, err := s.GetAllRemotesAuthStatus(ctx)
	if err == nil {
		s.mu.Lock()
		s.cachedStatus = res
		s.cachedStatusTime = time.Now()
		s.mu.Unlock()
	}
	return res, err
}

// InvalidateAuthCache clears cached auth state.
func (s *RemoteAuthService) InvalidateAuthCache() {
	s.mu.Lock()
	s.cachedStatusTime = time.Time{}
	s.mu.Unlock()
}

// GetAuditEvents retrieves the most recent OAuth audit lifecycle records.
func (s *RemoteAuthService) GetAuditEvents(limit int) []models.OAuthAuditEvent {
	return s.AuditLogger.GetEvents(limit)
}
