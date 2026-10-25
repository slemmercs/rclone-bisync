# Task Plan: Full-Codebase Modularization (<400 LOC Target)

## Goal
Decompose all 5 files identified in the repository LOC audit (`internal/services/sync_service.go`, `internal/services/explorer_service.go`, `internal/engine/runner/runner.go`, `dashboard/frontend/app.js`, and `dashboard/frontend/js/explorer.js`) so that zero files in the repository exceed 400 LOC while preserving 100% test passing and >88% statement coverage.

## Tasks

- [x] Task 1: Modularize `SyncService` & Low-level Utilities
  - **Scope**: Extract `TailFile` and `TailJSONLog` into `internal/services/filetail.go`. Extract `HandleAction` and `runBisyncSystemd` into `internal/services/sync_actions.go`. Move `FileCacheItem` and `LiveHealthCache` into `internal/services/telemetry_service.go`.
  - **Verification**: `go test -count=1 ./internal/services/...`

- [x] Task 2: Modularize `ExplorerService` & Security Jails
  - **Scope**: Extract `IsLocalPathJailed` and jail confinement logic into `internal/services/explorer_jail.go`. Extract `CopyItem` and `DeleteItem` into `internal/services/explorer_ops.go`. Keep browsing and initialization in `internal/services/explorer_service.go`.
  - **Verification**: `go test -count=1 ./internal/services/...`

- [x] Task 3: Decompose `SyncRunner` Engine Pipeline
  - **Scope**: Extract `handleSyncSuccess` and `handleSyncFailure` into `internal/engine/runner/runner_handlers.go`. Extract `recordAuditExecuted` and `recordAuditAborted` into `internal/engine/runner/runner_audit.go`. Keep concise `Run()` orchestrator in `runner.go`.
  - **Verification**: `go test -count=1 ./internal/engine/runner/...`

- [x] Task 4: Modularize Dashboard Frontend Architecture
  - **Scope**: Relocate review getters into `dashboard/frontend/js/review.js`. Extract daemon/bisync log modal and parsing into `dashboard/frontend/js/logs.js`. Extract 2-step delete modal and drag-and-drop into `dashboard/frontend/js/explorer_actions.js`. Update `dashboard/frontend/app.js` and `dashboard/frontend/index.html`.
  - **Verification**: Syntax check via `./scripts/check.sh` (`esbuild` and `tidy`).

- [x] Task 5: Repository-Wide Verification & Quality Gate
  - **Scope**: Audit all file sizes using `./scripts/check-loc.sh 400` and execute `./scripts/check.sh` (8-stage quality gate).
  - **Verification**: `./scripts/check-loc.sh 400 && ./scripts/check.sh`

## Status Updates

- 2026-10-25: Created fresh branch `refactor/codebase-modularization-under-400-loc` and initialized plans.
- 2026-10-25: Completed Tasks 1-4. All Go, JS, and HTML files now under 400 LOC.
- 2026-10-25: Completed Task 5. Full 8-stage quality gate passed (`./scripts/check.sh`) with 88.2% test coverage and 0 files exceeding 400 LOC. All tasks complete.

