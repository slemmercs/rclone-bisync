package services

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/audit"
	"github.com/slemmercs/rclone-bisync/internal/core/logger"
	"github.com/slemmercs/rclone-bisync/internal/models"
)

// AuditVerifyResult contains the outcome of a cryptographic ledger continuity audit.
type AuditVerifyResult struct {
	Valid        bool   `json:"valid"`
	TotalRecords int64  `json:"total_records"`
	CheckedAt    string `json:"checked_at"`
	HeadHash     string `json:"head_hash"`
	Error        string `json:"error,omitempty"`
}

// AuditService manages append-only cryptographic event logging, querying, verification, and export.
type AuditService struct {
	mu     sync.RWMutex
	DB     *sql.DB
	Logger *logger.Logger
}

// NewAuditService constructs an AuditService instance.
func NewAuditService(db *sql.DB, l *logger.Logger) *AuditService {
	return &AuditService{
		DB:     db,
		Logger: l,
	}
}

// RecordEvent writes an immutable, secret-sanitized audit entry to the SQLite database
// and mirrors the event to the unified bisync.log stream if a Logger is configured.
func (s *AuditService) RecordEvent(profileID string, action audit.Action, actor string, details map[string]any) (*audit.AuditRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.DB == nil {
		return nil, fmt.Errorf("audit database is not configured")
	}

	record, err := audit.Append(s.DB, profileID, action, actor, details)
	if err != nil {
		return nil, fmt.Errorf("failed to append audit record: %w", err)
	}

	// Mirror to bisync.log
	if s.Logger != nil {
		logEntry := models.AuditLogRecord{
			Time:      record.Timestamp,
			Level:     "info",
			Type:      "audit",
			Msg:       fmt.Sprintf("[%s] %s by %s", record.ProfileID, record.Action, record.Actor),
			SeqID:     record.SeqID,
			ProfileID: record.ProfileID,
			Action:    string(record.Action),
			Actor:     record.Actor,
			Hash:      record.Hash,
			PrevHash:  record.PrevHash,
			Details:   record.Details,
		}
		_ = s.Logger.RecordAuditEvent(logEntry)
	}

	return record, nil
}

// Query fetches paginated audit records matching the specified criteria.
func (s *AuditService) Query(filter audit.Filter) ([]audit.AuditRecord, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.DB == nil {
		return nil, 0, fmt.Errorf("audit database is not configured")
	}

	return audit.QueryRecords(s.DB, filter)
}

// Verify mathematically validates full cryptographic chain continuity from Genesis to Head.
func (s *AuditService) Verify() (*AuditVerifyResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now().UTC().Format(time.RFC3339)

	if s.DB == nil {
		return &AuditVerifyResult{
			Valid:     false,
			CheckedAt: now,
			Error:     "audit database is not configured",
		}, nil
	}

	valid, count, err := audit.VerifyLedger(s.DB)
	if err != nil {
		return &AuditVerifyResult{
			Valid:        false,
			TotalRecords: count,
			CheckedAt:    now,
			Error:        err.Error(),
		}, nil
	}

	headHash := audit.GenesisHash
	if count > 0 {
		_ = s.DB.QueryRow("SELECT hash FROM audit_logs WHERE seq_id = ?", count).Scan(&headHash)
	}

	return &AuditVerifyResult{
		Valid:        valid,
		TotalRecords: count,
		CheckedAt:    now,
		HeadHash:     headHash,
	}, nil
}

// Export produces formatted ledger data for external compliance and forensic inspection.
// Supported formats: "jsonl" (default) or "csv".
// Returns (data, mimeType, filename, error).
func (s *AuditService) Export(format string, filter audit.Filter) ([]byte, string, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.DB == nil {
		return nil, "", "", fmt.Errorf("audit database is not configured")
	}

	// Fetch up to 100,000 records for export
	filter.Limit = 100000
	filter.Offset = 0
	records, _, err := audit.QueryRecords(s.DB, filter)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to retrieve records for export: %w", err)
	}

	switch format {
	case "csv":
		var buf bytes.Buffer
		w := csv.NewWriter(&buf)
		_ = w.Write([]string{"seq_id", "timestamp", "profile_id", "action", "actor", "prev_hash", "hash", "details"})

		for _, r := range records {
			row := []string{
				strconv.FormatInt(r.SeqID, 10),
				r.Timestamp,
				r.ProfileID,
				string(r.Action),
				r.Actor,
				r.PrevHash,
				r.Hash,
				string(r.Details),
			}
			_ = w.Write(row)
		}
		w.Flush()
		return buf.Bytes(), "text/csv; charset=utf-8", "audit-ledger.csv", nil

	default: // "jsonl"
		var buf bytes.Buffer
		for _, r := range records {
			line, err := json.Marshal(r)
			if err != nil {
				return nil, "", "", fmt.Errorf("failed to marshal export record: %w", err)
			}
			buf.Write(line)
			buf.WriteByte('\n')
		}
		return buf.Bytes(), "application/x-ndjson; charset=utf-8", "audit-ledger.jsonl", nil
	}
}

// Close closes the underlying database handle.
func (s *AuditService) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.DB != nil {
		err := s.DB.Close()
		s.DB = nil
		return err
	}
	return nil
}
