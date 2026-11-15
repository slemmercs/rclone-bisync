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

- [ ] Task 3: Staged Actions Review Drawer & Live Batch Progress (`dashboard/frontend/`)
  - **Scope**: `dashboard/frontend/partials/modals/modal_dedupe.html`, `dashboard/frontend/js/dedupe_batch.js`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check-loc.sh 400`

- [ ] Task 4: Granular Audit Taxonomy & Per-File Logging in Log Dock (`internal/audit/`, `internal/services/`, `dashboard/frontend/`)
  - **Scope**: `internal/audit/taxonomy.go`, `internal/services/dedupe_service.go`, `internal/services/dedupe_service_test.go`, `dashboard/frontend/partials/components/ui_components.html`
  - **Verification**: `go test -count=1 ./internal/services/... -run "Test.*Dedupe.*"`

- [ ] Task 5: Correlated Daemon Log Badges (`RESYNC`, `DRY RUN`, `FORCE`) & Full Quality Gate
  - **Scope**: `dashboard/frontend/partials/modals/modal_daemon.html`, `dashboard/frontend/js/logs_daemon.js`, `./scripts/check.sh all`
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-11-15: Initialized plan for deduplication & preview UX hardening, audit granularity, and correlated log visibility.
- 2026-11-15: Completed Task 1 - Modal Stacking, Z-Index & Dismissal Isolation. Elevated `modal_confirm.html` to `z-[70]` and `modal_file_preview.html` to `z-[60]`, guarded ESC and click-away handlers in `modal_dedupe.html` so child prompt and preview dismissals do not close the parent Deduplication Workspace, and stopped ESC event bubbling in preview modal. Verified with templates check and LOC verification.
- 2026-11-15: Completed Task 2 - Rclone 2MB RC Ceiling Hard-Clamping & Preview Resilience. Clamped `maxLimit` and `count` to `RcloneGetFileMaxBytes` (2MB) in `CatStream`, resolved the `MaxTextPreviewSize + 1` overflow that caused HTTP 500 errors against Rclone RC, integrated `DownloadStream` image preview with fallback to clamped `CatFile`, and expanded `isLimitExceededError` detection with clean truncated preview guidance. Verified with client and service test suites.



















