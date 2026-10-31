# Implementation Plan: Deduplication Correlated Log Isolation

## Proposed Changes

When a user clicks "Correlated Log" from the Remote Deduplication modal (`modal_dedupe.html`), the dashboard currently displays the tail of `rclone-daemon.log` showing 500 lines from previous unrelated bisync runs instead of the deduplication log lines. This occurs because:
1. `GET /api/logs/daemon` ignores `run_id` and falls back to tail mode on `rclone-daemon.log` whenever `start_time` is absent.
2. `rclone dedupe` runs via RC API `core/command` and does not write to `rclone-daemon.log`; its outputs are captured by the server and streamed into `bisync.log`.
3. Preview (dry-run) deduplication (`mode == "list"`) in `handleRemoteDedupe` bypassed log writing into `bisync.log` due to a `mode != "list"` check.

We will isolate correlated logs by `run_id` so that viewing deduplication logs (or any correlated operation) displays strictly the log lines matching that specific operation.

### Affected Components & Packages

- `internal/services/log_service.go`:
  - Introduce `FilterLogsByRunID(logPath string, runID string, limit int) ([]string, error)` to extract entries matching `run_id` from JSON log files.
- `internal/server/handlers_telemetry.go`:
  - Update `handleDaemonLogs` to route `run_id` queries:
    - For `strings.HasPrefix(runID, "dedupe-")`: Query `bisync.log` via `FilterLogsByRunID` and return only matching deduplication entries.
    - For other `runID` values without explicit timestamps: Lookup start/end time in `bisync.log` to filter `rclone-daemon.log`, or fall back to matching `bisync.log` lines, returning empty rather than unrelated tail logs if not found.
    - Tail mode on `rclone-daemon.log` is strictly reserved for requests where `runID == ""` and `startTime.IsZero()`.
- `internal/server/handlers_auth.go`:
  - In `handleRemoteDedupe`, record structured log entries to `s.AuditService.Logger` for all deduplication modes (including `list` / preview) with `RunID: runID`, `Type: "dedupe"`, `Origin: "dedupe"`.
  - Ensure guaranteed status/completion messages even when no collisions exist.
- `dashboard/frontend/partials/modals/modal_daemon.html`:
  - Polish header title to dynamically display "Correlated Deduplication Logs" when `run_id` starts with `dedupe-`.
  - Polish footer window display to show `Run: <run_id>` when time window is absent.
- `dashboard/frontend/js/remotes.js`:
  - Pass `{ run_id: runId, remote: this.dedupeModal.remote, path: this.dedupeModal.path, type: "dedupe" }` in `viewDedupeLogs()`.
  - Adhere to the 400 LOC ceiling on `remotes.js`.
- Tests:
  - `internal/services/log_service_test.go`: Test `FilterLogsByRunID`.
  - `internal/server/server_test.go`: Test `handleDaemonLogs` with `run_id=dedupe-...`.

### Technical Strategy & Data Flow

```
[ Dedupe Modal / Entity Drawer ]
             │
             │ openDaemonModal({ run_id: "dedupe-1730355123" })
             ▼
[ GET /api/logs/daemon?run_id=dedupe-1730355123 ]
             │
             ├── If strings.HasPrefix(runID, "dedupe-"):
             │     LogService.FilterLogsByRunID(bisync.log, runID)
             │     └── Returns ONLY lines matching run_id
             │
             ├── If runID != "" (sync run):
             │     Lookup time window in bisync.log -> FilterDaemonLogs(rclone-daemon.log, start, end)
             │     └── Returns ONLY daemon logs for that sync run
             │
             └── If runID == "" and startTime.IsZero():
                   FilterDaemonLogs(rclone-daemon.log, 0, 0, limit)
                   └── General continuous daemon stream
```

### Safety & Error Handling

- Zero risk of leaking unrelated run logs to users requesting correlated logs.
- Safe buffer reading and bounded limits (clamped to max 1000 lines).
- Non-existent log files safely return empty arrays (`[]string{}`).
- Zero impact on advisory locking or sync execution.

## Verification Plan

- Unit tests: `go test -count=1 ./internal/services/...`
- Unit tests: `go test -count=1 ./internal/server/...`
- Template verification: `./scripts/check.sh templates`
- LOC budget verification: `./scripts/check.sh loc`
- Full pre-flight quality gate: `./scripts/check.sh all`
