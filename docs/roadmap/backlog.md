# Bisync Hub Engineering Backlog

This backlog maintains the tactical, prioritized engineering queue for **Bisync Hub**. It details concrete epics, estimated story points (SP), file scopes, and design document references ready for sprint planning and implementation.

---

## Document Alignment

- **Strategic Horizons & Vision**: Refer to [`ROADMAP.md`](./ROADMAP.md) for architectural direction and long-term milestones.
- **Historical Delivered Features**: Refer to [GitHub Releases](https://github.com/slemmercs/rclone-bisync/releases) for complete changelogs of all released versions (`v1.0.0-beta.1`–`beta.36`).
- **Active Sprint Planning**: Active micro-tasks are tracked in [`task_plan.md`](./task_plan.md) and designed in [`implementation_plan.md`](./implementation_plan.md).

---

## Prioritized Work Queue

### 1. Homelab Reverse Proxy Security Hardening & OpenTelemetry (OTel/OTLP)
- **Estimate**: 5–7h | 4 SP
- **Priority**: Active / In-Flight (Sprint Priority)
- **Objective**: Production self-hosted deployment security and standards-based observability.
- **Scope & Architecture**:
  - **Reverse Proxy Header Normalization**: Parse and sanitize `X-Forwarded-For`, `X-Real-IP`, and `X-Forwarded-Proto` guarded by configurable trusted CIDR proxy subnet checks.
  - **Authentication Gateway Support**: Optional HTTP basic authentication or forward-auth middleware support (`bisync.env`) for secure remote access behind Traefik, Caddy, or Nginx.
  - **OpenTelemetry Observability (OTel/OTLP)**: Native OpenTelemetry metrics and traces instrumentation exposing sync execution durations, transferred bytes, lock contention wait times, error counters, and daemon health via standard OTLP exporters (gRPC/HTTP).
- **Key Files**: `internal/server/middleware.go`, `internal/telemetry/otel.go`, `bisync.env`.

---

### 2. Native Browser Notifications
- **Estimate**: 3–4h | 2 SP
- **Priority**: Next / Minor UI
- **Objective**: HTML5 Web Notification API alerting operator when safety barriers trigger (`TOO_MANY_DELETES`), sync jobs fail, or conflict reviews are pending.
- **Scope & Architecture**:
  - Request permission workflow with settings toggle in Cockpit.
  - Dispatch browser notifications on sync failure, barrier trips, and conflict sessions.
  - Window focus handlers to bring Bisync Hub dashboard to front when clicking notifications.
- **Key Files**: `dashboard/frontend/src/composables/useNotifications.ts`, `dashboard/frontend/src/App.vue`.

---

### 3. Remote Mount Lifecycle Manager & Pooled Storage Engine
- **Estimate**: 8–10h | 8 SP
- **Priority**: Planned
- **Objective**: Resilient virtual storage endpoints and storage virtualization (`internal/mount`).
- **Scope & Architecture**:
  - **Stage A (Remote Mount Manager)**: Mount, inspect, configure, and unmount individual remotes to local paths via Rclone RC `/mount/*` endpoints. Keepalive probes with automatic reconnect on transient network drops.
  - **Stage B (Pooled Storage Engine)**: Provision stacked multi-cloud topologies (`union` + `chunker` + `crypt`) over RC `/config/create`. Combines disparate storage tiers into a single encrypted virtual drive guarded by Bisync Hub's safety probes.
- **Design Reference**: [`docs/design/remote_mount_manager_design_specification.md`](../design/remote_mount_manager_design_specification.md)

---

### 4. Local Inter-Process Communication (IPC) via Unix Domain Sockets
- **Estimate**: 4–6h | 3 SP
- **Priority**: Infrastructure
- **Objective**: Replace loopback TCP ports with local Unix domain sockets.
- **Scope & Architecture**:
  - Communicate over `unix:///run/user/$UID/rclone.sock` with strict `0600` permissions.
  - Eliminate local port contention and unauthorized loopback snooping.
  - Dual-mode support for bare-metal user sessions and containerized volume bind-mounts.

---

### 5. Native Desktop Application Packaging (Wails)
- **Estimate**: 4–6h | 3 SP
- **Priority**: Packaging
- **Objective**: Standalone native desktop application bundle without browser dependency.
- **Scope & Architecture**:
  - Wails v2 wrapper around Go backend and embedded frontend.
  - System tray icon with real-time sync status badges.
  - Native desktop notifications on conflict alerts and safety barrier trips.
