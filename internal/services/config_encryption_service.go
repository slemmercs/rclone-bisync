package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/core/client"
	"github.com/slemmercs/rclone-bisync/internal/models"
)

// CommandRunner defines the execution contract for running external CLI processes.
type CommandRunner func(ctx context.Context, name string, env []string, args ...string) (string, error)

// DefaultCommandRunner executes the system rclone binary with the provided arguments and environment.
func DefaultCommandRunner(ctx context.Context, name string, env []string, args ...string) (string, error) {
	// #nosec G204 - controlled configuration commands with system rclone binary
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

// ConfigEncryptionService manages rclone.conf password protection.
type ConfigEncryptionService struct {
	Config *config.Config
	Client *client.RcloneClient
	Runner CommandRunner
}

// NewConfigEncryptionService creates an instance of ConfigEncryptionService.
func NewConfigEncryptionService(cfg *config.Config, c *client.RcloneClient, runner CommandRunner) *ConfigEncryptionService {
	if runner == nil {
		runner = DefaultCommandRunner
	}
	return &ConfigEncryptionService{
		Config: cfg,
		Client: c,
		Runner: runner,
	}
}

// ResolveConfigFile determines the path of the active rclone.conf file.
func (s *ConfigEncryptionService) ResolveConfigFile(ctx context.Context) string {
	if s.Config != nil && s.Config.RcloneConfigFile != "" {
		return s.Config.RcloneConfigFile
	}
	if s.Client != nil {
		if paths, err := s.Client.GetConfigPaths(ctx); err == nil && paths != nil && paths.Config != "" {
			return paths.Config
		}
	}
	if env := os.Getenv("RCLONE_CONFIG"); env != "" {
		return env
	}
	if s.Config != nil {
		for i, arg := range s.Config.RcloneCommonArgs {
			if arg == "--config" && i+1 < len(s.Config.RcloneCommonArgs) {
				return s.Config.RcloneCommonArgs[i+1]
			}
			if strings.HasPrefix(arg, "--config=") {
				return strings.TrimPrefix(arg, "--config=")
			}
		}
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "rclone", "rclone.conf")
}

// GetStatus checks whether the active rclone.conf is encrypted with a password.
func (s *ConfigEncryptionService) GetStatus(ctx context.Context) (*models.EncryptionStatusResponse, error) {
	configFile := s.ResolveConfigFile(ctx)

	// #nosec G304 - reading configured rclone configuration file path
	f, err := os.Open(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return &models.EncryptionStatusResponse{
				IsEncrypted: false,
				ConfigFile:  configFile,
			}, nil
		}
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer f.Close()

	buf := make([]byte, 2048)
	n, _ := io.ReadFull(f, buf)
	content := string(buf[:n])

	isEncrypted := strings.Contains(content, "RCLONE_ENCRYPT_V0:") ||
		strings.Contains(content, "# Encrypted rclone configuration File")

	return &models.EncryptionStatusResponse{
		IsEncrypted: isEncrypted,
		ConfigFile:  configFile,
	}, nil
}

// SetPassword sets or updates the encryption password on rclone.conf.
func (s *ConfigEncryptionService) SetPassword(ctx context.Context, req models.SetEncryptionRequest) (*models.EncryptionActionResponse, error) {
	newPass := strings.TrimSpace(req.Password)
	if newPass == "" {
		return nil, fmt.Errorf("password cannot be empty")
	}

	configFile := s.ResolveConfigFile(ctx)

	env := append(os.Environ(),
		fmt.Sprintf("NEW_PASS=%s", newPass),
		fmt.Sprintf("OLD_PASS=%s", req.OldPassword),
	)

	// #nosec G101 - script pattern passed to rclone --password-command via environment variables
	pwCmd := `sh -c 'if [ "$RCLONE_PASSWORD_CHANGE" = "1" ]; then echo "$NEW_PASS"; else echo "$OLD_PASS"; fi'`

	args := []string{
		"config", "encryption", "set",
		"--config", configFile,
		"--ask-password=false",
		"--password-command", pwCmd,
	}

	_, err := s.Runner(ctx, "rclone", env, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to set encryption password: %w", err)
	}

	return &models.EncryptionActionResponse{
		Status:  "success",
		Message: "Configuration password set successfully",
	}, nil
}

// RemovePassword removes the encryption password from rclone.conf.
func (s *ConfigEncryptionService) RemovePassword(ctx context.Context, req models.RemoveEncryptionRequest) (*models.EncryptionActionResponse, error) {
	configFile := s.ResolveConfigFile(ctx)

	status, err := s.GetStatus(ctx)
	if err == nil && !status.IsEncrypted {
		return &models.EncryptionActionResponse{
			Status:  "success",
			Message: "Configuration file is not encrypted",
		}, nil
	}

	env := append(os.Environ(), fmt.Sprintf("OLD_PASS=%s", req.Password))
	// #nosec G101 - script pattern passed to rclone --password-command via environment variable
	pwCmd := `sh -c 'echo "$OLD_PASS"'`

	args := []string{
		"config", "encryption", "remove",
		"--config", configFile,
		"--ask-password=false",
		"--password-command", pwCmd,
	}

	_, err = s.Runner(ctx, "rclone", env, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to remove encryption password: %w", err)
	}

	return &models.EncryptionActionResponse{
		Status:  "success",
		Message: "Configuration password removed successfully",
	}, nil
}
