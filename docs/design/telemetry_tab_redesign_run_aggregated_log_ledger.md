# Design Specification: Telemetry Tab Redesign & Run-Aggregated Log Ledger

## Executive Summary

The **Telemetry Tab** provides real-time and historical operational visibility into `rclone bisync` executions, daemon health, and execution latency. 

This specification defines a major UI/UX redesign of the log ledger panel (`bisync.log`), transitioning it from a raw text stream into a **Run-Aggregated Table Ledger** matching the visual design language of the **Audit Ledger** (`view_audit.html`). 

Key enhancements include:
1. **Run-Centric Line Bundling:** Grouping disparate log lines generated during a single execution session into a single expandable summary row.
2. **Unified Audit & Daemon Badging:** Placing interactive **Audit pills** (`[ AUDIT: BARRIER_HOLD ]`, `[ AUDIT: RESOLVED ]`) directly alongside **Daemon Log pills** on each run row.
3. **Table UI/UX Alignment:** Restructuring logs into scannable table columns with structured badges and status indicators.
4. **Profile-Driven Workspace Scoping:** Automatically filtering telemetry in the Scoped Workspace (`/profiles/:id/telemetry`) while supporting global telemetry viewing in Fleet View (`/telemetry`).

---

## 1. Run Aggregation Model (Line Bundling Engine)

### 1.1 Rationale
A single `rclone bisync` run can emit dozens of raw log entries (init, lock acquisition, path scanning, transfer progress, canary checks, lock release). Displaying these as unbundled individual lines creates visual noise and obscures execution outcomes.

### 1.2 Aggregation Behavior
The backend log serializer or frontend state processor aggregates raw log lines sharing the same `run_id` (`run-YYYYMMDD-HHMMSS` or session hash) into a single **Primary Run Row**.

```
┌───────────────────────────────────────────────────────────────────────────────────────────────┐
│ PRIMARY RUN ROW (Summary)                                                                     │
│ │ SUCCESS │ #run-774 │ [Personal GDrive] │ Sync completed in 16.3s (3 files)          │
│            │          │          │                   │ [ AUDIT: CLEAN ]  [ DAEMON LOGS ]  [ ▾ ]  │
└────────────┴──────────┴──────────┴───────────────────┴────────────────────────────────────────┘
                                            │
                                            │ User clicks expand chevron [ ▾ ]
                                            ▼
┌───────────────────────────────────────────────────────────────────────────────────────────────┐
│ EXPANDED SUB-STREAM (Raw Chronological Logs for #run-774)                                     │
│ ├─ 22:11:20 [INFO]  Initializing bisync session for path1 -> path2                            │
│ ├─ 22:11:22 [INFO]  Canary file validation passed (RCLONE_TEST)                               │
│ ├─ 22:11:25 [INFO]  Transferring 3 files (Total: 14.2 MB)                                      │
│ └─ 22:11:36 [NOTICE] Lock released. Execution finished successfully in 16.3s                     │
└───────────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Table UI/UX Specification (Audit Ledger Alignment)

The log panel transitions to a structured, scannable table sharing CSS utilities and dark-mode tokens with `view_audit.html`:

```
┌─────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│  bisync.log (Unified Run Ledger)           Filter: [ All Runs ▾ ]   [ All Profiles ▾ ]      Tail: [ 100 Runs ▾ ]│
├─────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ TIME     │ STATUS    │ RUN ID   │ PROFILE           │ EXECUTION SUMMARY             │ AUDIT & DAEMON BADGES  │ ACT │
├──────────┼───────────┼──────────┼───────────────────┼───────────────────────────────┼────────────────────────┼─────┤
│ 22:11:36 │ SUCCESS   │ #run-774 │ [Personal GDrive] │ Succeeded in 16.3s (3 files)  │ [AUDIT: CLEAN] [DAEMON]│ [▾] │
│ 21:52:57 │ ✖ ERROR   │ #run-773 │ [OneDrive Photos] │ Safety barrier: --check-access│ [AUDIT: HOLD]  [DAEMON]│ [▾] │
│ 21:47:10 │ ℹ AUDIT   │ #run-772 │ [OneDrive Photos] │ Profile configuration deleted │ [AUDIT: SYS]   [LOGS]  │ [▾] │
│ 21:36:07 │ ⚠ SKIPPED │ #run-771 │ [Personal GDrive] │ Aborted: Lock contention      │ [AUDIT: NONE]  [DAEMON]│ [▾] │
└──────────┴───────────┴──────────┴───────────────────┴───────────────────────────────┴────────────────────────┴─────┘
```

### 2.1 Column Definitions

| Column | Format / Class | Content & Behavior |
| :--- | :--- | :--- |
| **TIME** | `font-mono text-xs text-slate-400` | ISO timestamp converted to local format (`HH:mm:ss`). |
| **STATUS** | Colored Status Badge | `SUCCESS` (Emerald), `ERROR` (Red), `BARRIER_HOLD` (Amber), `SKIPPED` (Slate/Yellow), `AUDIT` (Purple). |
| **RUN ID** | Interactive Pill (`bg-slate-800 text-indigo-300 font-mono`) | Clicking `#run-774` copies the Run ID or isolates session traces across views. |
| **PROFILE** | Profile Badge (`text-sky-400 bg-sky-950/50`) | Displays active profile context (e.g., `[Personal GDrive]`). Clicking filters telemetry by this profile. |
| **SUMMARY** | `text-slate-200 text-xs font-sans` | Aggregated run result text (duration, transferred file count, or barrier trigger reason). |
| **BADGES** | Dual Badge Container | Contains two primary action/status pills:<br>• **Audit Pill:** `[ AUDIT: HOLD ]` (Amber), `[ AUDIT: CLEAN ]` (Green), `[ AUDIT: SYS ]` (Purple).<br>• **Daemon Pill:** `[ DAEMON LOGS ]` (Blue/Slate). |
| **ACTIONS** | Expansion Toggle Button | Expands/collapses nested raw log lines for the specific run. |

---

## 3. Audit Pill & Daemon Link Mechanics

### 3.1 Audit Pill Behavior
The **Audit Pill** reflects whether the execution generated entries in the **Data Audit Ledger** or triggered an interactive review barrier:

* **`[ AUDIT: HOLD ]` (Amber/Warning):**
  * **Trigger:** Execution triggered a safety barrier (e.g., `--max-delete` exceeded or path conflict).
  * **Click Action:** Immediately opens the **Interactive Review Modal** (`modal_review.html`) for decision triage.
* **`[ AUDIT: RESOLVED ]` (Emerald/Success):**
  * **Trigger:** Execution required a review decision that was successfully authorized and committed by an operator.
  * **Click Action:** Opens the **Audit Ledger** pre-filtered to the associated `rev_id`.
* **`[ AUDIT: CLEAN ]` (Slate/Green):**
  * **Trigger:** Execution completed normally without triggering safety holds or audit flags.
  * **Click Action:** Opens the profile's **Audit Ledger** pre-filtered to `run_id`.

### 3.2 Daemon Pill Behavior
* **`[ DAEMON LOGS ]` (Blue/Indigo):**
  * **Click Action:** Opens the **Daemon Trace Modal** displaying stdout/stderr stream lines specifically filtered for that `run_id` execution window.

---

## 4. Dual-Context & Profile-Driven Architecture

In accordance with the **Hybrid Fleet Overview & Contextual Workspace** architecture:

1. **Global Fleet Telemetry (`/telemetry`):**
   * Displays aggregated sync runs across **all profiles**.
   * Includes a **Profile Filter Dropdown** in the header (`[ All Profiles ▾ ]`).
2. **Scoped Workspace Telemetry (`/profiles/:id/telemetry`):**
   * Automatically scopes log streams and metrics strictly to `WHERE profile_id = :id`.
   * Displays profile-specific latency graphs and execution statistics.

---

## 5. Data Transfer Objects & Backend Contracts (`internal/telemetry`)

### 5.1 Go DTO Schema (`internal/telemetry/dto.go`)

```go
package telemetry

import "time"

// RunSummary represents a bundled bisync execution session log entry.
type RunSummary struct {
	RunID            string    `json:"run_id"`
	ProfileID        string    `json:"profile_id"`
	ProfileName      string    `json:"profile_name"`
	StartTime        time.Time `json:"start_time"`
	EndTime          time.Time `json:"end_time"`
	DurationMs       int64     `json:"duration_ms"`
	Status           string    `json:"status"` // "SUCCESS", "ERROR", "BARRIER_HOLD", "SKIPPED"
	FilesTransferred int       `json:"files_transferred"`
	BytesTransferred int64     `json:"bytes_transferred"`
	TriggerSource    string    `json:"trigger_source"` // "SCHEDULER", "DASHBOARD", "CLI"
	
	// Audit Link State
	AuditState       string    `json:"audit_state"` // "CLEAN", "HOLD", "RESOLVED", "SYSTEM"
	RevID            string    `json:"rev_id,omitempty"`
	
	// Raw Log Lines (Populated on expansion or detailed query)
	LogLines         []LogLine `json:"log_lines,omitempty"`
}

type LogLine struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"` // "INFO", "WARN", "ERROR", "NOTICE"
	Source    string    `json:"source"`
	Message   string    `json:"message"`
}
```

### 5.2 REST API Endpoints

| Method | Endpoint | Description |
| :--- | :--- | :--- |
| `GET` | `/api/telemetry/runs` | Fetches aggregated run ledger entries (supports `profile_id`, `status`, `limit`). |
| `GET` | `/api/telemetry/runs/:run_id` | Fetches full raw log lines for a specific execution run. |
| `GET` | `/api/profiles/:id/telemetry` | Fetches telemetry metrics and run summaries scoped strictly to `:id`. |

---

## 6. Acceptance Criteria

- [ ] **Run Aggregation:** Log lines belonging to the same `run_id` are bundled into a single summary table row.
- [ ] **Expandable Rows:** Clicking a run summary row toggles the visibility of underlying raw log lines.
- [ ] **Table Styling:** Log panel matches the clean, scannable table design of `view_audit.html`.
- [ ] **Audit Pill Integration:** Each run row displays an interactive Audit pill (`[ AUDIT: HOLD ]`, `[ AUDIT: CLEAN ]`, `[ AUDIT: RESOLVED ]`) that links directly to safety review modals or audit ledger entries.
- [ ] **Daemon Pill Integration:** Each run row displays a `[ DAEMON LOGS ]` pill opening filtered daemon trace logs.
- [ ] **Profile-Driven Scoping:** Telemetry panel automatically filters by `profile_id` when viewed inside `/profiles/:id/telemetry`.