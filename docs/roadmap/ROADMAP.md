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
| **UI Design System, 10ms Telemetry & Inline Log Folding** | **Delivered** | PR #45 (`v1.0.0-beta.12`) |
| **Remote Hash Deduplication & Baseline Protection** | **Delivered** | PR #45 (`v1.0.0-beta.12`) |
| **Unified Logging, Profile Ledger & Telemetry De-Noising** | **Delivered** | PR #49–#53 (`v1.0.0-beta.14`–`beta.18`) |
| **Technical Debt Remediation & Strict Sub-400 LOC Modularization** | **Delivered** | PR #54 (`v1.0.0-beta.19`) |
| **Typed Service Contracts & Copyleft License Guard** | **Delivered** | PR #55 (`v1.0.0-beta.20`) |
| **Native Model Context Protocol (MCP) Server (`rclone-bisync mcp`)** | **Delivered** | PR #56 (`v1.0.0-beta.20`) |
| **Tier 1 Curated Options, Guided Flags Builder & Audit Ledger Sync** | **Delivered** | PR #59 (`v1.0.0-beta.21`) |
| **RC Argument Compiler, Rename Tracking & Review Cache Invalidation** | **Delivered** | `v1.0.0-beta.22` |
| **Configuration Snapshot & Rollback Engine** | **Active Priority (#1)** | Safety Pipeline & Config Snapshot API |
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
### Delivered Milestone: Remote Hash Deduplication & Baseline Protection (`v1.0.0-beta.12`)

Safely consolidates cloud remote collisions using native MD5 checksum deduplication without triggering bisync circuit breakers:
- **Interactive Storage View Action**: Quick-action `[Dedupe by Hash]` added to storage remote cards (`modal_dedupe.html`).
- **3-Strategy Execution**: Supports `oldest` (recommended - preserves original creation timestamps), `newest` (preserves latest modifications), and `list` (safe dry-run preview).
- **Automated Baseline Resync Safeguard**: Following remote deduplication, Bisync Hub refreshes internal baseline listings, completely preventing `TOO_MANY_DELETES` trips on subsequent sync runs.
- **Engine Process Isolation**: Process execution locks prevent concurrent sync jobs from corrupting cloud state during active deduplication.

---

### Delivered Milestone: Native Model Context Protocol (MCP) Server (`v1.0.0-beta.20`)

Integrates Bisync Hub directly into AI agent workflows (Claude Desktop, Cursor, Cline) via the official Model Context Protocol (MCP):
- **Single-Binary Dual Transport**: Pure-Go architecture supporting standard I/O (`stdio`) for local IDE/CLI agents and HTTP Server-Sent Events (`sse` on `--port 8081`) with graceful shutdown.
- **Cryptographic Confirmation Ticket Engine**: 2-step confirmation loop with exact parameter fingerprint validation (`expectedParams`) protecting destructive mutations (`trigger_sync` with `resync`/`force`, `resolve_review` with `execute=true`, and `dedupe_by_hash`).
- **Token-Optimized Read Tools & Resources**:
  - `bisync_get_status`: Compact health summary by default (195 characters, **~98.5% token reduction**), with opt-in flags for logs, config, and metrics.
  - `bisync_get_logs`: Dedicated tail-limited log retrieval tool with severity filtering (`ERROR`, `NOTICE`, `INFO`).
  - `bisync_explore_path`: Jail-confined filesystem explorer (`filepath.Rel`) with windowed pagination (`limit`, `offset`) and projected metadata.
  - `bisync_get_sync_history`: Historical ledger queries with stripped cryptographic hashes.
  - `bisync_inspect_conflict_session`: Unresolved candidate filtering and file diffs.
  - 5 MCP Resources: `bisync://profiles`, `bisync://status`, `bisync://profile/{id}`, `bisync://history/{id}`, `bisync://conflicts/{session_id}`.
- **Mutating Tools & ADR-004 Compliance**:
  - `bisync_trigger_sync`: Default dry-run mode with safety-gated resync and force flags.
  - `bisync_resolve_review`: Support for canonical decisions (`keep-local`, `keep-remote`, `keep-both`, `accept-delete`, `restore-item`, `skip`) and session-level `dismiss`.
  - `bisync_dedupe_by_hash`: Default safe `list` preview mode, 2,000-character context window output truncation, and bounded SQLite audit logging (zero audit records written for zero-deletion previews per ADR-004).

---

### Delivered Milestone: Tier 1 Curated Options, Guided Flags Builder & Audit Ledger Sync (`v1.0.0-beta.21`)

Streamlines profile operational flag management, guarantees boolean argument persistence, provides an interactive flag builder with real-time command preview, and fixes audit ledger counter synchronization:
- **Tier 1 Curated Provider Options**: Curated high-impact toggles for Google Drive (`--drive-shared-with-me`, `--drive-skip-gdocs`, `--drive-use-trash`), OneDrive, S3, and Dropbox, with clean lean fallbacks for other providers.
- **Explicit Boolean Serialization & Compiler Support**: Correctly serializes unchecked boolean overrides (e.g., `--drive-use-trash=false`) and enhances CLI-to-RC compiler to parse `=value` arguments into typed parameters.
- **Guided `+ Add Flag` Builder (`flags_builder.js`)**: Interactive row-by-row builder with autocomplete datalist, type-aware value inputs, validation tooltips, POSIX command-line tokenizer, and live generated `rclone bisync` command preview.
- **Dual Count/Percentage `MaxDeleteValue`**: Type-safe model supporting both count thresholds (e.g., `42`) and percentages (e.g., `10%`), serialized seamlessly with backwards compatibility.
- **Audit Ledger Counter Sync & CodeQL Remediation**: Initialized audit records on boot so navigation tab badges show total event count immediately, stabilized profile filtering, and fixed quote sanitization escaping (CodeQL #23).

---

### Delivered Milestone: RC Argument Compilation, Rename Tracking & Review Cache Invalidation (`v1.0.0-beta.22`)

Hardens CLI-to-RC argument compilation, rename tracking flags, in-memory review barrier synchronization, and dropdown rendering:
- **Rclone RC Parameter Compilation & Config Remapping**: Added `TypeParamMaxDelete` stripping `%` and emitting integer `maxDelete` at top-level AND setting `_config["MaxDelete"]` (`fs.ConfigInfo.MaxDelete`). Remapped `--track-renames` (`TypeConfigBool` -> `_config["TrackRenames"]`) and `--track-renames-strategy` (`TypeConfigStr` -> `_config["TrackRenamesStrategy"]`).
- **Review Session In-Memory Cache Invalidation**: Eliminated stale in-memory review barriers by auto-clearing `s.session = nil` when `active_review_session.json` is missing on disk or dismissed/resolved.
- **Flags Builder `<select>` DOM Parser Fix**: Replaced `<template x-for>` inside `<select>` with standard `<option>` tags, preventing Chromium/WebKit parser stripping and restoring dropdown options. Expanded curated catalog with `--track-renames` and `--track-renames-strategy`.
- **Headless Chromium E2E Testing**: Added automated E2E test verifying flags builder `<select>` retains catalog options.

---

### Active Strategic Priority (#1): Configuration Snapshot & Rollback Engine

Provides server-side configuration safety for global environments:
- **Atomic Configuration Snapshots**: Snapshot profile state before applying structural edits or sync actions.
- **Visual Rollback**: Compare active vs. previous profile revisions with diff visualization and one-click restoration.
  - Real-time POSIX tokenizer and security blocklist validation per row.

---

### Near-Term Priority 2: Configuration Snapshot & Rollback Engine

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