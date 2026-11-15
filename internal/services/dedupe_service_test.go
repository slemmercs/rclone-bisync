package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/models"
)

func TestDedupeBatch_Success(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")
	file3 := filepath.Join(tmpDir, "file3.txt")

	_ = os.WriteFile(file1, []byte("original"), 0600)
	_ = os.WriteFile(file2, []byte("duplicate to delete"), 0600)
	_ = os.WriteFile(file3, []byte("duplicate to rename"), 0600)

	cfg := &config.Config{LocalPath: tmpDir}
	expSvc := NewExplorerService(cfg, nil, nil)
	dedupeSvc := NewDedupeService(cfg, expSvc, nil, nil)

	req := models.DedupeBatchApplyRequest{
		Remote: "local",
		Path:   "",
		Actions: []models.DedupeStageAction{
			{Path: "file1.txt", Action: "keep"},
			{Path: "file2.txt", Action: "delete"},
			{Path: "file3.txt", Action: "rename", NewName: "file3_preserved.txt"},
		},
	}

	// Act
	resp, err := dedupeSvc.ExecuteDedupeBatch(context.Background(), req, tmpDir)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "success" {
		t.Errorf("expected success status, got: %s", resp.Status)
	}
	if resp.SuccessCount != 3 {
		t.Errorf("expected 3 successes, got: %d", resp.SuccessCount)
	}
	if resp.FailureCount != 0 {
		t.Errorf("expected 0 failures, got: %d (errors: %v)", resp.FailureCount, resp.Errors)
	}

	// Check file1 exists
	if _, err := os.Stat(file1); err != nil {
		t.Errorf("expected file1.txt to remain: %v", err)
	}
	// Check file2 deleted
	if _, err := os.Stat(file2); !os.IsNotExist(err) {
		t.Errorf("expected file2.txt to be deleted")
	}
	// Check file3 renamed
	renamedFile := filepath.Join(tmpDir, "file3_preserved.txt")
	if _, err := os.Stat(renamedFile); err != nil {
		t.Errorf("expected file3_preserved.txt to exist: %v", err)
	}
}

func TestDedupeBatch_CanaryProtection(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	canaryPath := filepath.Join(tmpDir, CanaryFilename)
	_ = os.WriteFile(canaryPath, []byte("canary"), 0600)

	cfg := &config.Config{LocalPath: tmpDir}
	expSvc := NewExplorerService(cfg, nil, nil)
	dedupeSvc := NewDedupeService(cfg, expSvc, nil, nil)

	req := models.DedupeBatchApplyRequest{
		Remote: "local",
		Path:   "",
		Actions: []models.DedupeStageAction{
			{Path: CanaryFilename, Action: "delete"},
			{Path: CanaryFilename, Action: "rename", NewName: "canary_renamed.txt"},
		},
	}

	// Act
	resp, err := dedupeSvc.ExecuteDedupeBatch(context.Background(), req, tmpDir)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "error" {
		t.Errorf("expected error status for canary tampering, got: %s", resp.Status)
	}
	if resp.FailureCount != 2 {
		t.Errorf("expected 2 failures, got: %d", resp.FailureCount)
	}
	// Canary must still exist
	if _, err := os.Stat(canaryPath); err != nil {
		t.Fatalf("canary file was corrupted or deleted: %v", err)
	}
}

func TestDedupeBatch_PathTraversal(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	cfg := &config.Config{LocalPath: tmpDir}
	expSvc := NewExplorerService(cfg, nil, nil)
	dedupeSvc := NewDedupeService(cfg, expSvc, nil, nil)

	req := models.DedupeBatchApplyRequest{
		Remote: "local",
		Path:   "",
		Actions: []models.DedupeStageAction{
			{Path: "../outside.txt", Action: "delete"},
			{Path: "safe.txt", Action: "rename", NewName: "../outside.txt"},
		},
	}

	// Act
	resp, err := dedupeSvc.ExecuteDedupeBatch(context.Background(), req, tmpDir)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.FailureCount != 2 {
		t.Errorf("expected 2 failures for path traversal, got: %d", resp.FailureCount)
	}
}
