# Implementation Plan: Full-Codebase Modularization (<400 LOC Target)

## Proposed Changes

Decompose all 5 files identified in the repository LOC audit (`internal/services/sync_service.go`, `internal/services/explorer_service.go`, `internal/engine/runner/runner.go`, `dashboard/frontend/app.js`, and `dashboard/frontend/js/explorer.js`) into cohesive, single-responsibility sub-components so that **zero files in the repository exceed 400 LOC**, while preserving 100% API compatibility, error handling, safety jails, and >88% test coverage.

### Affected Components & Packages

#### 1. Backend Service Layer (`internal/services/`)
- `sync_service.go` (460 LOC -> ~170 LOC): Core struct, constructor, `IsSyncing()`, and delegation facade.
- `filetail.go` (New, ~110 LOC): `TailFile` and `TailJSONLog` backwards-seeking I/O utilities.
- `sync_actions.go` (New, ~150 LOC): `HandleAction` and `runBisyncSystemd`.
- `telemetry_service.go`: Absorb `FileCacheItem` and `LiveHealthCache` struct definitions.
- `explorer_service.go` (438 LOC -> ~130 LOC): Core struct, constructor, and `BrowsePath`.
- `explorer_jail.go` (New, ~85 LOC): `IsLocalPathJailed` and path confinement validation.
- `explorer_ops.go` (New, ~225 LOC): `CopyItem` and `DeleteItem` with canary checks.

#### 2. Core Engine Pipeline (`internal/engine/runner/`)
- `runner.go` (412 LOC -> ~160 LOC): Core struct, constructor, and concise `Run()` pipeline.
- `runner_handlers.go` (New, ~175 LOC): `handleSyncSuccess` and `handleSyncFailure`.
- `runner_audit.go` (New, ~65 LOC): `recordAuditExecuted` and `recordAuditAborted`.

#### 3. Frontend Architecture (`dashboard/frontend/`)
- `app.js` (522 LOC -> ~220 LOC): Central Alpine.js state orchestrator and lifecycle polling.
- `dashboard/frontend/js/review.js` (309 LOC -> ~385 LOC): Absorb review computed getters (`reviewCandidates`, `filteredReviewCandidates`, etc.).
- `dashboard/frontend/js/logs.js` (New, ~160 LOC): Bisync/daemon log getters and daemon modal state/actions.
- `dashboard/frontend/js/explorer.js` (446 LOC -> ~340 LOC): Browsing, sorting, canary provisioning, item transfer.
- `dashboard/frontend/js/explorer_actions.js` (New, ~110 LOC): 2-step deletion modal and drag-and-drop handlers.
- `dashboard/frontend/index.html`: Include new script tags for `logs.js` and `explorer_actions.js`.

### Technical Strategy & Data Flow

- Retain identical public Go method receiver signatures (`*SyncService`, `*ExplorerService`, `*SyncRunner`) in the same packages to prevent call-site breakage.
- Maintain Alpine.js module composition via `window.LogsModule` and `window.ExplorerActionsModule`.
- Keep symlink resolution and path jail confinement strictly intact in `explorer_jail.go`.

## Verification Plan

- `go test -count=1 ./internal/services/...`
- `go test -count=1 ./internal/engine/runner/...`
- `go test -count=1 ./internal/server/...`
- `./scripts/check-loc.sh 400` (must report zero files exceeding 400 LOC).
- `./scripts/check.sh` (complete 8-stage pre-flight quality gate).

