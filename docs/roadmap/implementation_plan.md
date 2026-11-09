# Implementation Plan: Styled UI Confirm Modal & Unified Bisync Log Pipeline

## Overview
Replace the native browser `window.confirm(...)` popup with the existing dark-theme styled UI confirmation modal (`askConfirmation(...)` / `confirmModal`) when initial synchronization requires `--resync`. Additionally, harden the sync execution logging pipeline and sandbox configuration so that sync actions (initial resync requirement notices, sync start milestones, and completion/failure summaries) reliably stream into `bisync.log` and appear immediately in the Unified Ledger timeline.

## Proposed Changes

### 1. Frontend Confirmation Modal Migration
- Replace `window.confirm(...)` calls in `dashboard/frontend/app.js` and `dashboard/frontend/js/profiles_history.js` with `this.askConfirmation(...)`.
- Pass styled options: title `"Initial Resync Required"`, icon `"sync"`, action text `"Proceed with Resync"`, and proper async callbacks.

### 2. Log File Naming Parity & Sandbox Alignment
- Align `scripts/run-sandbox.sh` to use `LOG_FILE=$SANDBOX_DIR/bisync.log` (instead of `bisync-hub-log.json`) and initialize `touch "$SANDBOX_DIR/bisync.log"`.
- Set default fallback in `internal/config/config.go` to `filepath.Join(baseDir, "bisync.log")`.

### 3. Backend Intercept & Engine Sync Logging
- In `internal/server/handlers_sync.go` and `internal/server/handlers_profiles.go`, log a `REQUIRES_RESYNC` notice into `s.AuditService.Logger` whenever a sync request on an uninitialized profile is intercepted.
- In `internal/engine/runner/runner.go` (`RunWithError`), log an initial `RUNNING` status entry to `r.Logger` when the sync job starts.

## Affected Components & Packages
- `dashboard/frontend/app.js`: `triggerAction`
- `dashboard/frontend/js/profiles_history.js`: `triggerProfileSync`
- `scripts/run-sandbox.sh`: sandbox env and file setup
- `internal/config/config.go`: default `LogFile` fallback
- `internal/server/handlers_sync.go`: `handleAction`
- `internal/server/handlers_profiles.go`: `handleProfileSync`
- `internal/engine/runner/runner.go`: `RunWithError`
- `internal/server/handlers_profiles_sync_test.go`: test coverage for resync notice logging
- `internal/engine/runner/runner_test.go`: test coverage for running milestone logging

## Verification Plan
- Unit tests:
  - `go test -count=1 ./internal/server/... -run TestHandleAction_RequiresResync`
  - `go test -count=1 ./internal/server/... -run TestHandleProfileSync`
  - `go test -count=1 ./internal/engine/runner/... -run TestRunWithError`
- E2E smoke test:
  - `go test -count=1 ./tests/e2e/...`
- Full check suite:
  - `./scripts/check.sh all`
