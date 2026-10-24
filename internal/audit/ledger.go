package audit

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/core/security"
)

// GenesisHash is the 64-character hex zero-string used as the prev_hash for the first record.
const GenesisHash = "0000000000000000000000000000000000000000000000000000000000000000"

// AuditRecord represents a single immutable, cryptographically chained audit log entry.
type AuditRecord struct {
	SeqID     int64           `json:"seq_id"`
	Timestamp string          `json:"timestamp"`
	ProfileID string          `json:"profile_id"`
	Action    Action          `json:"action"`
	Actor     string          `json:"actor"`
	Details   json.RawMessage `json:"details"`
	PrevHash  string          `json:"prev_hash"`
	Hash      string          `json:"hash"`
}

// Filter specifies query criteria for fetching audit records.
type Filter struct {
	ProfileID string
	Action    Action
	Actor     string
	Limit     int
	Offset    int
}

// ComputeHash computes a deterministic SHA-256 digest over entry fields.
// Format: seq_id|timestamp|profile_id|action|actor|details|prev_hash
func ComputeHash(seqID int64, ts, profileID string, action Action, actor string, details []byte, prevHash string) string {
	h := sha256.New()
	payload := fmt.Sprintf("%d|%s|%s|%s|%s|%s|%s", seqID, ts, profileID, action, actor, string(details), prevHash)
	h.Write([]byte(payload))
	return hex.EncodeToString(h.Sum(nil))
}

// Append logs a new audit event within a safe database transaction.
// All keys within rawDetails undergo recursive secret sanitization prior to hashing and disk persistence.
func Append(db *sql.DB, profileID string, action Action, actor string, rawDetails map[string]any) (*AuditRecord, error) {
	if db == nil {
		return nil, fmt.Errorf("audit database is not initialized")
	}

	// 1. Redact sensitive credentials prior to hashing or disk persistence
	sanitizedDetails := security.RedactSecrets(rawDetails)
	detailsBytes, err := json.Marshal(sanitizedDetails)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal sanitized audit details: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 2. Fetch latest sequence and hash
	var lastSeq int64
	var prevHash string
	err = tx.QueryRow("SELECT seq_id, hash FROM audit_logs ORDER BY seq_id DESC LIMIT 1").Scan(&lastSeq, &prevHash)
	if err == sql.ErrNoRows {
		lastSeq = 0
		prevHash = GenesisHash
	} else if err != nil {
		return nil, fmt.Errorf("failed to query last audit record: %w", err)
	}

	nextSeq := lastSeq + 1
	now := time.Now().UTC().Format(time.RFC3339Nano)
	currHash := ComputeHash(nextSeq, now, profileID, action, actor, detailsBytes, prevHash)

	_, err = tx.Exec(
		`INSERT INTO audit_logs (seq_id, timestamp, profile_id, action, actor, details, prev_hash, hash) 
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		nextSeq, now, profileID, string(action), actor, string(detailsBytes), prevHash, currHash,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert audit record: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit audit record transaction: %w", err)
	}

	return &AuditRecord{
		SeqID:     nextSeq,
		Timestamp: now,
		ProfileID: profileID,
		Action:    action,
		Actor:     actor,
		Details:   detailsBytes,
		PrevHash:  prevHash,
		Hash:      currHash,
	}, nil
}

// QueryRecords retrieves paginated audit records matching the specified filter,
// along with the total count of matching records.
func QueryRecords(db *sql.DB, filter Filter) ([]AuditRecord, int64, error) {
	if db == nil {
		return nil, 0, fmt.Errorf("audit database is not initialized")
	}

	var whereClauses []string
	var args []any

	if filter.ProfileID != "" {
		whereClauses = append(whereClauses, "profile_id = ?")
		args = append(args, filter.ProfileID)
	}
	if filter.Action != "" {
		whereClauses = append(whereClauses, "action = ?")
		args = append(args, string(filter.Action))
	}
	if filter.Actor != "" {
		whereClauses = append(whereClauses, "actor = ?")
		args = append(args, filter.Actor)
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = " WHERE " + strings.Join(whereClauses, " AND ")
	}

	// 1. Get total count
	var total int64
	countQuery := "SELECT COUNT(*) FROM audit_logs" + whereSQL
	if err := db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count audit records: %w", err)
	}

	// 2. Query paginated records
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	selectSQL := fmt.Sprintf(
		"SELECT seq_id, timestamp, profile_id, action, actor, details, prev_hash, hash FROM audit_logs%s ORDER BY seq_id DESC LIMIT ? OFFSET ?",
		whereSQL,
	)
	queryArgs := append(args, limit, offset)

	rows, err := db.Query(selectSQL, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query audit records: %w", err)
	}
	defer rows.Close()

	var records []AuditRecord
	for rows.Next() {
		var r AuditRecord
		var detailsStr string
		var actionStr string
		if err := rows.Scan(&r.SeqID, &r.Timestamp, &r.ProfileID, &actionStr, &r.Actor, &detailsStr, &r.PrevHash, &r.Hash); err != nil {
			return nil, 0, fmt.Errorf("failed to scan audit record: %w", err)
		}
		r.Action = Action(actionStr)
		r.Details = json.RawMessage(detailsStr)
		records = append(records, r)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error reading audit rows: %w", err)
	}

	return records, total, nil
}

// VerifyLedger verifies mathematical continuity and cryptographic digest integrity across all recorded rows.
// It returns (valid, lastVerifiedSeqID, error).
func VerifyLedger(db *sql.DB) (bool, int64, error) {
	if db == nil {
		return false, 0, fmt.Errorf("audit database is not initialized")
	}

	rows, err := db.Query("SELECT seq_id, timestamp, profile_id, action, actor, details, prev_hash, hash FROM audit_logs ORDER BY seq_id ASC")
	if err != nil {
		return false, 0, fmt.Errorf("failed to query audit logs for verification: %w", err)
	}
	defer rows.Close()

	expectedPrevHash := GenesisHash
	var expectedSeq int64 = 1

	for rows.Next() {
		var r AuditRecord
		var detailsStr string
		var actionStr string
		if err := rows.Scan(&r.SeqID, &r.Timestamp, &r.ProfileID, &actionStr, &r.Actor, &detailsStr, &r.PrevHash, &r.Hash); err != nil {
			return false, 0, fmt.Errorf("failed to scan row %d: %w", expectedSeq, err)
		}
		r.Action = Action(actionStr)

		if r.SeqID != expectedSeq {
			return false, r.SeqID, fmt.Errorf("sequence gap at row %d (expected %d)", r.SeqID, expectedSeq)
		}

		if r.PrevHash != expectedPrevHash {
			return false, r.SeqID, fmt.Errorf("prev_hash mismatch at row %d (expected %s, got %s)", r.SeqID, expectedPrevHash, r.PrevHash)
		}

		computed := ComputeHash(r.SeqID, r.Timestamp, r.ProfileID, r.Action, r.Actor, []byte(detailsStr), r.PrevHash)
		if computed != r.Hash {
			return false, r.SeqID, fmt.Errorf("tampered payload at row %d: stored=%s, recomputed=%s", r.SeqID, r.Hash, computed)
		}

		expectedPrevHash = r.Hash
		expectedSeq++
	}

	if err := rows.Err(); err != nil {
		return false, 0, fmt.Errorf("error iterating audit rows: %w", err)
	}

	return true, expectedSeq - 1, nil
}
