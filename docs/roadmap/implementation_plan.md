# Implementation Plan: Dual-Pane Explorer Jump to File Navigation

## Proposed Changes

Fix file explorer deep navigation when clicking "Jump to file in Explorer" from correlated daemon logs, telemetry ledger, or deduplication modals. Ensure that switching to the explorer tab immediately mounts `<ViewExplorer>`, applies the target directory path across local and remote panes (respecting pane swapping and profile `remote_path` subdirectories), scrolls the pulsing file into view, and clears the pending jump so subsequent manual tab switches do not re-trigger old jumps.

### Affected Components & Packages

- `dashboard/frontend/src/stores/explorer.ts`:
  - Update `browse()` to return `Promise<BrowseResponse>` so panes receive their directory items directly without colliding over single shared store state.
  - Add `consumeTargetJumpPath()` to retrieve and clear pending jump targets atomically.
- `dashboard/frontend/src/views/ViewExplorer.vue`:
  - Implement `applyTargetJump()` helper to format paths for local and remote panes, strip `local_path` prefix if an absolute path is provided, account for `profile.remote_path` and `isSwapped`, and trigger pane refreshes.
  - Coordinate `activeProfile` watcher with pending `targetJumpPath` so initial mount navigates to the jump target rather than resetting to root `""`.
  - Update `targetJumpPath` watcher to call `applyTargetJump()` when jump is triggered while Explorer is already mounted.
  - Use returned `BrowseResponse` in `refreshPane()` to eliminate concurrency races.
- `dashboard/frontend/src/components/explorer/ExplorerTable.vue`:
  - Add automatic scrolling into view when an item matches `explorerStore.highlightName`.

### Technical Strategy & Data Flow

- User clicks jump link in `ModalDaemonLogs.vue`:
  1. `explorerStore.jumpToPath(path)` sets `highlightName` and `targetJumpPath`.
  2. `appStore.switchTab('explorer')` sets `activeTab = 'explorer'`.
  3. `<ViewExplorer>` mounts or receives updated `targetJumpPath`.
  4. `applyTargetJump()` formats directory paths:
     - Local directory: `cleanDir` (stripped of `profile.local_path` if absolute).
     - Remote directory: `(remote_path ? remote_path + '/' : '') + cleanDir`.
     - Assigns to `leftPath` / `rightPath` based on `isSwapped`.
     - Calls `consumeTargetJumpPath()`.
     - Calls `refreshPane('left')` and `refreshPane('right')`.
  5. In `ExplorerTable.vue`, the row corresponding to `highlightName` is rendered, pulses with electric blue outline, and scrolls into view.

### Safety & Error Handling

- Graceful fallback: If directory does not exist or has an error, `refreshPane` handles empty items cleanly without crashing.
- Single binary invariant: `npm --prefix dashboard/frontend run build` updates `dashboard/frontend/dist/` assets embedded into the Go binary.

## Verification Plan

- Frontend typecheck and build: `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build`
- E2E tests: `go test -count=1 ./tests/e2e/...`
- Full pre-commit check: `./scripts/check.sh all`





