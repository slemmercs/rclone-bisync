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
}
