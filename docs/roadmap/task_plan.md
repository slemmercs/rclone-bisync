# Task Plan: Deduplication Correlated Log Isolation

## Goal
Isolate correlated logs by `run_id` so that viewing deduplication logs (or specific run logs) displays strictly the corresponding operation log lines, eliminating user overwhelm caused by falling back to the entire unrelated daemon log tail.

## Tasks

- [x] Task 1: Deduplication Execution Logging in `handlers_auth.go`
  - **Scope**: `internal/server/handlers_auth.go`, `internal/server/handlers_audit_test.go`
  - **Verification**: `go test -count=1 ./internal/server/...`

- [x] Task 2: Implement `FilterLogsByRunID` in `LogService`
  - **Scope**: `internal/services/log_service.go`, `internal/services/log_service_test.go`
  - **Verification**: `go test -count=1 ./internal/services/...`

- [x] Task 3: Correlate `run_id` in `handleDaemonLogs`
  - **Scope**: `internal/server/handlers_telemetry.go`, `internal/server/server_test.go`
  - **Verification**: `go test -count=1 ./internal/server/...`

- [ ] Task 4: UI Modal Polish & Contextual Wiring
  - **Scope**: `dashboard/frontend/partials/modals/modal_daemon.html`, `dashboard/frontend/js/remotes.js`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh loc`

- [ ] Task 5: Full Pre-Flight Quality Gate & Release Readiness Validation
  - **Scope**: Full test suite and linting
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-10-31: Initialized plan from implementation_plan.md for Deduplication Correlated Log Isolation.
- 2026-10-31: Completed Task 1: Enabled guaranteed structured deduplication log stream recording in `handlers_auth.go` across all execution modes (including `list` preview mode) and added unit tests in `handlers_audit_test.go`.
- 2026-10-31: Completed Task 2: Implemented `FilterLogsByRunID` and `GetRunTimeWindow` in `LogService` with secret masking and unit test coverage in `log_service_runid_test.go`.
- 2026-10-31: Completed Task 3: Wired `run_id` correlation in `handleDaemonLogs` to route `dedupe-` runs to `bisync.log` via `FilterLogsByRunID`, resolve sync run windows, and prevent falling back to unrelated daemon tails. Added unit tests in `server_test.go`.
