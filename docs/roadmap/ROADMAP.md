# Roadmap

This roadmap represents the intended direction of Bisync Hub.

The roadmap may evolve as the project grows.

---

## Vision

Become the safest and most understandable way to operate Rclone Bisync.

**Review. Decide. Sync.**

---

## Community Participation

Roadmap priorities are primarily driven by:

1. User safety & data loss prevention
2. Reliability & crash-resilient atomicity
3. Usability & operational transparency
4. Maintainer capacity
5. Community feedback

Roadmap items are not commitments and may change based on project needs.

---

## Status Summary

| Epic / Feature | Status | Milestone / PR |
| :--- | :--- | :--- |
| Single-Binary Go Architecture & Embedded UI | **Delivered** | PR #9 |
| Test Coverage Uplift (>80% statement coverage across all packages) | **Delivered** | PR #17 |
| Remote OAuth Reconnect & Lifecycle Management | **Delivered** | PR #16 |
| SyncService Modularization & Domain Separation | **Delivered** | PR #18 |
| Bisync Hub Rebranding & Lucide SVG Design System | **Delivered** | PR #20 |
| **Interactive Sync Review & Deletion Diff Visualizer** | **Delivered** | Milestone 4 (`feat/interactive-sync-review`) |
| **Heuristic Conflict Recommendation Engine** | **Delivered** | Milestone 4 Extension (`feat/heuristic-conflict-engine`) |
| **Frontend Template Modularization (`html/template`)** | **Delivered** | Refactor (`refactor/modularize-frontend-templates`) |
| **Immutable Audit Ledger & Unified Log Architecture** | **Delivered** | PR #27 (`feat/immutable-audit-ledger-and-unified-logging`) |
| **Service Layer Decoupling & Architecture Modularization (sub-400 LOC)** | **Delivered** | PR #28, PR #29, PR #31 (`v1.0.0-beta.2`–`beta.4`) |
| **Go CLI Quality Gate (`cmd/check`) & E2E Testing Suite** | **Delivered** | PR #32 (`feat/alpine-boolean-lint-and-e2e`) |
| **Multi-Profile Backend Engine (Storage, Scheduler, Flags)** | **Delivered** | PR #37, PR #40, PR #42 (`v1.0.0-beta.7`–`beta.9`) |
| **Hybrid Fleet Overview, Provider Presets & Global Fleet Settings** | **Delivered** | PR #41, PR #43 (`v1.0.0-beta.8`–`beta.10`) |
| **UI Design System, 10ms Telemetry & Inline Log Folding** | **Delivered** | `feat/design-system-and-telemetry-perf` |
| **Remote Hash Deduplication & Baseline Protection** | **Delivered** | `feat/design-system-and-telemetry-perf` |
| **Bisync Hub Model Context Protocol (MCP) Server (`bisync-hub mcp`)** | **Active Priority (#1)** | Native Go stdio/SSE Agent Protocol |
| **Configuration Snapshot & Rollback Engine** | Planned | Safety Pipeline & Config Snapshot API |
| **In-App User Guidance, Info & Help Documents** | Planned | Contextual Assistance & Manual |
| Rclone RC Framework Hardening | Planned | Client Resiliency & Diagnostics |
| Remote Mount Manager & Pooled Virtual Storage Engine | Planned | Storage & VFS Extension |
| Local Inter-Process Communication (IPC) via Unix Sockets | Planned | Infrastructure Hardening |
| Auto-Generated Cryptographic Credentials Engine | Planned | Infrastructure Hardening |
| Desktop Application Packaging (Wails) | Future | Native Packaging |

---

## Active Strategic Priorities

### Delivered Milestone: Hybrid Fleet Overview, Provider Presets & Global Fleet Settings (`v1.0.0-beta.10`)

Transitions the Bisync Hub dashboard from single-tenancy to a workload-centric operational hierarchy:
- **Level 1 Fleet View**: Dashboard overview presenting system health summary counters (`Total`, `Healthy`, `Needs Review`, `Failing`, `Syncing`), interactive profile card grid, search/filter controls, and inline quick-actions.
- **Level 2 Scoped Workspace**: Context-aware workspace scoped to the selected profile with header context switcher and breadcrumb navigation.
- **Visual Operational Flags & Provider Presets**: 3-tier flag hierarchy configuration (Execution Controls, Profile Settings, Custom Flags Escape Hatch), provider-aware presets (`gdrive`, `onedrive`, `dropbox`, `s3`), and editable Global Fleet Settings (`modal_global_settings.html`) with live inheritance badges.
- Refer to [`docs/design/hybrid_fleet_overview_contextual_profile_workspace.md`](../design/hybrid_fleet_overview_contextual_profile_workspace.md) and [`docs/design/bisync_operational_flags.md`](../design/bisync_operational_flags.md).

---
### Delivered Milestone: Remote Hash Deduplication & Baseline Protection

Safely consolidates cloud remote collisions using native MD5 checksum deduplication without triggering bisync circuit breakers:
- **Interactive Storage View Action**: Quick-action `[Dedupe by Hash]` added to storage remote cards (`modal_dedupe.html`).
- **3-Strategy Execution**: Supports `oldest` (recommended - preserves original creation timestamps), `newest` (preserves latest modifications), and `list` (safe dry-run preview).
- **Automated Baseline Resync Safeguard**: Following remote deduplication, Bisync Hub refreshes internal baseline listings, completely preventing `TOO_MANY_DELETES` trips on subsequent sync runs.
- **Engine Process Isolation**: Process execution locks prevent concurrent sync jobs from corrupting cloud state during active deduplication.

---

### Active Strategic Priority (#1): Bisync Hub Model Context Protocol (MCP) Server (`bisync-hub mcp`)

Direct integration of Bisync Hub into AI agent workflows (Claude Desktop, Cursor, Cline) via the official [Model Context Protocol (MCP)](https://modelcontextprotocol.io/):
- **Single-Binary Zero-Dependency Architecture**: Compiled directly into the native Go `bisync-hub` binary; zero external Node.js, Python, or npm runtime requirements.
- **Dual Transport Support**: Operates over standard I/O (`stdio`) for local CLI agents and HTTP Server-Sent Events (`SSE`) for distributed network deployments.
- **First-Class MCP Tools**:
  - `bisync_get_health`: Inspect daemon connectivity, latency, active profile count, and circuit breaker trip states.
  - `bisync_trigger_sync`: Programmatically dispatch dry-run, normal sync, or force resync for specified profile workloads.
  - `bisync_review_queue`: Fetch pending sync review sessions, conflict items, and heuristic risk scores.
  - `bisync_resolve_review`: Stage decisions (`keep_local`, `keep_remote`, `delete`, `merge`) and execute atomic resolutions with backup rollback protection.
  - `bisync_dedupe_by_hash`: Execute remote MD5 checksum deduplication and trigger baseline cache rebuilds.
  - `bisync_audit_inspect`: Query and cryptographically verify immutable audit ledger blocks.
- **Read-Only MCP Resources**:
  - `bisync://telemetry`: Real-time streaming metrics, active transfers, and daemon operational health.
  - `bisync://profiles`: Detailed profile workload definitions, schedules, and effective operational flags.
  - `bisync://audit`: Cryptographically chained compliance event log.

---



### Near-Term Priority 1: Configuration Snapshot & Rollback Engine

Provides server-side configuration safety for global environments:
- **3-Layer Safety Pipeline**: Automated backup snapshots, SHA-256 integrity match, and JSON/env schema validation before applying new settings.
- **One-Click Rollback API**: Restore historical configurations immediately if a change fails verification.

---

### Near-Term Priority 3: In-App User Guidance, Info & Help Documents

Enhances dashboard accessibility for non-technical operators and new users:
- **Contextual Tooltips & Policy Explanations**: Embeds inline tooltips and modal banners across `partials/view_*.html` explaining safety mechanisms (`TOO_MANY_DELETES`, `MANDATORY_SUBDIR`, heuristic risk scores) in plain language.
- **In-Dashboard Reference Manual**: A dedicated slide-over panel summarizing CLI commands, Rclone flags (`--buffer-size`, `--tpslimit`), and step-by-step conflict recovery workflows.
- **Interactive Diagnostics & Troubleshooting**: Embedded help documents for resolving OAuth token drops, path permissions issues, and systemd service status.

---

### Near-Term Priority 4: Remote Mount Lifecycle Manager & Pooled Storage Engine

Programmatic management of VFS virtual drive mounts and multi-cloud storage pooling (`internal/mount`):
- **Stage A (Remote Mount Manager)**: Mount, inspect, configure, and unmount individual remotes to local paths via Rclone RC `/mount/*` endpoints. Keepalive probes with automatic reconnect on transient network drops.
- **Stage B (Pooled Storage Engine)**: Provision stacked multi-cloud topologies (`union` + `chunker` + `crypt`) over RC `/config/create`. Combines disparate storage tiers into a single encrypted virtual drive guarded by Bisync Hub's safety probes.
- Refer to [`docs/design/remote_mount_manager_design_specification.md`](../design/remote_mount_manager_design_specification.md).

---

## Secondary & Infrastructure Priorities

### Rclone Remote Control (RC) Framework Hardening

Improves connectivity resiliency between Bisync Hub and the background Rclone `rcd` daemon:
- Automated HTTP connection pooling and exponential backoff retry logic with jitter for HTTP 429 throttling.
- Dynamic RC API version compatibility checks and health probes.
- Proactive connection health monitoring across local Unix sockets and TCP loopbacks.

---

### Remote Mount Manager & Pooled Virtual Storage Engine (`internal/mount`)

Programmatic management of VFS virtual drive mounts and multi-cloud storage pooling:
- **Stage A (Remote Mount Manager)**: Mount, inspect, configure, and unmount individual remotes to local paths via Rclone RC `/mount/*` endpoints.
- **Stage B (Pooled Storage Engine)**: Provision stacked multi-cloud topologies (`union` + `chunker` + `crypt`) over RC `/config/create`. Combines disparate storage tiers into a single encrypted virtual drive guarded by Bisync Hub's safety probes.
- Refer to [`docs/design/remote_mount_manager_design_specification.md`](../design/remote_mount_manager_design_specification.md).

---

### Local Inter-Process Communication (IPC) via Unix Sockets

Replace loopback TCP ports with local Unix domain sockets:
- Communicate over `unix:///run/user/$UID/rclone.sock` with strict `0600` permissions.
- Eliminate local port contention and unauthorized loopback snooping.
- Dual-mode support for bare-metal user sessions and containerized volume bind-mounts.

---

### Auto-Generated Cryptographic Credentials Engine

Automated credential management without plaintext defaults:
- `install.sh` generation of cryptographically random 32-character secrets in secure directories (`0600`).
- Built-in Go CLI rotation command (`rclone-bisync rotate-creds`) utilizing `crypto/rand`.
- Automatic daemon reload on credential rotation.

---

## Long-Term Direction

Bisync Hub is explicitly not intended to become:
- A multi-tenant cloud SaaS platform.
- A proprietary replacement for the upstream Rclone engine.
- A complex binary diff merge engine (diff merges are left to specialized editors).

Its mission is to be the premier, safety-first desktop and server hub for Rclone Bisync operations.