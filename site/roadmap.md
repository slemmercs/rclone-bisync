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
  └──────────────────────────────┬───────────────────────────────┘
                                 │
                                 ▼
  ┌──────────────────────────────────────────────────────────────┐
  │             NEAR-TERM STRATEGIC PRIORITIES                   │
  │  1. Tier 1 Curated Options & Guided + Add Flag Row Builder   │
  │  2. In-Dashboard Configuration & Setup Manager (Rollback API)│
  │  3. In-App User Guidance & Interactive Diagnostic Manual     │
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

---

## 🟡 Near-Term Strategic Priorities (In Development)

The following capabilities represent our immediate development milestones:

### Priority 1: Tier 1 Curated Options & Guided `+ Add Flag` Row Builder
- **Objective**: Align provider options with Rclone backend standards, separate Tier 1 and Tier 2 user experiences, and replace free-form text input with a guided, interactive flag builder.
- **Architecture**:
  - **Tier 1 Providers (`drive`, `onedrive`, `dropbox`, `s3`)**: Curated bisync-critical toggles (Trash deletions, Skip Docs, Shortcuts, Shared With Me) with explicit boolean serialization.
  - **Tier 2 Providers**: Ultra-lean profile configuration directing advanced parameters to the row builder.
  - **Dynamic `[ + Add Flag ]` Builder**: Interactive row-by-row additions with provider-aware flag autocompletion, real-time POSIX tokenizer, and security blocklist validation.

### Priority 2: In-Dashboard Configuration & Setup Manager
- **Objective**: Provide safe visual editing of sync flags, intervals, and remote setup.
- **Architecture**:
  - **Visual Flag Editor**: Adjust `--buffer-size`, `--tpslimit`, conflict strategies, and webhook endpoints safely from the UI.
  - **Setup & Onboarding Wizard**: Guided visual setup wizard for initial Rclone RC remote pairing and environment configuration.
  - **3-Layer Safety Pipeline**: Automated backup snapshots, SHA-256 integrity match, and schema validation before applying changes.
  - **One-Click Rollback API**: Restore historical configurations immediately if a change fails verification.

### Priority 3: In-App User Guidance & Diagnostic Manual
- **Objective**: Enhance operational clarity and assisted recovery directly within the UI.
- **Architecture**:
  - **Contextual Tooltips & Policy Explanations**: Inline tooltips and modal banners explaining safety triggers (`TOO_MANY_DELETES`, `MANDATORY_SUBDIR`, heuristic risk scores) in plain language.
  - **In-Dashboard Reference Manual**: A dedicated slide-over panel summarizing CLI commands, Rclone flags, and step-by-step conflict recovery workflows.
  - **Interactive Diagnostics**: Integrated troubleshooting guides for resolving OAuth token drops, path permissions issues, and systemd service status.

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
