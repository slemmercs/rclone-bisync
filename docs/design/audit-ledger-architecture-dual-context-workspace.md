# Design Specification: Audit Ledger Architecture & Dual-Context Workspace

## Executive Summary

The **Audit Ledger** serves as the safety-first data transparency engine for **Bisync Hub**. It provides an immutable, auditable log of all file-level synchronization actions, safety barrier decisions, deletion approvals, and conflict resolutions.

This specification defines the migration of the Audit Ledger from a flat text/telemetry view to a **dual-context, unified database architecture**. To preserve operational clarity, the Audit Ledger focuses strictly on **data & sync operations**, while administrative profile configuration changes remain isolated in Profile Change History. 

The underlying storage uses a single, indexed SQLite table, while the frontend provides both a **Global Fleet Ledger View (`/ledger`)** and a **Scoped Profile Workspace Ledger (`/profiles/:id`)**.

---

## 1. Domain Scope & Separation of Concerns

To prevent telemetry noise during safety audits, logging is divided into two distinct domain boundaries:

```
┌────────────────────────────────────────────────────────────────────────┐
│                          Bisync Hub Telemetry                          │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │
         ┌──────────────────────────┴──────────────────────────┐
         ▼                                                     ▼
┌─────────────────────────────────┐           ┌─────────────────────────────────┐
│       Data Audit Ledger         │           │   Profile System Change Log     │
│   (Owned by Audit Framework)    │           │   (Owned by Profile Manager)    │
├─────────────────────────────────┤           ├─────────────────────────────────┤
│ • File / Folder Sync Actions    │           │ • Profile Creation / Deletion   │
│ • Safety Barrier Holds          │           │ • Flag & Rule Overrides         │
│ • Deletion Limit Approvals      │           │ • Schedule / Cron Modifications │
│ • Conflict Resolutions          │           │ • Global Default Restores       │
│ • File Revision Rollbacks       │           │ • OAuth Remote Mapping Changes  │
└─────────────────────────────────┘           └─────────────────────────────────┘
```

* **Data Audit Ledger (Scope of this Spec):** Tracks file and directory operations executed by `rclone bisync` or authorized by human operators.
* **Profile System Change Log:** Maintained separately within `Profile.History` (`profile-manager.md`) and displayed inside the Profile Editor settings tab.

---

## 2. Database Schema & Architecture (`internal/ledger`)

All audit events across all profiles are persisted in a single `audit_ledger` table within the primary SQLite database. Dynamic per-profile tables are explicitly prohibited to prevent schema fragmentation and migration complexity.

```sql
-- Main Audit Ledger Table
CREATE TABLE IF NOT EXISTS audit_ledger (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    profile_id   TEXT NOT NULL,
    run_id       TEXT,                             -- Associated bisync execution session
    rev_id       TEXT,                             -- Associated safety review decision ID
    timestamp    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    event_type   TEXT NOT NULL,                    -- e.g., 'barrier_hold', 'conflict_resolved', 'file_deleted', 'manual_sync'
    severity     TEXT NOT NULL DEFAULT 'INFO',     -- 'INFO', 'WARN', 'CRITICAL'
    source_path  TEXT NOT NULL,                    -- Local or Path1 reference
    target_path  TEXT NOT NULL,                    -- Remote or Path2 reference
    details_json TEXT,                             -- Structured JSON payload (file size, hash, resolution mode)
    
    FOREIGN KEY (profile_id) REFERENCES profiles(id) ON DELETE CASCADE
);

-- Fast lookup index for profile-scoped views ordered chronologically
CREATE INDEX IF NOT EXISTS idx_audit_profile_time 
ON audit_ledger (profile_id, timestamp DESC);

-- Fast lookup index for Run ID correlation cross-linking
CREATE INDEX IF NOT EXISTS idx_audit_run_id 
ON audit_ledger (run_id);

-- Fast lookup index for Review ID correlation
CREATE INDEX IF NOT EXISTS idx_audit_rev_id 
ON audit_ledger (rev_id);
```

---

## 3. Dual-Context UI & Navigation Model

### 3.1 Level 1: Global Fleet Ledger View (`/ledger`)

Accessible from the main application header. Displays cross-profile audit events for system-wide auditing and troubleshooting.

```
┌───────────────────────────────────────────────────────────────────────────────────────────────┐
│  Bisync Hub  /  Audit Ledger (Global Fleet)                                                   │
├───────────────────────────────────────────────────────────────────────────────────────────────┤
│  FILTERS                                                                                      │
│  Profile: [ All Profiles ▾ ]   Event: [ All Types ▾ ]   Severity: [ All ▾ ]   Search: [      ]│
├───────────────────────────────────────────────────────────────────────────────────────────────┤
│ TIME       │ PROFILE            │ RUN ID   │ EVENT               │ PATH / DETAILS   │ ACTION │
├────────────┼────────────────────┼──────────┼─────────────────────┼──────────────────┼────────┤
│ 22:15:02   │ Personal GDrive    │ #run-892 │ ⚠ Barrier Hold      │ 12 Deletions     │ [Review]│
│ 21:30:10   │ OneDrive Photos    │ #run-891 │ ✓ Conflict Resolved │ ~/photos/a.jpg   │ [Detail]│
│ 20:00:45   │ Server S3 Archive  │ #run-888 │ ✖ Sync Failed       │ Network Timeout  │ [Logs] │
└───────────────────────────────────────────────────────────────────────────────────────────────┘
```

### 3.2 Level 2: Scoped Profile Workspace Ledger (`/profiles/:id`)

Automatically opens when navigating to the **Audit Ledger** tab within a specific profile workspace (`/profiles/:id`). Queries are strictly bound to `WHERE profile_id = :id`.

```
┌───────────────────────────────────────────────────────────────────────────────────────────────┐
│  Bisync Hub  /  Profiles  /  Personal Google Drive           [ Profile: Personal GDrive ▾ ]   │
├───────────────────────────────────────────────────────────────────────────────────────────────┤
│  [ Telemetry ]   [ File Explorer ]   [ 📜 Audit Ledger (14) ]   [ Profile Settings ]          │
├───────────────────────────────────────────────────────────────────────────────────────────────┤
│ TIME       │ RUN ID   │ EVENT               │ SOURCE / TARGET PATH         │ ACTIONS          │
├────────────┼──────────┼─────────────────────┼──────────────────────────────┼──────────────────┤
│ 22:15:02   │ #run-892 │ ⚠ Barrier Hold      │ gdrive:photos/2026/          │ [ Review Diff ]  │
│ 21:00:00   │ #run-885 │ ✓ Clean Execution   │ 0 conflicts, 3 transferred   │ [ View Session ] │
│ 18:30:12   │ #run-870 │ ⚙ Force Overridden  │ User approved max-delete     │ [ Audit Detail ] │
└───────────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## 4. Deep Linking & Cross-Correlation Mechanics

To make the Audit Ledger actionable, entries include interactive deep links:

1. **Run ID Cross-Linking (`#run-XYZ`):**
   * Clicking a `#run-892` badge updates the active filter to isolate all audit events, logs, and telemetry charts belonging to that execution session.
2. **Revision ID Cross-Linking (`rev-XYZ`):**
   * Clicking a `rev-402` indicator opens a modal displaying the exact human decision record (who approved the deletion, timestamp, and diff preview).
3. **Deep Path Hyperlinking:**
   * File paths rendered in source/target columns are interactive links. Clicking a path dispatches a `navigate-explorer` event:
     * Switches the active workspace tab to **File Explorer**.
     * Automatically sets the directory path and highlights the target file in the respective pane.

---

## 5. Backend Data Transfer Objects & Service Contracts

### 5.1 Go DTO Schema (`internal/ledger/dto.go`)

```go
package ledger

import "time"

type AuditEvent struct {
	ID          int64     `json:"id"`
	ProfileID   string    `json:"profile_id"`
	ProfileName string    `json:"profile_name,omitempty"` // Populated on Fleet View joins
	RunID       string    `json:"run_id,omitempty"`
	RevID       string    `json:"rev_id,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
	EventType   string    `json:"event_type"` // "barrier_hold", "conflict_resolved", "file_deleted", etc.
	Severity    string    `json:"severity"`   // "INFO", "WARN", "CRITICAL"
	SourcePath  string    `json:"source_path"`
	TargetPath  string    `json:"target_path"`
	DetailsJSON string    `json:"details_json,omitempty"`
}

type AuditFilter struct {
	ProfileID string    `json:"profile_id"`
	RunID     string    `json:"run_id"`
	EventType string    `json:"event_type"`
	Severity  string    `json:"severity"`
	Search    string    `json:"search"`
	Page      int       `json:"page"`
	Limit     int       `json:"limit"`
}

type AuditResponse struct {
	Events     []AuditEvent `json:"events"`
	TotalCount int          `json:"total_count"`
	Page       int          `json:"page"`
	Limit      int          `json:"limit"`
}
```

### 5.2 REST API Contracts

| Method | Endpoint | Description | Query Parameters |
| :--- | :--- | :--- | :--- |
| `GET` | `/api/ledger` | Global Fleet Ledger feed | `profile_id`, `event_type`, `severity`, `search`, `page`, `limit` |
| `GET` | `/api/profiles/:id/ledger` | Scoped Profile Ledger feed | `event_type`, `severity`, `search`, `page`, `limit` |
| `POST` | `/api/ledger/event` | Internal logger endpoint | Post body: `AuditEvent` JSON payload |

---

## 6. Acceptance Criteria

- [ ] **Data Model:** SQLite database creates `audit_ledger` table with foreign key cascading and indices for `profile_id`, `run_id`, and `rev_id`.
- [ ] **Domain Isolation:** System/Profile configuration changes (e.g., flag updates) are excluded from `audit_ledger` and logged in `profile_history`.
- [ ] **Fleet Ledger (`/ledger`):** Global ledger view lists events across all profiles with working filter dropdowns (`Profile`, `Event Type`, `Severity`).
- [ ] **Scoped Ledger (`/profiles/:id`):** Profile workspace ledger automatically filters events strictly to the active `:id`.
- [ ] **Deep Linking:** Clicking a `Run ID` filters ledger by execution session; clicking a path dispatches navigation to the **File Explorer**.
- [ ] **Cascading Cleanup:** Deleting a profile automatically cleans up associated audit ledger entries via SQLite `ON DELETE CASCADE`.