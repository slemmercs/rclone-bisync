# Task Plan: Comprehensive HTTP Access Log Noise Demotion & Unified Ledger Filter

## Goal
Demote all successful read-only HTTP access requests (`GET`, `HEAD` with `status < 400`) to `slog.Debug`, and filter routine `http_access` records in the UI unified ledger so `bisync.log` displays only meaningful sync events, audit records, and operational mutations.

## Tasks

- [x] Task 1: Demote all routine read-only HTTP access logs to `slog.Debug` in Server Middleware & Update Tests
  - **Scope**: `internal/server/server.go`, `internal/server/server_test.go`
  - **Verification**: `go test -count=1 ./internal/server/...` and `./scripts/check.sh loc 400`

- [x] Task 2: Filter routine read-only `http_access` logs in Frontend Unified Ledger View
  - **Scope**: `dashboard/frontend/js/logs.js`, `dashboard/frontend/js/utils.js`
  - **Verification**: `go test -count=1 ./tests/e2e/...` and `./scripts/check.sh loc 400`

- [ ] Task 3: Full Quality Gate Verification & Release
  - **Scope**: Entire repository
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-10-31: Initialized plan from implementation_plan.md to eliminate all static asset and read-only GET/HEAD request noise from `bisync.log`.
- 2026-10-31: Completed Task 1: Demoted all successful read-only HTTP access requests (`GET`/`HEAD` with `status < 400`, including static assets, remotes, profiles, and queries) to `slog.Debug`, logging mutating requests at `slog.Info` and error status at `slog.Warn`. Updated `TestHTTPAccessLogLevel`. Compacted `internal/server/server.go` to 385 LOC (<400 budget). Verified with `go test -count=1 ./internal/server/...` and `./scripts/check.sh loc 400`.
- 2026-10-31: Completed Task 2: Filtered routine read-only `http_access` lines in `filteredBisyncLogs` and `parseJsonLogLine` (`logs.js` and `utils.js`), guaranteeing that historical or debug access entries for static assets or GET queries never clutter the UI unified business ledger. Verified with `go test -count=1 ./tests/e2e/...` and `./scripts/check.sh loc 400`.


