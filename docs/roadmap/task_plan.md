# Task Plan: Coverage Buffer (>86%), Multi-Upload, File Ops Filter, & File-Level Deep Nav

## Goal
Elevate test coverage to >86.0% (resolving the nightly deep quality GitHub Actions failure `84.9% < 85.0%`), and implement multi-file batch uploads, file-level deep navigation/pulsing in subdirectories, and strict "File Ops" log filtering.

## Tasks

- [x] Task 1: Boost Unit Test Coverage Buffer (>86.0%) Across Models, Services, and Server
  - **Scope**: `internal/models/provider_test.go`, `internal/services/explorer_ops_test.go`, `internal/services/explorer_preview_test.go`, `internal/server/handlers_explorer_test.go`
  - **Verification**: `MIN_COVERAGE=85.0 RUN_RACE=true ./scripts/check.sh test`

- [x] Task 2: Multi-File Selection & Desktop Drag-and-Drop Batch Upload
  - **Scope**: `dashboard/frontend/partials/view_explorer.html`, `dashboard/frontend/js/explorer_actions.js`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh loc 400`

- [ ] Task 3: Audit `item_path` Resolution, Subfolder Deep Navigation & Row Pulsing
  - **Scope**: `internal/server/handlers_explorer.go`, `dashboard/frontend/js/utils.js`, `dashboard/frontend/js/logs.js`
  - **Verification**: `go test -count=1 ./internal/server -run TestHandleExplorerUpload && ./scripts/check.sh loc 400`

- [ ] Task 4: Strict File/Folder Operation Log Filtering in Log Dock and Timeline Ledger
  - **Scope**: `dashboard/frontend/js/log_dock.js`, `dashboard/frontend/partials/components/component_log_dock.html`, `dashboard/frontend/partials/patterns/pattern_timeline_ledger.html`, `dashboard/frontend/js/logs.js`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh loc 400`

- [ ] Task 5: End-to-End Headless Smoke Tests & Full Quality Gate Pass
  - **Scope**: `tests/e2e/dashboard_smoke_test.go`
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-11-13: Initialized task plan from implementation_plan.md.
- 2026-11-13: Completed Task 1 - Added unit tests in `models`, `services`, `server`, `audit`, `config`, and `core/client`, elevating test statement coverage to 86.0% under `-race` to safely guard against CI threshold failures. Verified with `MIN_COVERAGE=85.0 RUN_RACE=true ./scripts/check.sh test`.

- 2026-11-13: Completed Task 2 - Added `multiple` attribute to explorer file inputs and upgraded `handleFileUpload` & `onDrop` for desktop file drops and batch uploading with progress toasts and LOC < 400 compliance.


