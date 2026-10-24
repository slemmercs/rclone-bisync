package audit

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/slemmercs/rclone-bisync/internal/core/security"
)

func setupTestDB(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	return filepath.Join(tmpDir, "audit_test.db")
}

func TestAppendAndVerify(t *testing.T) {
	dbPath := setupTestDB(t)
	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer db.Close()

	// Append 3 records across two profiles
	r1, err := Append(db, "gdrive-personal", ActionSyncExecuted, "SYSTEM", map[string]any{
		"files_synced": 5,
		"status":       "success",
	})
	if err != nil {
		t.Fatalf("failed to append record 1: %v", err)
	}
	if r1.SeqID != 1 {
		t.Fatalf("expected seq 1, got %d", r1.SeqID)
	}
	if r1.PrevHash != GenesisHash {
		t.Fatalf("expected genesis hash for record 1, got %s", r1.PrevHash)
	}
	if r1.Hash == "" {
		t.Fatal("expected non-empty hash for record 1")
	}

	r2, err := Append(db, "gdrive-personal", ActionConflictResolved, "USER", map[string]any{
		"path":       "Documents/notes.txt",
		"resolution": "keep_newer",
	})
	if err != nil {
		t.Fatalf("failed to append record 2: %v", err)
	}
	if r2.SeqID != 2 {
		t.Fatalf("expected seq 2, got %d", r2.SeqID)
	}
	if r2.PrevHash != r1.Hash {
		t.Fatalf("expected prev_hash %s, got %s", r1.Hash, r2.PrevHash)
	}

	r3, err := Append(db, "s3-backup", ActionDeletionAccepted, "USER", map[string]any{
		"count": 2,
	})
	if err != nil {
		t.Fatalf("failed to append record 3: %v", err)
	}
	if r3.SeqID != 3 || r3.PrevHash != r2.Hash {
		t.Fatalf("unexpected record 3 sequence or prev_hash: %+v", r3)
	}

	// Verify ledger integrity
	valid, count, err := VerifyLedger(db)
	if err != nil {
		t.Fatalf("verify ledger failed: %v", err)
	}
	if !valid || count != 3 {
		t.Fatalf("expected valid=true, count=3; got valid=%v, count=%d", valid, count)
	}

	// Query all records (ordered DESC)
	records, total, err := QueryRecords(db, Filter{})
	if err != nil {
		t.Fatalf("query records failed: %v", err)
	}
	if total != 3 || len(records) != 3 {
		t.Fatalf("expected 3 records, got total=%d, len=%d", total, len(records))
	}
	if records[0].SeqID != 3 || records[2].SeqID != 1 {
		t.Fatalf("expected records ordered DESC by seq_id; got [0]=%d, [2]=%d", records[0].SeqID, records[2].SeqID)
	}

	// Query with profile filter
	filtered, totalFiltered, err := QueryRecords(db, Filter{ProfileID: "gdrive-personal"})
	if err != nil {
		t.Fatalf("filtered query failed: %v", err)
	}
	if totalFiltered != 2 || len(filtered) != 2 {
		t.Fatalf("expected 2 filtered records, got total=%d, len=%d", totalFiltered, len(filtered))
	}

	// Query with pagination
	page, _, err := QueryRecords(db, Filter{Limit: 1, Offset: 1})
	if err != nil {
		t.Fatalf("paginated query failed: %v", err)
	}
	if len(page) != 1 || page[0].SeqID != 2 {
		t.Fatalf("expected record 2 on page, got %+v", page)
	}
}

func TestAppend_SecretRedaction(t *testing.T) {
	dbPath := setupTestDB(t)
	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer db.Close()

	secretPayload := map[string]any{
		"remote_name": "gdrive",
		"rc_password": "super-secret-password-123",
		"auth": map[string]any{
			"access_token":  "bearer-token-abc-999",
			"refresh_token": "rt-token-xyz-000",
		},
	}

	record, err := Append(db, "gdrive", ActionConfigUpdated, "USER", secretPayload)
	if err != nil {
		t.Fatalf("append with secrets failed: %v", err)
	}

	detailsStr := string(record.Details)
	if strings.Contains(detailsStr, "super-secret-password-123") {
		t.Fatal("password leaked in audit record details!")
	}
	if strings.Contains(detailsStr, "bearer-token-abc-999") || strings.Contains(detailsStr, "rt-token-xyz-000") {
		t.Fatal("tokens leaked in audit record details!")
	}
	if !strings.Contains(detailsStr, security.RedactedString) {
		t.Fatalf("expected %s in details, got %s", security.RedactedString, detailsStr)
	}

	valid, _, err := VerifyLedger(db)
	if err != nil || !valid {
		t.Fatalf("ledger with redacted secrets failed verification: %v", err)
	}
}

func TestTrigger_Immutability(t *testing.T) {
	dbPath := setupTestDB(t)
	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer db.Close()

	_, err = Append(db, "test", ActionSyncExecuted, "SYSTEM", map[string]any{"ok": true})
	if err != nil {
		t.Fatalf("failed to append record: %v", err)
	}

	// 1. Attempt UPDATE directly via SQL
	_, err = db.Exec("UPDATE audit_logs SET actor = 'ATTACKER' WHERE seq_id = 1")
	if err == nil {
		t.Fatal("expected UPDATE to be blocked by SQLite trigger, but it succeeded!")
	}
	if !strings.Contains(err.Error(), "immutable and cannot be updated") {
		t.Fatalf("expected trigger immutability error, got: %v", err)
	}

	// 2. Attempt DELETE directly via SQL
	_, err = db.Exec("DELETE FROM audit_logs WHERE seq_id = 1")
	if err == nil {
		t.Fatal("expected DELETE to be blocked by SQLite trigger, but it succeeded!")
	}
	if !strings.Contains(err.Error(), "immutable and cannot be deleted") {
		t.Fatalf("expected trigger deletion error, got: %v", err)
	}
}

func TestLedger_NilDBAndEdgeCases(t *testing.T) {
	_, err := Append(nil, "p", ActionSyncExecuted, "SYSTEM", nil)
	if err == nil {
		t.Fatal("expected error on nil db for Append")
	}

	_, _, err = QueryRecords(nil, Filter{})
	if err == nil {
		t.Fatal("expected error on nil db for QueryRecords")
	}

	_, _, err = VerifyLedger(nil)
	if err == nil {
		t.Fatal("expected error on nil db for VerifyLedger")
	}
}

func TestQueryRecords_Filters(t *testing.T) {
	dbPath := setupTestDB(t)
	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer db.Close()

	_, _ = Append(db, "prof1", ActionSyncExecuted, "SYSTEM", map[string]any{"v": 1})
	_, _ = Append(db, "prof2", ActionConflictResolved, "USER", map[string]any{"v": 2})

	// Filter by action
	recs, total, err := QueryRecords(db, Filter{Action: ActionSyncExecuted})
	if err != nil || total != 1 || len(recs) != 1 {
		t.Fatalf("failed action filter: %v, total=%d", err, total)
	}

	// Filter by actor
	recs, total, err = QueryRecords(db, Filter{Actor: "USER"})
	if err != nil || total != 1 || len(recs) != 1 {
		t.Fatalf("failed actor filter: %v, total=%d", err, total)
	}
}

func TestVerifyLedger_GapsAndHashMismatch(t *testing.T) {
	dbPath := setupTestDB(t)
	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer db.Close()

	_, _ = Append(db, "prof1", ActionSyncExecuted, "SYSTEM", map[string]any{"v": 1})
	_, _ = Append(db, "prof1", ActionSyncExecuted, "SYSTEM", map[string]any{"v": 2})

	// Drop triggers to test verification engine catching gap and prev_hash corruptions
	_, _ = db.Exec("DROP TRIGGER prevent_audit_update")
	_, _ = db.Exec("DROP TRIGGER prevent_audit_delete")

	// Test prev_hash mismatch
	_, _ = db.Exec("UPDATE audit_logs SET prev_hash = 'badhash' WHERE seq_id = 2")
	valid, rowID, err := VerifyLedger(db)
	if valid || rowID != 2 || err == nil {
		t.Fatalf("expected prev_hash mismatch error on row 2, got: valid=%v, row=%d, err=%v", valid, rowID, err)
	}

	// Test sequence gap (delete row 1)
	_, _ = db.Exec("DELETE FROM audit_logs WHERE seq_id = 1")
	valid, rowID, err = VerifyLedger(db)
	if valid || rowID != 2 || err == nil {
		t.Fatalf("expected sequence gap error, got: valid=%v, row=%d, err=%v", valid, rowID, err)
	}
}

func TestVerifyLedger_Tampering(t *testing.T) {
	dbPath := setupTestDB(t)
	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer db.Close()

	_, err = Append(db, "test", ActionSyncExecuted, "SYSTEM", map[string]any{"v": 1})
	if err != nil {
		t.Fatalf("failed to append 1: %v", err)
	}
	_, err = Append(db, "test", ActionSyncExecuted, "SYSTEM", map[string]any{"v": 2})
	if err != nil {
		t.Fatalf("failed to append 2: %v", err)
	}

	// Drop update trigger to simulate offline disk-level tampering
	_, err = db.Exec("DROP TRIGGER prevent_audit_update")
	if err != nil {
		t.Fatalf("failed to drop trigger for test: %v", err)
	}

	// Tamper row 1 details
	_, err = db.Exec("UPDATE audit_logs SET details = '{\"tampered\": true}' WHERE seq_id = 1")
	if err != nil {
		t.Fatalf("failed to execute tampering update: %v", err)
	}

	valid, rowID, err := VerifyLedger(db)
	if valid {
		t.Fatal("expected VerifyLedger to fail on tampered row, but it returned true!")
	}
	if rowID != 1 {
		t.Fatalf("expected failed row 1, got %d", rowID)
	}
	if !strings.Contains(err.Error(), "tampered payload") {
		t.Fatalf("expected tampered payload error, got: %v", err)
	}
}
