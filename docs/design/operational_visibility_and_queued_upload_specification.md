# Architecture & Design Specification: Operational Log Ledger, Remote Cloud Links & Queued Explorer Uploads

| Metadata | Details |
|---|---|
| **Status** | `Draft` / Backlog |
| **Domain** | Remote Explorer, Cloud Navigation & Upload Queues |
| **Governing ADRs** | [ADR-002: Rclone Remote Control API As Source Of Truth](../architecture/adr/ADR-002-Rclone-As-Source-Of-Truth.md), [ADR-010: Zero-Build Component Design System & Responsive Telemetry Architecture](../architecture/adr/ADR-010-Zero-Build-Design-System-And-Responsive-Telemetry.md) |
| **Core Principles** | [Product Principles](../architecture/PRODUCT_PRINCIPLES.md), [Architecture Principles](../architecture/ARCHITECTURE_PRINCIPLES.md) |

---

## 1. System Decomposition & Feature Scope

This specification formalizes three deeply integrated capabilities designed to elevate operator visibility, enable frictionless cloud navigation, and provide robust local and remote file upload capabilities without sync lock collisions:

1. **Phase 1: Unified Operational Log Ledger, Run ID Correlation & Deep Path Navigation**
   - Harmonize the unified log panel (`bisync.log`) to mirror the high-clarity table design of the Audit Ledger (`view_audit.html`).
   - Extract and cross-link execution Run IDs (`run-...`) and triage Review IDs (`rev-...`).
   - Parse and match filesystem paths in log entries (e.g., `Duplicate ignored`, `Deleted`, `Transferring`) with deep links that switch to the Explorer view, navigate to the target directory, and highlight the affected file.
2. **Phase 2: Remote Cloud Web Link Navigation ("Open on Remote in New Tab")**
   - Leverage Rclone RC `operations/publiclink` to retrieve canonical web URLs for cloud files and directories (e.g., Google Drive, OneDrive).
   - Embed an "Open on Remote" quick-action icon across the remote Explorer pane and next to remote path chips in the log ledger.
   - Handle provider link capabilities gracefully with automatic capability detection.
3. **Phase 3: Queued Dual-Pane File Explorer Direct Upload (Non-Blocking Deferred Execution)**
   - Enable manual file selection and desktop drag-and-drop uploads for both Local (jailed) and Remote directories.
   - Introduce an asynchronous **Upload Queue Manager** that detects active synchronization runs (`telemetry.is_syncing === true` or POSIX flock contention), safely holding files in a `queued (waiting for sync)` state instead of rejecting uploads.
   - Coordinate with the sync runner lock to prevent midway race collisions during active file writes.
4. **Deferred Backlog Items (Unspecified)**
   - Native Browser Notifications (Safety barrier raised, sync failures, completion pings).
   - File Content Preview (In-dashboard syntax viewer for text <= 2MB, image preview <= 5MB).

---

## 2. Phase 1 Design Spec: Unified Log Ledger & Deep Path Navigation

### 2.1 UI/UX Alignment with Audit Ledger
The unified log viewer card in `dashboard/frontend/partials/view_telemetry.html` will transition from a raw mono text container to a structured, scannable table matching `view_audit.html`:

| Column Header | Format / Styling | Content & Behavior |
| :--- | :--- | :--- |
| **TIME** | `font-mono text-xs text-slate-400` | ISO timestamp converted to local time (`HH:mm:ss.SSS`). |
| **LEVEL** | Badge (`bg-emerald-500/10 text-emerald-400`, `bg-rose-500/10 text-rose-400`, etc.) | `INFO`, `NOTICE`, `WARN`, `ERROR`. |
| **SOURCE / ORIGIN** | `text-slate-400 font-mono text-xs` | Origin tag (`bisync`, `daemon`, `audit`, `rclone-rc`). |
| **RUN ID** | Pill button (`bg-slate-800 text-indigo-300 font-mono hover:bg-slate-700`) | Click filters the Audit Ledger or opens the Correlated Run Trace in the Daemon modal. |
| **MESSAGE** | `text-slate-200 text-xs break-all` | Formatted log message with recognized file path chips. |
| **ACTIONS** | Action icons | Copy raw log entry; Jump to correlated trace. |

### 2.2 Run ID & Rev ID Semantic Disambiguation
* **`run_id` (`run-YYYYMMDD-HHMMSS`)**:
  - Emitted by `runner_audit.go` and logged during sync execution lifecycles.
  - Clicking `#<run_id>` dispatches `openDaemonModal(runId)` or sets Audit Ledger filter `profile_id` & `run_id`.
* **`rev_id` (`rev-[a-f0-9]{8}`)**:
  - Emitted when a safety threshold (`TOO_MANY_DELETES`) triggers an interactive review session.
  - Clicking `@<rev_id>` immediately opens the Historical Review Modal (`modal_review.html`) showing file-by-file triage decisions.

### 2.3 Deep Path Extraction & Navigation State Machine
```
[Log Message Stream]
        │
        ▼
[Contextual Tokenizer] ─── Matches: "Duplicate ignored: <path>", "Deleted: <path>", "Transferring: <path>"
        │
        ▼
[Jail & Target Resolver] ─── Resolves: Local jailed path OR Remote relative path (e.g. gdrive:dir/file.txt)
        │
        ▼
[Interactive Path Chip] ─── Rendered inline with: [Folder Jump Icon] and [External Cloud Link Icon]
        │
        ├── Click Folder Jump:
        │     1. window.dispatchEvent('explorer-navigate', { pane: 'left'|'right', path: 'dir', select: 'file.txt' })
        │     2. currentTab = 'explorer'
        │     3. Target pane triggers fetchDirectoryListing(path)
        │     4. Target row scrolls into center view & flashes highlight animation for 1500ms
        │
        └── Click External Cloud Link:
              Triggers GET /api/browse/link?remote=<remote>&path=<relPath> and opens window.open(url, '_blank')
```

---

## 3. Phase 2 Design Spec: Remote Cloud Web Links ("Open on Remote")

### 3.1 Rclone RC Integration (`operations/publiclink`)
* **Endpoint**: `POST /operations/publiclink`
  ```json
  {
    "fs": "gdrive:",
    "remote": "Work/Documents/Hatlapos.docx",
    "unlink": false
  }
  ```
* **Response**:
  ```json
  {
    "url": "https://drive.google.com/file/d/1B2C3D.../view?usp=drivesdk"
  }
  ```

### 3.2 Backend Service Contract (`internal/core/client/fs_ops.go`)
```go
type PublicLinkResult struct {
    Supported bool   `json:"supported"`
    URL       string `json:"url,omitempty"`
    Error     string `json:"error,omitempty"`
}

func (c *RcloneClient) GetPublicLink(ctx context.Context, remote, path string) (*PublicLinkResult, error)
```

### 3.3 Provider Compatibility Matrix & Graceful Degradation
* **Google Drive (`drive`)**: Supported for files and directories (canonical web link).
* **OneDrive (`onedrive`)**: Supported for files and folders (sharing web URL).
* **Dropbox / Box**: Supported (view links).
* **Local / SFTP / S3**: Not supported. Return `Supported: false` with HTTP 200 rather than throwing a 500 error. The UI will dynamically hide or disable the external link button for unsupported remotes.

---

## 4. Phase 3 Design Spec: Queued Dual-Pane File Explorer Direct Upload

### 4.1 Architecture & Concurrency Protection
To prevent sync corruption and eliminate user friction:
1. **Never Reject Active Sync Uploads**: Dropping files onto the explorer while `telemetry.is_syncing === true` places them into an **Upload Queue** with status `queued (waiting for sync to finish)`.
2. **Advisory Lock Coordination**: While uploads are in progress, an in-memory `ActiveUploadCounter` delays the scheduled sync runner by up to 60 seconds using the existing `sync_service.go` POSIX lock coordination.
3. **Strict Path Jailing**:
   - Local: All uploads must pass `filepath.Base(filename)` sanitization and `IsLocalPathJailed(targetDir)`.
   - Remote: Uploads stream directly through Rclone RC `/operations/uploadfile`.

### 4.2 Upload Queue State Machine
```
[User Drops / Selects Files]
        │
        ▼
[Enqueue Item in UploadManager] ─── Status: 'queued', Progress: 0%
        │
        ▼
[Queue Worker Poll]
        │
   Is Sync Active? ──YES──► Status: 'waiting_for_sync' (Hold queue, poll every 2s)
        │
       NO
        │
        ▼
[Acquire Upload Lock]
        │
        ▼
[Stream Multipart Payload] ─── Status: 'uploading', Progress: N%
        │
   ┌────┴───────────────────────────┐
   ▼                                ▼
[Local Target]               [Remote Target]
Write to disk under jail     Stream to RC /operations/uploadfile
   │                                │
   └────┬───────────────────────────┘
        ▼
[Release Upload Lock]
        │
[Status: 'completed'] ─── Refresh Explorer Pane View & Trigger Flash
```

### 4.3 Data Structures (`internal/models/upload.go`)
```go
package models

type UploadStatus string

const (
    UploadStatusQueued     UploadStatus = "queued"
    UploadStatusWaiting    UploadStatus = "waiting_for_sync"
    UploadStatusUploading  UploadStatus = "uploading"
    UploadStatusCompleted  UploadStatus = "completed"
    UploadStatusFailed     UploadStatus = "failed"
)

type UploadJob struct {
    ID           string       `json:"id"`
    ProfileID    string       `json:"profile_id"`
    PaneKey      string       `json:"pane_key"` // "left" (local) or "right" (remote)
    Remote       string       `json:"remote"`
    TargetDir    string       `json:"target_dir"`
    Filename     string       `json:"filename"`
    Size         int64        `json:"size"`
    Status       UploadStatus `json:"status"`
    ErrorMessage string       `json:"error_message,omitempty"`
    CreatedAt    int64        `json:"created_at"`
    CompletedAt  int64        `json:"completed_at,omitempty"`
}
```

### 4.4 Multi-Profile Compatibility Guarantee
All upload jobs and log deep links explicitly include a `ProfileID` attribute (`profile_id: "default"` today, seamlessly mapped to dynamic profile IDs when Priority #1 Multi-Profile lands). This ensures zero technical debt or redesign when multiple workloads run concurrently.

---

## 5. File Constraints & Architecture Governance
All upcoming files conform to the project's quality rules (`go run ./cmd/check loc`):
- All Go files will be split into single-responsibility units $\le$ 200 lines:
  - `internal/models/upload.go` (< 80 lines)
  - `internal/services/explorer_upload.go` (< 180 lines)
  - `internal/services/explorer_link.go` (< 140 lines)
- HTML templates will remain modularized under `partials/modals/` $\le$ 400 lines:
  - `dashboard/frontend/partials/modals/modal_upload_queue.html` (< 120 lines)
- JavaScript will be cleanly isolated into separate modules:
  - `dashboard/frontend/js/explorer_upload.js` (< 200 lines)
