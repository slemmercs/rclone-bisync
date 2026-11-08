# Task Plan: Sandbox Reliability & File Operations Remediation

## Goal
Resolve 7 issues from sandbox interactive testing: fix remote preview/download using `operations/getfile`, preserve folder containers on directory copy, fix destination path parsing on file moves, log resyncs in audit ledger, detect missing listings with a resync confirmation dialog, and isolate cache directories in sandbox.

## Tasks

- [x] Task 1: Fix Rclone RC File Preview & Download (operations/getfile & --rc-serve)
  - **Scope**: `internal/core/client/fs_read.go`, `internal/core/client/fs_read_test.go`, `internal/services/explorer_preview_test.go`, `scripts/run-sandbox.sh`
  - **Verification**: `go test -v -count=1 ./internal/core/client/... -run TestRcloneClient_Cat`

- [x] Task 2: Fix Directory Copying and Destination Path Placement
  - **Scope**: `internal/services/explorer_ops.go`, `internal/services/explorer_ops_test.go`
  - **Verification**: `go test -v -count=1 ./internal/services/... -run TestExplorerOps_Copy`

- [x] Task 3: Fix File Move / Rename Destination Parsing and Lucide Icon
  - **Scope**: `internal/services/explorer_move.go`, `internal/services/explorer_ops_test.go`, `dashboard/frontend/partials/view_explorer.html`, `dashboard/frontend/partials/components/component_context_menu.html`
  - **Verification**: `go test -v -count=1 ./internal/services/... -run TestExplorerOps_MoveItem`

- [x] Task 4: Audit Ledger Logging for Resync & User Actions
  - **Scope**: `internal/engine/runner/runner_handlers.go`, `internal/engine/runner/runner_test.go`
  - **Verification**: `go test -v -count=1 ./internal/engine/runner/... -run TestRunner_`

- [x] Task 5: Initial Sync / Missing Listings Detection & Resync Confirmation Modal
  - **Scope**: `internal/server/handlers_profiles.go`, `internal/server/handlers_profiles_test.go`, `dashboard/frontend/js/profiles_history.js`
  - **Verification**: `go test -v -count=1 ./internal/server/... -run TestHandler_ProfileSync`

- [x] Task 6: Sandbox Script Enhancements & Quality Verification
  - **Scope**: `scripts/run-sandbox.sh`, full quality check (`./scripts/check.sh all`)
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-11-08: Initialized 6-task plan addressing the 7 interactive sandbox issues.
- 2026-11-08: All 6 tasks completed and verified with all 7 quality gate checks passing and full e2e test suite passing.

