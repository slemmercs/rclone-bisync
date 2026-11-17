package audit

import (
	"path/filepath"
	"testing"
)

func TestRunCLI_SeedAndVerify(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_audit.db")
	localPath := filepath.Join(tempDir, "local")

	// Seed records
	err := RunCLI("seed", dbPath, localPath)
	if err != nil {
		t.Fatalf("RunCLI(seed) unexpected error: %v", err)
	}

	// Verify ledger records
	err = RunCLI("verify", dbPath, localPath)
	if err != nil {
		t.Fatalf("RunCLI(verify) unexpected error: %v", err)
	}

	// Invalid action
	err = RunCLI("invalid_action", dbPath, localPath)
	if err == nil {
		t.Fatalf("expected error for invalid action, got nil")
	}
}
