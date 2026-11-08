# Implementation Plan: Sandbox Reliability & File Operations Remediation

## Proposed Changes

Address 7 critical issues discovered during sandbox interactive testing:
1. **Remote File Preview & Streaming (Issue 1)**:
   - Rclone RC does not provide an `/operations/cat` endpoint (returns 404). The official Rclone RC API endpoint for reading single files is `/operations/getfile`, which supports `base64: true`, `offset`, `count`, and `maxSize`.
   - Update `internal/core/client/fs_read.go` to invoke `/operations/getfile` and decode base64 results for `CatFile` and `CatStream`.
   - Add `--rc-serve` to `scripts/run-sandbox.sh` and set `RCLONE_CACHE_DIR="$SANDBOX_DIR/cache"` for full ephemeral containment.

2. **Directory Copying & Tree Placement (Issues 2, 3, 4)**:
   - In `internal/services/explorer_ops.go` (`CopyItemWithOptions`), copying a directory (`isDir == true`) currently targets `dstFs` directly, causing Rclone's `/sync/copy` to dump files into the destination root instead of preserving the directory container.
   - Adjust `dstFs` and local target directory to append `filepath.Base(cleanSrcPath)` so the folder container is recreated at the destination.

3. **Missing Listings & First Sync Detection (Issue 5)**:
   - Running `rclone bisync` without `--resync` on an uninitialized profile fails with `NO_LISTINGS` ("cannot find prior Path1 or Path2 listings").
   - In `internal/server/handlers_profiles.go`, detect when an uninitialized profile (`p.State.LastSync == nil`) is triggered without `resync`, returning `requires_resync: true`.
   - In `dashboard/frontend/js/profiles_history.js`, intercept `requires_resync` and prompt the user with a confirmation dialog to initialize baseline listings via `--resync`.

4. **Audit Ledger Logging for Resync (Issue 6)**:
   - In `internal/engine/runner/runner_handlers.go`, `recordAuditExecuted` was gated strictly by `monitorRes.FilesSynced > 0`. A zero-transfer resync was therefore never recorded in the audit ledger.
   - Expand condition to `monitorRes.FilesSynced > 0 || resync || origin == "dashboard" || origin == "cli"`.

5. **File Move/Rename Path Construction & Lucide Icon (Issue 7)**:
   - In `internal/services/explorer_move.go` (`executeMove`), moving a file on a remote treated `cleanDstPath` as `dstFs` and appended `srcFile` again, creating nested folder structures (e.g. `test1/archive_info.json/archive_info.json`).
   - Parse `cleanDstPath` into destination directory and destination filename properly.
   - Ensure "New Folder" buttons use canonical Lucide `folder-plus` SVG markup.

### Affected Components & Packages

- `internal/core/client/fs_read.go`: Update `CatFile` and `CatStream` to use `/operations/getfile`.
- `internal/core/client/fs_read_test.go`: Update unit tests for `/operations/getfile`.
- `internal/services/explorer_preview_test.go`: Update mock server for `/operations/getfile`.
- `internal/services/explorer_ops.go`: Fix directory destination path in `CopyItemWithOptions`.
- `internal/services/explorer_move.go`: Fix destination path parsing in `executeMove`.
- `internal/engine/runner/runner_handlers.go`: Log audit events for resync and user-initiated runs.
- `internal/server/handlers_profiles.go`: Add initial sync check for missing listings.
- `dashboard/frontend/js/profiles_history.js`: Add confirmation dialog for initial resync.
- `dashboard/frontend/partials/view_explorer.html` & `component_context_menu.html`: Lucide icon consistency.
- `scripts/run-sandbox.sh`: Add `--rc-serve` and `RCLONE_CACHE_DIR`.

## Verification Plan

- Unit test `CatFile` and `CatStream` with mock Rclone `/operations/getfile`.
- Unit test directory copy and move destination paths.
- Unit test audit ledger recording for 0-transfer resync.
- Run full quality suite: `./scripts/check.sh all`.
- Verify in sandbox.

   - File: `internal/core/client/operations.go`
   - Purpose: Calls Rclone RC `operations/purge`.
4. `c.PublicLink(ctx context.Context, fs, remote string, unlink bool) (*models.PublicLinkResponse, error)`
   - File: `internal/core/client/operations.go`
   - Purpose: Calls Rclone RC `operations/publiclink` and returns shareable URL.
5. `e.MoveItem(ctx context.Context, req models.MoveRequest, localBase string) (models.MoveResponse, error)`
   - File: `internal/services/explorer_move.go`
   - Purpose: Validates local jail boundaries, blocks `RCLONE_TEST` canary moves, executes atomic `os.Rename` for local moves or RC client for cloud, audits with `SOURCE: WORKSPACE_STAGING`.
6. `e.PurgeDirectory(ctx context.Context, req models.PurgeRequest, localBase string) (models.PurgeResponse, error)`
   - File: `internal/services/explorer_purge.go`
   - Purpose: Recursively verifies no canary is contained before purging; audits action.
7. `e.GetPublicLink(ctx context.Context, req models.PublicLinkRequest) (models.PublicLinkResponse, error)`
   - File: `internal/services/explorer_link.go`
   - Purpose: Calls client public link; handles unsupported remotes gracefully without 500 error.
8. `dock.Enqueue(item models.UploadQueueItem) error` / `dock.GetStatus() models.UploadDockStatus`
   - File: `internal/services/upload_dock.go`
   - Purpose: In-memory upload queue tracking deferrals during active sync runs.
9. `s.handleMoveItem(w http.ResponseWriter, r *http.Request)`
   - File: `internal/server/handlers_operations.go`
   - Purpose: HTTP handler for `POST /api/operations/move`.
10. `s.handlePurgeDirectory(w http.ResponseWriter, r *http.Request)`
    - File: `internal/server/handlers_operations.go`
    - Purpose: HTTP handler for `POST /api/operations/purge`.
11. `s.handlePublicLink(w http.ResponseWriter, r *http.Request)`
    - File: `internal/server/handlers_operations.go`
    - Purpose: HTTP handler for `GET /api/operations/publiclink`.

### Modified Functions
1. `s.routes()`
   - File: `internal/server/server.go`
   - Modification: Call `s.registerOperationsRoutes()` to register new operational routes without exceeding 400 LOC.
2. `fetchExplorerDirectory(pane, target, path)`
   - File: `dashboard/frontend/js/explorer.js`
   - Modification: Check active review candidates and apply ambient badges.

## [Classes / Structs]

1. `ExplorerService`:
   - File: `internal/services/explorer_service.go`
   - Receives new methods across decomposed domain files (`explorer_move.go`, `explorer_purge.go`, `explorer_link.go`).
2. `UploadDock`:
   - File: `internal/services/upload_dock.go`
   - Thread-safe manager: `sync.RWMutex`, `items []models.UploadQueueItem`, checks `syncService.IsSyncing()`.
3. `RcloneClient`:
   - File: `internal/core/client/operations.go`
   - Receives methods `MoveFile`, `MoveDir`, `Purge`, `PublicLink`.

## [Dependencies]
- Zero external runtime or package dependencies; standard Go library (`net/http`, `os`, `path/filepath`, `sync`, `time`) and existing internal modules (`internal/core/client`, `internal/models`, `internal/security`).

## [Testing]

1. **Unit Tests**:
   - `internal/core/client/operations_test.go`:
     - `TestMoveFile_Success`, `TestMoveFile_Error`
     - `TestMoveDir_Success`
     - `TestPurge_Success`, `TestPurge_Error`
     - `TestPublicLink_Success`, `TestPublicLink_Unsupported`
   - `internal/services/explorer_move_test.go`:
     - `TestMoveItem_LocalToLocal_Success`
     - `TestMoveItem_CanaryBlocked`
     - `TestMoveItem_JailEscapeBlocked`
     - `TestPurgeDirectory_CanaryBlocked`
   - `internal/server/handlers_operations_test.go`:
     - Test HTTP endpoints with valid payloads and deliberate jail traversal attempts (`../`).
2. **Quality Gates**:
   - `./scripts/check.sh all` validating format, vet, staticcheck, LOC compliance (<400 LOC per file), template safety, and race detector.
3. **End-to-End Headless Chrome Tests**:
   - `tests/e2e/dashboard_smoke_test.go`:
     - Verify workspace dual panes reflect active profile local and remote paths.
     - Verify review candidate deep link opens workspace and highlights target.

## [Implementation Order]

1. **Step 1: Core Client Operation Wrappers & Models**
   - Create `internal/models/operations.go` and `internal/models/upload.go`.
   - Implement `internal/core/client/operations.go` and `internal/core/client/operations_test.go`.
   - Verify: `go test -v -count=1 ./internal/core/client/... -run "TestMove|TestPurge|TestPublicLink"`
2. **Step 2: Service Layer Profile-Bound Operations (Move, Purge, Link)**
   - Implement `internal/services/explorer_move.go`, `explorer_purge.go`, `explorer_link.go`.
   - Implement unit tests in `internal/services/explorer_move_test.go`.
   - Verify: `go test -v -count=1 ./internal/services/... -run "TestMoveItem|TestPurge|TestPublicLink"`
3. **Step 3: Server Handlers & Endpoint Routing**
   - Implement `internal/server/handlers_operations.go` and `internal/server/handlers_operations_test.go`.
   - Wire route registration in `internal/server/server.go`.
   - Verify: `go test -v -count=1 ./internal/server/... -run TestHandleOperations`
4. **Step 4: Non-Blocking Upload Dock Service**
   - Implement `internal/services/upload_dock.go` and unit tests in `internal/services/upload_dock_test.go`.
   - Add endpoint in `internal/server/handlers_operations.go`.
   - Verify: `go test -v -count=1 ./internal/services/... -run TestUploadDock`
5. **Step 5: Frontend Workspace Layout & Profile-Bound Panes**
   - Update `component_explorer_toolbar.html`, `view_explorer.html`, `explorer.js`, and `explorer_actions.js`.
   - Verify: `./scripts/check.sh templates && ./scripts/check.sh loc`
6. **Step 6: Sync Review Remediation Bridge**
   - Update `modal_review.html`, `component_context_menu.html`, and `review.js`.
   - Verify: `./scripts/check.sh templates && ./scripts/check.sh loc`
7. **Step 7: Full Quality Gates & E2E Smoke Testing**
   - Add E2E headless test scenarios in `tests/e2e/dashboard_smoke_test.go`.
   - Execute full pre-commit check: `./scripts/check.sh all`.
