# Implementation Plan: Provider-Aware "Open on Remote" Capability Filtering

## Proposed Changes

Filter out the "Open on Remote" context menu item when the active pane's storage provider does not support public sharing links (`operations/publiclink`), preventing user-facing errors on unsupported backends (such as local filesystem, alias to local, SFTP, raw S3).

### Affected Components & Packages

- `internal/core/client/remotes.go`: Add `GetRemoteFsInfo(ctx, remote)` querying Rclone RC `/operations/fsinfo` for backend features.
- `internal/services/telemetry_remotes.go`: In `GetRemotesSummaryCached`, evaluate `Features.PublicLink` from `GetRemoteFsInfo` with fallback to known provider types, populating `"can_public_link"` on each remote in `RemotesResponse`.
- `dashboard/frontend/js/explorer_operations.js`: Implement `canOpenOnRemote(paneKey)` to check remote capability; dynamically update `found.can_public_link = false` if runtime API returns unsupported.
- `dashboard/frontend/partials/components/component_context_menu.html`: Update template conditional from `target !== 'local'` to `canOpenOnRemote(contextMenu.paneKey)`.
- `internal/services/telemetry_service_test.go`: Add unit test assertions for `can_public_link` metadata in remotes summary.

### Technical Strategy & Data Flow

1. **Backend Capability Discovery**:
   - `c.GetRemoteFsInfo(ctx, remote)` queries `/operations/fsinfo`.
   - `telemetry_remotes.go` extracts `Features.PublicLink`. If `fsinfo` is unavailable (e.g. timeout or offline remote), falls back to provider type whitelist (`drive`, `onedrive`, `dropbox`, `box`, `pcloud`, `yandex`, `mega`, `mailru`, `opendrive`, `linkbox`, `pikpak`, `zoho`, `protondrive`).
   - `"can_public_link": bool` is included in the remote's JSON dictionary returned by `GET /api/remotes`.

2. **Frontend Filtering**:
   - `canOpenOnRemote(paneKey)` matches `explorer[paneKey].target` against `remotesData.remotes`.
   - If target is `"local"` or not found, returns `false`.
   - If `remote.can_public_link` is explicit, returns its boolean value; otherwise falls back to provider whitelist.
   - If `openOnRemote` ever returns `supported: false`, updates cached remote state to immediately suppress the item.

## Verification Plan

- `go test -v -count=1 ./internal/services/... -run TestGetRemotesSummary`
- `go test -v -count=1 ./tests/e2e/... -run TestDashboard_HeadlessChromeSmoke`
- `./scripts/check.sh all`
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
