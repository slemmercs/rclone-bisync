package services

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/slemmercs/rclone-bisync/internal/audit"
	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/core/logger"
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

func TestDedupeBatch_SubdirectoryPathConcatenation(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "Google Photos", "2015")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	file1 := filepath.Join(subDir, "SAM_0805-01.jpeg")
	file2 := filepath.Join(subDir, "SAM_0805-02.jpeg")
	_ = os.WriteFile(file1, []byte("photo1"), 0600)
	_ = os.WriteFile(file2, []byte("photo2"), 0600)

	cfg := &config.Config{LocalPath: tmpDir}
	expSvc := NewExplorerService(cfg, nil, nil)
	dedupeSvc := NewDedupeService(cfg, expSvc, nil, nil)

	req := models.DedupeBatchApplyRequest{
		Remote:        "local",
		Path:          "Google Photos/2015",
		TriggerResync: true,
		Actions: []models.DedupeStageAction{
			{Path: "SAM_0805-01.jpeg", Action: "keep"},
			{Path: "SAM_0805-02.jpeg", Action: "rename", NewName: "SAM_0805-02-copy2.jpeg"},
		},
	}

	// Act
	resp, err := dedupeSvc.ExecuteDedupeBatch(context.Background(), req, tmpDir)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "success" {
		t.Errorf("expected success status, got: %s (error: %s)", resp.Status, resp.Error)
	}
	if resp.SuccessCount != 2 {
		t.Errorf("expected 2 successes, got: %d", resp.SuccessCount)
	}
	if resp.FailureCount != 0 {
		t.Errorf("expected 0 failures, got: %d (errors: %v)", resp.FailureCount, resp.Errors)
	}

	// Verify the renamed file exists in the subdirectory
	renamedFile := filepath.Join(subDir, "SAM_0805-02-copy2.jpeg")
	if _, err := os.Stat(renamedFile); err != nil {
		t.Errorf("expected renamed file %s to exist: %v", renamedFile, err)
	}
}

func TestDedupeBatch_ResyncSkippedOnAllFailures(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	cfg := &config.Config{LocalPath: tmpDir}
	expSvc := NewExplorerService(cfg, nil, nil)
	dedupeSvc := NewDedupeService(cfg, expSvc, nil, nil)

	req := models.DedupeBatchApplyRequest{
		Remote:        "local",
		Path:          "nonexistent",
		TriggerResync: true,
		ProfileID:     "test-profile",
		Actions: []models.DedupeStageAction{
			{Path: "missing_file.txt", Action: "delete"},
		},
	}

	// Act
	resp, err := dedupeSvc.ExecuteDedupeBatch(context.Background(), req, tmpDir)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "error" {
		t.Errorf("expected error status, got: %s", resp.Status)
	}
	if resp.ResyncQueued {
		t.Errorf("expected ResyncQueued to be false when all actions fail")
	}
	if resp.ResyncMessage != "Resync skipped: no dedupe actions succeeded" {
		t.Errorf("unexpected resync message: %s", resp.ResyncMessage)
	}
	if len(resp.Errors) == 0 {
		t.Errorf("expected detailed error in resp.Errors")
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

func TestDedupeBatch_AuditAndLogging(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "audit.db")
	logPath := filepath.Join(tmpDir, "bisync.log")

	db, err := audit.OpenDB(dbPath)
	if err != nil {
		t.Fatalf("failed to open audit DB: %v", err)
	}
	defer db.Close()

	logSink, err := logger.New(logPath, false)
	if err != nil {
		t.Fatalf("failed to init logger: %v", err)
	}
	defer logSink.Close()

	auditSvc := NewAuditService(db, logSink)
	cfg := &config.Config{LocalPath: tmpDir}
	expSvc := NewExplorerService(cfg, nil, nil)
	dedupeSvc := NewDedupeService(cfg, expSvc, auditSvc, nil)

	file1 := filepath.Join(tmpDir, "keep_me.txt")
	file2 := filepath.Join(tmpDir, "del_me.txt")
	file3 := filepath.Join(tmpDir, "ren_me.txt")

	_ = os.WriteFile(file1, []byte("data1"), 0600)
	_ = os.WriteFile(file2, []byte("data2"), 0600)
	_ = os.WriteFile(file3, []byte("data3"), 0600)

	req := models.DedupeBatchApplyRequest{
		Remote:    "local",
		ProfileID: "prof_test",
		Path:      "",
		Actions: []models.DedupeStageAction{
			{Path: "keep_me.txt", Action: "keep"},
			{Path: "del_me.txt", Action: "delete"},
			{Path: "ren_me.txt", Action: "rename", NewName: "ren_me_renamed.txt"},
		},
	}

	// Act
	resp, err := dedupeSvc.ExecuteDedupeBatch(context.Background(), req, tmpDir)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.SuccessCount != 3 {
		t.Fatalf("expected 3 successes, got: %d", resp.SuccessCount)
	}

	// 1. Verify ActionDedupeBatchExecuted in audit ledger
	records, total, err := auditSvc.Query(audit.Filter{Action: audit.ActionDedupeBatchExecuted})
	if err != nil {
		t.Fatalf("failed to query audit records: %v", err)
	}
	if total != 1 || len(records) != 1 {
		t.Fatalf("expected 1 DEDUPE_BATCH_EXECUTED record, got %d", total)
	}
	if records[0].Action != audit.ActionDedupeBatchExecuted {
		t.Errorf("expected action %s, got %s", audit.ActionDedupeBatchExecuted, records[0].Action)
	}
	if records[0].ProfileID != "prof_test" {
		t.Errorf("expected profile prof_test, got %s", records[0].ProfileID)
	}

	// 2. Verify per-file deletion and rename logs in bisync.log
	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read bisync.log: %v", err)
	}
	logStr := string(logData)

	if !strings.Contains(logStr, `"action":"Deleted"`) || !strings.Contains(logStr, `"object":"del_me.txt"`) {
		t.Errorf("expected per-file deletion log entry in bisync.log, got:\n%s", logStr)
	}
	if !strings.Contains(logStr, `"action":"Renamed"`) || !strings.Contains(logStr, `"object":"ren_me_renamed.txt"`) {
		t.Errorf("expected per-file rename log entry in bisync.log, got:\n%s", logStr)
	}
	if !strings.Contains(logStr, "DEDUPE_BATCH_APPLIED") {
		t.Errorf("expected batch summary log entry in bisync.log, got:\n%s", logStr)
	}
}
