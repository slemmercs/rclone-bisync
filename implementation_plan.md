# Implementation Plan: Emit Synced Files in Log Dock with On-Demand Pagination

## Overview
Capture, record, and navigate individual file operations transferred during Rclone bisync runs within the Live Sync Log Dock and Unified Timeline Ledger, backed by on-demand log pagination and deep file jumping.

When a sync execution completes (initial resync or scheduled bi-directional run), operators currently see only an aggregate summary (e.g. `Resync succeeded in 2.0s (5 file(s) transferred)`). The specific files transferred are hidden within correlated daemon logs, preventing the Live Sync Log Dock's "File Ops" filter from exposing them and blocking operators from using one-click deep navigation to inspect newly synced files in the dual-pane Directory Explorer.

This implementation captures individual file transfers directly from the Rclone job output or daemon stream at sync completion, records up to 50 individual file operations into `bisync.log` with distinct `Object`, `Action`, and `Type: "sync_file"` metadata, and introduces on-demand pagination controls in the Log Dock to seamlessly load and step through historical pages of log entries without UI degradation or log explosion.

---

## Types

### 1. `internal/models/api.go`
```go
type SyncFileOp struct {
Path   string `json:"path"`   // Relative file path (e.g. "subfolder/nested.txt")
Action string `json:"action"` // Operation type: "COPIED", "REPLACED", "DELETED", "CONFLICT"
Msg    string `json:"msg"`    // Human-readable message (e.g. "subfolder/nested.txt: Copied (new)")
}

type BisyncLogEntry struct {
Time        string  `json:"time"`
Level       string  `json:"level"`
Type        string  `json:"type,omitempty"` // "sync", "sync_file", "audit"
ProfileID   string  `json:"profile_id,omitempty"`
ProfileName string  `json:"profile_name,omitempty"`
Msg         string  `json:"msg"`
RunID       string  `json:"run_id"`
JobID       int     `json:"job_id,omitempty"`
Origin      string  `json:"origin,omitempty"`
Status      string  `json:"status,omitempty"`     // SUCCESS, FAULT, ABORTED
ErrorType   string  `json:"error_type,omitempty"` // TOO_MANY_DELETES, RATE_LIMIT, etc.
DurationSec float64 `json:"duration_sec,omitempty"`
FilesSynced int     `json:"files_synced,omitempty"`
StartTime   string  `json:"start_time,omitempty"` // RFC3339
EndTime     string  `json:"end_time,omitempty"`   // RFC3339
Object      string  `json:"object,omitempty"`     // Relative path for deep navigation
Action      string  `json:"action,omitempty"`     // Operation action badge
}
```

### 2. `internal/core/client/monitor.go`
```go
type MonitorJobResult struct {
Success          bool
ErrorType        string
ErrorMessage     string
FilesSynced      int
TransferredFiles []models.SyncFileOp
}
```

---

## Proposed Changes

### Phase 1: Transfer File Parser & Extraction (`internal/models`, `internal/core/client`)
- Add `SyncFileOp` to `internal/models/api.go`.
- Implement `ExtractTransferredFiles(outputStr string, maxEntries int) []models.SyncFileOp` in `internal/core/client/monitor.go`.
- Update `MonitorJob` to populate `TransferredFiles` on `MonitorJobResult`.
- Unit tests in `internal/core/client/monitor_test.go`.

### Phase 2: Ingestion & 50-Item Capping in Runner (`internal/engine/runner`)
- In `handleSyncSuccess` in `internal/engine/runner/runner_handlers.go`, emit up to 50 individual `sync_file` records to `bisync.log`.
- Emit overflow notification if files exceed 50.
- Fallback to reading `rclone-daemon.log` if job output buffer was omitted.
- Unit tests in `internal/engine/runner/runner_test.go`.

### Phase 3: Log Dock Template & Pagination Controls (`component_log_dock.html`)
- Add pagination buttons (`[ < Page X of Y > ]`), entry counter, and badge styling in Log Dock.
- Validate template with `./scripts/check.sh templates`.

### Phase 4: Frontend Log Dock Engine & Stepping (`log_dock.js`, `logs.js`)
- Add `loadDockNextPage()` / `loadDockPrevPage()` in `dashboard/frontend/js/log_dock.js`.
- Include `type === "sync_file"` in `isFileOperationLog`.
- Verify LOC under 400 with `./scripts/check.sh loc 400`.

### Phase 5: Verification & End-to-End Tests (`dashboard_smoke_test.go`)
- Validate Log Dock pagination, sync file item jumping, and full test suite (`./scripts/check.sh all`).

---

## Safety & Error Handling
- **Cap Protection**: Strict capping at 50 file entries per sync run protects against log file explosion.
- **Fail-Safe Fallback**: If output parsing fails, summary logging remains intact and job execution is unaffected.
- **Confinement & Jail**: All deep navigation paths remain verified within profile boundaries.
