package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnvFile(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, "test.env")
	content := "TEST_KEY=test_value\n# Comment\nQUOTED_VAL=\"quoted\"\nEMPTY=\n"
	if err := os.WriteFile(envPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test env: %v", err)
	}

	if err := LoadEnvFile(envPath); err != nil {
		t.Fatalf("LoadEnvFile failed: %v", err)
	}

	if got := os.Getenv("TEST_KEY"); got != "test_value" {
		t.Errorf("expected test_value, got %s", got)
	}
	if got := os.Getenv("QUOTED_VAL"); got != "quoted" {
		t.Errorf("expected quoted, got %s", got)
	}
}

func TestLoadConfigMissingRCPass(t *testing.T) {
	tmpDir := t.TempDir()
	// Clear RC_PASS from environment
	orig := os.Getenv("RC_PASS")
	os.Unsetenv("RC_PASS")
	defer func() {
		if orig != "" {
			os.Setenv("RC_PASS", orig)
		}
	}()

	_, err := LoadConfig(tmpDir)
	if err == nil {
		t.Fatal("expected error when RC_PASS is missing, got nil")
	}
}

func TestLoadConfigSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	orig := os.Getenv("RC_PASS")
	os.Setenv("RC_PASS", "supersecret")
	defer func() {
		if orig != "" {
			os.Setenv("RC_PASS", orig)
		} else {
			os.Unsetenv("RC_PASS")
		}
	}()

	cfg, err := LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("expected successful config load, got: %v", err)
	}

	if cfg.RCPass != "supersecret" {
		t.Errorf("expected RCPass supersecret, got %s", cfg.RCPass)
	}
	if cfg.RCPort != 5572 {
		t.Errorf("expected default port 5572, got %d", cfg.RCPort)
	}
	if len(cfg.RcloneCommonArgs) == 0 {
		t.Errorf("expected non-empty RcloneCommonArgs")
	}
}

func TestConfigHelpersAndEnvExpansion(t *testing.T) {
	// Test str2bool
	if !str2bool("yes", false) || !str2bool("1", false) || !str2bool("true", false) {
		t.Fatal("expected true values")
	}
	if str2bool("false", true) || str2bool("0", true) || str2bool("no", true) || str2bool("off", true) {
		t.Fatal("expected false values")
	}
	if str2bool("invalid", true) != true {
		t.Fatal("expected fallback value for invalid string")
	}

	// Test getEnv helpers
	os.Setenv("TEST_INT_KEY", "42")
	os.Setenv("TEST_FLOAT_KEY", "3.14")
	os.Setenv("TEST_BOOL_KEY", "true")
	defer func() {
		os.Unsetenv("TEST_INT_KEY")
		os.Unsetenv("TEST_FLOAT_KEY")
		os.Unsetenv("TEST_BOOL_KEY")
	}()

	if getEnvInt("TEST_INT_KEY", 0) != 42 || getEnvInt("MISSING_INT", 99) != 99 {
		t.Fatal("getEnvInt mismatch")
	}
	if getEnvFloat("TEST_FLOAT_KEY", 0) != 3.14 || getEnvFloat("MISSING_FLOAT", 1.5) != 1.5 {
		t.Fatal("getEnvFloat mismatch")
	}
	if !getEnvBool("TEST_BOOL_KEY", false) || getEnvBool("MISSING_BOOL", true) != true {
		t.Fatal("getEnvBool mismatch")
	}

	// Test splitArgs
	args := splitArgs("  --arg1   --arg2=val   ")
	if len(args) != 2 || args[0] != "--arg1" || args[1] != "--arg2=val" {
		t.Fatalf("unexpected splitArgs result: %v", args)
	}

	// Test FilesToCleanup
	tmpDir := t.TempDir()
	os.Setenv("RC_PASS", "testsecret")
	os.Setenv("LOCAL_PATH", "~/my_gdrive")
	os.Setenv("FILES_TO_CLEANUP", "*.tmp, conflict*")
	defer func() {
		os.Unsetenv("RC_PASS")
		os.Unsetenv("LOCAL_PATH")
		os.Unsetenv("FILES_TO_CLEANUP")
	}()

	cfg, err := LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if len(cfg.FilesToCleanup) != 2 || cfg.FilesToCleanup[0] != "*.tmp" || cfg.FilesToCleanup[1] != "conflict*" {
		t.Fatalf("unexpected FilesToCleanup: %v", cfg.FilesToCleanup)
	}
}

