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
| **Heuristic Conflict Recommendation Engine** | **In Progress** | Phase 4 Extension (`feat/heuristic-conflict-engine`) |
| Multi-Profile Synchronization Hub | Planned | Phase 5 |
| Configuration Management & 3-Layer Safety Pipeline | Planned | Phase 5 |
| Local MVP Deployment Hardening (Unix Sockets & Crypto Credentials) | Planned | Phase 6 |
| Desktop Application Packaging (Wails) | Future | Phase 7 |

---

## Active Strategic Priority

### Heuristic Conflict Recommendation Engine (Phase 4 Extension)

Augments Phase 4 Interactive Sync Review with an intelligent decision pipeline ("Review. Decide. Sync."). When synchronization aborts due to conflicting modifications, candidate conflict pairs are evaluated offline using pure-Go deterministic heuristics.

Capabilities:
- **Offline Concurrent Analysis**: Bounded worker pool ($G=4$) evaluates candidates against a pipeline of deterministic analyzers.
- **Critical Risk Protection**: Automatically isolates potential 0-byte truncations, binary size divergences, and mass shrinkage with `RiskCritical` to prevent silent data loss.
- **Intelligent Pattern Matching**: Accurately recognizes application temporary locks (`~$*`, `*.tmp`, `*.swp`), DST hour jumps ($\pm 3600\text{s}$), and identical file hashes.
- **Confidence Scoring & Reasoning**: Computes normalized score ($-1.0$ to $+1.0$) and confidence ($0.00$ to $1.00$) with human-readable rationale.
- **Batch Resolution**: Enables one-click auto-staging of high-confidence, low-risk recommendations (`POST /api/review/action-batch`) while holding critical items for manual review.

Refer to [`docs/design/heuristic_conflict_recommendation_engine_design_specification.md`](../design/heuristic_conflict_recommendation_engine_design_specification.md) for detailed ruleset matrices and scoring formulas.


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

