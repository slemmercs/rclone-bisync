# Implementation Plan: UI Suite Polish & Workflow Alignment

## Overview
This implementation plan addresses the UI/UX refinements identified during Milestone 1 testing:
1. Relocating Global Fleet Settings tabs into a clean **multi-line tab pill arrangement** (`flex-wrap`) with zero horizontal scrollbars.
2. Adding graceful client and backend validation for **configuration encryption removal**, warning that the current passphrase is required instead of raising an execution error.
3. Removing the redundant **Engine Transfer Telemetry panel** from the Telemetry view so only the active synchronization progress bar is displayed when sync runs.
4. Streamlining File Explorer row actions so only the **Download button** and **`...` context menu trigger** appear on row hover, moving Copy and Delete exclusively into the context menu.
5. Aligning context menu icons to standard **Lucide SVGs with directional arrows** (pointing right in the left pane for "Copy to Remote", and pointing left in the right pane for "Copy to Local").
6. **Folder Download Evaluation & Resolution**: Downloading folders as zip archives to the browser adds unnecessary remote streaming overhead when users can already copy entire directories between remotes and local disks via the dual-pane explorer ("Copy to Local / Remote"). We will keep single-file downloads enabled for browser retrieval and retain directory transfers for filesystem operations, keeping the architecture lean and resilient.

---

## Proposed Changes

### 1. Global Fleet Settings Tab Navigation & Encryption Removal Validation
- `dashboard/frontend/partials/modals/modal_global_settings.html`:
  - Change tab container to `flex flex-wrap items-center gap-1.5 border-b border-slate-800 pb-2.5 text-xs font-semibold` (remove `overflow-x-auto` and `shrink-0`).
- `dashboard/frontend/js/settings.js`:
  - In `removeConfigPassword()`: Check if `old_password` is empty. If empty, trigger `showToast("Current passphrase is required to remove encryption.", "warning")` and return early without dispatching an API call.
- `internal/server/handlers_encryption.go`:
  - In `handleRemoveConfigEncryption()`: Validate that `req.Password` is non-empty. Return HTTP 400 Bad Request with a clear error payload if empty.
- `internal/server/handlers_encryption_test.go`:
  - Add test case verifying HTTP 400 is returned when removing encryption with an empty password.

### 2. Remove Redundant Telemetry Panel
- `dashboard/frontend/partials/view_telemetry.html`:
  - Remove `<!-- Core Daemon Bandwidth & Active Transfers Card -->` (the Engine Transfer Telemetry card).
  - Retain `<!-- Live Transfer Progress Card -->` (active synchronization progress bar with file name, speed, ETA, and percentage).

### 3. Explorer Row Actions Declutter & Directional Context Menu Icons
- `dashboard/frontend/partials/view_explorer.html`:
  - In left pane file rows: Remove inline Preview, Copy, and Delete buttons. Keep only Download and `...` More Actions button.
  - In right pane file rows: Remove inline Preview, Copy, and Delete buttons. Keep only Download and `...` More Actions button.
- `dashboard/frontend/partials/components/component_context_menu.html`:
  - For `contextMenu.paneKey === 'left'`: Render Lucide `arrow-right` SVG (`<path d="M5 12h14"/><path d="m12 5 7 7-7 7"/>`) for "Copy to Remote".
  - For `contextMenu.paneKey === 'right'`: Render Lucide `arrow-left` SVG (`<path d="m12 19-7-7 7-7"/><path d="M19 12H5"/>`) for "Copy to Local".
  - Ensure all other items use pure Lucide SVGs (Preview: `eye`, Download: `download`, Copy Path: `copy`, Delete: `trash-2`).

### 4. Quality Gate Verification & Smoke Testing
- `tests/e2e/dashboard_smoke_test.go`:
  - Update smoke tests to assert that only Download and `...` buttons are visible on row hover, and verify the multi-line settings tab wrapping.
- Run `./scripts/check.sh all` to ensure all 7 quality gates pass.

---

## Verification Plan

- Unit & handler tests: `go test -v -count=1 ./internal/server/... -run TestHandleRemoveConfigEncryption`
- LOC audit: `./scripts/check-loc.sh 400`
- Full check suite: `./scripts/check.sh all`
- E2E smoke tests: `go test -v -count=1 ./tests/e2e/...`

