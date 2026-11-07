# Implementation Plan: Milestone 2 - Dual-Pane File Commander & Ad-Hoc Transfers

## Overview
Milestone 2 advances Bisync Hub into an interactive dual-pane cloud storage commander:
1. **Interactive Dual-Pane Commander**: Side-by-side synchronized explorer panes (`leftPane`, `rightPane`) with independent browsing, path jails, breadcrumbs, and cross-pane transfer triggers.
2. **Atomic Move & Purge Operations**: Safe file/directory moves (`operations/movefile`, `operations/movedir`) and directory purging with canary protection and local jail validation.
3. **Non-Blocking Queued Upload Dock**: Asynchronous upload manager detecting active sync locks (`flock` contention or `telemetry.is_syncing === true`), staging transfers in a queued state instead of returning 409 errors, and auto-resuming once locks clear.
4. **Cloud Public Web Links**: Remote cloud web link resolution (`operations/publiclink`) for instant navigation to Google Drive and OneDrive web interfaces.

---

## Proposed Changes

### Affected Components & Packages

- `internal/models/operations.go`:
  - `MoveRequest` & `MoveResponse`: Source/destination remotes, paths, directory flag, and status.
  - `PublicLinkRequest` & `PublicLinkResponse`: Canonical remote cloud URL retrieval.
  - `QueuedUploadItem`: Transfer queue item representation, byte progress, and state (`queued`, `uploading`, `completed`, `failed`).

- `internal/core/client/operations.go`:
  - `MoveFile(ctx, srcFs, srcRemote, dstFs, dstRemote)` wrapping RC `/operations/movefile`.
  - `MoveDir(ctx, srcFs, dstFs)` wrapping RC `/operations/movedir`.
  - `PublicLink(ctx, fs, remote, unlink bool)` wrapping RC `/operations/publiclink`.

- `internal/services/explorer_ops.go`:
  - `MoveItem(ctx, srcRemote, srcPath, dstRemote, dstPath string, isDir bool, localBase ...string) (models.MoveResponse, error)` enforcing path jails and canary protection.
  - `GetPublicLink(ctx, remote, path string) (string, error)` with graceful degradation on unsupported tiers (local, SFTP, S3).

- `internal/services/upload_queue.go`:
  - `UploadQueueManager` maintaining an in-memory thread-safe queue of pending uploads.
  - Coordinates with `core/lock` to defer upload processing when sync runs are active, auto-resuming on completion.

- `internal/server/handlers_operations.go` & `internal/server/server.go`:
  - `POST /api/operations/move`: Validates payload, checks path jails/canaries, and executes atomic move.
  - `POST /api/operations/purge`: Purges directory with recursive canary protection.
  - `GET /api/operations/publiclink`: Returns canonical web link for cloud items.
  - `GET /api/explorer/queue` & `POST /api/explorer/upload-queued`: Multipart file upload staging into upload queue.

- `dashboard/frontend/partials/view_explorer.html`:
  - Dual-pane commander layout with responsive split-screen toggle (`Single Pane` vs `Dual Commander`).
  - Drag-and-drop file upload target zones on both left and right panes.
  - Commander action bar: `[Copy]`, `[Move]`, `[Delete]`, and upload queue status indicator.

- `dashboard/frontend/js/explorer.js` & `dashboard/frontend/js/explorer_actions.js`:
  - Dual-pane state management (`leftPane`, `rightPane`, active pane focus).
  - Cross-pane directional transfer handlers and keyboard shortcuts.
  - Non-blocking upload dock status pill and drawer.

---

### Technical Strategy & Data Flow

```
[Operator Action] ─── Drag & Drop / Move / Copy
         │
         ▼
[Frontend Controller] (explorer.js / explorer_actions.js)
         │
         ├── Check `telemetry.is_syncing`
         │     ├── If Idle: Direct execution via /api/operations/move or /api/explorer/upload
         │     └── If Sync Active: Route to /api/explorer/upload-queued
         │
         ▼
[Backend Service Layer] (ExplorerService / UploadQueueManager)
         │
         ├── Path Jail Validation (`isLocalPathJailed`)
         ├── Canary File Protection (`RCLONE_TEST` check)
         ├── Advisory Flock Lock Coordination
         │
         ▼
[Rclone RC API / Local Filesystem] ─── Atomic Rename / Transfer
```

### Safety & Error Handling

- **Local Filesystem Jail**: All file moves, copies, and uploads must verify source and destination paths using `filepath.Rel` and base prefix confinement (`isLocalPathJailed`).
- **Canary File Inviolability**: `RCLONE_TEST` cannot be moved, renamed, overwritten, or deleted. Any directory move or purge recursively checks for canary files before execution.
- **Flock Concurrency Coordination**: Background sync runs acquire exclusive locks. Explorer write actions check advisory lock state to prevent mid-sync corruption, enqueuing uploads if locks are held.
- **Code Governance Limits**: Every Go file and frontend module must remain strictly under 400 LOC.

---

## Verification Plan

### Automated Unit & Integration Tests
- `internal/services/explorer_ops_test.go`:
  - `TestMoveItem_LocalToLocal`: Verify atomic local moves and jail enforcement.
  - `TestMoveItem_CanaryProtection`: Verify moving `RCLONE_TEST` returns 403 / error.
  - `TestMoveItem_RemoteToRemote`: Mock RC `/operations/movefile` and verify request payload.
- `internal/services/upload_queue_test.go`:
  - Test queue state transitions when lock is held vs released.
- `internal/server/handlers_operations_test.go`:
  - Test `/api/operations/move` route and error responses.
  - Test `/api/operations/publiclink` route and fallback behavior.

### End-to-End Headless Chrome Tests (`tests/e2e/dashboard_smoke_test.go`)
- E2E smoke test verifying:
  - Commander dual-pane toggle renders both panes.
  - Drag-and-drop upload zone triggers exist.
  - Directional transfer buttons open confirmation modals with correct source/destination.

### Full Quality Gate Execution
- `./scripts/check.sh all`:
  - All 7 stages passing: format, vet, staticcheck, Alpine template safety, LOC check (<400 LOC), unit tests, and race detector.

- Full check suite: `./scripts/check.sh all`
- E2E smoke tests: `go test -v -count=1 ./tests/e2e/...`

