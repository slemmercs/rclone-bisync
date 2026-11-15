# Task Plan: Deduplication & Preview UX Hardening, Audit Granularity & Correlated Log Visibility

## Goal
Resolve 7 user-reported issues: isolate preview/dedupe modal dismissal, clamp Rclone `operations/getfile` maxSize to 2MB with graceful preview fallbacks, elevate rename prompt z-index above dedupe workspace, add staged actions review drawer and batch progress indicator, record granular per-file deletion logs and `DEDUPE_BATCH_EXECUTED` audit events, and show `--resync` badges in correlated daemon logs.

## Tasks

- [x] Task 1: Modal Stacking, Z-Index & Dismissal Isolation (`dashboard/frontend/`)
  - **Scope**: `dashboard/frontend/partials/modals/modal_confirm.html`, `dashboard/frontend/partials/modals/modal_file_preview.html`, `dashboard/frontend/partials/modals/modal_dedupe.html`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check-loc.sh 400`

- [x] Task 2: Rclone 2MB RC Ceiling Hard-Clamping & Preview Resilience (`internal/core/client/`, `internal/services/`)
  - **Scope**: `internal/core/client/fs_read.go`, `internal/services/explorer_preview.go`, `internal/services/explorer_preview_test.go`, `internal/core/client/fs_read_test.go`
  - **Verification**: `go test -count=1 ./internal/core/client/... ./internal/services/... -run "Test.*Preview.*|Test.*Cat.*"`

- [x] Task 3: Staged Actions Review Drawer & Live Batch Progress (`dashboard/frontend/`)
  - **Scope**: `dashboard/frontend/partials/modals/modal_dedupe.html`, `dashboard/frontend/partials/modals/modal_dedupe_drawer.html`, `dashboard/frontend/js/dedupe_batch.js`
  - **Verification**: `./scripts/check.sh templates`

- [x] Task 4: Granular Audit Taxonomy & Per-File Logging in Log Dock (`internal/audit/`, `internal/services/`, `dashboard/frontend/`)
  - **Scope**: `internal/audit/taxonomy.go`, `internal/services/dedupe_service.go`, `internal/services/dedupe_service_test.go`, `dashboard/frontend/partials/components/ui_components.html`
  - **Verification**: `go test -count=1 ./internal/services/... -run "Test.*Dedupe.*"`

- [x] Task 5: Correlated Daemon Log Badges (`RESYNC`, `DRY RUN`, `FORCE`) & Full Quality Gate
  - **Scope**: `dashboard/frontend/partials/modals/modal_daemon.html`, `dashboard/frontend/js/logs_daemon.js`, `./scripts/check.sh all`
  - **Verification**: `./scripts/check.sh all`

- [x] Task 6: Rename Confirmation Isolation & Modal Cooldown Protection (`dashboard/frontend/`)
  - **Scope**: `dashboard/frontend/js/app_state.js`, `dashboard/frontend/partials/modals/modal_confirm.html`, `dashboard/frontend/partials/modals/modal_dedupe.html`, `dashboard/frontend/partials/modals/modal_dedupe_explorer.html`
  - **Verification**: `./scripts/check.sh templates && go test -count=1 ./tests/e2e/...`

## Status Updates

- 2026-11-15: Initialized plan for deduplication & preview UX hardening, audit granularity, and correlated log visibility.
- 2026-11-15: Completed Task 1 - Modal Stacking, Z-Index & Dismissal Isolation. Elevated `modal_confirm.html` to `z-[70]` and `modal_file_preview.html` to `z-[60]`, guarded ESC and click-away handlers in `modal_dedupe.html` so child prompt and preview dismissals do not close the parent Deduplication Workspace, and stopped ESC event bubbling in preview modal. Verified with templates check and LOC verification.
- 2026-11-15: Completed Task 2 - Rclone 2MB RC Ceiling Hard-Clamping & Preview Resilience. Clamped `maxLimit` and `count` to `RcloneGetFileMaxBytes` (2MB) in `CatStream`, resolved the `MaxTextPreviewSize + 1` overflow that caused HTTP 500 errors against Rclone RC, integrated `DownloadStream` image preview with fallback to clamped `CatFile`, and expanded `isLimitExceededError` detection with clean truncated preview guidance. Verified with client and service test suites.
- 2026-11-15: Completed Task 3 - Staged Actions Review Drawer & Live Batch Progress. Built `modal_dedupe_drawer.html` partial with animated slide-up review list, action badges (delete/rename/keep), candidate unstage buttons, and animated live batch progress indicator; updated `dedupe_batch.js` with drawer state toggle, staged counts, and progress tracking; preserved strict <400 LOC modularity. Verified with templates check and server tests.
- 2026-11-15: Completed Task 4 - Granular Audit Taxonomy & Per-File Logging in Log Dock. Added `ActionDedupeBatchExecuted` in `internal/audit/taxonomy.go`, emitted per-file deletion and rename `BisyncLogEntry` events during batch execution so individual files show in the Log Dock and correlated logs, rendered dedicated `DEDUPE_BATCH_EXECUTED` summary card in `ui_components.html`, added filter option in `view_audit.html`, and verified with end-to-end audit ledger and log streaming tests.
- 2026-11-15: Completed Task 5 - Correlated Daemon Log Badges (`RESYNC`, `DRY RUN`, `FORCE`) & Full Quality Gate. Added reactive `daemonModalRunFlags` getter in `logs_daemon.js` inspecting execution metadata and daemon log output, styled `RESYNC`, `DRY RUN`, and `FORCE` badges in `modal_daemon.html`, formatted Go test suites, and verified with all 7 checks of `./scripts/check.sh all` and headless Chrome E2E tests passing.
- 2026-11-15: Completed Task 6 - Rename Confirmation Isolation & Modal Cooldown Protection. Recorded `confirmModal.closedAt = Date.now()` in `handleConfirmModal()` prior to resetting modal state, prevented Enter keydown propagation on prompt input with `.prevent.stop`, stopped mousedown/mouseup bubbling on prompt modal card and action buttons, added `.stop` to candidate dedupe buttons in both Candidates view and Directory Explorer view, and widened cooldown threshold to 400ms. Verified with `./scripts/check.sh all` and headless Chrome E2E tests.



















