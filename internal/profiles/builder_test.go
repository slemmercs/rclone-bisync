package profiles

import (
	"reflect"
	"strings"
	"testing"

	"github.com/slemmercs/rclone-bisync/internal/config"
)

func TestResolveFlags_Inheritance(t *testing.T) {
	globals := config.GlobalDefaults{
		MaxDelete:        50,
		CanaryValidation: true,
		ConflictResolve:  "newer",
		CompareMode:      "modtime",
		RemoveEmptyDirs:  true,
		CreateEmptyDirs:  true,
	}

	t.Run("inherits all globals when profile flags are nil", func(t *testing.T) {
		p := &Profile{
			ID: "prof_default",
		}
		res := ResolveFlags(p, globals, nil)

		if res.MaxDelete != 50 || !res.CanaryValidation || res.ConflictResolve != "newer" || res.CompareMode != "modtime" {
			t.Errorf("expected inherited values, got: %+v", res)
		}
		if res.DryRun || res.Resync || res.Force {
			t.Errorf("expected execution controls false by default")
		}
	})

	t.Run("applies profile overrides selectively", func(t *testing.T) {
		maxDel := 15
		cmpMode := "checksum"
		canary := false
		remEmpty := false

		p := &Profile{
			ID: "prof_overrides",
			Flags: ProfileFlags{
				MaxDelete:        &maxDel,
				CompareMode:      &cmpMode,
				CanaryValidation: &canary,
				RemoveEmptyDirs:  &remEmpty,
				CustomFlags:      "--bwlimit 10M",
			},
			ExcludePatterns: []string{"*.tmp", "node_modules/"},
		}

		res := ResolveFlags(p, globals, nil)

		if res.MaxDelete != 15 {
			t.Errorf("expected MaxDelete 15, got %d", res.MaxDelete)
		}
		if res.CompareMode != "checksum" {
			t.Errorf("expected CompareMode 'checksum', got %s", res.CompareMode)
		}
		if res.CanaryValidation {
			t.Errorf("expected CanaryValidation false")
		}
		if res.RemoveEmptyDirs {
			t.Errorf("expected RemoveEmptyDirs false")
		}
		if !res.CreateEmptyDirs {
			t.Errorf("expected CreateEmptyDirs inherited true")
		}
		if res.ConflictResolve != "newer" {
			t.Errorf("expected ConflictResolve inherited 'newer'")
		}
		if len(res.ExcludePatterns) != 2 {
			t.Errorf("expected 2 exclude patterns, got %d", len(res.ExcludePatterns))
		}
		if res.CustomFlags != "--bwlimit 10M" {
			t.Errorf("expected CustomFlags '--bwlimit 10M', got %s", res.CustomFlags)
		}
	})

	t.Run("applies execution overrides", func(t *testing.T) {
		p := &Profile{
			ID: "prof_exec",
		}
		dry := true
		resync := true
		force := true

		res := ResolveFlags(p, globals, &ExecutionOverrides{
			DryRun: &dry,
			Resync: &resync,
			Force:  &force,
		})

		if !res.DryRun || !res.Resync || !res.Force {
			t.Errorf("expected execution toggles true, got: dry=%v, resync=%v, force=%v", res.DryRun, res.Resync, res.Force)
		}
	})
}

func TestBuildCLIArgs(t *testing.T) {
	globals := config.DefaultGlobalDefaults()

	t.Run("builds standard arguments for inherited profile", func(t *testing.T) {
		p := &Profile{
			ID:         "prof_standard",
			Remote:     "drive:",
			RemotePath: "backup",
			LocalPath:  "/tmp/local",
		}

		args, err := BuildCLIArgs(p, globals, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := []string{
			"--conflict-resolve", "newer",
			"--max-delete", "50",
			"--check-access", "--check-filename", "RCLONE_TEST",
			"--compare", "modtime",
			"--remove-empty-dirs",
			"--create-empty-src-dirs",
		}

		if !reflect.DeepEqual(args, expected) {
			t.Errorf("got %v, expected %v", args, expected)
		}
	})

	t.Run("builds arguments with overrides, excludes, and custom flags", func(t *testing.T) {
		maxDel := 20
		cmp := "checksum"
		canary := true
		dry := true

		p := &Profile{
			ID:              "prof_custom",
			Remote:          "remote:",
			LocalPath:       "/tmp/local",
			ExcludePatterns: []string{"*.log"},
			Flags: ProfileFlags{
				MaxDelete:        &maxDel,
				CompareMode:      &cmp,
				CanaryValidation: &canary,
				CustomFlags:      `--bwlimit 20M --transfers 8`,
			},
		}

		args, err := BuildCLIArgs(p, globals, &ExecutionOverrides{DryRun: &dry})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := []string{
			"--dry-run",
			"--conflict-resolve", "newer",
			"--max-delete", "20",
			"--check-access", "--check-filename", "RCLONE_TEST",
			"--compare", "checksum",
			"--exclude", "*.log",
			"--remove-empty-dirs",
			"--create-empty-src-dirs",
			"--bwlimit", "20M",
			"--transfers", "8",
		}

		if !reflect.DeepEqual(args, expected) {
			t.Errorf("got %v, expected %v", args, expected)
		}
	})

	t.Run("fails when custom flags contain prohibited argument", func(t *testing.T) {
		p := &Profile{
			ID: "prof_evil",
			Flags: ProfileFlags{
				CustomFlags: "--config /etc/evil.conf",
			},
		}

		_, err := BuildCLIArgs(p, globals, nil)
		if err == nil {
			t.Fatalf("expected error for prohibited custom flag")
		}
		if !strings.Contains(err.Error(), "prohibited custom flag '--config'") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("compiles provider options for Tier 1 remotes alongside custom flags", func(t *testing.T) {
		p := &Profile{
			ID: "prof_tier1_gdrive",
			Flags: ProfileFlags{
				ProviderOptions: map[string]string{
					"drive_skip_shortcuts":    "true",
					"drive_use_trash":         "true",
					"drive_skip_gdocs":        "true",
					"drive_acknowledge_abuse": "true",
					"drive_export_formats":    "docx,xlsx,pptx",
				},
				CustomFlags: "--fast-list",
			},
		}

		args, err := BuildCLIArgs(p, globals, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		argsStr := strings.Join(args, " ")
		if !strings.Contains(argsStr, "--drive-skip-shortcuts") {
			t.Errorf("missing --drive-skip-shortcuts in args: %v", args)
		}
		if !strings.Contains(argsStr, "--drive-use-trash") {
			t.Errorf("missing --drive-use-trash in args: %v", args)
		}
		if !strings.Contains(argsStr, "--drive-skip-gdocs") {
			t.Errorf("missing --drive-skip-gdocs in args: %v", args)
		}
		if !strings.Contains(argsStr, "--drive-acknowledge-abuse") {
			t.Errorf("missing --drive-acknowledge-abuse in args: %v", args)
		}
		if !strings.Contains(argsStr, "--drive-export-formats docx,xlsx,pptx") {
			t.Errorf("missing --drive-export-formats in args: %v", args)
		}
		if !strings.Contains(argsStr, "--fast-list") {
			t.Errorf("missing custom flag --fast-list in args: %v", args)
		}
	})

	t.Run("compiles OneDrive and S3 provider options", func(t *testing.T) {
		pOD := &Profile{
			ID: "prof_od",
			Flags: ProfileFlags{
				ProviderOptions: map[string]string{
					"onedrive_no_versions": "true",
					"onedrive_chunk_size":  "20M",
				},
			},
		}
		argsOD, err := BuildCLIArgs(pOD, globals, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		argsStrOD := strings.Join(argsOD, " ")
		if !strings.Contains(argsStrOD, "--onedrive-no-versions") || !strings.Contains(argsStrOD, "--onedrive-chunk-size 20M") {
			t.Errorf("missing onedrive flags in args: %v", argsOD)
		}

		pS3 := &Profile{
			ID: "prof_s3",
			Flags: ProfileFlags{
				ProviderOptions: map[string]string{
					"s3_chunk_size":    "32M",
					"s3_storage_class": "STANDARD_IA",
				},
			},
		}
		argsS3, err := BuildCLIArgs(pS3, globals, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		argsStrS3 := strings.Join(argsS3, " ")
		if !strings.Contains(argsStrS3, "--s3-chunk-size 32M") || !strings.Contains(argsStrS3, "--s3-storage-class STANDARD_IA") {
			t.Errorf("missing s3 flags in args: %v", argsS3)
		}
	})
}
