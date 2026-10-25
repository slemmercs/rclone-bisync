# Implementation Plan: Fix Explorer Copy Button Disabled State & Reactivity

Fixes the issue where clicking the transfer/copy button for files (such as `Hatlapos.docx`) in the Dual-Pane File Explorer does not transfer the file to the opposing remote or local pane.

## Proposed Changes

### Root Cause Analysis

1. **Alpine.js Boolean Attribute Binding on Undefined Property**:
   In `dashboard/frontend/partials/view_explorer.html` (lines 185 and 363), the file transfer buttons are bound with:
   `:disabled="item._copying"`
   When items are loaded from `/api/browse`, the API returns directory entries (`Name`, `Path`, `Size`, `IsDir`, `ModTime`, `MimeType`) without a `_copying` property.
   In Alpine.js v3, evaluating an expression containing a dot (`.`) that resolves to `undefined` causes Alpine's internal evaluator to replace `undefined` with `""` (empty string). When applying `""` to boolean attributes (e.g. `disabled`, `checked`), Alpine treats any non-null, non-undefined, non-false value as presence of the attribute, setting `disabled="disabled"`.
   Consequently, every copy button in both left and right explorer panes was rendered with `disabled="disabled"`. While visually styled as interactive (opacity on hover, pointer cursor), HTML browsers suppress all user mouse click events on disabled buttons, so no action or network request was dispatched.

2. **Reactive State Initialization**:
   `item._copying` was uninitialized in the reactive items array, meaning dynamic mutation during copying was not guaranteed to trigger reactive UI updates.

3. **Asset Cache Busting**:
   Increment script asset cache busting parameters (`?v=18`) across `index.html` to guarantee existing browser sessions immediately load the fixed template logic and explorer scripts.

### Affected Components & Packages

- `dashboard/frontend/partials/view_explorer.html`:
  - Change `:disabled="item._copying"` to `:disabled="Boolean(item && item._copying)"` on both the left (line 185) and right (line 363) pane copy buttons.
  - Enforce `< 400 LOC` constraint.

- `dashboard/frontend/js/explorer.js`:
  - In `fetchExplorerDirectory`, map items to explicitly initialize `_copying: false` on every directory entry.
  - Enforce `< 400 LOC` constraint.

- `dashboard/frontend/index.html`:
  - Increment script cache busting query parameter from `?v=17` to `?v=18`.

- System Deployment & Service Restart:
  - Recompile single binary (`bisync-hub` and `/home/csaba/rclone-bisync/bin/rclone-bisync`).
  - Restart user systemd daemon service `rclone-dashboard.service`.

### Technical Strategy & Data Flow

- Explicit boolean coercion in Alpine bindings ensures `:disabled` is always strictly evaluated as a boolean (`false` by default, `true` only during an active transfer).
- With `disabled` removed from the DOM, mouse click events cleanly fire on the `<button>` element, invoking `transferItem(paneKey, item)`.
- `transferItem` executes `POST /api/browse/copy` via `window.API.post`, displays success toast, and refreshes the opposing remote directory pane (`fetchExplorerDirectory`), showing the transferred file.

### Safety & Error Handling

- Existing path confinement jails (`isLocalPathJailed`) and canary protection checks in `ExplorerService` are fully preserved.
- No live network requests during tests; unit tests mock HTTP endpoints.

## Verification Plan

- Run Chrome headless browser CDP automated test simulating real mouse press and release on `Hatlapos.docx` copy button:
  - Confirm `disabled="disabled"` attribute is absent.
  - Confirm real OS-level mouse click triggers `transferItem`.
  - Confirm toast notification displays success message.
  - Confirm opposite pane refreshes and contains `Hatlapos.docx`.
- Run codebase quality script: `./scripts/check.sh`.
- Run LOC checker: `./scripts/check-loc.sh 400`.



