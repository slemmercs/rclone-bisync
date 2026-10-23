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
3. Maintainer capacity
4. Community feedback
5. Long-term architectural direction

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
| **Interactive Sync Review & Deletion Diff Visualizer** | **In Progress** (`feat/interactive-sync-review`) | Phase 4 |
| Multi-Profile Synchronization Hub | Planned | Phase 5 |
| Configuration Management & 3-Layer Safety Pipeline | Planned | Phase 5 |
| Local MVP Deployment Hardening (Unix Sockets & Crypto Credentials) | Planned | Phase 6 |
| Desktop Application Packaging (Wails) | Future | Phase 7 |

---

## Active Strategic Priority

### Interactive Sync Review & Deletion Diff Visualizer (Phase 4)

Primary strategic feature transforming Bisync Hub from an observer dashboard into an interactive safety controller. When synchronization aborts due to safety circuit breakers (`TOO_MANY_DELETES` or unresolved file conflicts), the hub launches an interactive review session instead of requiring manual CLI `--force` intervention.

Capabilities:
- **Persistent Review Sessions**: Atomic JSON journal persistence (`internal/services/review_service.go`), surviving systemd restarts.
- **Selective Deletion Review**: Review deletion candidates per item (`Accept Deletion` vs. `Restore Item`).
- **Granular Conflict Resolution**: Per-file conflict decisions (`Keep Local`, `Keep Remote`, `Keep Both`).
- **Dual-Pane Explorer Visual Indicators**: Red badges for deletions, Amber for conflicts, Green for new items.
- **Safety Previews & Snapshots**: Automated backup snapshots prior to any mutations, with pre-sync rollback capabilities.
- **Crash Recovery & Auditability**: Idempotent resolution journal tracking every mutation state.
- **Pre-Sync Validation Barrier**: Bisync cannot restart until all safety conditions are resolved and verified.

Refer to [`docs/design/interactive-sync-review.md`](../design/interactive-sync-review.md) for detailed UX flows and architectural state machines.

---

## Medium-Term Priorities

### Multi-Profile Synchronization Hub

Support multiple isolated synchronization profiles from a single daemon and dashboard:
- Spawns independent goroutine runners per profile (e.g., Personal GDrive, Work OneDrive, NAS backup).
- Strict flock isolation (`/tmp/rclone-bisync-<profile>.lock`) and dedicated state directories.
- Profile-aware telemetry streams, charts, and explorer remote routing.
- Refer to [`docs/design/profile-manager.md`](../design/profile-manager.md).

---

### Configuration Management & 3-Layer Safety Pipeline

Safe, in-dashboard editing of synchronization flags, rate limits, and timing intervals:
- Visual adjustment for `--buffer-size`, `--tpslimit`, conflict strategies, and webhook endpoints.
- **3-Layer Safety Pipeline**: Automated backup snapshots, SHA-256 integrity match, and JSON/env schema validation before applying.
- One-click rollback API to restore historical configurations immediately.

---

## Infrastructure Hardening Priorities

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

