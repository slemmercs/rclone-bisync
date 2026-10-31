# Implementation Plan: Audit Ledger Reset, Telemetry Refinement & Defensive Defaults

## Proposed Changes

Enforce strict defensive defaults across global fleet settings and profile synchronization, clean up HTTP access log flooding by demoting background polling to debug level, fix unified log pagination in the dashboard, isolate unit/E2E test suites from host RC daemon, deduplicate profile filters in the audit view, and provide an immutable-safe audit ledger reset mechanism with automatic backup rotation.

### Affected Components & Packages

- `internal/config/dto.go` & `dto_test.go`:
  - Default `LockExplorerRemote` to `true` (locked to active profile remote by default).
  - Default `MaxDelete` to `0` (zero tolerance for deletions without explicit review/override).
- `internal/profiles/builder.go` & `internal/server/handlers_profiles.go`:
  - Update fallback condition from `<= 0` to `< 0` so that `MaxDelete = 0` is recognized as an explicit valid threshold and correctly emits `--max-delete 0`.
- `dashboard/frontend/js/settings.js` & `dashboard/frontend/js/profiles.js`:
  - Align frontend defaults for `lock_explorer_remote: true` and `max_delete: 0`.
- `internal/server/server.go`:
  - Refine HTTP access logging middleware: log background polling GET requests (`/api/telemetry`, `/api/health`, `/api/logs/*`, `/api/profiles/summary`) at `slog.Debug` level rather than `slog.Info`, preventing log spam.
- `dashboard/frontend/js/utils.js`:
  - Format `http_access` lines with method, path, status code, and duration instead of a bare string.
- `dashboard/frontend/partials/patterns/pattern_timeline_ledger.html`:
  - Gate the `Daemon` button on log rows so it only renders when the record contains a correlation `run_id` or `job_id`.
- `dashboard/frontend/app.js`:
  - Wire `fetchBisyncLogs(1)` into `initDashboard()` and `switchTab('telemetry')` to activate 50-line pagination.
- `tests/e2e/dashboard_smoke_test.go` & `internal/services/review_service_test.go`:
  - Replace unauthenticated references to live `http://127.0.0.1:5572` with isolated `httptest.NewServer` mocks.
- `dashboard/frontend/partials/view_audit.html`:
  - Remove duplicate hardcoded `<option value="default">Default Profile</option>` from the profile filter dropdown.
  - Add "Reset Ledger" button with confirmation modal.
- `internal/audit/` & `internal/server/handlers_audit.go`:
  - Implement `POST /api/audit/reset` endpoint that rotates existing `audit.db` to `audit.db.bak.<timestamp>` and initializes a clean ledger from Genesis block.

### Technical Strategy & Data Flow

```
[ Dashboard Client ]
       │
       ├── Background Polling (GET /api/telemetry, /api/health) ──► slog.Debug (omitted at INFO level)
       ├── User / State Action (POST, PUT, DELETE, /api/sync)   ──► slog.Info ("GET /path → 200 (1.2ms)")
       │
       └── POST /api/audit/reset 
              │
              ├── Lock DB / Acquire write mutex
              ├── Close active *sql.DB connection
              ├── Rename audit.db -> audit.db.bak.<timestamp> (immutable history preserved)
              ├── Run goose migrations to recreate tables & triggers
              └── Record fresh sequence #1 Genesis block
```

### Safety & Error Handling

- **Atomic Ledger Reset**: Reset preserves forensic history by renaming `audit.db` to timestamped backup before recreating schema. Triggers are never altered or dropped.
- **Defensive Defaults**: `max-delete: 0` ensures accidental mass deletions abort instantly unless overridden. `lock_explorer_remote: true` isolates operators to the active profile's remote.
- **Test Isolation**: Zero traffic directed to live host ports (`5572`), preventing spurious 401s in daemon logs.

## Verification Plan

- Unit tests for configuration and builder defaults: `go test -v ./internal/config/... ./internal/profiles/...`
- Server handler and audit reset verification: `go test -v ./internal/server/... ./internal/audit/...`
- Full pre-flight quality gate: `./scripts/check.sh all`



