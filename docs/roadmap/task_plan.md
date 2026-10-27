# Task Plan: Phase 1 - Multi-Profile UI & Engine Cohesion

## Goal
Stabilize the Multi-Profile Workspace Manager feature across frontend and backend: eliminate Alpine.js reactive null pointer exceptions on modal close, unify engine and telemetry sync status with profile scheduler workers, bind the cockpit engine badge to dynamic execution states, and forward profile exclude rules to the sync runner.

## Tasks

- [x] Task 1: Fix Alpine `profileModal.editing` Null Dereference on Close & Reset
  - **Scope**: `dashboard/frontend/js/profiles.js`, `dashboard/frontend/partials/modals/modal_profiles.html`
  - **Verification**: `./scripts/check.sh templates` and browser check for clean console on modal open/close

- [x] Task 2: Expose Scheduler Worker Execution State in Profiles Subsystem
  - **Scope**: `internal/profiles/scheduler.go`, `internal/profiles/service.go`, `internal/profiles/scheduler_test.go`
  - **Verification**: `go test -count=1 ./internal/profiles/...`

- [x] Task 3: Unify Telemetry Sync Status with Profile Scheduler Workers
  - **Scope**: `internal/server/handlers_telemetry.go`, `internal/server/server.go`, `internal/server/handlers_telemetry_test.go`
  - **Verification**: `go test -count=1 ./internal/server/...`

- [x] Task 4: Bind Cockpit Status Badge to Active Profile & Engine State
  - **Scope**: `dashboard/frontend/partials/view_telemetry.html`, `dashboard/frontend/app.js`
  - **Verification**: `./scripts/check.sh templates` and `./scripts/check.sh loc`

- [x] Task 5: Forward Profile Exclude Patterns to SyncRunner CLI Payload
  - **Scope**: `internal/server/server.go`, `internal/core/compiler/compiler.go`, `internal/server/handlers_profiles_test.go`
  - **Verification**: `go test -count=1 ./internal/server/...` and `go test -count=1 ./internal/core/compiler/...`

- [x] Task 6: Pre-Flight Quality Gate Audit (LOC, Lints, Tests, Leaks)
  - **Scope**: Entire repository
  - **Verification**: `./scripts/check.sh all`

- [x] Task 7: Replace Browser Popups with Styled Confirmation Modal
  - **Scope**: `dashboard/frontend/partials/modals/modal_confirm.html`, `dashboard/frontend/index.html`, `dashboard/frontend/app.js`, `dashboard/frontend/js/logs.js`, `dashboard/frontend/js/explorer_actions.js`, `dashboard/frontend/js/profiles.js`, `dashboard/frontend/js/review.js`
  - **Verification**: `./scripts/check.sh templates` and `./scripts/check.sh loc`

- [x] Task 8: Dynamic Active Remote Badge & Topology Binding to Active Profile
  - **Scope**: `dashboard/frontend/js/remotes.js`, `dashboard/frontend/partials/view_remotes.html`, `dashboard/frontend/partials/view_explorer.html`, `dashboard/frontend/partials/view_telemetry.html`, `dashboard/frontend/js/explorer.js`
  - **Verification**: `./scripts/check.sh templates` and `./scripts/check.sh loc`

## Status Updates

- 2026-10-27: Initialized Phase 1 Task Plan from four-phase implementation plan.
- 2026-10-27: Completed Task 1. Replaced modal form conditional mount with `<template x-if>` and added `createDefaultProfile()` fallback to prevent Alpine.js reactive null dereference errors.
- 2026-10-27: Completed Task 2. Added `atomic.Bool` sync state tracking in profile workers and exposed `IsProfileRunning(id)` and `IsAnyRunning()` in Scheduler and ProfileService.
- 2026-10-27: Completed Task 3. Integrated ProfileService sync state into `handleTelemetry` and added `execution_status: SYNCING | IDLE` in `TelemetryService`. Verified with unit test in `handlers_telemetry_test.go`.
- 2026-10-27: Completed Task 4. Bound cockpit engine status badge to dynamic profile and engine syncing states in `app.js` and `view_telemetry.html`.
- 2026-10-27: Completed Task 5. Added `--exclude` pattern mapping in `compiler.go` (`_filter["ExcludeRule"]`) and forwarded profile exclude patterns in `server.go` executor. Verified with end-to-end unit test `TestProfiles_ExecutorExcludeRulesForwarding`.
- 2026-10-27: Completed Task 6. Verified entire repository with all 6 quality gate checks passing via `./scripts/check.sh all`. Built binary `bisync-hub`.
- 2026-10-27: Completed Task 7. Replaced native browser `confirm(...)` dialogs across Telemetry resync, File Explorer directory sync, profile rollback, and review decisions with dark-mode `modal_confirm.html` and `askConfirmation()`.
- 2026-10-27: Completed Task 8. Added `isRemoteActiveTarget(remote)` and dynamic topology bindings to ensure `ACTIVE TARGET` and Local/Remote topology reflect the currently selected profile. All quality gates verified clean (`./scripts/check.sh all`).


