# Task Plan: Google Drive Deduplication Fixes, In-App Modals & Streaming Resilience

## Goal
Fix live Google Drive deduplication failures: resolve the path-truncation batch rename failure, prevent erroneous `--resync` triggers when dedupe batches fail, surface granular failure reasons, replace native browser `prompt()` with in-app dark-themed dialogs, fix blank dedupe directory explorer caused by Alpine `:key` collisions, lift streaming size caps (>2MB) on previews and downloads, and cache scan results across modal operations.

## Tasks

- [x] Task 1: Dedupe Batch Engine Path Concatenation, Resync Safety & Detailed Error Reporting (`internal/services/`)
  - **Scope**: `internal/services/dedupe_service.go`, `internal/services/dedupe_service_test.go`
  - **Verification**: `go test -count=1 ./internal/services/... -run "Test.*DedupeBatch.*"`

- [x] Task 2: Frontend In-App Styled Prompt Modal & Detailed Batch Error Diagnostics (`dashboard/frontend/`)
  - **Scope**: `dashboard/frontend/partials/modals/modal_confirm.html`, `dashboard/frontend/js/app_state.js`, `dashboard/frontend/js/dedupe_batch.js`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check-loc.sh 400`

- [x] Task 3: Dedupe Workspace Directory Explorer Resilience & Inline Candidate Actions (`dashboard/frontend/`)
  - **Scope**: `dashboard/frontend/partials/modals/modal_dedupe.html`, `dashboard/frontend/js/dedupe_workspace.js`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check-loc.sh 400`

- [x] Task 4: Large File (>2MB) Preview & Download Support with Rclone RC-Serve / MaxSize (`internal/core/client/`, `internal/services/`, `deploy/`)
  - **Scope**: `internal/core/client/fs_read.go`, `internal/services/explorer_preview.go`, `internal/services/explorer_download.go`, `internal/services/explorer_preview_test.go`, `deploy/systemd/rclone-daemon.service`
  - **Verification**: `go test -count=1 ./internal/services/... -run "Test.*Preview.*|Test.*Download.*"`

- [x] Task 5: Dedupe Scan Session Caching & Full Quality Pre-Flight (`dashboard/frontend/`, `scripts/`)
  - **Scope**: `dashboard/frontend/js/dedupe_workspace.js`, `dashboard/frontend/js/dedupe_explorer.js`, full quality checks
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-11-15: Initialized plan for Google Drive dedupe fixes, in-app modals, and streaming resilience.
- 2026-11-15: Completed Task 1 - Dedupe Batch Engine Path Concatenation, Resync Safety & Detailed Error Reporting. Resolved `req.Path` subdirectory path concatenation for batch actions, prevented erroneous `--resync` triggers when 0 actions succeed, populated concrete error details in `resp.Error`, and recorded audit logging for both successful and failed dedupe batches. Verified with unit tests.
- 2026-11-15: Completed Task 2 - Frontend In-App Styled Prompt Modal & Detailed Batch Error Diagnostics. Replaced browser native `prompt()` with dark-themed in-app input dialog via `askPrompt` in `modal_confirm.html`, surfaced granular per-action errors in toasts and dedupe execution console output. Verified with templates check and headless Chrome E2E smoke tests.
- 2026-11-15: Completed Task 3 - Dedupe Workspace Directory Explorer Resilience & Inline Candidate Actions. Implemented composite row keys (`getDedupeExplorerRowKey`) preventing duplicate DOM key collision on Google Drive duplicates, added `DUPLICATE` badges, candidate staged status badges, and inline Keep/Rename/Delete/Preview actions for direct folder browsing resolution. Verified with templates check, headless Chrome E2E smoke tests, and LOC audit.
- 2026-11-15: Completed Task 4 - Large File (>2MB) Preview & Download Support with Rclone RC-Serve / MaxSize. Forwarded explicit `maxSize` to `/operations/getfile` instead of Rclone's default 2MB limit, added `DownloadStream` supporting direct streaming via `--rc-serve` with fallback to `CatStream(maxSize)`, gracefully handled preview limit exceeded errors returning `Truncated: true` with direct download guidance, and added `--rc-serve` to systemd service unit. Verified with targeted client/service tests, full test suite pass, and LOC audit.
- 2026-11-15: Completed Task 5 - Dedupe Scan Session Caching & Full Quality Pre-Flight. Implemented persistent `sessionStorage` scan result caching with TTL and force-refresh controls in dedupe workspace, modularized dedupe explorer logic into `dedupe_explorer.js` and `modal_dedupe_explorer.html` maintaining strict <400 LOC limits, and passed all 7 stages of `./scripts/check.sh all` plus headless Chrome E2E smoke tests.


















