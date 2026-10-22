package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/core/state"
)

func TestDeletionService_MissingLog(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		BaseDir: filepath.Join(tmpDir, "nonexistent"),
	}
	svc := NewDeletionService(cfg)

	data := svc.ParsePendingDeletionsFromLog()
	if data.HasDeletions {
		t.Fatal("expected HasDeletions false when log is missing")
	}
	if data.TotalCount != 0 {
		t.Fatalf("expected TotalCount 0, got %d", data.TotalCount)
	}

	// Nil config safety
	nilSvc := NewDeletionService(nil)
	nilData := nilSvc.ParsePendingDeletionsFromLog()
	if nilData.HasDeletions {
		t.Fatal("expected false for nil config")
	}
}

func TestDeletionService_ParsePendingDeletionsFromLog(t *testing.T) {
	tmpDir := t.TempDir()
	daemonLog := filepath.Join(tmpDir, "rclone-daemon.log")

	// Create test daemon log with nested JSON and too many deletes
	nestedAbortMsg := map[string]interface{}{
		"msg": "ERROR : Bisync aborted. Too many deletes (3) on Path1",
	}
	nestedAbortJSON, _ := json.Marshal(nestedAbortMsg)

	logEntries := []string{
		`{"time":"2026-03-30T10:00:00Z","level":"info","msg":"Synching Path1"}`,
		`{"time":"2026-03-30T10:00:01Z","level":"info","msg":"File was deleted - doc1.pdf"}`,
		`{"time":"2026-03-30T10:00:02Z","level":"info","msg":"File was deleted - doc2.pdf"}`,
		`corrupted non-json line`,
		`{"time":"2026-03-30T10:00:03Z","level":"error","msg":"` + string(nestedAbortJSON) + `"}`,
	}

	content := ""
	for _, entry := range logEntries {
		content += entry + "\n"
	}
	_ = os.WriteFile(daemonLog, []byte(content), 0644)

	cfg := &config.Config{
		BaseDir: tmpDir,
	}
	svc := NewDeletionService(cfg)

	data := svc.ParsePendingDeletionsFromLog()
	if !data.HasDeletions {
		t.Fatal("expected pending deletions to be detected")
	}
	if data.TotalCount != 2 {
		t.Fatalf("expected 2 files, got %d", data.TotalCount)
	}
	if data.Files[0] != "doc1.pdf" || data.Files[1] != "doc2.pdf" {
		t.Fatalf("expected [doc1.pdf, doc2.pdf], got %v", data.Files)
	}
	if data.Source != "Google Drive (remote)" {
		t.Fatalf("expected remote source for Path1 abort, got %s", data.Source)
	}
}

func TestDeletionService_Path2AbortSource(t *testing.T) {
	tmpDir := t.TempDir()
	daemonLog := filepath.Join(tmpDir, "rclone-daemon.log")

	logEntries := []string{
		`{"time":"2026-03-30T10:00:00Z","level":"info","msg":"Synching Path1"}`,
		`{"time":"2026-03-30T10:00:01Z","level":"info","msg":"File was deleted - local_file.txt"}`,
		`{"time":"2026-03-30T10:00:02Z","level":"error","msg":"Bisync aborted. Too many deletes (1) on Path2"}`,
	}

	content := ""
	for _, entry := range logEntries {
		content += entry + "\n"
	}
	_ = os.WriteFile(daemonLog, []byte(content), 0644)

	cfg := &config.Config{
		BaseDir: tmpDir,
	}
	svc := NewDeletionService(cfg)

	data := svc.ParsePendingDeletionsFromLog()
	if !data.HasDeletions {
		t.Fatal("expected pending deletions to be detected")
	}
	if data.Source != "Local (~/gdrive)" {
		t.Fatalf("expected Local (~/gdrive) for Path2 abort, got %s", data.Source)
	}
	if data.TargetAction != "Will be deleted from Google Drive" {
		t.Fatalf("unexpected TargetAction: %s", data.TargetAction)
	}
}

func TestDeletionService_GetPendingDeletionsStructuredPrecedence(t *testing.T) {
	tmpDir := t.TempDir()
	structuredFile := filepath.Join(tmpDir, "pending-deletions.json")
	daemonLog := filepath.Join(tmpDir, "rclone-daemon.log")

	// Daemon log with 1 file
	_ = os.WriteFile(daemonLog, []byte(`{"time":"2026-03-30T10:00:00Z","msg":"File was deleted - from_log.txt\nBisync aborted. Too many deletes"}`+"\n"), 0644)

	// Structured state with 3 files
	structuredData := state.PendingDeletionsData{
		HasDeletions: true,
		TotalCount:   3,
		Source:       "Structured File",
		Files:        []string{"a.txt", "b.txt", "c.txt"},
	}
	_ = state.WriteJSONAtomic(structuredFile, structuredData)

	cfg := &config.Config{
		BaseDir:              tmpDir,
		PendingDeletionsFile: structuredFile,
	}
	svc := NewDeletionService(cfg)

	// GetPendingDeletions must prioritize structuredFile over daemon log
	result := svc.GetPendingDeletions()
	if result.Source != "Structured File" {
		t.Fatalf("expected structured file to take precedence, got source: %s", result.Source)
	}
	if len(result.Files) != 3 {
		t.Fatalf("expected 3 files from structured state, got %d", len(result.Files))
	}

	// Remove structured file -> falls back to daemon log
	_ = os.Remove(structuredFile)
	fallbackResult := svc.GetPendingDeletions()
	if len(fallbackResult.Files) != 1 || fallbackResult.Files[0] != "from_log.txt" {
		t.Fatalf("expected fallback to parse daemon log, got: %v", fallbackResult.Files)
	}
}
