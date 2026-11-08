# Implementation Plan: File Explorer Remote Target Synchronization & Directory Creation (Mkdir)

## Proposed Changes

Address two critical usability and reliability issues in the File Explorer Workspace:
1. **Explorer Remote Target Desynchronization & Self-Operation Safeguards**:
   - In the frontend, asynchronous loading of profiles and remotes caused the dual-pane file explorer to initialize before the active profile was loaded, defaulting the right pane to `""` which coerced to `"local"`. As a result, both left and right panes displayed the exact same local folder.
   - Performing a file move/copy when source and destination are identical caused file system collisions and errors.
   - Fix: Ensure `initExplorer()`, `fetchProfiles()`, and `fetchRemotes()` synchronize the right pane target with `activeProfile.remote` (or the first available discovered remote), and safeguard backend `CopyItemWithOptions` and `MoveItem` against identical source and destination paths.

2. **Add "Create Folder" (Mkdir) Capability**:
   - Provide a native "New Folder" action for both local storage and remote targets.
   - Add Go domain models (`MkdirRequest`, `MkdirResponse`), `ExplorerService.MakeDir` with strict path jailing for local storage and `RcloneClient.Mkdir` for remotes.
   - Add `POST /api/operations/mkdir` with audit logging.
   - Add `modal_explorer_mkdir.html` and UI buttons in both explorer pane headers.

### Affected Components & Packages

- `internal/models/operations.go`: Structs `MkdirRequest` and `MkdirResponse`.
- `internal/services/explorer_mkdir.go`: Implementation of `MakeDir(ctx, req, localBase)`.
- `internal/services/explorer_mkdir_test.go`: Unit tests for directory creation (local path jail, remote client mock, canary name rejection).
- `internal/services/explorer_ops.go`: Guard in `CopyItemWithOptions` against identical source and destination.
- `internal/services/explorer_move.go`: Guard in `MoveItem` against identical source and destination.
- `internal/server/handlers_operations.go`: Route `POST /api/operations/mkdir` and audit logging.
- `internal/server/handlers_operations_test.go`: HTTP handler tests for mkdir.
- `dashboard/frontend/partials/modals/modal_explorer_mkdir.html`: Modal dialog for creating new folders.
- `dashboard/frontend/index.html`: Include template `modal_explorer_mkdir`.
- `dashboard/frontend/partials/view_explorer.html`: "New Folder" button in left and right pane headers.
- `dashboard/frontend/js/explorer_operations.js`: State `mkdirModal`, `promptCreateFolder(paneKey)`, and `executeCreateFolder()`.
- `dashboard/frontend/js/explorer.js`: Target synchronization in `initExplorer()`, `syncExplorerWithActiveProfile()`, and `fetchExplorerDirectory()`.
- `dashboard/frontend/js/profiles.js`: Call `syncExplorerWithActiveProfile()` in `fetchProfiles()`.
- `dashboard/frontend/app.js`: Await profile and remote fetch before initial tab explorer check in `initDashboard()`.

### Technical Strategy & Data Flow

1. **Remote Target Synchronization**:
   - In `initDashboard()`, `await this.fetchProfiles()` and `await this.fetchRemotes(false)` before running tab initialization.
   - In `fetchProfiles()`, if the right pane target is unconfigured or equal to "local", invoke `syncExplorerWithActiveProfile()`.
   - In `initExplorer()`, if `profilesData.list` is empty, wait for profiles/remotes, and if `p.remote` is present, set right target to `p.remote.replace(/:+$/, "")`. If no profile remote exists, fall back to `remotesData.remotes[0].name`.
   - In `CopyItemWithOptions` and `MoveItem`, detect if `srcFs == dstFs && srcFile == dstFile` and return an explanatory error rather than dispatching a destructive self-copy/move.

2. **Create Folder Workflow**:
   - User clicks `+ New Folder` in left or right explorer pane header.
   - UI opens `mkdirModal` pre-scoped to `paneKey` and the current subpath.
   - On submission, UI posts to `/api/operations/mkdir` with `{ remote, path, profile_id }`.
   - Backend checks path traversal (`..`), verifies local path jail or remote target, creates directory (`os.MkdirAll` with `0755` or `c.Mkdir`), records audit event, and returns success.
   - UI closes modal, displays toast, and refreshes the directory listing of that pane.

### Safety & Error Handling

- **Path Traversal Jail**: Local directory creation enforces `isLocalPathJailed` within profile local root (`os.MkdirAll(safePath, 0755)`).
- **Canary Name Guard**: Folder creation matching `RCLONE_TEST` is strictly forbidden.
- **Identical Source/Destination Protection**: Moving or copying between identical source and destination directories/paths is intercepted with a clean validation error.

## Verification Plan

- Unit test `TestExplorerService_MakeDir` in `internal/services/explorer_mkdir_test.go`.
- Server test `TestHandler_MakeDir` in `internal/server/handlers_operations_test.go`.
- Browser E2E test covering New Folder creation and verified distinct panes.
- Quality gates: `./scripts/check.sh all`.

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
