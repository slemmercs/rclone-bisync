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
| **Immutable Audit Ledger & Unified Log Architecture** | **Next Priority (#1)** | Governance & Compliance |
| **Multi-Profile Workspace Manager** | **Next Priority (#2)** | Multi-Workload Pairing |
| **Configuration & Setup Manager** | **Next Priority (#3)** | Setup, Safety Pipeline & Flags |
| **In-App User Guidance, Info & Help Documents** | **Next Priority (#4)** | Contextual Assistance & Manual |
| Rclone RC Framework Hardening | Planned | Client Resiliency & Diagnostics |
| Remote Mount Manager & Pooled Virtual Storage Engine | Planned | Storage & VFS Extension |
| Local Inter-Process Communication (IPC) via Unix Sockets | Planned | Infrastructure Hardening |
| Auto-Generated Cryptographic Credentials Engine | Planned | Infrastructure Hardening |
| Desktop Application Packaging (Wails) | Future | Native Packaging |

---

## Active Strategic Priorities

### Near-Term Priority 1: Immutable Audit Ledger & Unified Log Architecture

Implements cryptographically chained SQLite audit logging (`internal/audit`) using a pure-Go driver (`modernc.org/sqlite`) and goose migrations (`pressly/goose/v3`):
- **Dual-Logging**: Critical milestones dual-logged to `bisync.log` alongside sync execution runs.
- **POSIX Log Rotator**: In-process size-bounded rotation (10 MB, 5 archives) for `bisync.log` and `rclone-daemon.log`.
- **Dedicated Audit Workspace**: Multi-stream terminal switcher plus a dedicated 4th top-level tab `[Audit Ledger]` with live cryptographic SHA-256 chain verification.

---

### Near-Term Priority 2: Multi-Profile Workspace Manager

Supports independent synchronization profiles from a single daemon and dashboard:
- Spawns isolated goroutine runners per profile (e.g., Personal GDrive, Work OneDrive, NAS backup).
- Pairs independent local folders to different remotes (e.g., `/home/user/Documents` -> `gdrive:`, `/home/user/Photos` -> `onedrive:`).
- Strict flock isolation (`/tmp/rclone-bisync-<profile>.lock`) and dedicated state directories.
- Profile-aware telemetry streams, speed charts, and file explorer remote routing.
- Refer to [`docs/design/profile-manager.md`](../design/profile-manager.md).

---

### Near-Term Priority 3: Configuration & Setup Manager

Provides safe, in-dashboard editing of synchronization flags, rate limits, timing intervals, and remote setup:
- **Visual Flag Editor**: Adjust `--buffer-size`, `--tpslimit`, conflict resolution strategies, and webhook endpoints safely from the UI.
- **Setup & Onboarding Wizard**: Guided visual setup wizard for initial Rclone RC remote pairing and environment configuration.
- **3-Layer Safety Pipeline**: Automated backup snapshots, SHA-256 integrity match, and JSON/env schema validation before applying new settings.
- **One-Click Rollback API**: Restore historical configurations immediately if a change fails verification.

---

### Near-Term Priority 4: In-App User Guidance, Info & Help Documents

Enhances dashboard accessibility for non-technical operators and new users:
- **Contextual Tooltips & Policy Explanations**: Embeds inline tooltips and modal banners across `partials/view_*.html` explaining safety mechanisms (`TOO_MANY_DELETES`, `MANDATORY_SUBDIR`, heuristic risk scores) in plain language.
- **In-Dashboard Reference Manual**: A dedicated slide-over panel summarizing CLI commands, Rclone flags (`--buffer-size`, `--tpslimit`), and step-by-step conflict recovery workflows.
- **Interactive Diagnostics & Troubleshooting**: Embedded help documents for resolving OAuth token drops, path permissions issues, and systemd service status.

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