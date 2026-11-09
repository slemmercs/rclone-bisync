# Task Plan: Styled UI Confirm Modal & Unified Bisync Log Pipeline

## Goal
Replace unstyled browser `window.confirm(...)` popups with the custom UI confirmation modal (`askConfirmation`), and harden the sync logging pipeline and sandbox environment so all sync lifecycle events (requires resync, running, and completion) stream cleanly into `bisync.log` and the dashboard Unified Ledger.

## Tasks

- [x] Task 1: Migrate Frontend Confirmation Popups to Custom UI Modal (`askConfirmation`)
  - **Scope**: `dashboard/frontend/app.js`, `dashboard/frontend/js/profiles_history.js`
  - **Verification**: `go test -count=1 ./tests/e2e/...`

- [x] Task 2: Standardize Sandbox & Default Log File Naming (`bisync.log`)
  - **Scope**: `scripts/run-sandbox.sh`, `internal/config/config.go`
  - **Verification**: `go test -count=1 ./internal/config/...`

- [x] Task 3: Backend Intercept & Resync Notice Logging
  - **Scope**: `internal/server/handlers_sync.go`, `internal/server/handlers_profiles.go`, `internal/server/handlers_profiles_sync_test.go`
  - **Verification**: `go test -count=1 ./internal/server/... -run "TestHandleAction_RequiresResync|TestHandler_ProfileSync"`

- [ ] Task 4: Engine Sync Running Milestone Logging
  - **Scope**: `internal/engine/runner/runner.go`, `internal/engine/runner/runner_test.go`
  - **Verification**: `go test -count=1 ./internal/engine/runner/... -run TestRunWithError`

- [ ] Task 5: Quality Gate Verification & Pre-Commit Validation
  - **Scope**: Full repository check suite
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-11-09: Initialized task plan from implementation_plan.md.
- 2026-11-09: Task 1 verified: Migrated uninitialized resync confirmation dialogs in `app.js` and `profiles_history.js` to styled `confirmModal`. E2E smoke tests passed.
- 2026-11-09: Task 2 verified: Standardized default LogFile to bisync.log in config, sandbox setup, and comments. All tests in internal/config passed.
- 2026-11-09: Task 3 verified: Added REQUIRES_RESYNC notice logging to bisync.log on intercepted uninitialized sync triggers in handlers_sync.go and handlers_profiles.go. Tests in internal/server passed.
