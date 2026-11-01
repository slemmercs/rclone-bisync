package services

import (
	"context"
	"fmt"

	"github.com/slemmercs/rclone-bisync/internal/audit"
	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/core/client"
	"github.com/slemmercs/rclone-bisync/internal/core/lock"
	"github.com/slemmercs/rclone-bisync/internal/core/logger"
	"github.com/slemmercs/rclone-bisync/internal/core/state"
	"github.com/slemmercs/rclone-bisync/internal/models"
)

var ProviderLabels = map[string]string{
	"drive":                "Google Drive",
	"onedrive":             "Microsoft OneDrive",
	"dropbox":              "Dropbox",
	"s3":                   "Amazon S3",
	"b2":                   "Backblaze B2",
	"box":                  "Box",
	"pcloud":               "pCloud",
	"mega":                 "Mega",
	"sftp":                 "SFTP",
	"smb":                  "SMB / CIFS",
	"webdav":               "WebDAV",
	"azureblob":            "Azure Blob Storage",
	"google cloud storage": "Google Cloud Storage",
}

// SyncService orchestrates bisync operations, exploration, and telemetry.
type SyncService struct {
	Config    *config.Config
	Client    *client.RcloneClient
	Canary    *CanaryService
	Deletion  *DeletionService
	Telemetry *TelemetryService
	Explorer  *ExplorerService
	Review    *ReviewService
	Audit     *AuditService
}

// NewSyncService initializes a new SyncService.
func NewSyncService(cfg *config.Config, rcClient *client.RcloneClient) *SyncService {
	if rcClient == nil {
		user := ""
		pass := ""
		if cfg != nil && cfg.RCAuthEnabled {
			user = cfg.RCUser
			pass = cfg.RCPass
		}
		rcURL := "http://localhost:5572"
		if cfg != nil && cfg.RCAPIURL != "" {
			rcURL = cfg.RCAPIURL
		}
		rcClient = client.NewRcloneClient(rcURL, user, pass)
	}
	canarySvc := NewCanaryService(cfg, rcClient)
	deletionSvc := NewDeletionService(cfg)
	telemetrySvc := NewTelemetryService(cfg, rcClient)
	explorerSvc := NewExplorerService(cfg, rcClient, canarySvc)
	reviewSvc := NewReviewService(cfg, rcClient)

	var auditSvc *AuditService
	if cfg != nil {
		dbPath := cfg.AuditDBPath
		if db, err := audit.OpenDB(dbPath); err == nil && db != nil {
			var runLogger *logger.Logger
			if cfg.LogFile != "" {
				runLogger, _ = logger.New(cfg.LogFile, true)
			}
			auditSvc = NewAuditService(db, runLogger)
			auditSvc.DBPath = dbPath
		}
	}
	if reviewSvc != nil {
		reviewSvc.Audit = auditSvc
	}

	return &SyncService{
		Config:    cfg,
		Client:    rcClient,
		Canary:    canarySvc,
		Deletion:  deletionSvc,
		Telemetry: telemetrySvc,
		Explorer:  explorerSvc,
		Review:    reviewSvc,
		Audit:     auditSvc,
	}
}

// IsSyncing checks if a sync is running via flock lockfile.
func (s *SyncService) IsSyncing() bool {
	return lock.IsLocked(s.Config.LockFile)
}

// GetDaemonHealth retrieves daemon health with caching.
func (s *SyncService) GetDaemonHealth(ctx context.Context) map[string]interface{} {
	return s.Telemetry.GetDaemonHealth(ctx)
}

// ParsePendingDeletionsFromLog scans daemon log for aborted deletions.
func (s *SyncService) ParsePendingDeletionsFromLog() state.PendingDeletionsData {
	return s.Deletion.ParsePendingDeletionsFromLog()
}

// GetPendingDeletions returns structured pending deletions or falls back to log parsing.
func (s *SyncService) GetPendingDeletions() state.PendingDeletionsData {
	return s.Deletion.GetPendingDeletions()
}

// GetRemotesSummary queries remotes and storage metrics.
func (s *SyncService) GetRemotesSummary(ctx context.Context) (models.RemotesResponse, error) {
	return s.Telemetry.GetRemotesSummary(ctx)
}

// GetTransferStats queries transfer statistics.
func (s *SyncService) GetTransferStats(ctx context.Context) *models.TransferStats {
	return s.Telemetry.GetTransferStats(ctx)
}

// GetCachedMetrics returns parsed and cached metrics lines.
func (s *SyncService) GetCachedMetrics(linesCount int) []map[string]interface{} {
	return s.Telemetry.GetCachedMetrics(linesCount)
}

// GetRuntimeConfig returns sanitized configuration map with masked credentials.
func (s *SyncService) GetRuntimeConfig() map[string]interface{} {
	return s.Telemetry.GetRuntimeConfig()
}

// GetTelemetryPayload aggregates daemon health, logs, metrics, config, and transfer stats.
func (s *SyncService) GetTelemetryPayload(ctx context.Context, limit int) map[string]interface{} {
	return s.Telemetry.GetTelemetryPayload(ctx, s.IsSyncing(), limit)
}

// BrowsePath lists directory items and verifies canary presence, enforcing path jails for local FS.
func (s *SyncService) BrowsePath(ctx context.Context, remote, path string) client.DirectoryListing {
	return s.Explorer.BrowsePath(ctx, remote, path)
}

// CreateCanary provisions an RCLONE_TEST canary file on target local or remote filesystem.
func (s *SyncService) CreateCanary(ctx context.Context, remote, path string) (models.CanaryResponse, error) {
	return s.Canary.CreateCanary(ctx, remote, path)
}

// CopyItem transfers a file or directory with path jail enforcement.
func (s *SyncService) CopyItem(ctx context.Context, srcRemote, srcPath, dstRemote, dstPath string, isDir bool) (models.CopyResponse, error) {
	return s.Explorer.CopyItem(ctx, srcRemote, srcPath, dstRemote, dstPath, isDir)
}

// DeleteItem removes a file or directory, safeguarding RCLONE_TEST canary and enforcing path jail.
func (s *SyncService) DeleteItem(ctx context.Context, remote, path string, isDir bool) (models.DeleteResponse, error) {
	return s.Explorer.DeleteItem(ctx, remote, path, isDir)
}

// GetActiveReviewSession returns the active review session, if any.
func (s *SyncService) GetActiveReviewSession(ctx context.Context) (*models.ReviewSession, error) {
	if s.Review == nil {
		return nil, fmt.Errorf("review service not initialized")
	}
	return s.Review.GetActiveSession(ctx)
}

// StageReviewDecision stages a decision on an active review candidate.
func (s *SyncService) StageReviewDecision(sessionID, candidateID string, decision models.ActionDecision) (*models.ReviewSession, error) {
	if s.Review == nil {
		return nil, fmt.Errorf("review service not initialized")
	}
	return s.Review.StageCandidateDecision(sessionID, candidateID, decision)
}

// ApplyReviewSession applies staged decisions on an active review session.
func (s *SyncService) ApplyReviewSession(ctx context.Context, sessionID string) (*models.ReviewSession, error) {
	if s.Review == nil {
		return nil, fmt.Errorf("review service not initialized")
	}
	return s.Review.ApplyReviewSession(ctx, sessionID)
}

// RollbackReviewSession performs Saga compensation on an applied review session.
func (s *SyncService) RollbackReviewSession(ctx context.Context, sessionID string) (*models.ReviewSession, error) {
	if s.Review == nil {
		return nil, fmt.Errorf("review service not initialized")
	}
	return s.Review.RollbackReviewSession(ctx, sessionID)
}

// DismissReviewSession dismisses and archives an active review session.
func (s *SyncService) DismissReviewSession(sessionID string) error {
	if s.Review == nil {
		return fmt.Errorf("review service not initialized")
	}
	return s.Review.DismissReviewSession(sessionID)
}
