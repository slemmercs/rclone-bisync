# Task Plan: Milestone 3 — Unified Cloud Remote & Sync Profile Provisioning Journey

## Goal
Deliver a unified, end-to-end operational journey for creating, configuring, and managing Rclone remotes and their corresponding sync profiles via both a browser-native UI wizard and the native MCP tool suite, plus complete parity for explorer file operations.

## Tasks

- [x] Task 1: Remote Wizard Service & REST API Endpoints
  - **Scope**: `internal/models/provider.go`, `internal/services/remote_wizard_service.go`, `internal/services/remote_wizard_service_test.go`, `internal/server/handlers_remotes_mgmt.go`, `internal/server/handlers_remotes_mgmt_test.go`, `internal/server/routes.go`
  - **Verification**: `go test -count=1 ./internal/services/... -run TestRemoteWizard && go test -count=1 ./internal/server/... -run TestRemoteMgmt`

- [x] Task 2: MCP Remote & Profile Lifecycle Tools
  - **Scope**: `internal/mcp/tools_remote.go`, `internal/mcp/tools_remote_test.go`, `internal/mcp/tools_profile.go`, `internal/mcp/tools_profile_test.go`, `internal/mcp/server.go`
  - **Verification**: `go test -count=1 ./internal/mcp/... -run "TestRemoteTools|TestProfileTools"`

- [x] Task 3: MCP File Explorer Action Tools (Parity with Milestone 2)
  - **Scope**: `internal/mcp/tools_explorer_ops.go`, `internal/mcp/tools_explorer_delete.go`, `internal/mcp/tools_explorer_ops_test.go`, `internal/mcp/server.go`
  - **Verification**: `go test -count=1 ./internal/mcp/... -run TestExplorerOpsTools`

- [ ] Task 4: Frontend Remote Creation Wizard Modal & Bridge UI
  - **Scope**: `dashboard/frontend/partials/modals/modal_create_remote.html`, `dashboard/frontend/js/remotes_wizard.js`, `dashboard/frontend/partials/view_remotes.html`, `dashboard/frontend/partials/modals/modal_profiles.html`, `dashboard/frontend/app.js`
  - **Verification**: `./scripts/check.sh template-lint`

- [ ] Task 5: MCP Inspector Showcase, E2E Journey Tests & Pre-commit Quality Gate
  - **Scope**: `dashboard/frontend/partials/modals/modal_mcp.html`, `site/usps/06-mcp-agent-server.md`, `tests/e2e/sandbox_ui_journey_test.go`
  - **Verification**: `./scripts/check.sh all && go test -count=1 ./tests/e2e/...`

## Status Updates

- 2026-11-10: Initialized plan from implementation_plan.md.
- 2026-11-10: Task 1 verified: Implemented RemoteWizardService, provider categorization, DTOs, audit taxonomy, and REST endpoints (/api/remotes/providers, POST /api/remotes/create, DELETE /api/remotes/{name}). All unit tests passed.
- 2026-11-10: Task 2 verified: Implemented MCP remote & profile lifecycle tools (bisync_list_providers, bisync_create_remote, bisync_delete_remote, bisync_create_profile, bisync_update_profile, bisync_delete_profile, bisync_rollback_profile), OAuth pending status detection, ticket confirmation safeguards, and comprehensive unit tests. All tests passed.
- 2026-11-10: Task 3 verified: Implemented MCP file explorer action tools (bisync_preview_file, bisync_mkdir, bisync_copy_item, bisync_move_item, bisync_delete_item, bisync_purge_dir) with canary guards, path traversal enforcement, ticket confirmations, and audit ledger tracking. All unit tests passed.

