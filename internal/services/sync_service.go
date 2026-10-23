package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/core/client"
	"github.com/slemmercs/rclone-bisync/internal/core/lock"
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

// FileCacheItem stores cached file content with mtime/size headers.
type FileCacheItem struct {
	MTime int64
	Size  int64
	Data  interface{}
}

// LiveHealthCache stores daemon health check with timestamp.
type LiveHealthCache struct {
	Time float64
	Data map[string]interface{}
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
}

// NewSyncService initializes a new SyncService.
func NewSyncService(cfg *config.Config, rcClient *client.RcloneClient) *SyncService {
	if rcClient == nil {
		user := ""
		pass := ""
		if cfg.RCAuthEnabled {
			user = cfg.RCUser
			pass = cfg.RCPass
		}
		rcClient = client.NewRcloneClient(cfg.RCAPIURL, user, pass)
	}
	canarySvc := NewCanaryService(cfg, rcClient)
	deletionSvc := NewDeletionService(cfg)
	telemetrySvc := NewTelemetryService(cfg, rcClient)
	explorerSvc := NewExplorerService(cfg, rcClient, canarySvc)
	reviewSvc := NewReviewService(cfg, rcClient)
	return &SyncService{
		Config:    cfg,
		Client:    rcClient,
		Canary:    canarySvc,
		Deletion:  deletionSvc,
		Telemetry: telemetrySvc,
		Explorer:  explorerSvc,
		Review:    reviewSvc,
	}
}

// IsSyncing checks if a sync is running via flock lockfile.
func (s *SyncService) IsSyncing() bool {
	return lock.IsLocked(s.Config.LockFile)
}

// HandleAction executes system actions (start_sync, stop_sync, force_resync, etc.).
func (s *SyncService) HandleAction(ctx context.Context, action string) (models.ActionResponse, error) {
	switch action {
	case "start_sync":
		if s.IsSyncing() {
			return models.ActionResponse{Status: "error", Msg: "Sync already in progress."}, nil
		}
		if s.Review != nil && s.Review.HasBlockingReview() {
			return models.ActionResponse{
				Status: "error",
				Msg:    "Sync blocked: unresolved interactive review session requires action. Review pending deletions or dismiss the session.",
			}, nil
		}
		if err := s.runBisyncSystemd(nil, "bisync-gdrive-manual", "dashboard"); err != nil {
			return models.ActionResponse{}, err
		}
		return models.ActionResponse{Status: "success", Msg: "Sync started via systemd (origin: dashboard)"}, nil

	case "stop_sync":
		stopped := 0
		for _, jobID := range s.Client.ListJobs(ctx) {
			if s.Client.StopJob(ctx, jobID) {
				stopped++
			}
		}
		_ = exec.Command("systemctl", "--user", "stop",
			"bisync-gdrive.service",
			"bisync-gdrive-manual.service",
			"bisync-gdrive-resync.service",
			"bisync-gdrive-force.service",
			"bisync-gdrive-dryrun.service",
		).Run()
		return models.ActionResponse{
			Status: "success",
			Msg:    fmt.Sprintf("Sync stopped (%d daemon jobs halted)", stopped),
		}, nil

	case "force_resync":
		if s.IsSyncing() {
			return models.ActionResponse{Status: "error", Msg: "Cannot resync: another sync is actively running."}, nil
		}
		if s.Review != nil && s.Review.HasBlockingReview() {
			return models.ActionResponse{
				Status: "error",
				Msg:    "Resync blocked: unresolved interactive review session requires action. Review pending deletions or dismiss the session.",
			}, nil
		}
		if err := s.runBisyncSystemd([]string{"--resync"}, "bisync-gdrive-resync", "dashboard"); err != nil {
			return models.ActionResponse{}, err
		}
		return models.ActionResponse{Status: "success", Msg: "Resync initiated via systemd (origin: dashboard)"}, nil

	case "force_sync":
		if s.IsSyncing() {
			return models.ActionResponse{Status: "error", Msg: "Cannot force sync: another sync is actively running."}, nil
		}
		if err := s.runBisyncSystemd([]string{"--force"}, "bisync-gdrive-force", "dashboard"); err != nil {
			return models.ActionResponse{}, err
		}
		return models.ActionResponse{Status: "success", Msg: "Force sync initiated via systemd (deletions authorized)"}, nil

	case "dry_run":
		if s.IsSyncing() {
			return models.ActionResponse{Status: "error", Msg: "Cannot start dry run: another sync is actively running."}, nil
		}
		if err := s.runBisyncSystemd([]string{"--dry-run"}, "bisync-gdrive-dryrun", "dashboard"); err != nil {
			return models.ActionResponse{}, err
		}
		return models.ActionResponse{Status: "success", Msg: "Dry-run simulation started via systemd (origin: dashboard)"}, nil

	case "remove_lock":
		if s.IsSyncing() {
			return models.ActionResponse{Status: "error", Msg: "Cannot remove lock: a synchronization run is actively in progress."}, nil
		}
		if _, err := os.Stat(s.Config.LockFile); err == nil {
			_ = os.Remove(s.Config.LockFile)
			return models.ActionResponse{Status: "success", Msg: "Lock file removed"}, nil
		}
		return models.ActionResponse{Status: "info", Msg: "No lock file found"}, nil

	case "clear_rate_limit":
		if state.ClearRateLimitBackoff(s.Config.RateLimitBackoffFile) {
			return models.ActionResponse{Status: "success", Msg: "Rate limit backoff cleared"}, nil
		}
		return models.ActionResponse{Status: "info", Msg: "No active rate limit flag found"}, nil

	default:
		return models.ActionResponse{Status: "error", Msg: fmt.Sprintf("Unknown action: '%s'", action)}, nil
	}
}

func (s *SyncService) runBisyncSystemd(extraArgs []string, unitName, origin string) error {
	_ = exec.Command("systemctl", "--user", "reset-failed", unitName).Run()

	homeDir, _ := os.UserHomeDir()
	binPath := filepath.Join(s.Config.BaseDir, "bin", "rclone-bisync")
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		// Fallback to rclone-bisync in BaseDir or PATH
		if _, err := os.Stat(filepath.Join(s.Config.BaseDir, "rclone-bisync")); err == nil {
			binPath = filepath.Join(s.Config.BaseDir, "rclone-bisync")
		} else if p, err := exec.LookPath("rclone-bisync"); err == nil {
			binPath = p
		}
	}

	execArgs := []string{binPath, "sync", fmt.Sprintf("--origin=%s", origin)}
	if len(extraArgs) > 0 {
		execArgs = append(execArgs, extraArgs...)
	}

	cmdArgs := []string{
		"systemd-run", "--user",
		fmt.Sprintf("--unit=%s", unitName),
		"--service-type=oneshot",
		fmt.Sprintf("--working-directory=%s", s.Config.BaseDir),
		fmt.Sprintf("--property=EnvironmentFile=-%s/.config/dbus-vars", homeDir),
		fmt.Sprintf("--property=EnvironmentFile=%s", filepath.Join(s.Config.BaseDir, "rc-secrets.env")),
		fmt.Sprintf("--property=EnvironmentFile=%s", filepath.Join(s.Config.BaseDir, "bisync.env")),
		"--property=MemoryMax=1G",
		"--property=CPUQuota=80%",
		"--property=NoNewPrivileges=true",
		"--property=PrivateTmp=true",
		"--property=StandardOutput=journal",
		"--property=StandardError=journal",
		"--property=SyslogIdentifier=bisync-gdrive",
	}
	cmdArgs = append(cmdArgs, execArgs...)

	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	return cmd.Run()
}

// GetDaemonHealth retrieves daemon health with caching.
func (s *SyncService) GetDaemonHealth(ctx context.Context) map[string]interface{} {
	return s.Telemetry.GetDaemonHealth(ctx)
}

// TailFile reads the last N whole lines from a file by seeking from EOF.
func TailFile(filepathStr string, linesCount int) []string {
	f, err := os.Open(filepathStr)
	if err != nil {
		return []string{fmt.Sprintf("Log file target not found at: %s", filepathStr)}
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil || stat.Size() == 0 {
		return []string{}
	}

	position := stat.Size()
	var bufferSize int64 = 8192
	var lines []string
	remainder := []byte{}

	for position > 0 && len(lines) < linesCount {
		toRead := bufferSize
		if position < toRead {
			toRead = position
		}
		position -= toRead

		_, _ = f.Seek(position, io.SeekStart)
		chunk := make([]byte, toRead)
		_, _ = io.ReadFull(f, chunk)
		chunk = append(chunk, remainder...)

		chunkLines := bytes.Split(chunk, []byte("\n"))
		remainder = chunkLines[0]

		for i := len(chunkLines) - 1; i >= 1; i-- {
			lineBytes := bytes.TrimRight(chunkLines[i], "\r")
			if len(lineBytes) > 0 {
				lines = append(lines, string(lineBytes))
				if len(lines) >= linesCount {
					break
				}
			}
		}
	}

	if len(remainder) > 0 && len(lines) < linesCount {
		trimmed := bytes.TrimRight(remainder, "\r")
		if len(trimmed) > 0 {
			lines = append(lines, string(trimmed))
		}
	}

	// Reverse to restore original chronological order
	for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
		lines[i], lines[j] = lines[j], lines[i]
	}

	return lines
}

// TailJSONLog reads log file, unwrapping nested JSON strings if present.
func TailJSONLog(filepathStr string, entriesCount int) []string {
	rawLines := TailFile(filepathStr, entriesCount*2)
	entries := make([]string, 0)

	for _, line := range rawLines {
		lineStr := strings.TrimSpace(line)
		if lineStr == "" {
			continue
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(lineStr), &parsed); err == nil {
			if msg, ok := parsed["msg"].(string); ok && (strings.Contains(msg, `{"`) || strings.Contains(msg, `{'`)) {
				sublines := strings.Split(msg, "\n")
				unwrappedAny := false
				for _, sub := range sublines {
					subTrim := strings.TrimSpace(sub)
					if strings.HasPrefix(subTrim, "{") && strings.HasSuffix(subTrim, "}") {
						var inner map[string]interface{}
						if err := json.Unmarshal([]byte(subTrim), &inner); err == nil {
							b, _ := json.Marshal(inner)
							entries = append(entries, string(b))
							unwrappedAny = true
						}
					}
				}
				if !unwrappedAny {
					b, _ := json.Marshal(parsed)
					entries = append(entries, string(b))
				}
			} else {
				b, _ := json.Marshal(parsed)
				entries = append(entries, string(b))
			}
		} else {
			entries = append(entries, lineStr)
		}
	}

	if len(entries) > entriesCount {
		return entries[len(entries)-entriesCount:]
	}
	return entries
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
func (s *SyncService) CreateCanary(ctx context.Context, remote, path string) (map[string]interface{}, error) {
	return s.Canary.CreateCanary(ctx, remote, path)
}

// CopyItem transfers a file or directory with path jail enforcement.
func (s *SyncService) CopyItem(ctx context.Context, srcRemote, srcPath, dstRemote, dstPath string, isDir bool) (models.CopyResponse, error) {
	return s.Explorer.CopyItem(ctx, srcRemote, srcPath, dstRemote, dstPath, isDir)
}

// DeleteItem removes a file or directory, safeguarding RCLONE_TEST canary and enforcing path jail.
func (s *SyncService) DeleteItem(ctx context.Context, remote, path string, isDir bool) (map[string]interface{}, error) {
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
