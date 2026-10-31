# Implementation Plan: Audit Ledger Noise Elimination, Unified JSON Logging (`bisync-hub-log.json`), and Proxy-Aware Security

## Proposed Changes

1. **Audit Ledger De-Spamming (Mutation-Only Invariant)**:
   - Eliminate `SYNC_ABORTED` spam from `audit.db`. The ledger must only record actual filesystem/cloud mutations (state changes).
   - Aborts due to lock contention, active rate-limit backoff, circuit breakers, or zero-transfer runs do NOT belong in `audit.db`; they belong strictly in `bisync-hub-log.json`.
   - In `internal/engine/runner/runner.go` and `runner_handlers.go`, remove calls to `recordAuditAborted`.
   - In `internal/server/handlers_auth.go` (`handleRemoteDedupe`), only emit `audit.ActionDedupeExecuted` if actual deletions/merging occurred (`mode != "list"` and `deleted > 0`). Dry-run / preview deduplication must NOT write to `audit.db`.
   - Ensure `audit.db` only receives mutations: `SYNC_EXECUTED` (when `files_synced > 0`), `DEDUPE_EXECUTED` (when `deleted > 0`), `CONFLICT_RESOLVED`, `DELETION_ACCEPTED`, `ROLLBACK_EXECUTED`.

2. **Unified JSON Logging (`bisync-hub-log.json`) & Dynamic Debug Mode**:
   - Change default log file in `internal/config/config.go` from `bisync.log` to `bisync-hub-log.json`.
   - Add dynamic `slog.LevelVar` to `internal/core/logger/logger.go` to support runtime level switching (Info vs. Debug).
   - Inject the shared `*logger.Logger` across the container (`app.Container`), making it available to all engine runners, schedulers, and servers.
   - Replace naked `log.Printf` across `runner.go`, `runner_handlers.go`, `monitor.go`, and `scheduler.go` with structured `slog` calls writing to `bisync-hub-log.json`.
   - Wire CLI `--debug` / `-v` flag and `DEBUG_MODE=true` environment variable to set `slog.LevelDebug` and pass `--verbose` (`-v`) to rclone bisync subprocesses.

3. **Dashboard Security (POSIX Permissions, Reverse Proxy Attribution & Optional API Key)**:
   - Call `syscall.Umask(0077)` at startup in `main.go` so all created files (`audit.db`, `profiles.json`, `bisync-hub-log.json`) have strict `0600`/`0700` POSIX permissions.
   - Extract user identity from proxy headers (`X-Forwarded-User`, `Remote-User`) in `internal/server/server.go` to attribute actions in the audit ledger.
   - Support optional `DASHBOARD_API_KEY` authentication middleware protecting mutating API routes (`POST`, `PUT`, `DELETE`).
   - Add lightweight HTTP JSON access logging middleware streaming directly into `bisync-hub-log.json`.

### Affected Components & Packages

- `internal/engine/runner/runner.go` & `runner_handlers.go`: Remove `recordAuditAborted`, gate audit ledger events strictly to file mutations.
- `internal/server/handlers_auth.go`: Gate `ActionDedupeExecuted` to `mode != "list"` and `deleted > 0`.
- `internal/config/config.go`: Update default log path to `bisync-hub-log.json`.
- `internal/core/logger/logger.go`: Add dynamic `slog.LevelVar`, support structured JSON output for all components.
- `internal/app/container.go`: Pass centralized logger into engine and server services.
- `internal/engine/monitor/monitor.go` & `internal/engine/scheduler/scheduler.go`: Replace `log.Printf` with structured logger.
- `main.go`: Apply `syscall.Umask(0077)`, wire `--debug` flag, pass logger to container.
- `internal/server/server.go` & `middleware.go`: Add HTTP request logging middleware, `X-Forwarded-User` actor extraction, and optional `DASHBOARD_API_KEY` validation.

### Technical Strategy & Data Flow

```
[ HTTP Client / Traefik Proxy ]
           │ (Headers: X-Forwarded-User, Authorization)
           ▼
[ Server Middleware ] ───▶ [ bisync-hub-log.json ] (HTTP access JSON entry)
           │
           ├── Validate DASHBOARD_API_KEY (if configured)
           ├── Extract actor: X-Forwarded-User || Remote-User || "USER"
           ▼
[ Engine / Runner ] ─────▶ [ bisync-hub-log.json ] (Component operational JSON logs)
           │
           │ (If files_synced > 0 || deleted > 0 || mutation)
           ▼
[ AuditService.RecordEvent ] ───▶ [ audit.db ] (Immutable forensic block)
```

### Safety & Error Handling

- **Umask Safety**: `syscall.Umask(0077)` ensures all newly written files default to private read/write for the executing user only.
- **Audit Invariant**: Zero-byte transfers or aborted attempts never pollute `audit.db`, preserving the cryptographic chain strictly for real changes.
- **Zero Global Pollution**: All logging is unified into a single rotating JSON file (`bisync-hub-log.json`) without external dependencies.

## Verification Plan

- Micro-Task 1: `go test -count=1 ./internal/engine/runner/... ./internal/services/... ./internal/server/...`
- Micro-Task 2: `go test -count=1 ./internal/core/logger/... ./internal/config/... ./internal/engine/...`
- Micro-Task 3: `go test -count=1 ./internal/server/... ./internal/core/security/...`
- Full Quality Gate: `./scripts/check.sh all`

