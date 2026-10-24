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
)

func setupTestAuditService(t *testing.T) (*AuditService, string, string) {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_audit.db")
	logPath := filepath.Join(tmpDir, "bisync.log")

	db, err := audit.OpenDB(dbPath)
	if err != nil {
		t.Fatalf("failed to open audit db: %v", err)
	}

	logSink, err := logger.New(logPath, false)
	if err != nil {
		t.Fatalf("failed to open logger: %v", err)
	}

	svc := NewAuditService(db, logSink)
	return svc, dbPath, logPath
}

func TestAuditService_RecordAndVerify(t *testing.T) {
	svc, _, logPath := setupTestAuditService(t)
	defer svc.Close()

	rec, err := svc.RecordEvent("gdrive-personal", audit.ActionConflictResolved, "USER", map[string]any{
		"path":       "notes.txt",
		"resolution": "newer",
	})
	if err != nil {
		t.Fatalf("RecordEvent failed: %v", err)
	}
	if rec.SeqID != 1 {
		t.Fatalf("expected SeqID 1, got %d", rec.SeqID)
	}

	// Verify ledger
	res, err := svc.Verify()
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if !res.Valid || res.TotalRecords != 1 {
		t.Fatalf("expected valid=true, total=1; got %+v", res)
	}
	if res.HeadHash != rec.Hash {
		t.Fatalf("expected head hash %s, got %s", rec.Hash, res.HeadHash)
	}

	// Verify mirrored in bisync.log
	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log: %v", err)
	}
	if !strings.Contains(string(logData), `"type":"audit"`) {
		t.Fatalf("log does not contain mirrored audit event: %s", string(logData))
	}
}

func TestAuditService_QueryAndExport(t *testing.T) {
	svc, _, _ := setupTestAuditService(t)
	defer svc.Close()

	_, _ = svc.RecordEvent("prof1", audit.ActionSyncExecuted, "SYSTEM", map[string]any{"v": 1})
	_, _ = svc.RecordEvent("prof2", audit.ActionDeletionAccepted, "USER", map[string]any{"v": 2})

	// Query
	recs, total, err := svc.Query(audit.Filter{})
	if err != nil || total != 2 || len(recs) != 2 {
		t.Fatalf("Query failed: %v, total=%d, len=%d", err, total, len(recs))
	}

	// Export JSONL
	dataJSONL, mimeJSONL, fileJSONL, err := svc.Export("jsonl", audit.Filter{})
	if err != nil {
		t.Fatalf("Export jsonl failed: %v", err)
	}
	if !strings.Contains(mimeJSONL, "ndjson") || fileJSONL != "audit-ledger.jsonl" {
		t.Fatalf("unexpected jsonl meta: %s, %s", mimeJSONL, fileJSONL)
	}
	lines := strings.Split(strings.TrimSpace(string(dataJSONL)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines in jsonl export, got %d", len(lines))
	}

	// Export CSV
	dataCSV, mimeCSV, fileCSV, err := svc.Export("csv", audit.Filter{})
	if err != nil {
		t.Fatalf("Export csv failed: %v", err)
	}
	if !strings.Contains(mimeCSV, "csv") || fileCSV != "audit-ledger.csv" {
		t.Fatalf("unexpected csv meta: %s, %s", mimeCSV, fileCSV)
	}
	csvLines := strings.Split(strings.TrimSpace(string(dataCSV)), "\n")
	if len(csvLines) != 3 { // Header + 2 rows
		t.Fatalf("expected 3 lines in csv export, got %d", len(csvLines))
	}
}

func TestAuditService_ReviewIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_audit.db")
	db, err := audit.OpenDB(dbPath)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	auditSvc := NewAuditService(db, nil)

	cfg := &config.Config{
		BaseDir:          tmpDir,
		RemotePath:       "gdrive:",
		ReviewDir:        tmpDir,
		ActiveReviewFile: filepath.Join(tmpDir, "active.json"),
		ReviewBackupsDir: tmpDir,
	}

	reviewSvc := NewReviewService(cfg, nil)
	reviewSvc.Audit = auditSvc

	sess, err := reviewSvc.DiscoverSession(context.Background(), "CONFLICT", "Conflict detected")
	if err != nil || sess == nil {
		t.Fatalf("DiscoverSession failed: %v", err)
	}

	if err := reviewSvc.DismissReviewSession(sess.ID); err != nil {
		t.Fatalf("DismissReviewSession failed: %v", err)
	}

	records, total, err := auditSvc.Query(audit.Filter{})
	if err != nil || total != 1 || len(records) != 1 {
		t.Fatalf("expected 1 audit record from dismiss, got total=%d, len=%d, err=%v", total, len(records), err)
	}
	if records[0].Action != audit.ActionReviewDismissed {
		t.Fatalf("expected action %s, got %s", audit.ActionReviewDismissed, records[0].Action)
	}
}
