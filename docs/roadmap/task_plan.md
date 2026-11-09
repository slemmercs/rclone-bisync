# Task Plan: Explorer Capabilities, Audit Parity, Initial-Sync Safety & Log Resilience

## Goal
Resolve 4 user-reported issues: (1) provider-aware "Open on Remote" context menu filtering, (2) File Explorer manual operations audit ledger logging, (3) universal baseline `--resync` enforcement on uninitialized profiles with sandbox cache isolation, and (4) resilient correlated daemon log retrieval.

## Tasks

- [x] Task 1: Backend Remote Capability Detection (`operations/fsinfo` & `can_public_link`)
  - **Scope**: `internal/core/client/remotes.go`, `internal/services/telemetry_remotes.go`, `internal/services/telemetry_service_test.go`
  - **Verification**: `go test -v -count=1 ./internal/services/... -run TestGetRemotesSummary`

- [x] Task 2: Frontend Context Menu Filtering & Dynamic Capability Degradation
  - **Scope**: `dashboard/frontend/js/explorer_operations.js`, `dashboard/frontend/partials/components/component_context_menu.html`, `tests/e2e/dashboard_smoke_test.go`
  - **Verification**: `go test -v -count=1 ./tests/e2e/... -run TestDashboard_HeadlessChromeSmoke`

- [x] Task 3: Audit Ledger Parity for File Explorer Operations
  - **Scope**: `internal/audit/taxonomy.go`, `internal/server/handlers_explorer.go`, `dashboard/frontend/js/audit.js`, `internal/server/handlers_explorer_test.go`
  - **Verification**: `go test -v -count=1 ./internal/server/... -run "TestExplorerAudit|TestHandleExplorerUpload"`

- [x] Task 4: Universal Uninitialized Profile Detection & Sandbox Cache Isolation
  - **Scope**: `internal/server/handlers_sync.go`, `dashboard/frontend/app.js`, `scripts/run-sandbox.sh`, `internal/server/handlers_profiles_sync_test.go`
  - **Verification**: `go test -v -count=1 ./internal/server/... -run "TestHandler_ProfileSync|TestHandleAction_RequiresResync"`

- [x] Task 5: Correlated Daemon Log Resilience & Error Formatting
  - **Scope**: `internal/server/handlers_telemetry.go`, `dashboard/frontend/js/logs.js`, `internal/server/handlers_telemetry_test.go`
  - **Verification**: `go test -v -count=1 ./internal/server/... -run TestHandler_DaemonLogs`

- [x] Task 6: Quality Gate Verification & Pre-Commit Check
  - **Scope**: Full repository quality check suite
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-11-08: Task 1 verified and committed (`git commit 6c2689c`).
- 2026-11-09: Initialized updated comprehensive plan covering all 4 interactive sandbox issues.
- 2026-11-09: Task 2 verified and committed (`git commit 2b45e2a`).
- 2026-11-09: Task 3 verified and committed (`git commit 46b4122`).
- 2026-11-09: Task 4 verified and committed (`git commit 3e11aa7`).
- 2026-11-09: Task 5 verified and committed (`git commit 816c1a2`).
- 2026-11-09: Task 6 verified: all 7 quality gate checks in `./scripts/check.sh all` passed.


