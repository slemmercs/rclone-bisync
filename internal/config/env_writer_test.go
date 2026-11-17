package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormatEnvValue(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"true", "true"},
		{"false", "false"},
		{"1200", "1200"},
		{"1.5", "1.5"},
		{"size,modtime", `"size,modtime"`},
		{`hello "world"`, `"hello \"world\""`},
		{`"already quoted"`, `"already quoted"`},
		{"--tpslimit 10", `"--tpslimit 10"`},
	}

	for _, c := range cases {
		got := FormatEnvValue(c.input)
		if got != c.expected {
			t.Errorf("FormatEnvValue(%q) = %q; want %q", c.input, got, c.expected)
		}
	}
}

func TestUpdateEnvKeys_ModifyAndAppend(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, "bisync.env")

	initialContent := `# Sample header
LOCAL_PATH=~/old_path
JOB_TIMEOUT_SECONDS=1800
# Comment here
DEBUG_MODE=false
`
	if err := os.WriteFile(envPath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("failed to write initial file: %v", err)
	}

	updates := map[string]string{
		"LOCAL_PATH":          "~/new_path",
		"JOB_TIMEOUT_SECONDS": "3600",
		"GLOBAL_MAX_DELETE":   "30",
	}

	if err := UpdateEnvKeys(envPath, updates); err != nil {
		t.Fatalf("UpdateEnvKeys failed: %v", err)
	}

	parsed, err := ParseEnvFile(envPath)
	if err != nil {
		t.Fatalf("ParseEnvFile failed: %v", err)
	}

	if parsed["LOCAL_PATH"] != "~/new_path" {
		t.Errorf("LOCAL_PATH = %q; want ~/new_path", parsed["LOCAL_PATH"])
	}
	if parsed["JOB_TIMEOUT_SECONDS"] != "3600" {
		t.Errorf("JOB_TIMEOUT_SECONDS = %q; want 3600", parsed["JOB_TIMEOUT_SECONDS"])
	}
	if parsed["DEBUG_MODE"] != "false" {
		t.Errorf("DEBUG_MODE = %q; want false", parsed["DEBUG_MODE"])
	}
	if parsed["GLOBAL_MAX_DELETE"] != "30" {
		t.Errorf("GLOBAL_MAX_DELETE = %q; want 30", parsed["GLOBAL_MAX_DELETE"])
	}

	// Verify comments are preserved
	content, _ := os.ReadFile(envPath)
	if !strings.Contains(string(content), "# Comment here") {
		t.Errorf("expected comment '# Comment here' to be preserved in file content: %s", string(content))
	}
}

func TestUpdateEnvKeys_CreateNew(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, "new.env")

	updates := map[string]string{
		"JOB_TIMEOUT_SECONDS": "2400",
		"DEBUG_MODE":          "true",
	}

	if err := UpdateEnvKeys(envPath, updates); err != nil {
		t.Fatalf("UpdateEnvKeys on new file failed: %v", err)
	}

	parsed, err := ParseEnvFile(envPath)
	if err != nil {
		t.Fatalf("ParseEnvFile failed: %v", err)
	}

	if parsed["JOB_TIMEOUT_SECONDS"] != "2400" {
		t.Errorf("JOB_TIMEOUT_SECONDS = %q; want 2400", parsed["JOB_TIMEOUT_SECONDS"])
	}
	if parsed["DEBUG_MODE"] != "true" {
		t.Errorf("DEBUG_MODE = %q; want true", parsed["DEBUG_MODE"])
	}
}
