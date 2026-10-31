# Design Specification: Bisync Operational Flags & Custom Flag Escape Hatch

| Metadata | Details |
|---|---|
| **Status** | `Shipped` (v1.0.0-beta.12) |
| **Domain** | Execution Engine & Configuration |
| **Governing ADR** | [ADR-007: 3-Tier Cascading Operational Flags Hierarchy](../architecture/adr/ADR-007-Cascading-Operational-Flags-Hierarchy.md) |
| **Core Principles** | [Product Principles](../architecture/PRODUCT_PRINCIPLES.md), [Security Principles](../architecture/SECURITY_PRINCIPLES.md) |

---

## Executive Summary

This design specification details the integration of `rclone bisync` CLI flags into **Bisync Hub**. Following the **Profile Manager** architecture (`profile-manager.md`), operational flags are strictly owned by Bisync Hub profiles—never by Rclone's remote connectivity layer (`rclone.conf`).

Rather than exposing all ~80+ Rclone CLI flags in the UI, Bisync Hub applies an opinionated Pareto model:
* **Tier 1 (Execution Controls):** Exposed directly on the dashboard execution bar and review modal for real-time operation.
* **Tier 2 (Profile Settings):** Maintained within the Profile configuration schema for profile-specific synchronization behavior.
* **Tier 3 (Custom Flags Escape Hatch):** A safe, audited text input for passing arbitrary CLI flags without cluttering the interface.

---

## 1. Governance & Hierarchy Model

To combine central configuration convenience with workload isolation, operational flags are managed via a two-tier hierarchy:

```
┌────────────────────────────────────────────────────────────────────────┐
│                   Global Application Defaults                          │
│                   (Managed in Dashboard Settings)                      │
│                                                                        │
│   max_delete: 50%          canary_validation: true                     │
│   conflict_resolve: "newer" compare_mode: "modtime"                    │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │ Inherited on Profile Creation
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│                        Bisync Hub Profile                              │
│                                                                        │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │                    Profile-Level Overrides                       │  │
│  │                                                                  │  │
│  │   Inherits Global Defaults UNLESS explicitly overridden:        │  │
│  │   - compare_mode: "checksum"  [OVERRIDDEN]                       │  │
│  │   - max_delete: 10%           [OVERRIDDEN]                       │  │
│  │   - canary_validation: true   [INHERITED]                        │  │
│  │   - custom_flags: "--bwlimit 10M" [PROFILE SPECIFIC]             │  │
│  └──────────────────────────────────────────────────────────────────┘  │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │ Resolves & Validates
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│                      Synchronization Orchestrator                      │
│                                                                        │
│  1. Resolves Effective Settings (Profile Override > Global Default)    │
│  2. Applies Execution-Time Overrides (e.g. Dry Run toggle in UI)       │
│  3. Validates against Security Blocklist & shlex parser                │
│  4. Executes `rclone bisync <path1> <path2> [resolved_args...]`        │
└────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Updated Data Structures (`internal/profiles/dto.go` & `internal/config/dto.go`)

### Global Application Defaults (`internal/config/dto.go`)

Stored centrally in `config.json` / `config.yaml` owned by Bisync Hub.

```go
package config

// GlobalDefaults defines fallback operational values for all new profiles.
type GlobalDefaults struct {
	MaxDelete        int    `json:"max_delete" yaml:"max_delete"`                 // Default: 50
	ConflictResolve  string `json:"conflict_resolve" yaml:"conflict_resolve"`     // Default: "newer"
	CanaryValidation bool   `json:"canary_validation" yaml:"canary_validation"`   // Default: true
	CompareMode      string `json:"compare_mode" yaml:"compare_mode"`             // Default: "modtime"
	RemoveEmptyDirs  bool   `json:"remove_empty_dirs" yaml:"remove_empty_dirs"`   // Default: true
	CreateEmptyDirs  bool   `json:"create_empty_dirs" yaml:"create_empty_dirs"`   // Default: true
}
```

### Profile Data Model (`internal/profiles/dto.go`)

Profile flags use pointer types (`*T`) for override fields. A `nil` value indicates that the profile inherits the Global Application Default.

```go
package profiles

import "[github.com/slemmercs/rclone-bisync/internal/config](https://github.com/slemmercs/rclone-bisync/internal/config)"

// Profile represents a managed synchronization workload.
type Profile struct {
	ID          string          `json:"id" yaml:"id"`
	Name        string          `json:"name" yaml:"name"`
	Enabled     bool            `json:"enabled" yaml:"enabled"`
	RemotePath1 string          `json:"remote_path1" yaml:"remote_path1"`
	RemotePath2 string          `json:"remote_path2" yaml:"remote_path2"`
	Settings    ProfileSettings `json:"settings" yaml:"settings"`
}

type ProfileSettings struct {
	Schedule       string      `json:"schedule" yaml:"schedule"`
	ReviewRequired bool        `json:"review_required" yaml:"review_required"`
	Notifications  string      `json:"notifications" yaml:"notifications"`
	Flags          BisyncFlags `json:"flags" yaml:"flags"`
}

// BisyncFlags contains nullable fields allowing explicit overrides or inheritance.
type BisyncFlags struct {
	// Tier 1 Overrides (nil = inherit global default)
	DryRun          bool    `json:"dry_run" yaml:"dry_run"`                     // Always explicit per run
	Resync          bool    `json:"resync" yaml:"resync"`                       // Always explicit per run
	Force           bool    `json:"force" yaml:"force"`                         // Always explicit per run
	ConflictResolve *string `json:"conflict_resolve,omitempty" yaml:"conflict_resolve,omitempty"`
	MaxDelete       *int    `json:"max_delete,omitempty" yaml:"max_delete,omitempty"`

	// Tier 2 Overrides (nil = inherit global default)
	CanaryValidation *bool    `json:"canary_validation,omitempty" yaml:"canary_validation,omitempty"`
	CompareMode      *string  `json:"compare_mode,omitempty" yaml:"compare_mode,omitempty"`
	ExcludePatterns  []string `json:"exclude_patterns" yaml:"exclude_patterns"`
	FiltersFilePath  string   `json:"filters_file_path" yaml:"filters_file_path"`
	RemoveEmptyDirs  *bool    `json:"remove_empty_dirs,omitempty" yaml:"remove_empty_dirs,omitempty"`
	CreateEmptyDirs  *bool    `json:"create_empty_dirs,omitempty" yaml:"create_empty_dirs,omitempty"`

	// Tier 3 Custom Flags Escape Hatch
	CustomFlags string `json:"custom_flags" yaml:"custom_flags"`
}

// Resolve merges global defaults, profile overrides, and execution controls.
func (pf *BisyncFlags) Resolve(globals config.GlobalDefaults) ResolvedFlags {
	res := ResolvedFlags{
		DryRun:           pf.DryRun,
		Resync:           pf.Resync,
		Force:            pf.Force,
		ConflictResolve:  globals.ConflictResolve,
		MaxDelete:        globals.MaxDelete,
		CanaryValidation: globals.CanaryValidation,
		CompareMode:      globals.CompareMode,
		RemoveEmptyDirs:  globals.RemoveEmptyDirs,
		CreateEmptyDirs:  globals.CreateEmptyDirs,
		ExcludePatterns:  pf.ExcludePatterns,
		FiltersFilePath:  pf.FiltersFilePath,
		CustomFlags:      pf.CustomFlags,
	}

	if pf.ConflictResolve != nil { res.ConflictResolve = *pf.ConflictResolve }
	if pf.MaxDelete != nil { res.MaxDelete = *pf.MaxDelete }
	if pf.CanaryValidation != nil { res.CanaryValidation = *pf.CanaryValidation }
	if pf.CompareMode != nil { res.CompareMode = *pf.CompareMode }
	if pf.RemoveEmptyDirs != nil { res.RemoveEmptyDirs = *pf.RemoveEmptyDirs }
	if pf.CreateEmptyDirs != nil { res.CreateEmptyDirs = *pf.CreateEmptyDirs }

	return res
}

type ResolvedFlags struct {
	DryRun           bool
	Resync           bool
	Force            bool
	ConflictResolve  string
	MaxDelete        int
	CanaryValidation bool
	CompareMode      string
	RemoveEmptyDirs  bool
	CreateEmptyDirs  bool
	ExcludePatterns  []string
	FiltersFilePath  string
	CustomFlags      string
}
```

---

## 3. Tier 1 Flag Specifications (Execution Controls)

Tier 1 flags directly govern active synchronization safety and conflict resolution. They are exposed in both the Profile Settings modal and as **one-time overrides** in the dashboard execution toolbar.

### Tier 1 Matrix

| Flag | Rclone Argument | Default | UI Control Type | Location | Behavior & Purpose |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Dry Run** | `--dry-run` | `false` | Toggle Switch | Dashboard / Execution Modal | Simulates sync execution without altering files. Generates preview logs for Safety Review. |
| **Resync** | `--resync` | `false` | Action Button / Checkbox | Run Dialog & Safety Recovery | Re-initializes bisync listing database. Used when state files are corrupted or out of sync. |
| **Force** | `--force` | `false` | Switch / Safety Override | Safety Review Barrier | Bypasses safety checks (such as max delete limit breaches) after human confirmation. |
| **Conflict Resolve** | `--conflict-resolve` | `newer` | Select Dropdown | Profile Settings & Review Modal | Determines automatic conflict winner: `newer`, `loser`, `path1`, `path2`, or `none` (trigger review). |
| **Max Delete** | `--max-delete` | `50` | Number Input (%) | Profile Settings / Safety Bar | Triggers a Safety Review Barrier if deletions exceed this percentage threshold. |

---

## 4. Tier 2 Flag Specifications (Profile Settings)

Tier 2 flags define profile-level behavior. They are configured during Profile creation or editing (`Profile Manager -> Settings Tab`).

### Tier 2 Matrix

| Flag Group | Rclone Arguments | Default | UI Control Type | Validation Rules |
| :--- | :--- | :--- | :--- | :--- |
| **Canary Validation** | `--check-access`, `--check-filename` | `true` | Checkbox Toggle | Ensures canary files (`RCLONE_TEST`) exist on both endpoints before starting sync. |
| **Comparison Mode** | `--compare` | `modtime` | Radio Group (`modtime`, `size`, `checksum`) | Defaults to `modtime`. Select `checksum` for bit-for-bit cloud verification. |
| **File Exclusions** | `--exclude`, `--filters-file` | Empty | Tag List / Path Input | Strips leading slashes. Rejects wildcard patterns that match critical system root paths. |
| **Directory Cleanup** | `--remove-empty-dirs`, `--create-empty-src-dirs` | `true` / `true` | Dual Checkbox | Maintains identical directory hierarchy across local and remote endpoints. |

---

## 5. Custom Flags Escape Hatch Specification

### Concept & Guardrails

The **Custom Flags Escape Hatch** provides a single freeform text field (`custom_flags`) in Profile settings, allowing advanced users to append arbitrary Rclone arguments safely (`--bwlimit 10M`, `--transfers 4`, etc.).

#### 1. Prohibited Flag Blocklist

The following flags are strictly rejected by the validator:

```go
var ProhibitedFlags = map[string]string{
	"--config":   "Rclone configuration is managed exclusively by Rclone RC Framework",
	"--use-mmap": "System-level memory overrides are prohibited",
	"--rc":       "Internal Remote Control server configuration cannot be overridden",
	"--rc-addr":  "Internal Remote Control port cannot be overridden",
	"--rc-user":  "RC authentication is managed internally",
	"--rc-pass":  "RC authentication is managed internally",
	"--log-file": "Logging is intercepted and handled by Bisync Hub Telemetry Engine",
	"--json":     "Log formatting is managed internally for structured parsing",
}
```

#### 2. Shell Injection Prevention
* Raw strings are tokenized using POSIX shell-quote rules (`shlex` parser) rather than passed to shell sub-processes.
* Arguments are passed directly as a discrete string slice (`[]string`) to `exec.Command("rclone", args...)`.

---

## 6. Flag Resolution & CLI Builder Engine (`internal/profiles/builder.go`)

```go
package profiles

import (
	"fmt"
	"strings"
	"[github.com/google/shlex](https://github.com/google/shlex)"
	"[github.com/slemmercs/rclone-bisync/internal/config](https://github.com/slemmercs/rclone-bisync/internal/config)"
)

// BuildCLIArgs generates the complete rclone bisync execution argument slice.
func BuildCLIArgs(p *Profile, globals config.GlobalDefaults, overrides *BisyncFlags) ([]string, error) {
	effective := p.Settings.Flags.Resolve(globals)

	// Apply execution-time single-run overrides (e.g., UI toolbar buttons)
	if overrides != nil {
		if overrides.DryRun { effective.DryRun = true }
		if overrides.Resync { effective.Resync = true }
		if overrides.Force { effective.Force = true }
	}

	args := []string{"bisync", p.RemotePath1, p.RemotePath2}

	// Tier 1 Flags
	if effective.DryRun { args = append(args, "--dry-run") }
	if effective.Resync { args = append(args, "--resync") }
	if effective.Force { args = append(args, "--force") }
	if effective.ConflictResolve != "" {
		args = append(args, "--conflict-resolve", effective.ConflictResolve)
	}
	if effective.MaxDelete > 0 {
		args = append(args, "--max-delete", fmt.Sprintf("%d", effective.MaxDelete))
	}

	// Tier 2 Flags
	if effective.CanaryValidation {
		args = append(args, "--check-access", "--check-filename", "RCLONE_TEST")
	}
	if effective.CompareMode != "" {
		args = append(args, "--compare", effective.CompareMode)
	}
	for _, pattern := range effective.ExcludePatterns {
		if strings.TrimSpace(pattern) != "" {
			args = append(args, "--exclude", pattern)
		}
	}
	if effective.RemoveEmptyDirs { args = append(args, "--remove-empty-dirs") }
	if effective.CreateEmptyDirs { args = append(args, "--create-empty-src-dirs") }

	// Tier 3 Custom Flags Escape Hatch
	if strings.TrimSpace(effective.CustomFlags) != "" {
		customTokens, err := shlex.Split(effective.CustomFlags)
		if err != nil {
			return nil, fmt.Errorf("invalid custom_flags syntax: %w", err)
		}

		for _, token := range customTokens {
			flagName := strings.Split(token, "=")[0]
			if reason, blocked := ProhibitedFlags[flagName]; blocked {
				return nil, fmt.Errorf("prohibited custom flag '%s': %s", flagName, reason)
			}
			args = append(args, token)
		}
	}

	return args, nil
}
```

---

## 7. UI / UX Integration for Inheritance

In the Profile Editor form, each flag input displays its current resolution status so users know whether they are using the **Global Default** or a **Profile Override**:

```
┌────────────────────────────────────────────────────────────────────────┐
│ Profile Settings: Personal Google Drive                                │
├────────────────────────────────────────────────────────────────────────┤
│ Max Deletion Threshold (%):                                            │
│ [ 10 ] %   (● Override active | Global Default: 50%)                  │
│ [ Reset to Global Default ]                                            │
│                                                                        │
│ File Comparison Mode:                                                  │
│ ( ) Inherit Global Default (Modification Time)                         │
│ (•) Override: Checksum (Strict Hash Comparison)                        │
│                                                                        │
│ Custom Flags Escape Hatch:                                             │
│ ┌────────────────────────────────────────────────────────────────────┐ │
│ │ --bwlimit 10M --transfers 4                                        │ │
│ └────────────────────────────────────────────────────────────────────┘ │
│ ℹ Appended directly to rclone bisync command line.                      │
└────────────────────────────────────────────────────────────────────────┘
```

---

## 8. Acceptance Criteria

- [ ] **Global Config:** Dashboard Settings tab allows configuring global application defaults (`MaxDelete`, `CompareMode`, `CanaryValidation`, etc.).
- [ ] **Inheritance Logic:** Profiles default to `nil` overrides and inherit global settings automatically upon resolution.
- [ ] **Profile Overrides:** Profile editor permits overriding global defaults per profile with visual indicator badges (`Override Active` vs. `Inherited`).
- [ ] **CLI Resolution Engine:** `BuildCLIArgs` resolves effective values (`Profile Override > Global Default > Execution Control`) and builds valid command slices.
- [ ] **Security Validation:** Custom flags pass `shlex` parsing and enforce the prohibited flag blocklist.
``GLOBAL_FLAG_OVERRIDES``