package remoteauth

import (
	"context"
	"sync"

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

// GetAuthStatus queries the complete authentication health state across all configured remotes.
func (s *RemoteAuthService) GetAuthStatus(ctx context.Context) (models.AuthStatusResponse, error) {
	return s.GetAllRemotesAuthStatus(ctx)
}

// GetAuditEvents retrieves the most recent OAuth audit lifecycle records.
func (s *RemoteAuthService) GetAuditEvents(limit int) []models.OAuthAuditEvent {
	return s.AuditLogger.GetEvents(limit)
}
