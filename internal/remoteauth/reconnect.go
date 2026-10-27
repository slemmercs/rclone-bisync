package remoteauth

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/models"
)

// CommandRunner defines the execution contract for running external CLI processes.
type CommandRunner func(ctx context.Context, name string, env []string, args ...string) (string, error)

// DefaultCommandRunner executes the system rclone binary with the provided arguments and environment.
func DefaultCommandRunner(ctx context.Context, name string, env []string, args ...string) (string, error) {
	// #nosec G204 - external process execution wrapper for rclone authorization
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = env
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := strings.TrimSpace(stdout.String())
	if err != nil {
		errStr := strings.TrimSpace(stderr.String())
		if errStr != "" {
			return output, fmt.Errorf("%w: %s", err, errStr)
		}
		return output, err
	}
	return output, nil
}

// collectSessionEnvironment extracts desktop session variables to enable browser launching.
func collectSessionEnvironment() []string {
	// Base environment
	env := os.Environ()

	// If systemd user session file exists, merge DBus session vars
	home, err := os.UserHomeDir()
	if err == nil {
		dbusPath := filepath.Join(home, ".config", "dbus-vars")
		// #nosec G304 - reading fixed user config file for dbus session variables
		if f, err := os.Open(filepath.Clean(dbusPath)); err == nil {
			defer f.Close()
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line != "" && !strings.HasPrefix(line, "#") && strings.Contains(line, "=") {
					env = append(env, line)
				}
			}
		}
	}

	return env
}

// ReconnectRemote executes the desktop-integrated OAuth re-authentication flow for a remote.
func (s *RemoteAuthService) ReconnectRemote(ctx context.Context, remoteName string) (models.ReconnectResponse, error) {
	cleanName := strings.TrimRight(remoteName, ":")
	if cleanName == "" {
		return models.ReconnectResponse{
			Status: "error",
			Error:  "Remote name cannot be empty",
		}, fmt.Errorf("remote name cannot be empty")
	}

	// Prevent overlapping concurrent reconnects for the same remote
	s.mu.Lock()
	if _, exists := s.activeReconnects[cleanName]; exists {
		s.mu.Unlock()
		return models.ReconnectResponse{
			Status: "error",
			Remote: cleanName,
			Error:  fmt.Sprintf("Reconnect flow already in progress for remote '%s'", cleanName),
		}, fmt.Errorf("reconnect flow already in progress for remote '%s'", cleanName)
	}

	// Create cancellable context with 3-minute timeout for interactive browser authorization
	reconnectCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	s.activeReconnects[cleanName] = cancel
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.activeReconnects, cleanName)
		s.mu.Unlock()
	}()

	s.AuditLogger.LogEvent("oauth_reconnect_started", cleanName, "started", "Initiating desktop OAuth reconnect flow")

	// Check if rclone RC already has an active OAuth server running with an auth URL
	_, authURL, _ := s.Client.OAuthStatus(ctx)

	// Execute `rclone config reconnect <remoteName>: --auto-confirm`
	env := collectSessionEnvironment()
	args := []string{"config", "reconnect", fmt.Sprintf("%s:", cleanName), "--auto-confirm"}

	output, err := s.Runner(reconnectCtx, "rclone", env, args...)
	if err != nil {
		s.AuditLogger.LogEvent("oauth_reconnect_failed", cleanName, "failed", err.Error())
		return models.ReconnectResponse{
			Status:  "error",
			Remote:  cleanName,
			AuthURL: authURL,
			Error:   fmt.Sprintf("Reconnect failed: %v", err),
		}, err
	}

	// Log success and validate
	s.AuditLogger.LogEvent("oauth_reconnect_success", cleanName, "success", "OAuth authorization completed successfully")

	return models.ReconnectResponse{
		Status:  "success",
		Remote:  cleanName,
		AuthURL: authURL,
		Msg:     fmt.Sprintf("Successfully re-authenticated remote '%s'. %s", cleanName, output),
	}, nil
}

// CancelReconnect cancels any ongoing reconnect runner for a specific remote and releases its lock.
func (s *RemoteAuthService) CancelReconnect(remoteName string) bool {
	cleanName := strings.TrimRight(remoteName, ":")
	s.mu.Lock()
	cancel, exists := s.activeReconnects[cleanName]
	if exists {
		cancel()
		delete(s.activeReconnects, cleanName)
	}
	s.mu.Unlock()

	if exists {
		s.AuditLogger.LogEvent("oauth_reconnect_cancelled", cleanName, "info", "User cancelled OAuth reconnect process")
	}
	return exists
}

// StopOAuth cancels any ongoing reconnect runners and stops the rclone OAuth web server.
func (s *RemoteAuthService) StopOAuth(ctx context.Context) error {
	s.mu.Lock()
	for name, cancel := range s.activeReconnects {
		cancel()
		delete(s.activeReconnects, name)
	}
	s.mu.Unlock()

	err := s.Client.OAuthStop(ctx)
	s.AuditLogger.LogEvent("oauth_stop", "all", "info", "Stopped OAuth authentication processes")
	return err
}
