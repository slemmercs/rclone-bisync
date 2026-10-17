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
