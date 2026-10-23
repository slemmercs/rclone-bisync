
# Task Plan: Phase 4 — Interactive Sync Review & Deletion Diff Visualizer

## Goal
Implement an interactive synchronization review and resolution engine ("Review. Decide. Sync.") that empowers users to inspect, approve, or resolve mass deletion events and file conflicts before synchronization proceeds, replacing risky manual command-line `--force` workarounds with an auditable, rollback-capable dashboard workflow.

## Tasks
- [x] Task 1: Review Domain Models & Configuration Paths
  - **Scope**: Create `internal/models/review.go` defining all DTOs and enums; update `internal/config/config.go` with `ReviewDir` and `ActiveReviewFile`.
  - **Verification**: `go test -count=1 ./internal/models ./internal/config`
- [x] Task 2: Review Core Domain Package (FSM, Discovery & WAL Journal)
  - **Scope**: Implement `session.go` (state machine), `discovery.go` (candidate discovery), and `journal.go` (atomic persistence & append-only WAL) in `internal/review/`.
  - **Verification**: `go test -count=1 ./internal/review`
- [x] Task 3: Dual Backup Strategy Provider & Saga Resolution Executor
  - **Scope**: Implement `backup.go` (local zero-byte hardlink + cloud server-side copy + `statfs`), `executor.go` (reversible commands), and `rollback.go` in `internal/review/`.
  - **Verification**: `go test -count=1 ./internal/review -run TestExecutor`
- [ ] Task 4: Service Facade & Pre-Sync Safety Barrier
  - **Scope**: Implement `internal/services/review_service.go`, wire into `SyncService.HandleAction`, connect sync abort hooks in `internal/engine/runner/runner.go`.
  - **Verification**: `go test -count=1 ./internal/services`
- [ ] Task 5: Server REST Handlers & Route Integration
  - **Scope**: Implement `internal/server/handlers_review.go`, mount routes in `server.go`, add integration tests in `server_test.go`.
  - **Verification**: `go test -count=1 ./internal/server`
- [ ] Task 6: Frontend Review Module & Dual-Pane Explorer Highlighting
  - **Scope**: Create `dashboard/frontend/js/review.js`, update `dashboard/frontend/js/explorer.js` with color status badges, add Review Modal and banner to `dashboard/frontend/index.html`, wire into `dashboard/frontend/app.js`.
  - **Verification**: `./scripts/check.sh`
- [ ] Task 7: Full Quality Gate & End-to-End Verification
  - **Scope**: Run full `./scripts/check.sh` (>80% coverage across all packages, clean static analysis, native and cross-platform builds); finalize documentation.
  - **Verification**: `./scripts/check.sh`

## Next Up
- **Task 4: Service Facade & Pre-Sync Safety Barrier**

## Status Updates
- 2026-10-23: Phase 3 and Maintenance Milestone (Quality hardening, security jails, modularization) merged to `main`.
- 2026-10-23: Switched to feature branch `feat/interactive-sync-review`. Verified clean baseline via `./scripts/check.sh` (84.8% coverage). Initialized Phase 4 plan.
- 2026-10-23: Completed Task 1: Defined `ReviewSession`, `DeletionCandidate`, `ConflictCandidate`, `ResolutionAction`, `WALEntry`, and API DTOs in `internal/models/review.go`. Added review paths and backup TTL configuration to `internal/config/config.go`. Verified via `go test -count=1 ./internal/models ./internal/config` and `./scripts/check.sh`.
- 2026-10-23: Completed Task 2: Implemented `session.go` (formal FSM lifecycle, candidate decision staging), `discovery.go` (structured state and log-based deletion/conflict discovery), and `journal.go` (atomic persistence, WAL append/read, crash recovery reconciliation) in `internal/review/`. Verified via `go test -count=1 ./internal/review` (86.3% statement coverage) and `./scripts/check.sh` (85.0% total statement coverage).
- 2026-10-23: Completed Task 3: Implemented `backup.go` (dual local zero-byte hardlink + cloud server-side copy + statfs safety), `executor.go` (reversible resolution executor), and `rollback.go` (Saga compensation runner in reverse order) in `internal/review/`. Verified via `go test -count=1 ./internal/review -run TestExecutor` and `./scripts/check.sh` (84.0% repository statement coverage).














