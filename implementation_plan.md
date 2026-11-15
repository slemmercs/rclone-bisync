# Implementation Plan: Deduplication Edge-Cases, Smart Safety Recommendations & Staged Batch Review Engine

## Proposed Changes

Deliver an enterprise-grade deduplication workflow that eliminates Google Drive duplicate stream collisions (HTTP 500), fixes UI directory explorer stalling and multi-select collisions, alerts users to data-loss content divergence on same-name duplicates, and introduces a transactional Staged Batch Review & Resync Bridge.

### 1. Backend Duplicate Stream Interception & Safe Error Handling
- In `internal/models/api_explorer.go`:
  - Add `IsDuplicateCollision bool` and `CollisionCount int` to `FilePreviewResponse`.
- In `internal/services/explorer_preview.go`:
  - Catch `"found more than one file"` when calling `CatFile` on Google Drive.
  - Return a valid `FilePreviewResponse` with `IsDuplicateCollision: true`, file metadata, and actionable guidance rather than propagating an unhandled error.
- In `internal/services/explorer_download.go` & `internal/server/handlers_preview.go`:
  - Catch `"found more than one file"` and return HTTP 409 Conflict with a structured error payload advising deduplication resolution instead of HTTP 500.

### 2. Deduplication Batch Execution Engine & Endpoints
- In `internal/models/api_dedupe.go`:
  - Define `DedupeStageAction`, `DedupeBatchApplyRequest`, and `DedupeBatchApplyResponse`.
- In `internal/services/dedupe_service.go`:
  - Implement `ExecuteDedupeBatch(ctx, req, localBase)` to execute staged renames and deletions transactionally, with canary checks, jail bounds, and audit ledger tracking.
  - If `TriggerResync` is requested, invoke profile resync safely.
- In `internal/server/handlers_dedupe.go` and `routes.go`:
  - Add `POST /api/remotes/dedupe/batch` handler with confirmation token safeguards.

### 3. Frontend UID Disambiguation & Explorer Target Row Highlighting
- In `dashboard/frontend/js/explorer.js`:
  - In `fetchExplorerDirectory`, assign a unique `_uid` (`item.ID || (item.Path + '_' + item.Size + '_' + item.ModTime + '_' + index)`) to every item.
  - Identify duplicate names within the same folder and mark `item.IsDuplicate = true`.
  - Support `highlightedFile` with auto-scroll via `$nextTick`.
- In `dashboard/frontend/js/explorer_compare.js`:
  - Update `isItemSelected` and `toggleSelectItem` to track selection by `_uid` rather than path, preventing duplicate filename collision.
- In `dashboard/frontend/partials/view_explorer.html`:
  - Display `⚠️ Duplicate` badge for duplicate items.
  - Add highlight animation for `highlightedFile`.

### 4. Dedupe Workspace Directory Explorer Resilience & Content Divergence Detection
- In `dashboard/frontend/js/dedupe_workspace.js`:
  - Implement `switchDedupeTab(tabName)` with automatic loading of directory contents.
  - Add `hasContentDivergence(collision)` comparing candidate sizes and hashes.
  - Fix `jumpToExplorerFile` to call `await this.fetchExplorerDirectory(paneKey, targetRemote, dirPath)` and set `highlightedFile`.
  - Gracefully display explorer errors with retry and parent navigation buttons.
- In `dashboard/frontend/partials/modals/modal_dedupe.html`:
  - Render Content Divergence warning alert when sizes differ across duplicate candidates.
  - Clarify Same-Folder Names vs Cross-Folder Hashes with explanatory tooltips.

### 5. Staged Batch Review UI & Post-Dedupe Resync Bridge
- In `dashboard/frontend/js/dedupe_workspace.js`:
  - Implement staging queue methods: `stageCandidateAction`, `removeStagedAction`, `clearStagedActions`, `applyStagedBatch`.
  - Connect "Compare Copies" to `ExplorerCompareModule.compareDirect(...)`.
- In `dashboard/frontend/partials/modals/modal_dedupe.html`:
  - Add inline "Stage Rename" and "Stage Delete" action controls to candidate cards.
  - Add Staged Review Drawer showing pending operations, execution button, and post-batch resync button.
- In `dashboard/frontend/partials/modals/modal_preview.html`:
  - Render dedupe resolution quick-link when preview encounters duplicate collision.

### 6. Local Sandbox Fixtures & Verification
- In `scripts/run-sandbox.sh`:
  - Seed hash-duplicate and duplicate-named files in test folders for offline testing.

## Verification Plan

- Unit tests:
  - `go test -count=1 ./internal/services/... -run "Test.*Preview|Test.*Dedupe"`
  - `go test -count=1 ./internal/server/...`
- E2E and Quality gates:
  - `./scripts/check.sh templates`
  - `./scripts/check-loc.sh 400`
  - `./scripts/check.sh all`



