# Implementation Plan: Explorer Capabilities, Audit Parity, Initial-Sync Safety & Log Resilience

## Overview
Implement provider-aware capability filtering for File Explorer remote public sharing links, achieve full cryptographic audit ledger parity across all manual File Explorer operations, enforce universal uninitialized profile detection requiring baseline `--resync` across all sync triggers, and isolate the sandbox cache directory to prevent host listing collisions and log fetch failures.

## Types

### 1. Audit Taxonomy Extensions (`internal/audit/taxonomy.go`)
Extend the standardized audit action enumeration to cover all manual staging mutations initiated from the File Explorer:
```go
const (
    // Workspace Manual Staging Operations
    ActionWorkspaceItemMoved     Action = "WORKSPACE_ITEM_MOVED"
    ActionWorkspaceDirPurged     Action = "WORKSPACE_DIR_PURGED"
    ActionWorkspaceDirCreated    Action = "WORKSPACE_DIR_CREATED"
    ActionWorkspaceFileUploaded  Action = "WORKSPACE_FILE_UPLOADED"
    ActionWorkspaceItemCopied    Action = "WORKSPACE_ITEM_COPIED"
    ActionWorkspaceItemDeleted   Action = "WORKSPACE_ITEM_DELETED"
)
```

### 2. Remote Public Link Capability Contract (`internal/models/remote.go` / `internal/services/telemetry_remotes.go`)
Ensure the boolean field `can_public_link` is exposed on each remote dictionary returned by `GET /api/remotes`:
```go
type RemoteInfo struct {
    Name          string `json:"name"`
    Type          string `json:"type"`
    CanPublicLink bool   `json:"can_public_link"`
}
```

## Files

### Existing Files to Modify:
1. `dashboard/frontend/js/explorer_operations.js`:
   - Implement `canOpenOnRemote(paneKey)` helper matching `explorer[paneKey].target` against `remotesData.remotes` and checking `can_public_link`.
   - Update `openOnRemote()`: if the backend returns `supported: false` or error, dynamically set `remote.can_public_link = false` on the cached remote object to immediately hide the action.
2. `dashboard/frontend/partials/components/component_context_menu.html`:
   - Replace `<template x-if="contextMenu.item && contextMenu.paneKey && explorer[contextMenu.paneKey].target !== 'local'">` with `<template x-if="contextMenu.item && contextMenu.paneKey && canOpenOnRemote(contextMenu.paneKey)">`.
3. `internal/audit/taxonomy.go`:
   - Add `ActionWorkspaceFileUploaded`, `ActionWorkspaceItemCopied`, and `ActionWorkspaceItemDeleted`.
4. `internal/server/handlers_explorer.go`:
   - In `handleExplorerUpload`: upon successful upload, invoke `s.AuditService.RecordEvent(profileID, audit.ActionWorkspaceFileUploaded, s.resolveActor(r), map[string]any{ ... })`.
   - In `handleCopyItem`: record `audit.ActionWorkspaceItemCopied`.
   - In `handleDeleteItem`: record `audit.ActionWorkspaceItemDeleted`.
5. `dashboard/frontend/js/audit.js`:
   - Update `getAuditActionBadgeClass` to style `WORKSPACE_FILE_UPLOADED`, `WORKSPACE_ITEM_COPIED`, and `WORKSPACE_ITEM_DELETED`.
6. `internal/server/handlers_sync.go`:
   - In `handleAction` for `case "start_sync"`: check whether `prof.State.LastSync == nil || prof.State.LastSync.IsZero()`. If uninitialized, return `requires_resync: true` and `status: "requires_resync"`.
7. `dashboard/frontend/app.js`:
   - In `triggerAction(actionStr)`: intercept `data.requires_resync`. If user confirms, cascade to `triggerAction("force_resync")` or `triggerProfileSync(profId, false, true, false)`.
8. `scripts/run-sandbox.sh`:
   - Add `--cache-dir "$SANDBOX_DIR/cache"` explicitly to the `rclone rcd` command arguments to prevent fallback to `~/.cache/rclone`.
9. `internal/server/handlers_telemetry.go`:
   - In `handleDaemonLogs`: enforce defensive handling against missing log files and return empty log arrays instead of hanging or returning 500.
10. `dashboard/frontend/js/logs.js`:
    - In `openDaemonModal`: catch connection errors with user-friendly error formatting.

## Functions

### New Functions:
- `canOpenOnRemote(paneKey)` (`dashboard/frontend/js/explorer_operations.js`):
  Evaluates whether the target storage provider of `paneKey` supports public links using `this.remotesData.remotes` and whitelist fallback.

### Modified Functions:
- `handleExplorerUpload(w, r)` (`internal/server/handlers_explorer.go`):
  Audits file upload with remote, target path, filename, size, and actor.
- `handleCopyItem(w, r)` (`internal/server/handlers_explorer.go`):
  Audits copy operation with src/dst remotes, paths, and is_dir flag.
- `handleDeleteItem(w, r)` (`internal/server/handlers_explorer.go`):
  Audits delete operation with remote, path, and is_dir flag.
- `handleAction(w, r)` (`internal/server/handlers_sync.go`):
  Guards `start_sync` against uninitialized profiles missing baseline listings.
- `triggerAction(actionStr)` (`dashboard/frontend/app.js`):
  Prompts user when `requires_resync` is returned by `/api/action`.
- `openOnRemote(paneKey, item)` (`dashboard/frontend/js/explorer_operations.js`):
  Dynamically mutates `remote.can_public_link = false` upon runtime unsupported response.

## Classes
No new Go classes or services required. `AuditService`, `ExplorerService`, and `SyncService` integrate the above changes within their existing contracts.

## Dependencies
Native Go standard library and existing internal modules (`internal/audit`, `internal/services`, `internal/server`). Zero external runtime dependencies.

## Testing
1. `internal/server/handlers_explorer_test.go`:
   - Verify `handleExplorerUpload`, `handleCopyItem`, and `handleDeleteItem` write audit records to the test audit database.
2. `internal/server/handlers_profiles_sync_test.go`:
   - Verify `/api/action` with `start_sync` returns `requires_resync: true` when a profile has `LastSync == nil`.
3. `tests/e2e/dashboard_smoke_test.go`:
   - Headless Chrome test asserting that context menu on unsupported or local remotes suppresses "Open on Remote".
4. Quality gate:
   - Full pass: `./scripts/check.sh all`.

## Implementation Order
1. Complete Context Menu Capability Filtering (`explorer_operations.js`, `component_context_menu.html`).
2. Audit Ledger Parity for File Explorer Operations (`taxonomy.go`, `handlers_explorer.go`, `audit.js`, `handlers_explorer_test.go`).
3. Universal Uninitialized Profile Detection & Sandbox Cache Isolation (`handlers_sync.go`, `app.js`, `run-sandbox.sh`, `handlers_profiles_sync_test.go`).
4. Correlated Log Resilience & Error Formatting (`handlers_telemetry.go`, `logs.js`).
5. Quality Gate Verification (`./scripts/check.sh all`).


### Modified Functions

