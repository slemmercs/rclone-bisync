package services

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/slemmercs/rclone-bisync/internal/core/state"
	"github.com/slemmercs/rclone-bisync/internal/models"
)

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
	// #nosec G204 - systemd unit reset using systemctl with controlled unitName
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

	// #nosec G204 - systemd-run invocation with strictly structured parameters
	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		outStr := strings.TrimSpace(string(out))
		if outStr != "" {
			return fmt.Errorf("%s: %s", err, outStr)
		}
		return err
	}
	return nil
}
