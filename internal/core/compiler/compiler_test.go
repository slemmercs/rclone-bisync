package compiler

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCompileBasicParameters(t *testing.T) {
	args := []string{
		"--check-first",
		"--track-renames",
		"--remove-empty-dirs",
		"--resilient",
		"--fast-list",
		"--buffer-size", "64M",
		"--retries", "5",
		"--tpslimit", "10",
	}

	payload, err := CompileBisyncPayload(args, "/tmp/test_local", "gdrive:", false, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if payload["path1"] != "gdrive:" {
		t.Fatalf("expected gdrive:, got %v", payload["path1"])
	}
	if payload["checkFirst"] != true {
		t.Fatalf("expected checkFirst true")
	}
	cfg, ok := payload["_config"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing _config in payload")
	}
	if cfg["BufferSize"] != "64M" || cfg["Retries"] != 5 {
		t.Fatalf("invalid config options: %v", cfg)
	}
}

func TestCanaryGuard(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "canary_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	args := []string{"--check-access"}
	// Should fail when missing
	_, err = CompileBisyncPayload(args, tmpDir, "gdrive:", false, false, false)
	if err == nil {
		t.Fatalf("expected error for missing canary")
	}

	// Create canary
	canaryPath := filepath.Join(tmpDir, "RCLONE_TEST")
	_ = os.WriteFile(canaryPath, []byte("canary"), 0644)

	payload, err := CompileBisyncPayload(args, tmpDir, "gdrive:", false, false, false)
	if err != nil {
		t.Fatalf("unexpected error with canary present: %v", err)
	}
	if payload["checkAccess"] != true {
		t.Fatalf("expected checkAccess true")
	}
}
