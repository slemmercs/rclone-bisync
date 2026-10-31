package compiler

import (
	"os"
	"path/filepath"
	"strings"
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

func TestCompileAdvancedParameters(t *testing.T) {
	args := []string{
		"--use-json-log",
		"--color", "NEVER",
		"--drive-skip-gdocs",
		"--drive-use-trash",
		"--drive-export-formats", "docx,xlsx",
		"--conflict-resolve", "newer",
		"--conflict-loser", "num",
		"--tpslimit", "15.5",
		"--tpslimit-burst", "20",
		"--recover",
	}

	payload, err := CompileBisyncPayload(args, "/tmp/local", "gdrive:", true, true, true)
	if err != nil {
		t.Fatalf("unexpected compile error: %v", err)
	}

	if payload["resync"] != true || payload["force"] != true || payload["dryRun"] != true {
		t.Fatalf("expected flags to be true: %v", payload)
	}

	// Path1 should have backend options appended
	path1, ok := payload["path1"].(string)
	if !ok || path1 == "gdrive:" {
		t.Fatalf("expected backend options attached to path1, got: %v", path1)
	}

	cfg, ok := payload["_config"].(map[string]interface{})
	if !ok {
		t.Fatal("missing _config in payload")
	}
	if cfg["TPSLimit"] != 15.5 || cfg["TPSLimitBurst"] != 20 {
		t.Fatalf("expected TPSLimit 15.5 and TPSLimitBurst 20, got: %v", cfg)
	}
}
func TestCompileTier1ProviderOptions(t *testing.T) {
	// Test OneDrive with subpath
	oneDriveArgs := []string{
		"--onedrive-no-versions",
		"--onedrive-chunk-size", "10M",
	}
	payloadOD, err := CompileBisyncPayload(oneDriveArgs, "/tmp/local", "onedrive:Documents/Work", false, false, false)
	if err != nil {
		t.Fatalf("unexpected error compiling OneDrive: %v", err)
	}
	pathOD, _ := payloadOD["path1"].(string)
	if !strings.HasPrefix(pathOD, "onedrive,") || !strings.HasSuffix(pathOD, ":Documents/Work") {
		t.Fatalf("expected backend options on onedrive path, got: %s", pathOD)
	}
	if !strings.Contains(pathOD, "no_versions=true") || !strings.Contains(pathOD, "chunk_size=10M") {
		t.Fatalf("missing onedrive options in path1: %s", pathOD)
	}

	// Test S3
	s3Args := []string{
		"--s3-chunk-size", "16M",
		"--s3-storage-class", "STANDARD_IA",
	}
	payloadS3, err := CompileBisyncPayload(s3Args, "/tmp/local", "mys3:backup", false, false, false)
	if err != nil {
		t.Fatalf("unexpected error compiling S3: %v", err)
	}
	pathS3, _ := payloadS3["path1"].(string)
	if !strings.Contains(pathS3, "storage_class=STANDARD_IA") || !strings.Contains(pathS3, "chunk_size=16M") {
		t.Fatalf("missing s3 options in path1: %s", pathS3)
	}

	// Test Dropbox
	dbArgs := []string{
		"--dropbox-chunk-size", "48M",
	}
	payloadDB, err := CompileBisyncPayload(dbArgs, "/tmp/local", "dropbox:", false, false, false)
	if err != nil {
		t.Fatalf("unexpected error compiling Dropbox: %v", err)
	}
	pathDB, _ := payloadDB["path1"].(string)
	if pathDB != "dropbox,chunk_size=48M:" {
		t.Fatalf("expected dropbox,chunk_size=48M:, got: %s", pathDB)
	}
}

func TestCompileExcludeRules(t *testing.T) {
	args := []string{
		"--exclude", ".git/**",
		"--fast-list",
		"--exclude", "*.tmp",
		"--exclude", ".DS_Store",
	}

	payload, err := CompileBisyncPayload(args, "/tmp/local", "gdrive:", false, false, false)
	if err != nil {
		t.Fatalf("unexpected compile error: %v", err)
	}

	filterMap, ok := payload["_filter"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected _filter map in payload, got: %v", payload["_filter"])
	}

	rules, ok := filterMap["ExcludeRule"].([]string)
	if !ok {
		t.Fatalf("expected []string ExcludeRule, got: %T", filterMap["ExcludeRule"])
	}

	if len(rules) != 3 {
		t.Fatalf("expected 3 exclude rules, got %d: %v", len(rules), rules)
	}
	if rules[0] != ".git/**" || rules[1] != "*.tmp" || rules[2] != ".DS_Store" {
		t.Fatalf("unexpected rules content: %v", rules)
	}
}

func TestCompileVerboseFlags(t *testing.T) {
	// Test --verbose
	args1 := []string{"--verbose"}
	payload1, err := CompileBisyncPayload(args1, "/tmp/local", "gdrive:", false, false, false)
	if err != nil {
		t.Fatalf("unexpected error compiling --verbose: %v", err)
	}
	cfg1, ok := payload1["_config"].(map[string]interface{})
	if !ok || cfg1["Verbose"] != 1 {
		t.Fatalf("expected Verbose: 1 in _config, got: %v", cfg1)
	}

	// Test -v
	args2 := []string{"-v"}
	payload2, err := CompileBisyncPayload(args2, "/tmp/local", "gdrive:", false, false, false)
	if err != nil {
		t.Fatalf("unexpected error compiling -v: %v", err)
	}
	cfg2, ok := payload2["_config"].(map[string]interface{})
	if !ok || cfg2["Verbose"] != 1 {
		t.Fatalf("expected Verbose: 1 in _config, got: %v", cfg2)
	}
}
