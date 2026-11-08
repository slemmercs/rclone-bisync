# Task Plan: Pure Profile Sync Staging & Remediation Workspace

## Goal
Implement Milestone 2: Pure Profile Sync Staging & Remediation Workspace with atomic move/purge operations, canary inviolability, non-blocking upload dock, cloud public links, and deep-linked review triage.

## Tasks

- [x] Task 1: Core RC Client Operations Wrappers & Data Models
  - **Scope**: `internal/models/operations.go`, `internal/models/upload.go`, `internal/core/client/operations.go`, `internal/core/client/operations_test.go`
  - **Verification**: `go test -v -count=1 ./internal/core/client/... -run "TestMove|TestPurge|TestPublicLink"`

- [x] Task 2: Service Layer Profile-Bound Operations (Move, Purge, Link)
  - **Scope**: `internal/services/explorer_move.go`, `internal/services/explorer_purge.go`, `internal/services/explorer_ops_test.go`
  - **Verification**: `go test -v -count=1 ./internal/services/... -run "TestExplorerOps"`

- [ ] Task 3: HTTP API Handlers & Routing for Workspace Operations
  - **Scope**: `internal/server/handlers_operations.go`, `internal/server/handlers_operations_test.go`, `internal/server/server.go`
  - **Verification**: `go test -v -count=1 ./internal/server/... -run TestHandleOperations`

- [ ] Task 4: Lightweight Upload Dock Service & Mid-Sync Deferral
  - **Scope**: `internal/services/upload_dock.go`, `internal/services/upload_dock_test.go`, `internal/server/handlers_operations.go`
  - **Verification**: `go test -v -count=1 ./internal/services/... -run TestUploadDock`

- [ ] Task 5: Frontend Profile-Bound Workspace & Staging Layout
  - **Scope**: `dashboard/frontend/partials/view_explorer.html`, `dashboard/frontend/partials/components/component_explorer_toolbar.html`, `dashboard/frontend/js/explorer.js`, `dashboard/frontend/js/explorer_actions.js`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh loc`

- [ ] Task 6: Sync Review Remediation Bridge & Ambient Candidate Highlighting
  - **Scope**: `dashboard/frontend/partials/modals/modal_review.html`, `dashboard/frontend/partials/components/component_context_menu.html`, `dashboard/frontend/js/review.js`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh loc`

- [ ] Task 7: Full Quality Gates & E2E Smoke Testing
  - **Scope**: `tests/e2e/dashboard_smoke_test.go`
  - **Verification**: `./scripts/check.sh all && go test -v -count=1 ./tests/e2e/...`

## Status Updates

- 2026-11-07: Initialized Milestone 2 Task Plan from approved Option 1 implementation plan.
