package audit

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	// DefaultDBFilename is the filename for the SQLite audit database.
	DefaultDBFilename = "audit.db"
	// AppDirName is the directory name under user config for Bisync Hub.
	AppDirName = "bisync-hub"
	// EnvAuditDBPath is the environment variable override for the audit DB location.
	EnvAuditDBPath = "BISYNC_HUB_AUDIT_DB_PATH"
)

// ResolveDBPath resolves the absolute audit database path and guarantees directory existence with 0700 permissions.
// Resolution precedence:
// 1. Explicit override path (if non-empty)
// 2. BISYNC_HUB_AUDIT_DB_PATH environment variable (if non-empty)
// 3. User config directory ($XDG_CONFIG_HOME/bisync-hub/audit.db or ~/.config/bisync-hub/audit.db)
func ResolveDBPath(overridePath string) (string, error) {
	if overridePath != "" {
		absPath, err := filepath.Abs(overridePath)
		if err != nil {
			return "", fmt.Errorf("failed to resolve absolute path for override: %w", err)
		}
		if err := os.MkdirAll(filepath.Dir(absPath), 0700); err != nil {
			return "", fmt.Errorf("failed to create directory for custom audit db: %w", err)
		}
		return absPath, nil
	}

	if envPath := os.Getenv(EnvAuditDBPath); envPath != "" {
		absPath, err := filepath.Abs(envPath)
		if err != nil {
			return "", fmt.Errorf("failed to resolve absolute path for %s: %w", EnvAuditDBPath, err)
		}
		// #nosec G703 - directory creation for operator-configured audit db environment variable
		if err := os.MkdirAll(filepath.Dir(absPath), 0700); err != nil {
			return "", fmt.Errorf("failed to create directory for env audit db: %w", err)
		}
		return absPath, nil
	}

	baseConfigDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine user config directory: %w", err)
	}

	targetDir := filepath.Join(baseConfigDir, AppDirName)
	if err := os.MkdirAll(targetDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create default config directory: %w", err)
	}

	return filepath.Join(targetDir, DefaultDBFilename), nil
}
