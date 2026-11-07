# Implementation Plan: Pure Profile Sync Staging & Remediation Workspace

## [Overview]
Transform the explorer into a profile-bound Sync Staging & Remediation Workspace where the left pane is strictly jailed to the active profile's local storage and the right pane is bound to the active profile's cloud remote, enabling atomic file/directory moves, directory purges, cloud public web links, non-blocking upload coordination during active sync, and seamless deep-link triage from review sessions.

The workspace operates as a 1:1 visual counterpart to the sync engine. Changing the selected profile simultaneously shifts both panes to that profile's root paths. Staging actions (dragging files left-to-right to push changes, or right-to-left to pull changes) directly manipulate disk/cloud state without triggering a full bisync run. Safety is guaranteed by canary inviolability (`RCLONE_TEST`), strict local path confinement (`filepath.Rel`), contextual deletion warnings, and audit logging with distinct staging vs remediation provenance.

## [Types]

### 1. `internal/models/operations.go`
- `MoveRequest`:
  ```go
  type MoveRequest struct {
      ProfileID string `json:"profile_id"`
      SrcRemote string `json:"src_remote"` // "local" or remote name
      SrcPath   string `json:"src_path"`
      DstRemote string `json:"dst_remote"` // "local" or remote name
      DstPath   string `json:"dst_path"`
      IsDir     bool   `json:"is_dir"`
  }
  ```
- `MoveResponse`:
  ```go
  type MoveResponse struct {
      Status string `json:"status"` // "success" or "error"
      Src    string `json:"src"`
      Dst    string `json:"dst"`
      Error  string `json:"error,omitempty"`
  }
  ```
- `PurgeRequest`:
  ```go
  type PurgeRequest struct {
      ProfileID string `json:"profile_id"`
      Remote    string `json:"remote"` // "local" or remote name
      Path      string `json:"path"`
  }
  ```
- `PurgeResponse`:
  ```go
  type PurgeResponse struct {
      Status string `json:"status"`
      Path   string `json:"path"`
      Error  string `json:"error,omitempty"`
  }
  ```
- `PublicLinkRequest`:
  ```go
  type PublicLinkRequest struct {
      ProfileID string `json:"profile_id"`
      Remote    string `json:"remote"`
      Path      string `json:"path"`
      Unlink    bool   `json:"unlink,omitempty"`
  }
  ```
- `PublicLinkResponse`:
  ```go
  type PublicLinkResponse struct {
      URL       string `json:"url,omitempty"`
      Supported bool   `json:"supported"`
      Status    string `json:"status"`
      Error     string `json:"error,omitempty"`
  }
  ```

### 2. `internal/models/upload.go`
- `UploadQueueItem`:
  ```go
  type UploadQueueItem struct {
      ID         string    `json:"id"`
      ProfileID  string    `json:"profile_id"`
      Remote     string    `json:"remote"`
      Path       string    `json:"path"`
      FileName   string    `json:"file_name"`
      Size       int64     `json:"size"`
      Status     string    `json:"status"` // "queued", "waiting_for_sync", "uploading", "completed", "failed"
      Error      string    `json:"error,omitempty"`
      CreatedAt  time.Time `json:"created_at"`
  }
  ```
- `UploadDockStatus`:
  ```go
  type UploadDockStatus struct {
      PendingCount int               `json:"pending_count"`
      IsSyncActive bool              `json:"is_sync_active"`
      Items        []UploadQueueItem `json:"items"`
  }
  ```

## [Files]

### New Files
- `internal/models/operations.go`: Request and response contract DTOs for Move, Purge, and PublicLink.
- `internal/models/upload.go`: Request and response contract DTOs for non-blocking upload dock.
- `internal/core/client/operations.go`: Rclone RC client wrappers for `/operations/movefile`, `/operations/movedir`, `/operations/purge`, `/operations/publiclink`.
- `internal/core/client/operations_test.go`: Unit tests for operations client using `httptest.Server`.
- `internal/services/explorer_move.go`: Domain service method `MoveItem` handling local atomic renames, cloud moves, canary blocks, and local jail validation.
- `internal/services/explorer_move_test.go`: Unit tests verifying `MoveItem` under valid moves, canary blocks, and jail escape attempts.
- `internal/services/explorer_purge.go`: Domain service method `PurgeDirectory` enforcing canary recursion protection and profile jail bounds.
- `internal/services/explorer_link.go`: Domain service method `GetPublicLink` with graceful degradation for unsupported remotes.
- `internal/services/upload_dock.go`: Lightweight thread-safe in-memory upload dock coordinating with sync engine state.
- `internal/services/upload_dock_test.go`: Unit tests verifying lock detection and state transitions.
- `internal/server/handlers_operations.go`: HTTP endpoints for `/api/operations/move`, `/api/operations/purge`, `/api/operations/publiclink`, `/api/explorer/dock`.
- `internal/server/handlers_operations_test.go`: HTTP tests covering authorization, jail containment, and error responses.

### Modified Existing Files
- `internal/server/server.go`: Call `s.registerOperationsRoutes()` to attach new endpoints while staying strictly under 400 LOC.
- `dashboard/frontend/partials/components/component_explorer_toolbar.html`:
  - Bind to active profile with lock indicator (`Profile: [Name] (Local <-> Remote)`).
  - Add Upload Dock status indicator (`Uploads: N`).
  - Active Remediation Cockpit Banner when navigating from a conflict review item.
- `dashboard/frontend/partials/view_explorer.html`:
  - Render ambient conflict/deletion badges on file rows matching candidate items.
  - Contextual sync impact warnings on delete/purge actions.
- `dashboard/frontend/partials/modals/modal_review.html`:
  - Add "Inspect in Workspace" button (🔍) to candidate rows.
- `dashboard/frontend/partials/components/component_context_menu.html`:
  - Add inline review decision triggers (`Keep Local`, `Keep Remote`, `Accept Delete`) when right-clicking candidate rows.
- `dashboard/frontend/js/explorer.js` & `dashboard/frontend/js/review.js`:
  - Methods `inspectCandidateInWorkspace(candidate)` and `getReviewCandidate(path)`.
- `docs/roadmap/task_plan.md`: Sequential, atomic micro-tasks tracking each step.

## [Functions]

### New Functions
1. `c.MoveFile(ctx context.Context, srcFs, srcRemote, dstFs, dstRemote string) error`
   - File: `internal/core/client/operations.go`
   - Purpose: Calls Rclone RC `operations/movefile` with normalized fs endpoints.
2. `c.MoveDir(ctx context.Context, srcFs, dstFs string) error`
   - File: `internal/core/client/operations.go`
   - Purpose: Calls Rclone RC `operations/movedir`.
3. `c.Purge(ctx context.Context, fs, remote string) error`
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
