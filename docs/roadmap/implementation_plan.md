# Implementation Plan: Deduplication Workspace Redesign & Link/Whitespace Edge-Case Resolution

## Proposed Changes

### High-Level Strategy & Architecture Impact
1. **Google Drive Link & Trailing Whitespace Edge-Case Resolution**:
   - In `internal/services/explorer_service.go`, replace aggressive `strings.TrimSpace(path)` trimming with slash-only trimming `strings.Trim(path, "/")` so that folders with legitimate trailing spaces (e.g. `"MOHU Szemétszállítás "`) are preserved accurately.
   - Implement graceful 404 transparent auto-resolution in `BrowsePath`: when Rclone returns `directory not found` for a remote path, the backend automatically queries the parent directory, detects if a matching folder exists with trailing whitespace or as a Google Drive shortcut, resolves it, and returns its contents with the recovered path transparently.
   - Tag items with `IsShortcut: true` when Rclone reports composite IDs (e.g. `<shortcut_id>\t<target_id>`) or mime type `application/vnd.google-apps.shortcut`.

2. **Deduplication Workspace 3-Section Redesign**:
   - Redesign `modal_dedupe.html` from a cramped single-column dialog into an expansive, responsive 3-section workspace (`max-w-6xl` / `max-w-7xl`):
     - **Section 1 (Left Rail, ~280px)**: Configuration & Scope (Target remote, folder selector with browser chips, match mode, resolution strategy, paired bidirectional dedupe toggle, execution triggers, and metrics summary).
     - **Section 2 (Center Panel, ~360px)**: Collision Queue / Ledger (Search/filter bar, collision cards showing filename, path, copies, size, and `🔗 Shortcut` indicator; clicking an item selects it).
     - **Section 3 (Right Panel, ~520px+)**: Inbuilt File Explorer & Duplicate Inspector (Mini-breadcrumbs, duplicate candidate comparison cards with `KEEP` vs `DELETE` recommendation badges, directory contents mini-explorer, preview/download actions, and in-situ 404 recovery fallback).

3. **Frontend Modularization (<400 LOC Ceiling)**:
   - Extract deduplication workspace logic from `dashboard/frontend/js/remotes_modals.js` into `dashboard/frontend/js/dedupe_workspace.js` (~220 LOC), keeping both modules well below the strict 400 LOC ceiling.
   - Register `dedupe_workspace.js` in `dashboard/frontend/index.html` and `app.js`.

### Affected Components & Packages

- `internal/services/`:
  - `explorer_service.go`: Preserve folder whitespace; add parent-lookup fallback for 404 directory errors; tag shortcut metadata.
  - `explorer_service_test.go`: Add unit tests for whitespace preservation and shortcut detection.
- `internal/models/`:
  - `api_explorer.go`: Add `IsShortcut` boolean field to directory listing item models.
- `dashboard/frontend/`:
  - `partials/modals/modal_dedupe.html`: Complete 3-section responsive layout overhaul.
  - `js/dedupe_workspace.js`: New modular deduplication workspace state and action handlers (master-detail selection, candidate comparison, inbuilt folder browsing).
  - `js/remotes_modals.js`: Remove dedupe modal logic, delegating to `dedupe_workspace.js`.
  - `js/dedupe_parser.js`: Parse duplicate timestamps, IDs, and hashes for side-by-side inspection.
  - `index.html`: Register `dedupe_workspace.js`.
  - `app.js`: Include `DedupeWorkspaceModule` in dashboard state composition.
- `tests/e2e/`:
  - `dashboard_smoke_test.go`: Add E2E tests for 3-section dedupe workspace navigation and inbuilt folder exploration.

### Safety & Error Handling

- **Non-Destructive Dry-Run Preservation**: "Preview (Dry Run)" remains the safe default (`--dedupe-mode list`), requiring explicit user confirmation before any file mutation.
- **Path Confinement & Jail Bounds**: Inbuilt explorer within the dedupe modal strictly uses existing `/api/browse` endpoint with path traversal safeguards (`..` checks and root confinement).
- **Graceful Error Recovery**: If an explorer path fails to load (e.g. broken shortcut or invalid link), Section 3 displays a recovery card with options to browse parent folder or retry without throwing raw JSON errors.

## Verification Plan

- Unit tests:
  - `go test -count=1 ./internal/services/... -run TestBrowsePath_WhitespacePreservation`
  - `go test -count=1 ./internal/server/...`
- Full Quality Gate:
  - `./scripts/check.sh templates`
  - `./scripts/check.sh loc 400`
  - `./scripts/check.sh all`
- End-to-End Headless Chrome Testing:
  - Verify dedupe preview scan, collision selection, candidate comparison card rendering (`KEEP`/`DELETE`), and embedded directory browsing.


