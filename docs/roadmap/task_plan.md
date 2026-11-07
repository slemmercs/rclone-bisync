# Task Plan: UI Suite Polish & Workflow Alignment

## Goal
Implement UI/UX refinements identified during testing: restructure Global Fleet Settings modal tabs into a multi-line pill layout, add empty password validation/warning when removing encryption, remove redundant Engine Transfer Telemetry card, streamline explorer row actions to Download and `...` buttons only, and align context menu Lucide icons with directional arrows.

## Tasks

- [x] Task 1: Global Fleet Settings Multi-Line Tabs & Passphrase Validation
  - **Scope**: `dashboard/frontend/partials/modals/modal_global_settings.html`, `dashboard/frontend/js/settings.js`, `internal/server/handlers_encryption.go`, `internal/server/handlers_encryption_test.go`
  - **Verification**: `go test -v -count=1 ./internal/server/... -run TestHandleRemoveConfigEncryption`

- [x] Task 2: Remove Redundant Engine Transfer Telemetry Card
  - **Scope**: `dashboard/frontend/partials/view_telemetry.html`
  - **Verification**: `./scripts/check.sh templates` and `./scripts/check.sh loc`

- [x] Task 3: Explorer Row Actions Declutter & Directional Context Menu Icons
  - **Scope**: `dashboard/frontend/partials/view_explorer.html`, `dashboard/frontend/partials/components/component_context_menu.html`
  - **Verification**: `./scripts/check.sh templates` and `./scripts/check.sh loc`

- [ ] Task 4: Full Quality Gate Verification & Smoke Testing
  - **Scope**: `tests/e2e/dashboard_smoke_test.go`
  - **Verification**: `./scripts/check.sh all` and `go test -v -count=1 ./tests/e2e/...`

## Status Updates

- 2026-11-07: Initialized UI Suite Polish task plan from approved implementation_plan.md.
- 2026-11-07: Completed Task 1 - Global Fleet Settings Multi-Line Tabs & Passphrase Validation. Verified with unit tests, templates check, and loc audit.
- 2026-11-07: Completed Task 2 - Remove Redundant Engine Transfer Telemetry Card. Verified with templates check and loc audit.
- 2026-11-07: Completed Task 3 - Explorer Row Actions Declutter & Directional Context Menu Icons. Verified with templates check and loc audit.

