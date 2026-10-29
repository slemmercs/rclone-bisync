# Bisync Hub Design System: Operational Patterns

## 1. Pattern Overview

Operational patterns represent composite UI workflows assembled from Design System components to solve key systems administration needs.

---

## 2. Health Banner Pattern (`pattern_health_banner`)

Provides instant situational awareness of fleet health across all configured synchronization profiles.

- **Placement**: Top of the Fleet View (`view_fleet.html`) and Engine Overview.
- **Metrics Displayed**:
  - Global Daemon Ping & Health (`LIVE` vs `OFFLINE`).
  - Active Engine Status (`Running`, `Idle`, `Blocked`).
  - Health counts: `Healthy: N`, `Warning: N`, `Critical: N`.
- **Interactions**:
  - Clicking filter badges filters the profile grid below to that status subset.

---

## 3. Sync Summary Pattern (`pattern_sync_summary`)

Real-time synchronization progress strip rendered whenever background or manual file sync is active.

- **Trigger**: `x-show="telemetry.is_syncing && liveTransferStats"`.
- **Metrics**:
  - Transfer speed (MB/s).
  - Bytes transferred / Total bytes.
  - Percentage progress bar.
  - ETA calculation.
  - Currently active transferring file name with monospace truncation.

---

## 4. Timeline Ledger Pattern (`pattern_timeline_ledger`)

Chronological unified event stream combining bisync engine milestones, rclone operations, and immutable cryptographic audit events.

- **50-Line On-Demand Pagination**:
  - Default view requests 50 entries (`page=1&limit=50`).
  - Page 2 and older records load on demand, reducing browser memory usage by >80%.
- **Inline Log Deduplication & Folding**:
  - Consecutive identical notices (e.g. `Duplicate directory found: march/march.go:350`) are grouped into collapsible clusters with `×N repeated` counters.
  - Keeps chronological timeline order intact without opening nested modals.
  - Global `Fold Repeated` toggle and `Expand / Collapse All` master button.
- **Filter Pills**:
  - `All`: Unified stream.
  - `Sync`: Engine operations only.
  - `Audit`: Cryptographic ledger events only.

---

## 5. Entity Inspector Pattern (Slide-Over Drawer)

Replaces modal-on-modal overlays for entity inspection.
- When an operator clicks `Ledger →` on an audit event in the timeline, the **Entity Drawer** slides in from the right edge.
- Displays full SHA-256 block hash, previous anchor hash, target object path, and formatted JSON parameters.
- Context remains visible on the left; pressing `Esc` dismisses the drawer immediately.
