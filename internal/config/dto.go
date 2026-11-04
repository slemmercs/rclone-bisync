package config

import "github.com/slemmercs/rclone-bisync/internal/models"

// GlobalDefaults represents baseline operational settings applied across all profiles
// unless explicitly overridden at the profile or execution level.
type GlobalDefaults = models.GlobalDefaults

// DefaultGlobalDefaults returns the canonical default operational configuration.
func DefaultGlobalDefaults() GlobalDefaults {
	return GlobalDefaults{
		MaxDelete:          0,
		CanaryValidation:   true,
		ConflictResolve:    "newer",
		CompareMode:        "size,modtime",
		RemoveEmptyDirs:    true,
		CreateEmptyDirs:    true,
		LockExplorerRemote: true,
	}
}
