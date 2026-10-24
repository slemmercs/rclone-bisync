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
  │  • Zero-Dependency Single Go Binary Distribution             │
  │  • Embedded Alpine.js / Tailwind CSS Management Cockpit      │
  │  • POSIX Advisory Locking & Local Path Confinement Jails     │
  │  • Real-Time Telemetry & Sync Status Reporting               │
  └──────────────────────────────┬───────────────────────────────┘
                                 │
                                 ▼
  ┌──────────────────────────────────────────────────────────────┐
  │             NEAR-TERM STRATEGIC PRIORITIES                   │
  │  1. Dual-Stage Customizable Safety Policies & Canaries       │
  │  2. Automated Remote Mount Lifecycle Manager (FUSE / WinFsp) │
  │  3. Multi-Channel Webhook Notifications (Discord, Telegram)  │
  │  4. Cloud Multi-Provider Optimization (Google Drive, OneDrive│
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

---

## 🟡 Near-Term Strategic Priorities (In Development)

The following capabilities represent our immediate development milestones following public launch:

### Priority 1: Dual-Stage Customizable Safety Policies
- **Objective**: Declarative circuit breakers stopping runaway syncs before damage occurs.
- **Architecture**:
  - **Stage 1 (Pre-Execution Gate)**: Single-call RC `operations/stat` lookup verifying required mount canaries (`RCLONE_TEST`) in $< 200\text{ms}$ before scanning.
  - **Stage 2 (Pre-Mutation Gate)**: Strict configurable deletion quota bounds (`SAFETY_MAX_DELETE_COUNT`, `SAFETY_MAX_DELETE_RATIO`).
  - **Adaptive Rate Throttling**: Exponential backoff with jitter responding immediately to provider HTTP 429 rate limits.

### Priority 2: Remote Mount Lifecycle Manager
- **Objective**: Turn fragile OS FUSE mounts into resilient, self-healing storage endpoints.
- **Architecture**:
  - Pure-Go daemon managing `rclone mount` subprocess lifecycles on Linux (FUSE) and Windows (WinFsp).
  - Background keepalive probes with automatic reconnect on transient network drops.
  - Clean unmount handlers (`fusermount -u`) preventing stuck mountpoints during unexpected crashes.

### Priority 3: Multi-Channel Webhook Notifications
- **Objective**: Real-time alerting when interactive reviews are required or sync runs fail.
- **Architecture**:
  - Outbound dispatchers for Discord Webhooks and Telegram Bot APIs.
  - Dead-man snitch ping integration with Healthchecks.io on successful sync completions.
  - Configurable alert severity filtering (warnings, errors, review halts).

### Priority 4: Cloud Multi-Provider Optimizations
- **Objective**: Provider-specific resilience and skew compensation across leading cloud backends.
- **Architecture**:
  - Dedicated Microsoft OneDrive Graph API rate limit smoothing and 2-second timestamp window alignment.
  - Google Drive v3 API token refresh coordination and change-polling optimizations.

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
