---
layout: default
title: Roadmap
nav_order: 3
description: "Bisync Hub engineering roadmap, shipped capabilities, and near-term strategic priorities."
---

# Project Roadmap & Capabilities

Bisync Hub is developed using disciplined, test-driven micro-iterations with strict zero-regression quality gates (>80% test coverage target).

```text
  ┌──────────────────────────────────────────────────────────────┐
  │                   SHIPPED & AVAILABLE                        │
  │  • Interactive Review & Deletion Diff Modal                  │
  │  • 4-Analyzer Heuristic Conflict Confidence Engine           │
  │  • Cryptographic Immutable Audit Ledger (SQLite + SHA-256)   │
  │  • Go CLI Quality Engine (cmd/check) & License Guard         │
  │  • Decoupled Service Architecture (Strictly <400 LOC)        │
  │  • Zero-Dependency Single Go Binary Distribution             │
  │  • Embedded Alpine.js / Tailwind CSS Management Cockpit      │
  │  • POSIX Advisory Locking & Local Path Confinement Jails     │
  │  • Hybrid Fleet Overview & Multi-Profile Workspace (`beta.10`)│
  │  • Remote MD5 Hash Deduplication (`beta.12`)                 │
  │  • Unified Logging Ledger & Telemetry De-Noising (`beta.18`) │
  │  • Native Model Context Protocol (MCP) Server (`beta.20`)    │
  │  • Tier 1 Curated Options & Flags Builder (`beta.21`)        │
  │  • RC Argument Compilation & Rename Tracking (`beta.22`)     │
  │  • Security Hardening & MCP Inspector Cockpit (`beta.23`)    │
  │  • Sandbox Integration Suite & POSIX Journal Locks (`beta.27`)│
  └──────────────────────────────┬───────────────────────────────┘
                                 │
                                 ▼
  ┌──────────────────────────────────────────────────────────────┐
  │             NEAR-TERM STRATEGIC PRIORITIES                   │
  │  1. In-App User Guidance & Interactive Diagnostic Manual     │
  │  2. In-Dashboard Configuration Snapshot & Rollback API       │
  │  3. Homelab Reverse Proxy Hardening & OpenTelemetry (OTel)   │
  │  4. Remote Mount Lifecycle Manager & Pooled Storage Engine   │
  └──────────────────────────────────────────────────────────────┘
```

---

## 🟢 Shipped Capabilities (Available Now)

The following core modules are implemented, tested, and included in current binary releases:

### 1. Interactive Sync Review & Mutation Inspection
- **Dry-Run Inspection Stream**: Captures dry-run stdout/stderr, categorizing every action into `+ CREATE`, `~ MODIFY`, or `- DELETE`.
- **Operator Review Modal**: Halts destructive execution when deletion thresholds are tripped, allowing granular per-file approval or full-run abort.
- **Rollback Resilience**: Zero-byte hard link backups and recovery staging prevent accidental local wipes.

### 2. Multi-Analyzer Heuristic Conflict Engine
- **4 Parallel Heuristic Analyzers**:
  - `SizeDivergenceAnalyzer` (30% weight): Detects unexpected file truncations and major additions.
  - `BinarySignatureAnalyzer` (25% weight): Inspects MIME and magic bytes to reject corrupted partial writes.
  - `ClockSkewAnalyzer` (25% weight): Compensates for provider-specific API clock rounding (e.g. OneDrive 2-second timestamp windows and DST shifts).
  - `VelocityAnalyzer` (20% weight): Evaluates modification frequency to differentiate active work from background touches.
- **Graduated Confidence Bands**: Automatically commits resolutions scoring $\ge 85\%$, stages backups for $60\% - 84\%$, and safely quarantines $< 60\%$ collisions for manual operator review.

### 3. Cryptographic Immutable Audit Ledger
- **Append-Only SQLite Storage**: Schema-level `BEFORE UPDATE` and `BEFORE DELETE` triggers prevent modifying or purging existing ledger entries.
- **SHA-256 Cryptographic Hash Chaining**: Links each block to its predecessor (`H_n = SHA256(H_{n-1} || payload || timestamp)`), ensuring forensic tamper evidence.
- **In-Process POSIX Log Rotator**: Thread-safe size-bounded rotation (`bisync.log`) with lazy initialization and dual-stream audit logging (`"type": "audit"`).
- **Interactive Audit Explorer**: Dedicated Web UI tab with live cryptographic chain verification, actor/action filtering, and CSV/JSONL compliance export.
- **Recursive Secret Sanitization**: Deep masking (`security.RedactSecrets`) strips all tokens, secrets, and auth headers before persistence.

### 4. Production Go Architecture & Embedded UI
- **Single Binary Artifact**: Pure Go binary with embedded dashboard assets (`go:embed`). No Node.js, Python, or external runtime required.
- **Advisory POSIX Locking**: Race-free process exclusion via `syscall.Flock` ensures zero metadata collisions between background daemons and interactive CLI invocations.
- **Path Confinement Jails**: Directory traversal (`../`) prevention protects local storage roots.
- **Rclone Integration**: Seamless inter-process communication with Rclone RC API and local filesystem synchronization.

### 5. High-Performance Go Quality Engine (`cmd/check`) & E2E Testing
- **Parallel Quality Gate**: Concurrent Go CLI check runner validating formatting, templates, line-count budgets, linters, and tests in under 10 seconds.
- **Multi-Line HTML Attribute Tokenizer**: True AST-style token parsing ignoring comments and detecting dotted boolean attributes (`:disabled`, `:readonly`).
- **Headless Chrome E2E Suite**: Smoke tests using `chromedp` executing against the live embedded dashboard to verify real browser DOM reactivity.
- **Coverage Assurance**: Statement coverage actively verified at 88.3% across all packages.

### 6. Decoupled Service Architecture (Sub-400 LOC Enforcement)
- **Bounded Contexts**: Core logic decoupled into focused domain engines (`TelemetryService`, `CanaryService`, `DeletionService`, `ReviewService`, `ExplorerService`, `SyncService`).
- **Strict Maintainability Limits**: All Go files strictly enforced to remain under ~400 lines of code to prevent architectural drift.

### 7. Multi-Profile Backend & 3-Tier Operational Flags Engine
- **Independent Workload Scheduling**: Spawns isolated runner goroutines per profile with dedicated advisory lock confinement (`/tmp/rclone-bisync-<id>.lock`) and SQLite state tracking.
- **3-Tier Operational Flags Architecture**: Fine-grained flag inheritance model (`Execution Override > Profile Override > Global Default > Hardcoded Fallback`).
- **POSIX Shell Argument Tokenizer & Security Blocklist**: Safe tokenizer with hard blocklist preventing command injection and unauthorized override of daemon flags (`--config`, `--rc`, `--use-mmap`, `--log-file`, `--json`).

### 8. Hybrid Fleet Overview & Remote Hash Deduplication (`beta.10`–`beta.12`)
- **Fleet-Wide Command Deck**: Centralized dashboard presenting real-time counters (`Total`, `Healthy`, `Needs Review`, `Failing`, `Syncing`), interactive profile cards, and global fleet settings.
- **Remote MD5 Checksum Deduplication**: Safe interactive remote file deduplication (`oldest`, `newest`, `list`) with process locking and automated baseline cache resync preventing `TOO_MANY_DELETES` trips.

### 9. Native Model Context Protocol (MCP) Server (`beta.20`)
- **Zero-Dependency Dual Transport**: Native pure-Go MCP server operating over `stdio` for local agents and streaming `sse` over HTTP on `--port 8081`.
- **Confirmation Ticket Engine**: Two-step confirmation loop with cryptographic parameter fingerprinting for destructive actions (`trigger_sync` with `resync`/`force`, `resolve_review`, `dedupe_by_hash`).
- **Token-Optimized Operations**: High-signal default status (195 characters, ~98.5% token reduction), windowed explorer pagination, level-filtered log inspection, and bounded SQLite ledger records.

### 10. Tier 1 Curated Options, Guided Flags Builder & Audit Sync (`beta.21`)
- **Curated Cloud Provider Toggles**: Curated high-impact bisync options for Google Drive (`--drive-shared-with-me`, `--drive-skip-gdocs`, `--drive-use-trash`), OneDrive, S3, and Dropbox with clean lean fallbacks for other remotes.
- **Explicit Boolean Serialization & Compiler Support**: Properly emits `=false` when unchecking boolean options and parses `=value` arguments into typed parameters in the CLI-to-RC compiler.
- **Guided `+ Add Flag` Builder (`flags_builder.js`)**: Interactive row-by-row builder with autocomplete datalist, type-aware value inputs, validation tooltips, POSIX command-line tokenizer, and live generated `rclone bisync` command preview.
- **Dual Count/Percentage `MaxDeleteValue`**: Type-safe model supporting both count thresholds (e.g., `42`) and percentages (e.g., `10%`), serialized seamlessly with backwards compatibility.
- **Audit Ledger Counter Sync & Security Remediation**: Instant ledger counter synchronization on load and polling, and resolved CodeQL quote sanitization escaping.

### 11. RC Argument Compilation, Rename Tracking & Review Invalidation (`beta.22`)
- **Rclone RC Parameter Compilation**: Strips `%` from `--max-delete` and compiles typed values into both root payload and `_config["MaxDelete"]` (`fs.ConfigInfo.MaxDelete`).
- **Rename Tracking Configuration**: Remaps `--track-renames` (`_config["TrackRenames"]`) and `--track-renames-strategy` (`_config["TrackRenamesStrategy"]`).
- **Review Barrier Cache Invalidation**: Guarantees in-memory review session cache (`s.session = nil`) clears immediately upon session file deletion or dismissal.
- **DOM Parsing Safety**: Direct `<option>` elements in provider flags builder dropdown preventing browser parsing stripping.
- **End-to-End Headless Verification**: Automated Chromium test suite checking dropdown preservation and multi-tab cockpit navigation.

### 12. Security Hardening (CodeQL #24) & MCP Inspector Cockpit (`beta.23`)
- **CodeQL #24 Path Injection Remediation**: Strict character validation (`^[a-zA-Z0-9_-]+$`), directory containment checks, and HTTP 400 rejection for review session identifiers.
- **Configurable MCP Read-Only Mode (`MCP_READ_ONLY`)**: Direct environment variable configuration in `bisync.env` and `--read-only` CLI override.
- **Interactive MCP Inspector Modal**: Clickable dashboard header pill opening real-time connection status, 1-click client configuration snippets (Claude Desktop, Cline, Cursor), and complete 11-tool catalog.
- **Explicit MCP Abort Diagnostics**: Structured classified sync failure causes (`TOO_MANY_DELETES`, canary missing, lock contention) returned directly in tool calls with review barrier recovery guidance.
- **Profile & Run-Scoped Log Filtering**: Multi-profile log scoping and single-run trace isolation in `bisync_get_logs`.
- **Compiler Flag Validation Tool**: Read-only `bisync_validate_flags` tool previewing compiled RC parameters and `_config` dictionaries without executing sync.
- **Explorer Pagination Metadata**: Added `total_count` and `has_more` indicators to `bisync_explore_path`.
- **Unified Background Serving**: Embedded the native MCP SSE server directly into the dashboard daemon service, consolidating systemd units and eliminating duplicate schedulers and multi-daemon file races.

### 13. Architecture Hardening, Real Rclone Sandbox Suite & Embedded MCP (`beta.27`)
- **POSIX Advisory File Locking**: Race-free `syscall.Flock` concurrency protection across review sessions (`active_review_session.json.flock`) and write-ahead log (`review.wal.flock`).
- **Tier 4 Real Local Rclone Sandbox**: Ephemeral dynamic-port integration tests against real `rclone rcd` instances with `--config /dev/null`, verifying trailing whitespace, 404 idempotency, and mixed directory/file conflict reviews.
- **REST Schema Governance**: Standardized typed Go struct models (`internal/models/api.go`) across all configuration and telemetry endpoints.
- **Daemon Health Watchdog**: Resilient background watchdog with exponential backoff surviving transient background RC daemon restarts.
- **Embedded MCP SSE Server**: Unified single-process architecture embedding the MCP server on `:8081` inside `rclone-dashboard.service`.

---

## 🟡 Near-Term Strategic Priorities (In Development)

The following capabilities represent our immediate development milestones:

### Priority 1: In-App User Guidance & Diagnostic Manual
- **Objective**: Enhance operational clarity and assisted recovery directly within the UI.
- **Architecture**:
  - **Contextual Tooltips & Policy Explanations**: Inline tooltips and modal banners explaining safety triggers (`TOO_MANY_DELETES`, `MANDATORY_SUBDIR`, heuristic risk scores) in plain language.
  - **In-Dashboard Reference Manual**: A dedicated slide-over panel summarizing CLI commands, Rclone flags, and step-by-step conflict recovery workflows.
  - **Interactive Diagnostics**: Integrated troubleshooting guides for resolving OAuth token drops, path permissions issues, and systemd service status.

### Priority 2: In-Dashboard Configuration Snapshot & Rollback Engine
- **Objective**: Provide server-side configuration safety and instant rollback for global and profile settings.
- **Architecture**:
  - **Visual Flag Editor**: Adjust `--buffer-size`, `--tpslimit`, conflict strategies, and webhook endpoints safely from the UI.
  - **3-Layer Safety Pipeline**: Automated backup snapshots, SHA-256 integrity match, and JSON/env schema validation before applying changes.
  - **Atomic Configuration Snapshots**: Snapshot profile state before applying structural edits or sync actions, archived in `config_snapshots/` with timestamped revisions.
  - **One-Click Rollback API**: Visual comparison of active configuration against historical snapshots with one-click restoration to instantly revert breaking changes.

### Priority 3: Homelab Reverse Proxy Security Hardening & OpenTelemetry (OTel)
- **Objective**: Harden production homelab exposure behind reverse proxies with standards-based telemetry and trace support.
- **Architecture**:
  - **Reverse Proxy Header Normalization**: Support for standard proxy headers (`X-Forwarded-For`, `X-Real-IP`, `X-Forwarded-Proto`) with configurable trusted CIDR proxy subnets.
  - **Authentication Integration**: Optional HTTP basic authentication or forward-auth middleware support for secure exposure behind reverse proxies (Traefik, Caddy, Nginx).
  - **OpenTelemetry Observability (OTel/OTLP)**: Native OpenTelemetry metrics and traces instrumentation exposing sync execution durations, cumulative transferred bytes, lock contention wait times, error counters, and daemon health via standard OTLP exporters.

### Priority 4: Remote Mount Lifecycle Manager & Storage Pooling (`internal/mount`)
- **Objective**: Turn fragile OS FUSE mounts into resilient, self-healing virtual storage endpoints.
- **Architecture**:
  - **Stage A (Remote Mount Manager)**: Pure-Go daemon managing `rclone mount` subprocess lifecycles on Linux (FUSE) and Windows (WinFsp) with keepalive probes and clean unmount handlers (`fusermount -u`).
  - **Stage B (Pooled Storage Engine)**: Provision stacked multi-cloud topologies (`union` + `chunker` + `crypt`) over RC `/config/create`.
  - Refer to [Remote Mount Manager Design Specification](https://github.com/slemmercs/rclone-bisync/blob/main/docs/design/remote_mount_manager_design_specification.md).

<div class="screenshot-grid">
  <div class="screenshot-frame">
    <img src="{{ '/assets/images/screenshots/bisync-hub_google-drive-oauth-configuration.png' | relative_url }}" alt="Google Drive OAuth Configuration" />
    <div class="caption">
      <span class="title">Google Drive OAuth</span>
      <span>Client ID, Secret & scope setup</span>
    </div>
  </div>
  <div class="screenshot-frame">
    <img src="{{ '/assets/images/screenshots/bisync-hub_microsoft-onedrive-oauth-configuration.png' | relative_url }}" alt="Microsoft OneDrive OAuth Configuration" />
    <div class="caption">
      <span class="title">OneDrive OAuth</span>
      <span>Personal & Business Graph API setup</span>
    </div>
  </div>
</div>


---

## 🔮 Future Vision & Research
- **Peer-to-Peer State Coordination**: Multi-node cluster coordination for organizations synchronizing across distributed branch offices.
- **Encrypted Remote Telemetry**: Zero-knowledge telemetry syncing enabling multi-device sync status monitoring without exposing file metadata.
