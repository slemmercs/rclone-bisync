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
| **Interactive Sync Review & Deletion Diff Visualizer** | **Delivered** | Phase 4 (`feat/interactive-sync-review`) |
| **Heuristic Conflict Recommendation Engine** | **Delivered** | Phase 4 Extension (`feat/heuristic-conflict-engine`) |
| **Frontend Template Modularization (`html/template`)** | **In Progress** | Refactor (`refactor/modularize-frontend-templates`) |
| **Immutable Audit Ledger & Unified Log Architecture** | **Next Milestone** | Phase 4/5 Governance & Compliance |
| Multi-Profile Synchronization Hub | Planned | Phase 5 |
| Configuration Management & 3-Layer Safety Pipeline | Planned | Phase 5 |
| Local MVP Deployment Hardening (Unix Sockets & Crypto Credentials) | Planned | Phase 6 |
| Desktop Application Packaging (Wails) | Future | Phase 7 |

---

## Active Strategic Priority

### Frontend Template Modularization (`html/template`)

Decomposes the monolithic 1,500+ LOC `dashboard/frontend/index.html` into clean, maintainable Go standard library `html/template` partials (`partials/view_*.html`, `partials/modals/*.html`).

Capabilities & Architecture:
- **Zero Node.js/npm Dependencies**: Pure Go server-side assembly using `embed.FS` and `html/template`.
- **Zero DOM Shift / Pop-in**: Server assembles and caches templates at boot, serving full HTML in a single HTTP request.
- **Maintainable Architecture**: Separates Cockpit Telemetry, Storage Remotes, File Explorer, and interactive Modals into modular files under 300 LOC.
- **Clean Foundation for Audit Ledger**: Enables upcoming `partials/view_audit.html` to be plugged into the 4th top-level navigation tab with zero clutter.

---

### Upcoming Strategic Priority: Immutable Audit Ledger & Unified Log Architecture

Implements cryptographically chained SQLite audit logging (`internal/audit`) with pure-Go driver (`modernc.org/sqlite`) and goose migrations (`pressly/goose/v3`).
- **Dual-Logging**: Critical milestones dual-logged to `bisync.log` alongside sync execution runs.
- **POSIX Log Rotator**: In-process size-bounded rotation (10 MB, 5 archives) for `bisync.log` and `rclone-daemon.log`.
- **Option 1 UI**: Multi-stream cockpit terminal switcher (`All`, `Sync`, `Audit`) plus a dedicated 4th top-level tab `[Audit Ledger]` with live cryptographic integrity verification.


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

