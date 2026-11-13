# Task Plan: Emit Synced Files in Log Dock with On-Demand Pagination

## Goal
Extract and surface individual transferred files from bisync executions in the Live Sync Log Dock with deep navigation pills, 50-file capping safety, and on-demand pagination controls.

## Tasks

- [x] Task 0: Branch Switch & Merge Prerequisite
  - **Scope**: Git worktree (`feat/deep-path-nav-log-dock`)
  - **Action**: Fast-forward merge `fix/bisync-transfer-stats-and-ledger` into `feat/deep-path-nav-log-dock`.
  - **Verification**: `git status && git log -n 3 --oneline`

- [x] Task 1: Transfer File Parser & Client Extraction (`internal/core/client/`)
  - **Scope**: `internal/models/api.go`, `internal/core/client/monitor.go`, `internal/core/client/monitor_test.go`
  - **Action**: Implement `ExtractTransferredFiles` supporting JSON objects and plain-text regex fallback, capped at 50. Populate `TransferredFiles` on `MonitorJobResult`.
  - **Verification**: `go test -count=1 -v ./internal/core/client -run TestExtractTransferredFiles`

- [x] Task 2: Sync Runner File Operation Ingestion & Capping (`internal/engine/runner/`)
  - **Scope**: `internal/engine/runner/runner_handlers.go`, `internal/engine/runner/runner_test.go`
  - **Action**: In `handleSyncSuccess`, record up to 50 `sync_file` log entries to `bisync.log` with `Object`, `Action`, and `Msg`. Add overflow notice and daemon log fallback.
  - **Verification**: `go test -count=1 -v ./internal/engine/runner -run TestSyncRunner`

- [x] Task 3: Log Dock Template & Pagination Controls (`component_log_dock.html`)
  - **Scope**: `dashboard/frontend/partials/components/component_log_dock.html`
  - **Action**: Add pagination pill controls (`[ < Page X of Y > ]`), entry count badge, and action badge styling for `sync_file` rows.
  - **Verification**: `./scripts/check.sh templates`

- [x] Task 4: Log Dock Pagination Engine & File Ops Filtering (`log_dock.js` & `logs.js`)
  - **Scope**: `dashboard/frontend/js/log_dock.js`, `dashboard/frontend/js/logs.js`
  - **Action**: Add `loadDockNextPage()`, `loadDockPrevPage()`, and update `isFileOperationLog` to include `sync_file`. Ensure files remain strictly under 400 LOC.
  - **Verification**: `./scripts/check.sh loc 400`

- [x] Task 5: End-to-End Verification & Full Quality Gates (`dashboard_smoke_test.go`)
  - **Scope**: `tests/e2e/dashboard_smoke_test.go`
  - **Action**: Verify Log Dock pagination, `sync_file` path chip rendering, and deep explorer navigation in headless Chrome.
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-11-13: Initialized task plan for surfacing synced files in Log Dock.
- 2026-11-13: Completed Task 0 - Checked out feat/deep-path-nav-log-dock and fast-forward merged fix/bisync-transfer-stats-and-ledger.
- 2026-11-13: Completed Task 1 - Implemented ExtractTransferredFiles in client/monitor.go parsing JSON objects and text lines, added SyncFileOp to models/api.go, and verified with go test.
- 2026-11-13: Completed Task 2 - Ingested TransferredFiles into bisync.log as sync_file entries with 50-file capping, overflow warnings, and daemon log fallback in handleSyncSuccess; verified with comprehensive unit tests.
- 2026-11-13: Completed Task 3 - Updated component_log_dock.html with pagination controls, safe helper bindings, sync_file action badges, overflow notice, and path navigation chips. Passed ./scripts/check.sh templates.
- 2026-11-13: Completed Task 4 - Implemented pagination engine (loadDockNextPage, loadDockPrevPage, getDockTotalPages, isDockPrevDisabled, isDockNextDisabled), action badges (getDockActionBadgeClass, getDockActionLabel), sync_file recognition in isFileOperationLog, and navigateToExplorer alias in log_dock.js. Verified with ./scripts/check.sh loc 400.
- 2026-11-13: Completed Task 5 - Verified Log Dock pagination, action badge styling, sync_file file op recognition, and page boundary stepping in dashboard_smoke_test.go. All 7/7 repository quality gates passed cleanly via ./scripts/check.sh all.
