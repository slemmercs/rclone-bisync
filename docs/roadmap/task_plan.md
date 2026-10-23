
# Task Plan: Phase 4 — Interactive Sync Review & Deletion Diff Visualizer

## Goal
Implement the **Interactive Sync Review & Resolution Workflow** ([`docs/design/interactive-sync-review.md`](../design/interactive-sync-review.md)). When Rclone Bisync encounters dangerous synchronization conditions (`TOO_MANY_DELETES` or unresolved file conflicts), Bisync Hub captures the event, provisions an atomic, persistent `ReviewSession`, visualizes affected items in the Dual-Pane Explorer, allows granular per-item user decisions (`keep-local`, `keep-remote`, `keep-both`, `accept-delete`, `restore-item`), stages backups, and supports rollback prior to final synchronization.

## Tasks
- [ ] Task 1: Review Domain Models & Contract Schemas
  - **Scope**: Create `internal/models/review.go` defining `ReviewSession`, `DeletionCandidate`, `ConflictCandidate`, `ResolutionAction`, `ReviewSessionStatus`, and resolution action types.
  - **Verification**: `go test -count=1 ./internal/models`
- [ ] Task 2: Review Service Engine & Persistent Journal
  - **Scope**: Create `internal/services/review_service.go` with thread-safe session lifecycle, atomic JSON journal persistence (`.tmp.<pid>` + `os.Rename`), crash recovery on startup, and pre-sync validation barrier.
  - **Verification**: `go test -count=1 ./internal/services -run TestReviewService`
- [ ] Task 3: Candidate Discovery & Abort Interceptor
  - **Scope**: Enhance `internal/services/sync_service.go` and `internal/engine/runner/runner.go` to intercept `TOO_MANY_DELETES` and conflict aborts, extract affected paths from bisync execution logs, and automatically provision a populated `ReviewSession`.
  - **Verification**: `go test -count=1 ./internal/services -run TestReviewService_CandidateDiscovery`
- [ ] Task 4: Resolution Executor, Backup Staging & Rollback Engine
  - **Scope**: Implement execution logic in `ReviewService` utilizing Rclone RC operations, pre-mutation file backups into `.bisync-backup/<session-id>/`, and atomic rollback to restore backups.
  - **Verification**: `go test -count=1 ./internal/services -run TestReviewService_ExecuteAndRollback`
- [ ] Task 5: REST API Endpoints & Request Routing
  - **Scope**: Implement `GET /api/review/session`, `POST /api/review/action`, `POST /api/review/execute`, `POST /api/review/rollback`, and `POST /api/review/dismiss` in `internal/server/server.go`.
  - **Verification**: `go test -count=1 ./internal/server -run TestReviewEndpoints`
- [ ] Task 6: Frontend Dual-Pane Explorer Highlighting & Interactive Review Modal
  - **Scope**: Update `dashboard/frontend/` (`index.html`, `js/explorer.js`, `js/app.js`) with visual highlighting (Red=deletion, Amber=conflict, Green=new) and dedicated Review Modal.
  - **Verification**: `go test -count=1 ./internal/server -run TestStaticAndIndexEmbedding`
- [ ] Task 7: Full Verification Gate & Service Deployment
  - **Scope**: Run full `./scripts/check.sh` quality gate, verify statement test coverage >80%, compile `bin/rclone-bisync`, deploy to user service, and verify live endpoints.
  - **Verification**: `./scripts/check.sh` && `systemctl --user status rclone-dashboard.service`

## Status Updates
- 2026-10-23: Phase 3 (OAuth Reconnect, SyncService modularization PR #18, Rebrand to Bisync Hub PR #20) completed and merged into `main`.
- 2026-10-23: Initialized Phase 4 Interactive Sync Review task plan based on `docs/design/interactive-sync-review.md`.












