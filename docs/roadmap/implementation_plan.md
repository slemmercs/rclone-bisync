# Implementation Plan: Architecture Modularization & Unified Log Ledger Run-Level Consolidation

## Proposed Changes

### High-Level Strategy & Architecture Impact
1. **Architectural File Modularization (<400 LOC Ceiling)**:
   Decompose 10 priority files nearing the 400 LOC ceiling into cohesive, single-responsibility modules:
   - Backend Go files:
     - `internal/models/api.go` (396 LOC) $\rightarrow$ `api_explorer.go`, `api_diagnostics.go`, `api_logs.go`, and core `api.go`.
     - `internal/profiles/store.go` (395 LOC) $\rightarrow$ `store_history.go` (snapshot history & rollback) and `store.go`.
     - `internal/server/handlers_auth.go` (366 LOC) $\rightarrow$ `handlers_dedupe.go` (dedupe HTTP handler) and `handlers_auth.go`.
     - `internal/config/config.go` (378 LOC) $\rightarrow$ `env.go` (environment variable parsers) and `config.go`.
     - `internal/services/explorer_preview.go` (377 LOC) $\rightarrow$ `explorer_download.go`, `mime_types.go`, and `explorer_preview.go`.
     - `internal/mcp/tools_write.go` (386 LOC) $\rightarrow$ `tools_review.go`, `tools_dedupe.go`, and `tools_write.go`.
   - Frontend JS files:
     - `dashboard/frontend/js/utils.js` (391 LOC) $\rightarrow$ `dedupe_parser.js` and `utils.js`.
     - `dashboard/frontend/app.js` (392 LOC) $\rightarrow$ `app_state.js` and `app.js`.
     - `dashboard/frontend/js/explorer_actions.js` (394 LOC) $\rightarrow$ `explorer_preview.js` and `explorer_actions.js`.
     - `dashboard/frontend/js/logs.js` (392 LOC) $\rightarrow$ `logs_clustering.js`, `logs_daemon.js`, and `logs.js`.

2. **Unified Log Ledger Consolidation & Interactive Deep-Link Badges**:
   - Synthesize multiple log lifecycle traces of a sync execution (`run_id`) into a single primary row.
   - Combine the audit milestone (`action: SYNC_EXECUTED`, `seq_id`, `actor`) and telemetry (`run_id`, `duration_sec`, `files_synced`, `origin`).
   - Unpack nested `parsed.details` in `utils.js` so audit records have top-level `run_id`, `duration_sec`, etc.
   - Convert all badges on the consolidated ledger line to interactive navigation triggers:
     - `action` (`SYNC_EXECUTED`): opens Audit Drawer (`openAuditDrawer(entry)`).
     - `run_id` (`run:sync_261`): opens Correlated Daemon Logs modal (`openDaemonModal(entry)`).
     - `profile` (`[prof_sandbox]`): sets profile filter.
     - `duration`: displays elapsed time pill.
     - `files_synced`: displays transferred count pill.
     - `origin` (`UI`, `CLI`, `CRON`): displays source badge.
   - Suppress redundant polling `slog.Debug` traces from polluting `bisync.log`.

### Affected Components & Packages

- `internal/models/`:
  - `api_explorer.go`: File browsing, transfer, deletion, compare, and canary DTOs.
  - `api_diagnostics.go`: Diagnostic status enums, check structs, and diagnostics response.
  - `api_logs.go`: Paginated logs, daemon logs, audit records, and dedupe request/responses.
  - `api.go`: Core action, health, remotes, telemetry, and BisyncLogEntry.
- `internal/profiles/`:
  - `store_history.go`: `GetHistory` and `Rollback` methods.
  - `store.go`: Core profile persistence, locks, and validation.
- `internal/server/`:
  - `handlers_dedupe.go`: `handleDedupe` handler.
  - `handlers_auth.go`: Auth, profile actions, and tokens.
- `internal/config/`:
  - `env.go`: `str2bool`, `getEnv`, `getEnvInt`, `getEnvFloat`, `getEnvBool`, `splitArgs`, `LoadEnvFile`.
  - `config.go`: Config struct and core initialization.
- `internal/services/`:
  - `explorer_download.go`: `DownloadStream`.
  - `mime_types.go`: `DetectMimeType`, `isBinaryData`.
  - `explorer_preview.go`: `PreviewItem`.
- `internal/mcp/`:
  - `tools_review.go`: `bisync_resolve_review` and `handleResolveReview`.
  - `tools_dedupe.go`: `bisync_dedupe_by_hash` and `handleDedupeByHash`.
  - `tools_write.go`: Core write tools (`bisync_delete_item`, `bisync_purge_dir`, `bisync_copy_item`, etc.).
- `internal/engine/runner/`:
  - `runner_handlers.go`: Tag `RecordSyncRun` with `Action: "SYNC_EXECUTED"` and unify completion emission.
- `internal/core/client/`:
  - `monitor.go`: Streamline debug polling output to avoid `bisync.log` pollution.
- `dashboard/frontend/`:
  - `js/dedupe_parser.js`: `parseDedupeOutput`, `extractPathFromLogMsg`.
  - `js/app_state.js`: Baseline Alpine.js reactive state factory.
  - `js/explorer_preview.js`: Explorer preview modal and download logic.
  - `js/logs_clustering.js`: Run/job level consolidation and message normalization.
  - `js/logs_daemon.js`: Daemon log inspection modal handlers and exports.
  - `js/utils.js`: Unpack `parsed.details` in `parseJsonLogLine`.
  - `partials/patterns/pattern_timeline_ledger.html`: Consolidated row layout with interactive badges.
  - `index.html`: Register new modules.

### Safety & Error Handling

- **Zero Breaking Changes**: Package-internal Go splits maintain identical exported APIs, method receivers, and struct definitions.
- **Path Confinement**: Explorer preview, downloads, and dedupe retain strict jail confinement (`filepath.Rel` and `strings.HasPrefix(.., "..")`).
- **Audit Immutability**: Cryptographic hash chaining in SQLite audit ledger remains fully intact.
- **Frontend Isolation**: Sub-modules attach to `window` and merge cleanly into Alpine data model.

## Verification Plan

- Unit tests:
  - `go test -count=1 ./internal/models/...`
  - `go test -count=1 ./internal/profiles/...`
  - `go test -count=1 ./internal/server/...`
  - `go test -count=1 ./internal/config/...`
  - `go test -count=1 ./internal/services/...`
  - `go test -count=1 ./internal/mcp/...`
  - `go test -count=1 ./internal/engine/runner/...`
- Full Quality Gate:
  - `./scripts/check.sh loc 400` (all Go and JS files strictly < 400 lines)
  - `./scripts/check.sh templates`
  - `./scripts/check.sh all` (all 7 quality stages)

