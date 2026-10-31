# Implementation Plan: De-Noise Unified Ledger & Sync Milestones

## Proposed Changes

Eliminate UI noise and redundant log emissions so that sync runs (such as dry-runs) present a single, clean, reassuring milestone entry in the dashboard Unified Ledger. Demote backend daemon job polling milestones and successful HTTP access records to `slog.LevelDebug`, eliminate duplicate `log.Printf` vs `RecordSyncRun` logging in `runner_handlers.go`, contextually label the sync mode (`Dry run`, `Resync`, `Sync`), and ensure the frontend Unified Ledger filters out internal HTTP access entries.

### Affected Components & Packages

- `internal/core/client/monitor.go`:
  - Demote `slog.Info("Monitoring bisync job", ...)` and `slog.Info("Job completed successfully", ...)` to `slog.Debug(...)`.
- `internal/engine/runner/runner_handlers.go`:
  - Remove duplicate `log.Printf` calls from `handleSyncSuccess` and `handleSyncFailure`.
  - Contextually prefix milestone messages: `"Dry run succeeded..."` for dry-runs, `"Resync succeeded..."` for resyncs, and `"Sync succeeded..."` for standard runs.
- `internal/engine/runner/runner_test.go`:
  - Align test assertions with contextual mode prefixes.
- `internal/server/server.go`:
  - In `ServeHTTP`, demote all successful HTTP access logging (`status < 400`) to `slog.LevelDebug`, retaining `slog.LevelWarn` / `slog.LevelError` for errors (`status >= 400`).
- `internal/server/server_test.go`:
  - Update `TestHTTPAccessLogLevel` to test that successful mutating requests log at `slog.LevelDebug`.
- `dashboard/frontend/js/logs.js`:
  - In `filteredBisyncLogs`, filter out `l.type === "http_access"` across Unified Ledger views.

### Technical Strategy & Data Flow

1. **Backend De-Noising**:
   - `monitor.go` job polling steps log at `slog.LevelDebug`.
   - `server.go` HTTP access middleware logs successful requests (`status < 400`) at `slog.LevelDebug`.
   - `runner_handlers.go` logs sync outcomes once via `r.Logger.RecordSyncRun`.
2. **Contextual Action Labeling**:
   - `actionLabel` determines mode: `if dryRun { return "Dry run" } else if resync { return "Resync" } return "Sync"`.
   - The user sees: `[OneDrive (Test)] Dry run succeeded in 2.0s (0 file(s) transferred)`.
3. **Frontend Signal Isolation**:
   - `filteredBisyncLogs` in `logs.js` filters out `http_access` records from the Unified Ledger, keeping HTTP server access logs strictly in diagnostic/daemon streams.

### Safety & Error Handling

- Respect strict `<400` LOC limit across all modified Go and JS files.
- Failed HTTP requests (`status >= 400`) and failed sync runs (`FAULT`, `ABORTED`) retain `WARN` / `ERROR` levels and visibility.
- Tracing is preserved: running `--debug` still records all debug logs to `bisync.log`.

## Verification Plan

- Unit tests:
  - `go test -count=1 ./internal/core/client/...`
  - `go test -count=1 ./internal/engine/runner/...`
  - `go test -count=1 ./internal/server/...`
- File size check:
  - `./scripts/check-loc.sh 400`
- End-to-end smoke test:
  - `go test -count=1 ./tests/e2e/...`
- Full quality check:
  - `./scripts/check.sh all`




