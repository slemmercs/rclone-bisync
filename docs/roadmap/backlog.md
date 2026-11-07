# Bisync Hub Engineering Backlog

This backlog maintains the tactical, prioritized engineering queue for **Bisync Hub**. It details concrete epics, estimated story points (SP), file scopes, and design document references ready for sprint planning and implementation.

---

## Document Alignment

- **Strategic Horizons & Vision**: Refer to [`ROADMAP.md`](./ROADMAP.md) for architectural direction and long-term milestones.
- **Historical Delivered Features**: Refer to [GitHub Releases](https://github.com/slemmercs/rclone-bisync/releases) for complete changelogs of all released versions (`v1.0.0-beta.1`–`beta.27`).
- **Active Sprint Planning**: Active micro-tasks are tracked in [`task_plan.md`](./task_plan.md) and designed in [`implementation_plan.md`](./implementation_plan.md).

---

## Prioritized Work Queue

### 1. Configuration Snapshot & One-Click Rollback Engine
- **Estimate**: 5–7h | 4 SP
- **Priority**: Active / Sprint Ready
- **Objective**: Server-side configuration safety for profiles and global settings.
- **Scope & Architecture**:
  - **3-Layer Safety Pipeline**: Automated configuration backup, SHA-256 integrity verification, and JSON/env schema validation before applying updates.
  - **Atomic Snapshots**: Save timestamped snapshots in `config_snapshots/` prior to any profile edit or global config change.
  - **Rollback API & Visual Diff**: REST endpoints (`/api/config/snapshots`, `/api/config/rollback/{id}`) and modal diff viewer in `modal_global_settings.html` to instantly revert breaking changes.
- **Key Files**: `internal/profiles/store.go`, `internal/server/handlers_config.go`, `dashboard/frontend/partials/modals/modal_global_settings.html`.

---

### 2. Homelab Reverse Proxy Security Hardening & OpenTelemetry (OTel/OTLP)
- **Estimate**: 5–7h | 4 SP
- **Priority**: Sprint Ready
- **Objective**: Production self-hosted deployment security and standards-based observability.
- **Scope & Architecture**:
  - **Reverse Proxy Header Normalization**: Parse and sanitize `X-Forwarded-For`, `X-Real-IP`, and `X-Forwarded-Proto` guarded by configurable trusted CIDR proxy subnet checks.
  - **Authentication Gateway Support**: Optional HTTP basic authentication or forward-auth middleware support (`bisync.env`) for secure remote access behind Traefik, Caddy, or Nginx.
  - **OpenTelemetry Observability (OTel/OTLP)**: Native OpenTelemetry metrics and traces instrumentation exposing sync execution durations, transferred bytes, lock contention wait times, error counters, and daemon health via standard OTLP exporters (gRPC/HTTP).
- **Key Files**: `internal/server/middleware.go`, `internal/telemetry/otel.go`, `bisync.env`.

---

### 3. Unified Operational Log Ledger, Run ID Correlation & Deep Path Navigation
- **Estimate**: 4–5h | 3 SP
- **Priority**: Sequenced
- **Objective**: High-density structured operational event visibility aligned with the Audit Ledger.
- **Scope & Architecture**:
  - **Table Layout Restructuring**: Transform `bisync.log` card in `view_telemetry.html` into a structured table format matching `view_audit.html` (`TIME`, `LEVEL`, `SOURCE`, `RUN ID`, `MESSAGE`, `ACTIONS`).
  - **Run ID & Review ID Correlation**: Clickable Run ID chips (`#run-...`) jumping directly to filtered Audit Ledger records and correlated daemon execution traces; Review IDs (`@rev-...`) linking to conflict reviews.
  - **Deep Path Navigation**: Contextual path tokenizer recognizing file references (`Duplicate ignored`, `Deleted`, `Transferring`) with deep links that switch to File Explorer, load the directory, and highlight the file row.
- **Design Reference**: [`docs/design/operational_visibility_and_queued_upload_specification.md`](../design/operational_visibility_and_queued_upload_specification.md)

---

### 4. Remote Cloud Web Link Navigation ("Open on Remote in New Tab")
- **Estimate**: 2–3h | 2 SP
- **Priority**: Sequenced
- **Objective**: Instant one-click navigation to cloud provider web interfaces.
- **Scope & Architecture**:
  - **Rclone RC Public Link Integration**: Connect to Rclone RC `operations/publiclink` to retrieve canonical web URLs for cloud files and directories (Google Drive, OneDrive).
  - **Explorer & Log Integration**: "Open on Remote" quick-action icon embedded in remote File Explorer table rows and rendered next to remote path chips in the unified log ledger.
  - **Graceful Degradation**: Dynamic capability detection hiding external link actions on storage tiers lacking web interfaces (local, SFTP, raw S3) without throwing errors.
- **Design Reference**: [`docs/design/operational_visibility_and_queued_upload_specification.md`](../design/operational_visibility_and_queued_upload_specification.md)

---

### 5. Queued Dual-Pane File Explorer Direct Upload
- **Estimate**: 5–6h | 4 SP
- **Priority**: Sequenced
- **Objective**: Direct file upload capabilities with zero sync lock collisions.
- **Scope & Architecture**:
  - **Dual-Pane Upload Targets**: Support uploading to both Local (jailed) and Remote directories via file selector and drag-and-drop dropzones.
  - **Non-Blocking Asynchronous Upload Queue**: Automatically queues file uploads during active synchronization (`telemetry.is_syncing === true` or POSIX flock contention) in a floating dock instead of rejecting them with HTTP 409.
  - **Sequential Auto-Resume**: Automatically triggers sequential uploads once active sync runs finish and lock contention releases.
  - **Mid-Upload Lock Protection**: Advisory upload coordination delaying scheduled sync runs while active multipart streams are in-flight.
- **Design Reference**: [`docs/design/operational_visibility_and_queued_upload_specification.md`](../design/operational_visibility_and_queued_upload_specification.md)

---

### 6. Remote Mount Lifecycle Manager & Pooled Storage Engine
- **Estimate**: 8–10h | 8 SP
- **Priority**: Planned
- **Objective**: Resilient virtual storage endpoints and storage virtualization (`internal/mount`).
- **Scope & Architecture**:
  - **Stage A (Remote Mount Manager)**: Mount, inspect, configure, and unmount individual remotes to local paths via Rclone RC `/mount/*` endpoints. Keepalive probes with automatic reconnect on transient network drops.
  - **Stage B (Pooled Storage Engine)**: Provision stacked multi-cloud topologies (`union` + `chunker` + `crypt`) over RC `/config/create`. Combines disparate storage tiers into a single encrypted virtual drive guarded by Bisync Hub's safety probes.
- **Design Reference**: [`docs/design/remote_mount_manager_design_specification.md`](../design/remote_mount_manager_design_specification.md)

---

### 7. Local Inter-Process Communication (IPC) via Unix Domain Sockets
- **Estimate**: 4–6h | 3 SP
- **Priority**: Infrastructure
- **Objective**: Replace loopback TCP ports with local Unix domain sockets.
- **Scope & Architecture**:
  - Communicate over `unix:///run/user/$UID/rclone.sock` with strict `0600` permissions.
  - Eliminate local port contention and unauthorized loopback snooping.
  - Dual-mode support for bare-metal user sessions and containerized volume bind-mounts.

---

### 8. Native Desktop Application Packaging (Wails)
- **Estimate**: 4–6h | 3 SP
- **Priority**: Packaging
- **Objective**: Standalone native desktop application bundle without browser dependency.
- **Scope & Architecture**:
  - Wails v2 wrapper around Go backend and embedded frontend.
  - System tray icon with real-time sync status badges.
  - Native desktop notifications on conflict alerts and safety barrier trips.

---

### 9. Native Browser Notifications
- **Estimate**: 3–4h | 2 SP
- **Priority**: Minor UI
- **Objective**: HTML5 Web Notification API alerting operator when safety barriers trigger (`TOO_MANY_DELETES`), sync jobs fail, or conflict reviews are pending.
- **Scope**: Window focus handlers to bring Bisync Hub dashboard to front when clicking notifications.

---

### 10. File Explorer Content Preview
- **Estimate**: 4–5h | 3 SP
- **Priority**: Minor UI
- **Objective**: Modal content viewer for text/code files (up to 2MB) with syntax highlighting, and inline preview for common image formats (PNG, JPG, SVG, WebP up to 5MB) directly from local jail and remote streams.
