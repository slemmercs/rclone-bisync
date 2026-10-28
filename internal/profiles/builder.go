package profiles

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/slemmercs/rclone-bisync/internal/config"
)

// ResolveFlags resolves the effective operational configuration for a profile
// by combining ExecutionOverrides > ProfileFlags > GlobalDefaults > Hardcoded Defaults.
func ResolveFlags(p *Profile, globals config.GlobalDefaults, overrides *ExecutionOverrides) ResolvedFlags {
	res := ResolvedFlags{
		ConflictResolve:  globals.ConflictResolve,
		MaxDelete:        globals.MaxDelete,
		CanaryValidation: globals.CanaryValidation,
		CompareMode:      globals.CompareMode,
		RemoveEmptyDirs:  globals.RemoveEmptyDirs,
		CreateEmptyDirs:  globals.CreateEmptyDirs,
		ExcludePatterns:  p.ExcludePatterns,
		FiltersFilePath:  p.Flags.FiltersFilePath,
		CustomFlags:      p.Flags.CustomFlags,
	}

	// Fallback to canonical hardcoded defaults if globals are uninitialized
	if res.ConflictResolve == "" {
		res.ConflictResolve = "newer"
	}
	if res.MaxDelete <= 0 {
		res.MaxDelete = 50
	}
	if res.CompareMode == "" {
		res.CompareMode = "modtime"
	}

	// Apply Profile-Level Overrides (non-nil pointers)
	if p.Flags.ConflictResolve != nil && *p.Flags.ConflictResolve != "" {
		res.ConflictResolve = *p.Flags.ConflictResolve
	}
	if p.Flags.MaxDelete != nil {
		res.MaxDelete = *p.Flags.MaxDelete
	}
	if p.Flags.CanaryValidation != nil {
		res.CanaryValidation = *p.Flags.CanaryValidation
	}
	if p.Flags.CompareMode != nil && *p.Flags.CompareMode != "" {
		res.CompareMode = *p.Flags.CompareMode
	}
	if p.Flags.RemoveEmptyDirs != nil {
		res.RemoveEmptyDirs = *p.Flags.RemoveEmptyDirs
	}
	if p.Flags.CreateEmptyDirs != nil {
		res.CreateEmptyDirs = *p.Flags.CreateEmptyDirs
	}

	// Apply Single-Run Execution Overrides
	if overrides != nil {
		if overrides.DryRun != nil && *overrides.DryRun {
			res.DryRun = true
		}
		if overrides.Resync != nil && *overrides.Resync {
			res.Resync = true
		}
		if overrides.Force != nil && *overrides.Force {
			res.Force = true
		}
	}

	return res
}

// BuildCLIArgs generates the complete rclone bisync operational arguments based on
// resolved flag inheritance and validated custom flags.
func BuildCLIArgs(p *Profile, globals config.GlobalDefaults, overrides *ExecutionOverrides) ([]string, error) {
	effective := ResolveFlags(p, globals, overrides)

	var args []string

	// Tier 1: Execution Controls & Safety Thresholds
	if effective.DryRun {
		args = append(args, "--dry-run")
	}
	if effective.Resync {
		args = append(args, "--resync")
	}
	if effective.Force {
		args = append(args, "--force")
	}
	if effective.ConflictResolve != "" {
		args = append(args, "--conflict-resolve", effective.ConflictResolve)
	}
	if effective.MaxDelete > 0 {
		args = append(args, "--max-delete", strconv.Itoa(effective.MaxDelete))
	}

	// Tier 2: Profile Settings
	if effective.CanaryValidation {
		args = append(args, "--check-access", "--check-filename", "RCLONE_TEST")
	}
	if effective.CompareMode != "" {
		args = append(args, "--compare", effective.CompareMode)
	}
	for _, pattern := range effective.ExcludePatterns {
		if tr := strings.TrimSpace(pattern); tr != "" {
			args = append(args, "--exclude", tr)
		}
	}
	if strings.TrimSpace(effective.FiltersFilePath) != "" {
		args = append(args, "--filters-file", strings.TrimSpace(effective.FiltersFilePath))
	}
	if effective.RemoveEmptyDirs {
		args = append(args, "--remove-empty-dirs")
	}
	if effective.CreateEmptyDirs {
		args = append(args, "--create-empty-src-dirs")
	}

	// Tier 3: Custom Flags Escape Hatch
	customTokens, err := ValidateCustomFlags(effective.CustomFlags)
	if err != nil {
		return nil, fmt.Errorf("failed to validate custom flags for profile %q: %w", p.ID, err)
	}
	if len(customTokens) > 0 {
		args = append(args, customTokens...)
	}

	return args, nil
}
