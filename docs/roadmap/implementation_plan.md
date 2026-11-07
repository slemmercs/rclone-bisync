# Implementation Plan

## [Overview]
Recreate the core functional capabilities of `rclone/rclone-ui` (Features 1 through 9) natively within Bisync Hub's single-binary Go backend and zero-build Alpine.js frontend across four phased, incrementally verified milestones.

While `rclone-ui` is a standalone Tauri/React desktop application that wraps raw Rclone commands without safety barriers, Bisync Hub will implement these capabilities as cohesive domain services governed by POSIX advisory locking (`flock`), local filesystem path confinement jails, and cryptographic audit logging. This expansion turns Bisync Hub from a sync-specialized manager into a comprehensive cloud storage cockpit with zero runtime dependencies.

The implementation is structured into 4 sequential phases:
- **Phase 1: Operational Controls & File Inspection** (Dynamic Bandwidth Throttling, File Previews/Download, Configuration Encryption)
- **Phase 2: Dual-Pane File Commander & Ad-Hoc Transfers** (Dual-pane browsing, direct drag-and-drop upload, one-way copy/move/purge)
- **Phase 3: Storage Virtualization & Protocol Servers** (VFS Remote Mount Manager with auto-unmount watchdog, WebDAV/SFTP/HTTP/DLNA servers)
- **Phase 4: Remote Provisioning & Multi-Host Fleet** (Interactive Remote Creation Wizard via `/config/providers`, multi-host daemon switching)

---

## [Types]

### 1. Bandwidth & Core Stats Models (`internal/models/core.go`)
```go
package models

type BandwidthLimitRequest struct {
    Rate string `json:"rate"` // e.g. "10M", "500k", "off"
}

type BandwidthLimitResponse struct {
    BytesPerSecond int64  `json:"bytes_per_second"`
    RateFormatted  string `json:"rate_formatted"`
    Status         string `json:"status"`
}

type CoreStatsResponse struct {
    Bytes         int64              `json:"bytes"`
    Speed         float64            `json:"speed"`
    Transfers     int                `json:"transfers"`
    Checks        int                `json:"checks"`
    Errors        int                `json:"errors"`
    FatalErrors   bool               `json:"fatal_errors"`
    Transferring  []ActiveTransfer   `json:"transferring,omitempty"`
}

type ActiveTransfer struct {
    Name       string  `json:"name"`
    Size       int64   `json:"size"`
    Bytes      int64   `json:"bytes"`
    Percentage int     `json:"percentage"`
    Speed      float64 `json:"speed"`
    ETA        int64   `json:"eta"`
}
```

### 2. File Preview & Direct Download Models (`internal/models/preview.go`)
```go
package models

type FilePreviewResponse struct {
    Remote      string `json:"remote"`
    Path        string `json:"path"`
    Name        string `json:"name"`
    Size        int64  `json:"size"`
    MimeType    string `json:"mime_type"`
    IsBinary    bool   `json:"is_binary"`
    Content     string `json:"content,omitempty"`      // UTF-8 string for text/markdown/json
    DataURL     string `json:"data_url,omitempty"`     // Base64 data URI for images <= 5MB
    Truncated   bool   `json:"truncated"`
    Error       string `json:"error,omitempty"`
}
```

### 3. VFS Mount Models (`internal/models/mount.go`)
```go
package models

import "time"

type MountPoint struct {
    ID          string    `json:"id"`
    Remote      string    `json:"remote"`
    RemotePath  string    `json:"remote_path"`
    MountPoint  string    `json:"mount_point"`
    MountedAt   time.Time `json:"mounted_at"`
    Status      string    `json:"status"` // "mounted", "unmounting", "error"
    VFSCacheMode string   `json:"vfs_cache_mode"`
    Error       string    `json:"error,omitempty"`
}

type MountRequest struct {
    Remote       string `json:"remote"`
    RemotePath   string `json:"remote_path"`
    MountPoint   string `json:"mount_point"`
    VFSCacheMode string `json:"vfs_cache_mode,omitempty"` // default "full"
    ReadOnly     bool   `json:"read_only,omitempty"`
}

type UnmountRequest struct {
    MountPoint string `json:"mount_point"`
    Force      bool   `json:"force,omitempty"`
}
```

### 4. Protocol Serve Models (`internal/models/serve.go`)
```go
package models

import "time"

type ServeInstance struct {
    ID        string    `json:"id"`
    Protocol  string    `json:"protocol"` // "webdav", "http", "sftp", "dlna"
    Remote    string    `json:"remote"`
    Path      string    `json:"path"`
    Addr      string    `json:"addr"`     // e.g. "127.0.0.1:8080"
    User      string    `json:"user,omitempty"`
    StartedAt time.Time `json:"started_at"`
    Status    string    `json:"status"`
}

type ServeStartRequest struct {
    Protocol string `json:"protocol"`
    Remote   string `json:"remote"`
    Path     string `json:"path"`
    Addr     string `json:"addr"`
    User     string `json:"user,omitempty"`
    Pass     string `json:"pass,omitempty"`
}

type ServeStopRequest struct {
    ID string `json:"id"`
}
```

### 5. Ad-Hoc Operations & Upload Models (`internal/models/operations.go`)
```go
package models

type MoveRequest struct {
    SrcRemote string `json:"src_remote"`
    SrcPath   string `json:"src_path"`
    DstRemote string `json:"dst_remote"`
    DstPath   string `json:"dst_path"`
    IsDir     bool   `json:"is_dir"`
}

type MoveResponse struct {
    Status string `json:"status"`
    Src    string `json:"src"`
    Dst    string `json:"dst"`
    Error  string `json:"error,omitempty"`
}

type PurgeRequest struct {
    Remote string `json:"remote"`
    Path   string `json:"path"`
}

type PurgeResponse struct {
    Status string `json:"status"`
    Path   string `json:"path"`
    Error  string `json:"error,omitempty"`
}
```

### 6. Config Encryption Models (`internal/models/encryption.go`)
```go
package models

type EncryptionStatusResponse struct {
    IsEncrypted bool   `json:"is_encrypted"`
    ConfigFile  string `json:"config_file"`
}

type SetEncryptionRequest struct {
    Password string `json:"password"`
}
```

### 7. Multi-Host Daemon Models (`internal/models/hosts.go`)
```go
package models

type RemoteDaemonHost struct {
    ID        string `json:"id"`
    Name      string `json:"name"`
    URL       string `json:"url"`
    User      string `json:"user,omitempty"`
    IsActive  bool   `json:"is_active"`
}
```

---

## [Files]

### New Files
- `internal/models/core.go`: Data structures for bandwidth and transfer stats.
- `internal/models/preview.go`: Data structures for file preview and streaming.
- `internal/models/mount.go`: Data structures for VFS mount management.
- `internal/models/serve.go`: Data structures for protocol serve instances.
- `internal/models/operations.go`: Data structures for move and purge operations.
- `internal/models/encryption.go`: Data structures for `rclone.conf` encryption state.
- `internal/models/hosts.go`: Data structures for multi-daemon host profiles.
- `internal/services/core_service.go`: Business logic for `/core/bwlimit` and `/core/stats`.
- `internal/services/core_service_test.go`: Unit tests for bandwidth calculation and mock responses.
- `internal/services/explorer_preview.go`: Logic for safe text/image preview extraction with jail containment.
- `internal/services/explorer_preview_test.go`: Unit tests for text truncation and mime-type detection.
- `internal/services/mount_service.go`: Lifecycle supervisor for VFS mounts, auto-reconnect, and signal unmounting.
- `internal/services/mount_service_test.go`: Unit tests for mount registry and mock RC calls.
- `internal/services/serve_service.go`: Supervisor for protocol serve instances (WebDAV, SFTP, HTTP, DLNA).
- `internal/services/serve_service_test.go`: Unit tests for serve lifecycle.
- `internal/services/config_encryption_service.go`: Handler for `rclone.conf` password encryption status and updates.
- `internal/services/host_pool_service.go`: Multi-host daemon manager for remote `rclone rcd` instances.
- `internal/server/handlers_core.go`: HTTP endpoints for `/api/core/bwlimit` and `/api/core/stats`.
- `internal/server/handlers_mount.go`: HTTP endpoints for `/api/mounts/*`.
- `internal/server/handlers_serve.go`: HTTP endpoints for `/api/serve/*`.
- `internal/server/handlers_operations.go`: HTTP endpoints for move, purge, upload, and download.
- `dashboard/frontend/js/commander.js`: Dual-pane controller, synchronized directory navigation, and drag-and-drop.
- `dashboard/frontend/js/mounts.js`: Reactive Alpine controller for active mounts and VFS metrics.
- `dashboard/frontend/js/serves.js`: Reactive Alpine controller for active protocol shares.
- `dashboard/frontend/partials/modals/modal_file_preview.html`: In-app file viewer for code, markdown, and images.
- `dashboard/frontend/partials/modals/modal_create_remote.html`: Step-by-step remote creation wizard with provider fields.
- `dashboard/frontend/partials/modals/modal_serve.html`: Share modal for exposing paths over WebDAV/SFTP.
- `dashboard/frontend/partials/view_mounts.html`: Dedicated VFS Mount Manager view card.

### Existing Files to Modify
- `internal/core/client/client.go`: Add `SetBandwidthLimit`, `GetCoreStats`, and multi-target support.
- `internal/core/client/fs_ops.go`: Add `MoveFile`, `MoveDir`, and streaming reader for download.
- `internal/services/explorer_service.go`: Integrate preview and upload methods.
- `internal/services/explorer_ops.go`: Add `MoveItem` with path jail protection.
- `internal/app/container.go`: Register `CoreService`, `MountService`, `ServeService`, and `HostPoolService`.
- `internal/server/server.go`: Wire new routes under `/api/core/*`, `/api/mounts/*`, `/api/serve/*`, `/api/operations/*`.
- `dashboard/frontend/app.js`: Initialize bandwidth poller, active host selector, and mount status badges.
- `dashboard/frontend/index.html`: Register new script tags (`commander.js`, `mounts.js`, `serves.js`).
- `dashboard/frontend/partials/view_explorer.html`: Add toggle for Dual-Pane Commander layout with left/right dropzones.
- `dashboard/frontend/partials/modals/modal_global_settings.html`: Add bandwidth default slider and configuration encryption management.

---

## [Functions]

### New Functions

#### `internal/core/client/client.go`
- `SetBandwidthLimit(ctx context.Context, rate string) (*models.BandwidthLimitResponse, error)`: Calls `POST /core/bwlimit` with `{"rate": rate}`.
- `GetCoreStats(ctx context.Context) (*models.CoreStatsResponse, error)`: Calls `POST /core/stats` and parses active transfers.
- `MountRemote(ctx context.Context, fs, mountPoint string, vfsOpt map[string]interface{}) error`: Calls `POST /mount/mount`.
- `UnmountRemote(ctx context.Context, mountPoint string) error`: Calls `POST /mount/unmount`.
- `ListMounts(ctx context.Context) ([]string, error)`: Calls `POST /mount/listmounts`.
- `StartServe(ctx context.Context, protocol, fs string, opt map[string]interface{}) (int, error)`: Calls `POST /serve/<protocol>`.
- `StopServe(ctx context.Context, jobID int) error`: Calls `POST /job/stop` or `/serve/stop`.

#### `internal/services/explorer_preview.go`
- `PreviewItem(ctx context.Context, remote, path string, localBase ...string) (models.FilePreviewResponse, error)`: Inspects file size, reads up to 2MB for text/code or 5MB for images (converted to data URI), with strict path confinement.

#### `internal/services/explorer_ops.go`
- `MoveItem(ctx context.Context, srcRemote, srcPath, dstRemote, dstPath string, isDir bool, localBase ...string) (models.MoveResponse, error)`: Confines source and destination paths, verifies canary file protection, and executes `MoveFile` or `MoveDir`.
- `UploadItem(ctx context.Context, targetRemote, targetDir, filename string, r io.Reader, localBase ...string) error`: Writes uploaded file stream into local jail or streams to remote via Rclone RC `/operations/uploadfile`.

#### `internal/services/mount_service.go`
- `NewMountService(cfg *config.Config, client *client.RcloneClient) *MountService`: Initializes mount supervisor.
- `Mount(ctx context.Context, req models.MountRequest) (*models.MountPoint, error)`: Validates mount point directory, checks for existing mounts, issues RC mount command, and registers supervisor probe.
- `Unmount(ctx context.Context, mountPoint string, force bool) error`: Issues RC unmount with fallback to `fusermount -u` on Linux.
- `ListMounts(ctx context.Context) ([]models.MountPoint, error)`: Merges active Rclone mounts with tracked metadata.
- `Shutdown(ctx context.Context) error`: Unmounts all active mounts cleanly when Bisync Hub exits.

#### `internal/services/serve_service.go`
- `StartServe(ctx context.Context, req models.ServeStartRequest) (*models.ServeInstance, error)`: Validates port availability, launches protocol server via RC, and registers instance.
- `StopServe(ctx context.Context, id string) error`: Halts active protocol server.
- `ListServes(ctx context.Context) []models.ServeInstance`: Lists running protocol servers.

### Modified Functions
- `internal/server/server.go:routes()`: Mount `/api/core/*`, `/api/mounts/*`, `/api/serve/*`, `/api/explorer/preview`, `/api/explorer/download`, `/api/explorer/upload`, `/api/operations/move`, `/api/operations/purge`.
- `internal/services/explorer_ops.go:DeleteItem()`: Add `PurgeDir` support when `isDir=true` with canary safety validation.

---

## [Classes] (Go Domain Structs)

- `CoreService`: Manages `/core/bwlimit`, `/core/stats`, and speed presets.
  - File: `internal/services/core_service.go`
  - Methods: `GetStats(ctx)`, `SetLimit(ctx, rate)`, `GetLimit(ctx)`.
- `MountService`: Supervises VFS mount points, FUSE health, and clean OS unmounting.
  - File: `internal/services/mount_service.go`
  - Methods: `Mount(ctx, req)`, `Unmount(ctx, path, force)`, `ListMounts(ctx)`, `WatchdogLoop(ctx)`.
- `ServeService`: Manages local protocol sharing (WebDAV, SFTP, HTTP, DLNA).
  - File: `internal/services/serve_service.go`
  - Methods: `Start(ctx, req)`, `Stop(ctx, id)`, `List(ctx)`.
- `ConfigEncryptionService`: Checks and mutates `rclone.conf` password protection.
  - File: `internal/services/config_encryption_service.go`
  - Methods: `GetStatus(ctx)`, `SetPassword(ctx, pass)`, `RemovePassword(ctx)`.
- `HostPoolService`: Coordinates connections to multiple remote `rclone rcd` instances.
  - File: `internal/services/host_pool_service.go`
  - Methods: `GetActiveClient()`, `SwitchHost(hostID)`, `ListHosts()`.

---

## [Dependencies]

- **Zero new external Go module dependencies**: Standard library (`net/http`, `io`, `os`, `path/filepath`, `sync`, `os/signal`) and existing Rclone RC client.
- **Frontend**: Zero build step. Uses existing Tailwind utility classes, Alpine.js reactive components, and Lucide icons.

---

## [Testing]

### Test Requirements
- `internal/services/core_service_test.go`: Mock Rclone RC `/core/bwlimit` and `/core/stats` returning speed and transfer byte counts. Verify rate string formatting (`10M` -> `10485760 B/s`).
- `internal/services/explorer_preview_test.go`:
  - Test text files < 2MB return complete UTF-8 content.
  - Test text files > 2MB return truncated content with `Truncated: true`.
  - Test image files (PNG, JPEG) return base64 data URLs.
  - Test path traversal attempts (`../../etc/passwd`) are rejected with 403 Forbidden.
- `internal/services/mount_service_test.go`: Verify mount command generation, error handling for busy mount points, and SIGTERM cleanup simulation.
- `internal/services/serve_service_test.go`: Verify protocol validation (rejecting invalid protocols) and clean job tracking.
- `tests/e2e/dashboard_smoke_test.go`:
  - E2E assertion: Dual-pane toggle switches between single explorer and side-by-side Commander.
  - E2E assertion: File preview modal opens on file click and displays content.
  - E2E assertion: Bandwidth throttle slider correctly calls API and updates status pill.
- `./scripts/check.sh all`: Run all 7 quality stages (format, vet, staticcheck, templates, LOC limit, unit tests, race detector).

---

## [Implementation Order]

### Milestone 1: Operational Controls & File Inspection (Features 3, 4, 9)
1. Implement bandwidth and core stats models (`internal/models/core.go`) and client RC methods (`SetBandwidthLimit`, `GetCoreStats`).
2. Implement `CoreService` (`internal/services/core_service.go`) and HTTP handlers (`internal/server/handlers_core.go`).
3. Add Bandwidth speed slider and live transfer counter to header in `view_telemetry.html` and `app.js`.
4. Implement file preview models (`internal/models/preview.go`) and service (`internal/services/explorer_preview.go`) with path jail validation.
5. Create `modal_file_preview.html` and add preview/download buttons to `view_explorer.html`.
6. Implement config encryption status endpoint and UI indicator in `modal_global_settings.html`.
7. Verify Milestone 1 via unit tests and `./scripts/check.sh all`.

### Milestone 2: Dual-Pane File Commander & Ad-Hoc Transfers (Features 1, 6)
8. Extend `internal/services/explorer_ops.go` with `MoveItem` and multipart `UploadItem`.
9. Create `internal/server/handlers_operations.go` exposing `/api/operations/move`, `/api/operations/purge`, `/api/explorer/upload`, and `/api/explorer/download`.
10. Refactor `dashboard/frontend/partials/view_explorer.html` to add Commander mode with two synchronized panes (`leftPane`, `rightPane`).
11. Implement drag-and-drop file upload zones with lock coordination dock (`commander.js`).
12. Add Commander toolbar actions: `[Copy]`, `[Move]`, `[Delete/Purge]`, `[Preview]`.
13. Verify Milestone 2 via unit tests, E2E drag-and-drop tests, and `./scripts/check.sh all`.

### Milestone 3: Storage Virtualization & Protocol Servers (Features 2, 5)
14. Implement mount models (`internal/models/mount.go`) and `MountService` (`internal/services/mount_service.go`) with background health watchdog and clean shutdown unmount handlers.
15. Implement `handlers_mount.go` exposing `/api/mounts/*`.
16. Implement serve models (`internal/models/serve.go`), `ServeService` (`internal/services/serve_service.go`), and `handlers_serve.go`.
17. Build `view_mounts.html` and `modal_serve.html` with Alpine controller `mounts.js` and `serves.js`.
18. Verify Milestone 3 via mock mount tests and `./scripts/check.sh all`.

### Milestone 4: Cloud Remote Provisioning & Multi-Host Fleet (Features 7, 8)
19. Implement `remote_wizard_service.go` querying Rclone `/config/providers` to render dynamic OAuth and parameter schemas.
20. Create `modal_create_remote.html` step-by-step wizard.
21. Implement `HostPoolService` (`internal/services/host_pool_service.go`) supporting remote Rclone RC endpoints.
22. Add Host Switcher dropdown to dashboard header.
23. Run complete pre-flight suite (`./scripts/check.sh all`) and update documentation.
