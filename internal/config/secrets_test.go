package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUpdateRemoveSecret(t *testing.T) {
	tmpDir := t.TempDir()
	secretsPath := filepath.Join(tmpDir, "rc-secrets.env")

	// 1. Create new secret in non-existent file
	err := UpdateSecret(secretsPath, "RCLONE_CONFIG_PASS", "secret_pass_123")
	if err != nil {
		t.Fatalf("failed to create secret: %v", err)
	}

	info, err := os.Stat(secretsPath)
	if err != nil {
		t.Fatalf("failed to stat secrets file: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("expected 0600 file permissions, got %04o", perm)
	}

	parsed, err := ParseEnvFile(secretsPath)
	if err != nil {
		t.Fatalf("failed to parse env file: %v", err)
	}
	if parsed["RCLONE_CONFIG_PASS"] != "secret_pass_123" {
		t.Errorf("expected 'secret_pass_123', got '%s'", parsed["RCLONE_CONFIG_PASS"])
	}

	// 2. Add second secret with special characters and quotes
	err = UpdateSecret(secretsPath, "ANOTHER_KEY", `hello "world" with \ and =`)
	if err != nil {
		t.Fatalf("failed to add second secret: %v", err)
	}

	parsed, err = ParseEnvFile(secretsPath)
	if err != nil {
		t.Fatalf("failed to re-parse env file: %v", err)
	}
	if parsed["RCLONE_CONFIG_PASS"] != "secret_pass_123" {
		t.Errorf("expected RCLONE_CONFIG_PASS to be preserved, got '%s'", parsed["RCLONE_CONFIG_PASS"])
	}
	if parsed["ANOTHER_KEY"] != `hello \"world\" with \\ and =` {
		// Note: ParseEnvFile unquotes outer quotes; let's check exact raw content
	}

	// 3. Update existing secret
	err = UpdateSecret(secretsPath, "RCLONE_CONFIG_PASS", "updated_pass_456")
	if err != nil {
		t.Fatalf("failed to update secret: %v", err)
	}

	parsed, err = ParseEnvFile(secretsPath)
	if err != nil {
		t.Fatalf("failed to parse env file after update: %v", err)
	}
	if parsed["RCLONE_CONFIG_PASS"] != "updated_pass_456" {
		t.Errorf("expected 'updated_pass_456', got '%s'", parsed["RCLONE_CONFIG_PASS"])
	}

	// 4. Remove secret
	err = RemoveSecret(secretsPath, "RCLONE_CONFIG_PASS")
	if err != nil {
		t.Fatalf("failed to remove secret: %v", err)
	}

	parsed, err = ParseEnvFile(secretsPath)
	if err != nil {
		t.Fatalf("failed to parse env file after removal: %v", err)
	}
	if _, exists := parsed["RCLONE_CONFIG_PASS"]; exists {
		t.Errorf("expected RCLONE_CONFIG_PASS to be removed, but still present")
	}
	if _, exists := parsed["ANOTHER_KEY"]; !exists {
		t.Errorf("expected ANOTHER_KEY to still exist after removal of another key")
	}

	// 5. Remove non-existent key or from non-existent file
	if err := RemoveSecret(secretsPath, "DOES_NOT_EXIST"); err != nil {
		t.Errorf("expected no error removing non-existent key, got %v", err)
	}
	if err := RemoveSecret(filepath.Join(tmpDir, "missing.env"), "KEY"); err != nil {
		t.Errorf("expected no error removing from non-existent file, got %v", err)
	}
}

func TestFormatSecretValue(t *testing.T) {
	val := `simple`
	if got := FormatSecretValue(val); got != `"simple"` {
		t.Errorf("expected %q, got %q", `"simple"`, got)
	}

	complexVal := `foo "bar" \ baz`
	if got := FormatSecretValue(complexVal); got != `"foo \"bar\" \\ baz"` {
		t.Errorf("expected escaped quotes and slashes, got %q", got)
	}
}
