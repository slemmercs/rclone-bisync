# Implementation Plan: Tier 4 Rclone Sandbox UI Journey E2E Suite

## Proposed Changes

Implement a comprehensive browser-driven end-to-end integration test (`tests/e2e/sandbox_ui_journey_test.go`) that connects headless Chromium/Chrome to a live, ephemeral `rclone rcd` daemon and a live Bisync Hub server operating exclusively on real local folders (`t.TempDir()`).

This suite bridges the gap between the headless browser UI tests (which used mock servers) and the Tier 4 backend sandbox integration tests, validating the full user journey:
1. Directory exploration against real filesystem structures.
2. File move/rename via modal directly modifying disk.
3. Directory purge with safety confirmation removing disk folders.
4. Canary protection (`RCLONE_TEST`) actively blocking destructive actions in the UI.
5. In-flight upload dock staging and mid-sync deferral.

### Affected Components & Packages

- `tests/e2e/sandbox_ui_journey_test.go`: Real sandbox E2E test file launching `rclone rcd`, Bisync Hub server, and Chromedp runner.
- `docs/roadmap/task_plan.md`: Atomic tasks tracking sandbox UI journey execution.

### Technical Strategy & Data Flow

1. **Ephemeral Real Rclone Daemon**:
   - Verify `rclone` and Chrome/Chromium availability in PATH (`t.Skip` gracefully if absent).
   - Write dynamic isolated `rclone.conf` defining a `local` remote target:
     ```ini
     [sandrem]
     type = local
     ```
   - Spawn `rclone rcd` with `--rc-addr 127.0.0.1:0 --rc-no-auth --config <rclone.conf> --cache-dir <tempDir>/cache`.
   - Wait for healthy state via `client.WaitForHealthy`.

2. **Seeded Local Sandbox Environment**:
   - `localDir`: `<tempDir>/local` (profile `LocalPath`), seeded with:
     - `file_alpha.txt` ("alpha content")
     - `docs/nested.txt` ("nested doc")
     - `to_rename.txt` ("rename me")
   - `remoteDir`: `<tempDir>/remote` (profile `RemotePath` on `sandrem:`), seeded with:
     - `remote_file.txt` ("remote content")
     - `purge_dir/target.txt` ("purge payload")
     - `canary_dir/RCLONE_TEST` (canary marker)

3. **Bisync Hub Configuration & Real Wiring**:
   - Instantiate `config.Config` pointing to `localDir`, remote `sandrem:`, and real Rclone RC URL.
   - Configure active profile with ID `sandbox-profile`.
   - Launch `httptest.NewServer(server.NewServer(cfg, syncSvc, frontendFS))`.

4. **Headless Chrome UI Journey Execution**:
   - **Step 1: Workspace Initialization & Real Directory Fetch**:
     Navigate to dashboard, switch to Explorer tab, verify both left pane (`local`) and right pane (`sandrem:`) populate with real filesystem entries.
   - **Step 2: Real File Move/Rename via Modal**:
     Trigger `promptMoveItem` for `to_rename.txt` $\to$ `renamed_success.txt`, execute move via modal, verify UI updates, and assert via `os.Stat` that `renamed_success.txt` exists on disk and `to_rename.txt` is gone.
   - **Step 3: Real Directory Purge**:
     Trigger `promptPurgeDirectory` for `purge_dir`, verify directory and contents are deleted from disk (`os.Stat` returns `os.IsNotExist`).
   - **Step 4: Canary Protection in UI**:
     Trigger `promptPurgeDirectory` or `promptMoveItem` on canary items, verify warning toast appears in UI and file remains intact on disk.
   - **Step 5: Upload Dock Staging & Execution**:
     Stage upload payload via `/api/explorer/dock`, verify pending badge in UI, flush dock, and verify file is written to destination.

### Safety & Error Handling

- **Zero Cloud Impact**: Uses purely local storage and ephemeral dynamic ports.
- **Process Cleanup**: `t.Cleanup()` ensures all background processes (`rclone rcd`, test servers, browser contexts) terminate cleanly.
- **Canary Inviolability**: Verifies canary files are protected even when requested through live UI events.

## Verification Plan

- Fast unit/e2e run: `go test -v -count=1 ./tests/e2e -run TestDashboard_SandboxUIJourney`
- Full quality check: `./scripts/check.sh all`

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
