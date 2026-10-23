# Design Specification: Immutable Audit Ledger

**Feature Name**: Immutable Audit Ledger  
**Status**: Draft / Proposed  
**Target Milestone**: Phase 4 / Phase 5  
**Package Path**: `internal/audit`  
**Storage Driver**: Pure-Go SQLite (`modernc.org/sqlite`) + Embedded Migration Orchestrator (`pressly/goose/v3`)  

---

## 1. Executive Summary & Purpose

Bisync Hub prioritizes **operational intelligence, data safety, and user accountability** over generic cloud file management. When users resolve sync conflicts, accept proposed file deletions, or update critical configuration profiles, these choices carry data-loss risk if unrecorded or improperly audited.

The **Immutable Audit Ledger** provides a cryptographically verifiable, append-only historical log of every critical decision made within Bisync Hub. By combining pure-Go SQLite with SHA-256 hash chaining (`prev_hash` $\rightarrow$ `hash`) and database-level triggers, the ledger guarantees tamper-evidence and fast indexed UI querying while adhering to Bisync Hub's **Single Binary First**, **Zero CGO**, and **Minimal Runtime Dependencies** principles.

---

## 2. Core Architecture & Design Principles

```
  +-----------------------------------------------------------------------+
  |                             Bisync Hub                                |
  |                                                                       |
  |  +------------------+     +------------------+     +---------------+  |
  |  |  Sync Engine /   |     |  Dashboard UI /  |     |  CLI Engine   |  |
  |  |  Review Service  |     |  REST Handlers   |     |  ('bisync')   |  |
  |  +--------+---------+     +--------+---------+     +-------+-------+  |
  |           |                        |                       |          |
  |           +-------------------+    |    +------------------+          |
  |                               |    |    |                             |
  |                               v    v    v                             |
  |                     +--------------------------+                      |
  |                     |  internal/audit Service  |                      |
  |                     +------------+-------------+                      |
  |                                  |                                    |
  |                +-----------------+-----------------+                  |
  |                |                                   |                  |
  |                v                                   v                  |
  |   +--------------------------+        +--------------------------+    |
  |   | Security Redaction Gate  |        | Hash Chain Engine        |    |
  |   | (security.RedactSecrets) |        | (SHA-256 HMAC / Digest)  |    |
  |   +------------+-------------+        +------------+-------------+    |
  |                |                                   |                  |
  |                +-----------------+-----------------+                  |
  |                                  |                                    |
  |                                  v                                    |
  |                     +--------------------------+                      |
  |                     |   pressly/goose Driver   |                      |
  |                     +------------+-------------+                      |
  |                                  |                                    |
  +----------------------------------|------------------------------------+
                                     v
                       +---------------------------+
                       |    SQLite DB (audit.db)   |
                       | - WAL Mode Enabled        |
                       | - BEFORE UPDATE Triggers  |
                       | - BEFORE DELETE Triggers  |
                       +---------------------------+
```

1. **Single Binary & Zero CGO**: Uses `modernc.org/sqlite` to compile down into a pure Go binary without requiring C toolchains or system library dependencies.
2. **Embedded Migrations**: Embeds schema migration SQL scripts via `//go:embed` and automatically runs upgrades on application boot using `pressly/goose/v3`.
3. **Cryptographic Tamper-Evidence**: Each row includes a SHA-256 hash of its payload and the preceding entry's hash. Any disk-level file tampering or row deletion invalidates the chain.
4. **Engine-Level Defense**: SQLite `BEFORE UPDATE` and `BEFORE DELETE` triggers reject direct updates or row removals at the database layer.
5. **Secret Redaction at Boundary**: All event detail payloads are passed through `security.RedactSecrets()` before hashing and persistence to ensure secrets never permanently enter the cryptographic ledger.

---

## 3. Storage & Path Resolution

### 3.1 Path Resolution Hierarchy
The audit database file location is resolved dynamically on application startup in the following priority order:

1. **Explicit CLI Flag**: `--audit-db-path=/custom/path/audit.db`
2. **Environment Variable**: `BISYNC_HUB_AUDIT_DB_PATH=/custom/path/audit.db`
3. **Configuration File**: `audit_db_path` field in `bisync.env` or JSON config.
4. **Default System Path**: `$XDG_CONFIG_HOME/bisync-hub/audit.db` (Linux default: `~/.config/bisync-hub/audit.db`), managed via Go's `os.UserConfigDir()`.

### 3.2 Directory Initialization Code (`internal/audit/path.go`)

```go
package audit

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	DefaultDBFilename = "audit.db"
	AppDirName        = "bisync-hub"
)

// ResolveDBPath resolves the absolute audit database path and guarantees directory existence with 0700 permissions.
func ResolveDBPath(overridePath string) (string, error) {
	if overridePath != "" {
		absPath, err := filepath.Abs(overridePath)
		if err != nil {
			return "", fmt.Errorf("failed to resolve absolute path for override: %w", err)
		}
		if err := os.MkdirAll(filepath.Dir(absPath), 0700); err != nil {
			return "", fmt.Errorf("failed to create directory for custom audit db: %w", err)
		}
		return absPath, nil
	}

	baseConfigDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine user config directory: %w", err)
	}

	targetDir := filepath.Join(baseConfigDir, AppDirName)
	if err := os.MkdirAll(targetDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create default config directory: %w", err)
	}

	return filepath.Join(targetDir, DefaultDBFilename), nil
}
```

---

## 4. Database Schema & Migration Orchestration

Schema migrations are managed by `pressly/goose/v3` and embedded in `internal/audit/migrations/`.

### 4.1 Initial Migration (`00001_initial_audit_schema.sql`)

```sql
-- +goose Up
-- Enable Write-Ahead Logging (WAL) for non-blocking concurrent reads during sync operations
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA busy_timeout = 5000;

CREATE TABLE IF NOT EXISTS audit_logs (
    seq_id      INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp   TEXT NOT NULL,         -- ISO-8601 UTC timestamp (RFC3339Nano)
    profile_id  TEXT NOT NULL,         -- Profile or remote identifier (e.g. "gdrive-personal")
    action      TEXT NOT NULL,         -- Standardized event taxonomy code
    actor       TEXT NOT NULL,         -- "USER", "SYSTEM", or "API"
    details     TEXT NOT NULL,         -- Secret-sanitized JSON string payload
    prev_hash   TEXT NOT NULL,         -- SHA-256 hash of previous row (or Genesis hash)
    hash        TEXT NOT NULL          -- SHA-256 digest of current entry
);

CREATE INDEX IF NOT EXISTS idx_audit_profile_id ON audit_logs(profile_id);
CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_logs(timestamp);

-- SQLite Triggers enforcing append-only immutability
CREATE TRIGGER IF NOT EXISTS prevent_audit_update
BEFORE UPDATE ON audit_logs
BEGIN
    SELECT RAISE(ABORT, 'Audit ledger entries are immutable and cannot be updated');
END;

CREATE TRIGGER IF NOT EXISTS prevent_audit_delete
BEFORE DELETE ON audit_logs
BEGIN
    SELECT RAISE(ABORT, 'Audit ledger entries are immutable and cannot be deleted');
END;

-- +goose Down
DROP TRIGGER IF EXISTS prevent_audit_delete;
DROP TRIGGER IF EXISTS prevent_audit_update;
DROP INDEX IF EXISTS idx_audit_timestamp;
DROP INDEX IF EXISTS idx_audit_action;
DROP INDEX IF EXISTS idx_audit_profile_id;
DROP TABLE IF EXISTS audit_logs;
```

### 4.2 Migration Orchestrator (`internal/audit/migrations.go`)

```go
package audit

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

// RunMigrations executes pending database schema upgrades automatically on application boot.
func RunMigrations(db *sql.DB) error {
	goose.SetBaseFS(embedMigrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("failed to set goose sqlite3 dialect: %w", err)
	}

	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("failed to apply audit schema migrations: %w", err)
	}
	return nil
}
```

---

## 5. Cryptographic Hash Chaining & Verification Engine

### 5.1 Genesis Seed & Digest Calculation
The initial entry (`seq_id = 1`) uses a fixed 64-character hex zero-string as its `prev_hash`:
`0000000000000000000000000000000000000000000000000000000000000000`

The SHA-256 digest format concatenates fields with explicit delimiters to prevent canonical payload shifting:
`payload = seq_id + "|" + timestamp + "|" + profile_id + "|" + action + "|" + actor + "|" + details + "|" + prev_hash`

### 5.2 Implementation (`internal/audit/ledger.go`)

```go
package audit

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/core/security"
)

const GenesisHash = "0000000000000000000000000000000000000000000000000000000000000000"

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

// ComputeHash computes a deterministic SHA-256 digest over entry fields.
func ComputeHash(seqID int64, ts, profileID string, action Action, actor string, details []byte, prevHash string) string {
	h := sha256.New()
	payload := fmt.Sprintf("%d|%s|%s|%s|%s|%s|%s", seqID, ts, profileID, action, actor, string(details), prevHash)
	h.Write([]byte(payload))
	return hex.EncodeToString(h.Sum(nil))
}

// Append logs a new audit event within a safe database transaction.
func Append(db *sql.DB, profileID string, action Action, actor string, rawDetails map[string]any) (*AuditRecord, error) {
	// 1. Redact secrets prior to hashing or disk persistence
	sanitizedDetails := security.RedactSecrets(rawDetails)
	detailsBytes, err := json.Marshal(sanitizedDetails)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal sanitized audit details: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, err
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
		return nil, err
	}

	nextSeq := lastSeq + 1
	now := time.Now().UTC().Format(time.RFC3339Nano)
	currHash := ComputeHash(nextSeq, now, profileID, action, actor, detailsBytes, prevHash)

	_, err = tx.Exec(
		`INSERT INTO audit_logs (seq_id, timestamp, profile_id, action, actor, details, prev_hash, hash) 
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		nextSeq, now, profileID, action, actor, string(detailsBytes), prevHash, currHash,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert audit record: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
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

// VerifyLedger verifies mathematical continuity across all recorded rows.
func VerifyLedger(db *sql.DB) (bool, int64, error) {
	rows, err := db.Query("SELECT seq_id, timestamp, profile_id, action, actor, details, prev_hash, hash FROM audit_logs ORDER BY seq_id ASC")
	if err != nil {
		return false, 0, err
	}
	defer rows.Close()

	expectedPrevHash := GenesisHash
	var expectedSeq int64 = 1

	for rows.Next() {
		var r AuditRecord
		var detailsStr string
		if err := rows.Scan(&r.SeqID, &r.Timestamp, &r.ProfileID, &r.Action, &r.Actor, &detailsStr, &r.PrevHash, &r.Hash); err != nil {
			return false, 0, err
		}

		if r.SeqID != expectedSeq {
			return false, r.SeqID, fmt.Errorf("sequence gap at row %d (expected %d)", r.SeqID, expectedSeq)
		}

		if r.PrevHash != expectedPrevHash {
			return false, r.SeqID, fmt.Errorf("prev_hash mismatch at row %d", r.SeqID)
		}

		computed := ComputeHash(r.SeqID, r.Timestamp, r.ProfileID, r.Action, r.Actor, []byte(detailsStr), r.PrevHash)
		if computed != r.Hash {
			return false, r.SeqID, fmt.Errorf("tampered payload at row %d: stored=%s, recomputed=%s", r.SeqID, r.Hash, computed)
		}

		expectedPrevHash = r.Hash
		expectedSeq++
	}

	return true, expectedSeq - 1, nil
}
```

---

## 6. Standardized Event Taxonomy

All events logged to the ledger must use standardized `Action` constants declared in `internal/audit/taxonomy.go`:

```go
package audit

type Action string

const (
	// Conflict & Deletion Decisions
	ActionConflictResolved Action = "CONFLICT_RESOLVED"
	ActionDeletionAccepted Action = "DELETION_ACCEPTED"
	ActionDeletionRestored Action = "DELETION_RESTORED"
	ActionReviewDismissed  Action = "REVIEW_SESSION_DISMISSED"

	// Sync Operations
	ActionSyncExecuted     Action = "SYNC_EXECUTED"
	ActionSyncAborted      Action = "SYNC_ABORTED"
	ActionRollbackExecuted Action = "ROLLBACK_EXECUTED"

	// Configuration & Remote Auth Lifecycle
	ActionConfigUpdated    Action = "CONFIG_UPDATED"
	ActionOAuthReconnected Action = "OAUTH_RECONNECTED"
	ActionCanaryProvisioned Action = "CANARY_PROVISIONED"
)
```

---

## 7. Operational Recommendations & Guardrails

### 7.1 Boundary Secret Redaction
Before any entry payload is serialized, the `AuditService` passes the input map through `security.RedactSecrets(details)`. This guarantees that API keys, OAuth tokens, and BasicAuth credentials can never be permanently embedded in cryptographic hashes or saved to disk.

### 7.2 WAL Mode Concurrency Tuning
During sync execution, background tasks log telemetry while the user interacts with the dashboard UI. Enabling Write-Ahead Logging (`PRAGMA journal_mode = WAL`) prevents HTTP request handlers from blocking on background database writes.

### 7.3 Archival & Checkpoint Command
Because database triggers block `DELETE` operations, log archiving is performed by a dedicated CLI command:
`bisync-hub audit archive --before=YYYY-MM-DD --out=audit-archive-2026.db`

* **Checkpoint Mechanism**: When historical data is archived, the active database is reset with a special `GENESIS_CHECKPOINT` entry carrying over the last verified hash of the pruned dataset as its `prev_hash`.

### 7.4 Log Export Pipeline
The API and CLI provide JSON Lines (`.jsonl`) and CSV export capabilities:
* `bisync-hub audit export --format=jsonl --out=audit.jsonl`
* `GET /api/audit/export?format=csv`

---

## 8. REST API Endpoints

| Method | Endpoint | Description |
| :--- | :--- | :--- |
| `GET` | `/api/audit/records` | Query paginated audit log entries (`?limit=20&page=1&action=...`) |
| `GET` | `/api/audit/verify` | Execute cryptographic integrity verification over full ledger |
| `GET` | `/api/audit/export` | Download exported ledger (`?format=jsonl` or `?format=csv`) |

### Example Response (`GET /api/audit/verify`)

```json
{
  "valid": true,
  "total_records": 142,
  "checked_at": "2026-10-23T20:30:00Z",
  "head_hash": "a1b2c3d4e5f67890123456789abcdef0123456789abcdef0123456789abcdef0"
}
```

---

## 9. Next Steps & Implementation Roadmap

1. **Sprint 1 (`internal/audit`)**: Implement `path.go`, `migrations/`, `ledger.go`, and `taxonomy.go` with unit tests achieving >85% statement coverage.
2. **Sprint 2 (`internal/server`)**: Add HTTP REST handlers (`handlers_audit.go`) and integrate secret redaction.
3. **Sprint 3 (`dashboard/frontend`)**: Build the **Audit Trail Viewer** component in the Alpine.js SPA with real-time verification indicators.
4. **Sprint 4 (`cli`)**: Register CLI commands (`bisync-hub audit verify`, `bisync-hub audit export`).