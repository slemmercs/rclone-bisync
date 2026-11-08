# Task Plan: File Explorer Remote Target Synchronization & Directory Creation (Mkdir)

## Goal
Fix file explorer remote target desynchronization, safeguard operations against self-collisions, and add "New Folder" (mkdir) creation across local and remote storage.

## Tasks

- [x] Task 1: Domain Models & Explorer Mkdir Service Implementation
  - **Scope**: `internal/models/operations.go`, `internal/services/explorer_mkdir.go`, `internal/services/explorer_mkdir_test.go`, `internal/audit/taxonomy.go`
  - **Verification**: `go test -v -count=1 ./internal/services/... -run TestExplorerService_MakeDir`

- [ ] Task 2: Mkdir Server Handler & Identical Path Safeguards
  - **Scope**: `internal/server/handlers_operations.go`, `internal/server/handlers_operations_test.go`, `internal/services/explorer_ops.go`, `internal/services/explorer_move.go`
  - **Verification**: `go test -v -count=1 ./internal/server/... -run TestHandler_MakeDir`

- [ ] Task 3: Frontend Remote Target Synchronization & Mkdir Modal Integration
  - **Scope**: `dashboard/frontend/app.js`, `dashboard/frontend/js/explorer.js`, `dashboard/frontend/js/profiles.js`, `dashboard/frontend/js/explorer_operations.js`, `dashboard/frontend/partials/modals/modal_explorer_mkdir.html`, `dashboard/frontend/index.html`, `dashboard/frontend/partials/view_explorer.html`
  - **Verification**: `go test -v -count=1 ./tests/e2e/... -run TestDashboard_Smoke`

- [ ] Task 4: Sandbox Script Update & Quality Verification
  - **Scope**: `scripts/run-sandbox.sh`, full quality check (`./scripts/check.sh all`)
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-11-08: Initialized plan for Explorer Remote Target Synchronization & Mkdir creation.

