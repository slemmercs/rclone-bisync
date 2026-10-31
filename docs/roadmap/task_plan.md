# Task Plan: Audit Ledger Noise Elimination, Unified JSON Logging (`bisync-hub-log.json`), and Proxy-Aware Security

## Goal
Enforce mutation-only invariant on the SQLite Audit Ledger (`audit.db`) by dropping spammy `SYNC_ABORTED` entries and non-mutating preview dedupe records, unify application logging into a rotating `bisync-hub-log.json` with dynamic debug level switching, and harden dashboard security via `umask 0077`, reverse proxy user attribution (`X-Forwarded-User`), and optional `DASHBOARD_API_KEY`.

## Tasks

- [x] Task 1: Audit Ledger De-Spamming & Mutation-Only Gating
  - **Scope**: `internal/engine/runner/runner.go`, `internal/engine/runner/runner_handlers.go`, `internal/engine/runner/runner_audit.go`, `internal/server/handlers_auth.go`, unit tests in `runner_test.go` and `handlers_audit_test.go`
  - **Verification**: `go test -count=1 ./internal/engine/runner/... ./internal/server/...`

- [ ] Task 2: Unified JSON Logging (`bisync-hub-log.json`) & Dynamic Debug Mode
  - **Scope**: `internal/config/config.go`, `internal/core/logger/logger.go`, `internal/app/container.go`, `internal/engine/runner/...`, `internal/engine/monitor/...`, `internal/engine/scheduler/...`, `main.go`
  - **Verification**: `go test -count=1 ./internal/core/logger/... ./internal/config/... ./internal/engine/...`

- [ ] Task 3: Dashboard Security: POSIX Umask, Reverse Proxy Attribution & Optional API Key
  - **Scope**: `main.go`, `internal/server/server.go`, `internal/server/middleware.go`, `internal/config/config.go`, `internal/server/server_test.go`
  - **Verification**: `go test -count=1 ./internal/server/...`

- [ ] Task 4: Full Pre-Flight Quality Gate & Release Readiness Validation
  - **Scope**: Entire repository test suite, linters, templates, and LOC budgets
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-10-31: Initialized plan from implementation_plan.md for Audit Ledger Noise Elimination, Unified JSON Logging, and Proxy-Aware Security.
- 2026-10-31: Completed Task 1: Eliminated `SYNC_ABORTED` entries and gated `DEDUPE_EXECUTED` to `deleted > 0` in `handlers_auth.go`, enforcing the mutation-only invariant on `audit.db` while preserving execution logs in `bisync.log`. Verified via `runner_test.go` and `handlers_audit_test.go`.

