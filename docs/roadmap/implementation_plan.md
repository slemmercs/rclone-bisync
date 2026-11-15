# Implementation Plan: Deduplication & Preview UX Hardening, Audit Granularity & Correlated Log Visibility

## Proposed Changes

Resolve live Google Drive deduplication issues identified in testing:
1. Modal stacking & dismissal isolation: prevent closing the file preview or prompt modal from unexpectedly closing parent Deduplication Workspace; elevate prompt modal z-index above dedupe workspace.
2. Rclone 2MB RC ceiling resilience: respect Rclone's hard 2MB limit on `operations/getfile`, fall back to `--rc-serve` streaming or return clean truncated previews with direct download guidance rather than 500 errors.
3. Staged dedupe review drawer & live progress: provide an interactive staged actions review drawer with per-action unstaging controls and animated chunked batch progress tracking.
4. Granular audit taxonomy & file logging: introduce `DEDUPE_BATCH_EXECUTED` taxonomy, emit per-file deletion and rename events to the bisync log stream so deleted files show up in the Log Dock and correlated logs.
5. Correlated daemon log badges: display `--resync`, `--dry-run`, and `--force` operational flags prominently in the correlated logs modal header.
6. Rename Confirmation Isolation: Ensure `handleConfirmModal()` sets `confirmModal.closedAt = Date.now()` prior to closing, stop Enter keydown bubbling on input with `.stop`, stop mousedown/mouseup propagation on prompt dialog buttons and cards, and add `.stop` to inline candidate dedupe action buttons so confirming rename cannot trigger backdrop click or bleed through to parent dedupe workspace.

### Affected Components & Packages

- `dashboard/frontend/js/app_state.js`: Record `confirmModal.closedAt = Date.now()` in `handleConfirmModal()` before resetting state.
- `dashboard/frontend/partials/modals/modal_confirm.html`: Add `.stop` to `@keydown.enter.prevent`, add `@mousedown.stop @mouseup.stop` to card and buttons.
- `dashboard/frontend/partials/modals/modal_dedupe.html`: Add `@click.stop` to candidate action buttons and expand cooldown threshold to 400ms.
- `dashboard/frontend/partials/modals/modal_dedupe_explorer.html`: Add `@click.stop` to explorer candidate dedupe buttons.
- `dashboard/frontend/partials/modals/modal_file_preview.html`: Set stacking priority to `z-[60]`, stop ESC bubbling.
- `dashboard/frontend/partials/modals/modal_dedupe.html`: Guard ESC and click-away listeners so child modals do not trigger dedupe dismissal; add slide-up staged actions drawer.
- `dashboard/frontend/partials/modals/modal_daemon.html`: Add operational flags badges (`RESYNC`, `DRY RUN`, `FORCE`).
- `dashboard/frontend/js/dedupe_batch.js`: Implement staged actions drawer state and live progress tracking.
- `internal/core/client/fs_read.go`: Clamp `operations/getfile` `maxSize` parameter to 2MB ceiling.
- `internal/services/explorer_preview.go`: Support streaming via `DownloadStream` when available, handle built-in limit error messages gracefully.
- `internal/audit/taxonomy.go`: Add `ActionDedupeBatchExecuted`.
- `internal/services/dedupe_service.go`: Log individual file deletions and renames, record `ActionDedupeBatchExecuted`.
- `dashboard/frontend/partials/components/ui_components.html`: Render dedicated card for `DEDUPE_BATCH_EXECUTED`.

## Verification Plan

- `./scripts/check.sh templates`
- `./scripts/check-loc.sh 400`
- `go test -count=1 ./internal/core/client/... ./internal/services/...`
- `./scripts/check.sh all`





