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

// createPasswordHelperScript creates an ephemeral POSIX script to safely feed passwords
// to rclone's --password-command without shell quotation or parsing issues.
func createPasswordHelperScript() (string, error) {
	tmpFile, err := os.CreateTemp("", "rclone-pw-helper-*.sh")
	if err != nil {
		return "", fmt.Errorf("failed to create temp password helper: %w", err)
	}
	scriptContent := `#!/bin/sh
if [ "$RCLONE_PASSWORD_CHANGE" = "1" ]; then
    echo "$NEW_PASS"
else
    echo "$OLD_PASS"
fi
`
	if _, err := tmpFile.WriteString(scriptContent); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to write temp password helper: %w", err)
	}
	if err := tmpFile.Chmod(0700); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to chmod temp password helper: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to close temp password helper: %w", err)
	}
	return tmpFile.Name(), nil
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

	helperScript, err := createPasswordHelperScript()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = os.Remove(helperScript)
	}()

	args := []string{
		"config", "encryption", "set",
		"--config", configFile,
		"--ask-password=false",
		"--password-command", helperScript,
	}

	_, err = s.Runner(ctx, "rclone", env, args...)
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

	env := append(os.Environ(),
		fmt.Sprintf("OLD_PASS=%s", req.Password),
		fmt.Sprintf("NEW_PASS=%s", ""),
	)

	helperScript, err := createPasswordHelperScript()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = os.Remove(helperScript)
	}()

	args := []string{
		"config", "encryption", "remove",
		"--config", configFile,
		"--ask-password=false",
		"--password-command", helperScript,
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
