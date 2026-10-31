# Design Specification: Remote Mount Manager & Pooled Storage Engine (`internal/mount`)

| Metadata | Details |
|---|---|
| **Status** | `Draft` / Strategic Proposal |
| **Domain** | Remote Mounts, VFS Virtual File Systems & Storage Pooling |
| **Governing ADR** | [ADR-002: Rclone Remote Control API As Source Of Truth](../architecture/adr/ADR-002-Rclone-As-Source-Of-Truth.md) |
| **Core Principles** | [Product Principles](../architecture/PRODUCT_PRINCIPLES.md), [Architecture Principles](../architecture/ARCHITECTURE_PRINCIPLES.md) |

---

## 1. Executive Summary & Vision

Bisync Hub operates primarily as a synchronization orchestrator using `rclone bisync` to maintain 1:1 local copies of cloud storage. However, power users and homelab administrators frequently require **on-demand streaming access** (VFS Virtual File System mounts) without consuming local physical disk space.

Furthermore, advanced users often possess multiple disparate cloud accounts (e.g., 400 GB Google Drive + 20 GB Dropbox + 5 GB OneDrive) and desire a unified, encrypted, pooled virtual drive.

This specification outlines a two-stage architecture:

1. **Stage A: Remote Mount Manager** — Programmatically mount, inspect, configure, and unmount individual Rclone remotes to local mountpoints using the Rclone RC API.
2. **Stage B: Pooled Virtual Storage Engine** — Construct layered virtual topologies (`union` + `chunker` + `crypt`) directly in Rclone configuration via RC APIs, enabling multi-cloud storage pooling with Bisync Hub safety guardrails.

---

## 2. Architectural Principles & Guarantees

* **Rclone RC Supremacy**: All mounts are created, tracked, and unmounted via Rclone's HTTP RC API (`mount/*`). Bisync Hub never executes raw OS `mount` binaries directly unless fallback unmounting is required.
* **Safety Before Automation**: Mounting or unmounting a remote must not disrupt active `bisync` runs. Stale mountpoints or unmounted cloud providers must trigger Stage 1 pre-execution safety halts before synchronization runs can proceed.
* **Zero Runtime Overhead**: Mount management and topology assembly are built directly into the single Go executable with zero CGO or external process wrapper requirements.
* **Local OS Abstraction**: Abstracts Linux/macOS FUSE (`fuse3`, `macFUSE`) and Windows WinFSP differences behind standard Rclone mount parameter DTOs.

---

## 3. Domain Model & Data Contracts (`internal/models/mount.go`)

```go
package models

import "time"

// VFSCacheMode defines Rclone VFS cache levels.
type VFSCacheMode string

const (
	VFSCacheOff     VFSCacheMode = "off"
	VFSCacheMinimal VFSCacheMode = "minimal"
	VFSCacheWrites  VFSCacheMode = "writes"
	VFSCacheFull    VFSCacheMode = "full"
)

// MountOptions contains configuration flags passed to Rclone's mount/mount endpoint.
type MountOptions struct {
	MountPoint      string        `json:"mount_point"`        // e.g., "/mnt/gdrive" or "Z:"
	VFSCacheMode    VFSCacheMode  `json:"vfs_cache_mode"`     // Default: "full"
	VFSCacheMaxAge  time.Duration `json:"vfs_cache_max_age"`  // Default: 1h
	VFSCacheMaxSize int64         `json:"vfs_cache_max_size"` // Max disk space for VFS cache in bytes
	ReadOnly        bool          `json:"read_only"`
	AllowOther      bool          `json:"allow_other"`
	DirCacheTime    time.Duration `json:"dir_cache_time"`     // Default: 5m
}

// ActiveMount represents an active virtual drive tracked by Rclone and Bisync Hub.
type ActiveMount struct {
	ID         string       `json:"id"`          // Unique identifier
	Fs         string       `json:"fs"`          // Remote name (e.g., "gdrive:" or "secure_pool:")
	MountPoint string       `json:"mount_point"` // Local path or drive letter
	MountedAt  time.Time    `json:"mounted_at"`
	IsPooled   bool         `json:"is_pooled"`
	Options    MountOptions `json:"options"`
}

// UnionPolicy defines creation and write strategies for pooled storage.
type UnionPolicy string

const (
	PolicyMostFreeSpace UnionPolicy = "mfs" // Most Free Space
	PolicyLeastUsed     UnionPolicy = "lu"  // Least Used
	PolicyFirstFound    UnionPolicy = "ff"  // First Found
)

// PooledTopology defines a multi-cloud storage stack.
type PooledTopology struct {
	Name             string       `json:"name"`              // Target remote name (e.g., "secure_pool")
	MemberRemotes    []string     `json:"member_remotes"`    // Base remotes: ["gdrive:data", "dropbox:data", "onedrive:data"]
	CreatePolicy     UnionPolicy  `json:"create_policy"`     // Default: "mfs"
	EnableChunking   bool         `json:"enable_chunking"`   // Enable rclone chunker
	ChunkSizeMB      int64        `json:"chunk_size_mb"`     // e.g., 1024 (1 GiB)
	EnableEncryption bool         `json:"enable_encryption"` // Enable rclone crypt
	Passphrase       string       `json:"passphrase,omitempty"` // Masked in JSON responses
	MountOnStart     bool         `json:"mount_on_start"`   // Auto-mount on Bisync Hub daemon start
	MountOptions     MountOptions `json:"mount_options"`
}
```

---

## 4. Stage A Specification: Standalone Remote Mount Manager

Stage A introduces programmatic VFS mounting for individual remotes already configured in Rclone (`gdrive:`, `onedrive:`, `dropbox:`).

```text
                      +-------------------------------+
                      |   Bisync Hub Dashboard UI     |
                      |  [Storage Remotes / Mounts]   |
                      +---------------+---------------+
                                      |
                           HTTP REST  | POST /api/mounts
                                      v
                      +-------------------------------+
                      |      internal/server          |
                      |   (handlers_mount.go)         |
                      +---------------+---------------+
                                      |
                                      v
                      +-------------------------------+
                      |   internal/services/          |
                      |     MountService              |
                      +---------------+---------------+
                                      |
                           RC API     | POST /mount/mount
                                      v
                      +-------------------------------+
                      |      Rclone Daemon (rcd)      |
                      |    [VFS / FUSE / WinFSP]      |
                      +---------------+---------------+
                                      |
                                      v
                        +---------------------------+
                        | Local Virtual Mount Point |
                        |   (e.g., /mnt/onedrive)   |
                        +---------------------------+
```

### 4.1 Backend Engine (`internal/core/client/mounts.go` & `internal/services/mount_service.go`)

* **Mount API Call**: Calls Rclone RC `/mount/mount` with parameters:

  ```json
  {
    "fs": "onedrive:",
    "mountPoint": "/home/csaba/mounts/onedrive",
    "vfsOpt": {
      "CacheMode": 3,
      "CacheMaxAge": 3600000000000
    }
  }
  ```

* **Unmount API Call**: Calls Rclone RC `/mount/unmount` passing `mountPoint`. If FUSE locks cause a timeout, fallback attempts `fusermount -u <path>` (Linux) or `umount -f <path>` (macOS).
* **Liveness Poller**: Periodic 10-second ticker calling `/mount/listmounts` to verify mount health and detect external unmounts or stale FUSE handles.

### 4.2 UI Integration (`partials/view_remotes.html` & `partials/modals/modal_mount.html`)

* **Remote Cards**: Adds a **`[ Mount Virtual Drive ]`** button on each Storage Remote card.
* **Mount Modal**: Allows users to select/enter a local directory (e.g., `~/mounts/onedrive`), choose VFS Cache Mode (`Full`, `Writes`, `Minimal`), and set Read-Only toggles.
* **Active Mounts Bar**: Displays active mounts with real-time IO statistics and a 1-click **`[ Unmount ]`** button.

---

## 5. Stage B Specification: Pooled Storage & Virtual Union Engine

Stage B introduces dynamic multi-cloud pooling by programmatically building stacked Rclone remotes via `config/create` API calls.

```text
                   +-----------------------------------------------+
                   |           User Virtual Drive Endpoint         |
                   |               "secure_pool:"                  |
                   +-----------------------+-----------------------+
                                           |
                                           v
                   +-----------------------------------------------+
                   |                rclone crypt                   |
                   |  - Encrypts filenames & content (Zero-Trust)  |
                   +-----------------------+-----------------------+
                                           |
                                           v
                   +-----------------------------------------------+
                   |               rclone chunker                  |
                   |  - Splits files >1 GiB into smaller parts     |
                   +-----------------------+-----------------------+
                                           |
                                           v
                   +-----------------------------------------------+
                   |                 rclone union                  |
                   |  - Creation Policy: "mfs" (Most Free Space)   |
                   +-------+---------------+---------------+-------+
                           |               |               |
                           v               v               v
                   +---------------+---------------+---------------+
                   |  gdrive:data  |  dropbox:data | onedrive:data |
                   |   (400 GB)    |    (20 GB)    |    (5 GB)     |
                   +---------------+---------------+---------------+
```

### 5.1 Dynamic Topology Provisioning Engine (`internal/services/pool_service.go`)

When a user defines a Pooled Remote, Bisync Hub executes a 3-step atomic configuration transaction over Rclone RC API (`/config/create`):

1. **Step 1: Create `union` Remote (`<pool>_union`)**:

   ```json
   {
     "name": "secure_pool_union",
     "type": "union",
     "parameters": {
       "upstreams": "gdrive:data dropbox:data onedrive:data",
       "action_policy": "mfs",
       "create_policy": "mfs",
       "search_policy": "ff"
     }
   }
   ```
2. **Step 2: Create `chunker` Remote (`<pool>_chunked`)**:

   ```json
   {
     "name": "secure_pool_chunked",
     "type": "chunker",
     "parameters": {
       "remote": "secure_pool_union:",
       "chunk_size": "1GiB"
     }
   }
   ```
3. **Step 3: Create `crypt` Remote (`<pool>`)**:

   ```json
   {
     "name": "secure_pool",
     "type": "crypt",
     "parameters": {
       "remote": "secure_pool_chunked:",
       "filename_encryption": "standard",
       "directory_name_encryption": "true",
       "password": "<encrypted-user-passphrase>"
     }
   }
   ```

### 5.2 Multi-Cloud Partial Outage Guardrail

A primary danger of pooled storage is **partial cloud disappearance** (e.g., Dropbox token expires while Google Drive remains active). If a `bisync` run executes against a partially degraded pool, missing files are interpreted as remote deletions.

Bisync Hub addresses this with an integrated **Pool Canary Probe**:

* Before mounting or syncing a pooled remote, `PoolService` executes lightweight `operations/stat` requests against **all individual member remotes** (`gdrive:`, `dropbox:`, `onedrive:`).
* If any single member remote fails authorization or connectivity, the entire pool mount/sync is **HALTED IMMEDIATELY**, displaying a `MEMBER_REMOTE_OFFLINE` alert on the Cockpit Telemetry banner.

---

## 6. REST API Endpoints (`internal/server/handlers_mount.go`)

| Method | Endpoint | Description |
| :--- | :--- | :--- |
| `GET` | `/api/mounts` | List all active Rclone VFS mounts |
| `POST` | `/api/mounts` | Create a new VFS mount for a remote |
| `DELETE` | `/api/mounts/{id}` | Unmount an active virtual drive |
| `GET` | `/api/mounts/pools` | List configured pooled storage topologies |
| `POST` | `/api/mounts/pools` | Provision a new pooled remote stack (`union`+`chunker`+`crypt`) |
| `POST` | `/api/mounts/pools/{name}/verify` | Execute canary probe across member remotes |

---

## 7. Summary of System Benefits

1. **Stage A** empowers users to stream files instantly from any configured remote without performing full local disk synchronizations.
2. **Stage B** unlocks complete multi-cloud aggregation, enabling users to combine small free cloud tiers into a single, encrypted, high-capacity virtual drive protected by Bisync Hub's signature review workflows and pre-execution safety gates.
