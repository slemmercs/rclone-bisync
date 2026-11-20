package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigrateEnvFile_AlreadyUpToDate(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	templatePath := filepath.Join(tmpDir, "template.env")
	deployedPath := filepath.Join(tmpDir, "deployed.env")

	templateContent := `# Template config
KEY_A=value_a
KEY_B=value_b
`
	deployedContent := `# Deployed config
KEY_A=custom_a
KEY_B=custom_b
`
	if err := os.WriteFile(templatePath, []byte(templateContent), 0640); err != nil {
		t.Fatalf("failed to write template: %v", err)
	}
	if err := os.WriteFile(deployedPath, []byte(deployedContent), 0640); err != nil {
		t.Fatalf("failed to write deployed: %v", err)
	}

	// Act
	res, err := MigrateEnvFile(deployedPath, templatePath, false)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.AlreadyUpToDate {
		t.Errorf("expected AlreadyUpToDate to be true, got false")
	}
	if res.AddedCount != 0 {
		t.Errorf("expected AddedCount = 0, got %d", res.AddedCount)
	}
	if res.BackupPath != "" {
		t.Errorf("expected no backup when already up to date, got %s", res.BackupPath)
	}

	content, _ := os.ReadFile(deployedPath)
	if string(content) != deployedContent {
		t.Errorf("deployed content was modified: got %s", string(content))
	}
}

func TestMigrateEnvFile_DryRun(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	templatePath := filepath.Join(tmpDir, "template.env")
	deployedPath := filepath.Join(tmpDir, "deployed.env")

	templateContent := `# Header
KEY_A=value_a
# Concurrency tuning
KEY_B=value_b
# Telemetry
KEY_C=value_c
`
	deployedContent := `KEY_A=custom_a
`
	if err := os.WriteFile(templatePath, []byte(templateContent), 0640); err != nil {
		t.Fatalf("failed to write template: %v", err)
	}
	if err := os.WriteFile(deployedPath, []byte(deployedContent), 0640); err != nil {
		t.Fatalf("failed to write deployed: %v", err)
	}

	// Act
	res, err := MigrateEnvFile(deployedPath, templatePath, true)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.DryRun {
		t.Errorf("expected DryRun = true")
	}
	if res.AlreadyUpToDate {
		t.Errorf("expected AlreadyUpToDate = false")
	}
	if res.AddedCount != 2 {
		t.Errorf("expected AddedCount = 2, got %d", res.AddedCount)
	}
	if len(res.MissingKeys) != 2 || res.MissingKeys[0] != "KEY_B" || res.MissingKeys[1] != "KEY_C" {
		t.Errorf("expected missing keys [KEY_B, KEY_C], got %v", res.MissingKeys)
	}
	if res.BackupPath != "" {
		t.Errorf("expected no backup created on dry-run, got %s", res.BackupPath)
	}

	content, _ := os.ReadFile(deployedPath)
	if string(content) != deployedContent {
		t.Errorf("deployed file was modified during dry run: %s", string(content))
	}
}

func TestMigrateEnvFile_AppendsMissingKeysAndCreatesBackup(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	templatePath := filepath.Join(tmpDir, "template.env")
	deployedPath := filepath.Join(tmpDir, "deployed.env")

	templateContent := `# Header comment
KEY_A=default_a

# --- Important Section ---
# Explanation for B
KEY_B=default_b

# Explanation for C
KEY_C="default c"
`
	deployedContent := `# My personal settings
KEY_A=my_custom_a
`
	if err := os.WriteFile(templatePath, []byte(templateContent), 0640); err != nil {
		t.Fatalf("failed to write template: %v", err)
	}
	if err := os.WriteFile(deployedPath, []byte(deployedContent), 0640); err != nil {
		t.Fatalf("failed to write deployed: %v", err)
	}

	// Act
	res, err := MigrateEnvFile(deployedPath, templatePath, false)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.AddedCount != 2 {
		t.Errorf("expected AddedCount = 2, got %d", res.AddedCount)
	}
	if res.BackupPath == "" {
		t.Fatal("expected BackupPath to be set")
	}

	backupBytes, err := os.ReadFile(res.BackupPath)
	if err != nil {
		t.Fatalf("failed to read backup file %s: %v", res.BackupPath, err)
	}
	if string(backupBytes) != deployedContent {
		t.Errorf("backup content mismatch: got %q, want %q", string(backupBytes), deployedContent)
	}

	parsed, err := ParseEnvFile(deployedPath)
	if err != nil {
		t.Fatalf("failed to parse migrated env file: %v", err)
	}

	if parsed["KEY_A"] != "my_custom_a" {
		t.Errorf("KEY_A custom value overwritten! got %q, want my_custom_a", parsed["KEY_A"])
	}
	if parsed["KEY_B"] != "default_b" {
		t.Errorf("KEY_B default missing or wrong: got %q", parsed["KEY_B"])
	}
	if parsed["KEY_C"] != "default c" {
		t.Errorf("KEY_C default missing or wrong: got %q", parsed["KEY_C"])
	}

	content, _ := os.ReadFile(deployedPath)
	contentStr := string(content)
	if !strings.Contains(contentStr, "# Explanation for B") {
		t.Errorf("expected comment for KEY_B to be present in migrated file:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "Migrated Configuration Defaults") {
		t.Errorf("expected migration header banner in migrated file:\n%s", contentStr)
	}
}

func TestMigrateEnvFile_InitializesMissingFile(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	templatePath := filepath.Join(tmpDir, "template.env")
	deployedPath := filepath.Join(tmpDir, "deployed.env")

	templateContent := `# Canonical template
BOOTSTRAP_LOCAL=~/sync
BOOTSTRAP_REMOTE=remote:
`
	if err := os.WriteFile(templatePath, []byte(templateContent), 0640); err != nil {
		t.Fatalf("failed to write template: %v", err)
	}

	// Act
	res, err := MigrateEnvFile(deployedPath, templatePath, false)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.AddedCount != 2 {
		t.Errorf("expected AddedCount = 2, got %d", res.AddedCount)
	}
	if res.BackupPath != "" {
		t.Errorf("expected no backup when initializing fresh file, got %s", res.BackupPath)
	}

	content, err := os.ReadFile(deployedPath)
	if err != nil {
		t.Fatalf("failed to read newly initialized file: %v", err)
	}
	if string(content) != templateContent {
		t.Errorf("initialized file does not match template: got %s", string(content))
	}
}

func TestMigrateEnvFile_MissingTemplateReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := MigrateEnvFile(filepath.Join(tmpDir, "deployed.env"), filepath.Join(tmpDir, "nonexistent.env"), false)
	if err == nil {
		t.Error("expected error when template does not exist, got nil")
	}
}
