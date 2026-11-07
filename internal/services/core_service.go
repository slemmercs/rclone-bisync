package services

import (
"context"
"fmt"
"strings"

"github.com/slemmercs/rclone-bisync/internal/config"
"github.com/slemmercs/rclone-bisync/internal/core/client"
"github.com/slemmercs/rclone-bisync/internal/models"
)

// CoreService provides high-level domain operations for Rclone core runtime controls (bandwidth, stats).
type CoreService struct {
	Config *config.Config
	Client *client.RcloneClient
}

// NewCoreService constructs a CoreService with configuration and RC client references.
func NewCoreService(cfg *config.Config, rcloneClient *client.RcloneClient) *CoreService {
	return &CoreService{
		Config: cfg,
		Client: rcloneClient,
	}
}

// GetBandwidthLimit queries the currently active bandwidth rate limit from the Rclone RC daemon.
func (s *CoreService) GetBandwidthLimit(ctx context.Context) (*models.BandwidthLimitResponse, error) {
	if s.Client == nil {
		return nil, fmt.Errorf("rclone client is not configured")
	}
	return s.Client.GetBandwidthLimit(ctx)
}

// SetBandwidthLimit sets a dynamic bandwidth limit on the Rclone RC daemon.
func (s *CoreService) SetBandwidthLimit(ctx context.Context, rate string) (*models.BandwidthLimitResponse, error) {
	if s.Client == nil {
		return nil, fmt.Errorf("rclone client is not configured")
	}
	cleanRate := strings.TrimSpace(rate)
	if cleanRate == "" {
		return nil, fmt.Errorf("rate cannot be empty")
	}
	return s.Client.SetBandwidthLimit(ctx, cleanRate)
}

// GetCoreStats retrieves aggregated transfer metrics and active file transfers from the Rclone daemon.
func (s *CoreService) GetCoreStats(ctx context.Context) (*models.CoreStatsResponse, error) {
	if s.Client == nil {
		return nil, fmt.Errorf("rclone client is not configured")
	}
	return s.Client.GetParsedCoreStats(ctx)
}
