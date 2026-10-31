# Bisync Hub UI Design System Specification

| Metadata | Details |
|---|---|
| **Status** | `Shipped` (v1.0.0-beta.12) |
| **Domain** | UI / UX Design System & Front-End Architecture |
| **Governing ADR** | [ADR-010: Zero-Build Component Design System & Responsive Telemetry Architecture](../architecture/adr/ADR-010-Zero-Build-Design-System-And-Responsive-Telemetry.md) |
| **Core Principles** | [Product Principles](../architecture/PRODUCT_PRINCIPLES.md), [Architecture Principles](../architecture/ARCHITECTURE_PRINCIPLES.md) |

---

## Purpose

This specification defines the visual, interaction, and architectural standards used throughout the Bisync Hub user interface.

The design system supports the product vision:

> Review. Decide. Sync.

The interface must optimize for:

- Operational clarity
- Decision support
- Safety
- Consistency
- Maintainability

---

# Design Principles

## Operations First

The interface exists to support operational decisions.

Design should prioritize:

- state visibility
- risk awareness
- actionable choices
- efficient workflows

over decorative visuals.

---

## Consistency Over Creativity

Existing patterns should always be reused before creating new ones.

Before creating a component ask:

```text
Can this be solved using:

Card
Button
Badge
Alert
Modal
Table
Drawer
Empty State
```

If yes:

```text
Reuse.
```

---

## Progressive Disclosure

Display information using:

```text
Summary
↓
Details
↓
Deep Inspection
```

Users should never be forced to process detailed data first.

---

## Dark Mode First

Primary operating environment:

```text
Desktop
Dark Theme
Long-Lived Sessions
```

Dark mode is the primary visual target.

---

# Design Tokens

## Semantic Colors

Use semantic meaning.

Never:

```text
red
green
blue
yellow
```

Use:

```text
success
warning
danger
info
```

### Required Tokens

```css
--color-success
--color-warning
--color-danger
--color-info

--color-surface
--color-surface-secondary

--color-border
--color-text
```

---

# Spacing Scale

Allowed spacing values:

```text
4px
8px
16px
24px
32px
```

Avoid arbitrary spacing.

---

# Typography

## Page Title

Used once per page.

```text
Fleet Overview
Conflict Review
Profile Management
```

## Section Title

Used inside pages.

## Card Title

Used inside cards.

---

# Approved Status Vocabulary & Normalization

In a safety-critical bidirectional cloud synchronization tool where operators manage destructive file mutations, status terminology must be unambiguous, concise, and predictable.

Status indicators communicate two orthogonal dimensions:
1. **Health State**: Is the system or workload functioning reliably?
2. **Execution State**: Is an operational process currently running?

Every badge, tile, indicator dot, and alert across Bisync Hub must use **only** one of these seven canonical labels:

| Canonical Label | Semantic Meaning | Color Token | CSS Badge Class | Disallowed Legacy Terms |
|---|---|---|---|---|
| **`Healthy`** | Normal operation, listings aligned, baseline clean | Emerald (`--color-success`) | `.ds-badge-healthy` | `OPERATIONAL`, `ONLINE`, `OK`, `READY`, `PASSING` |
| **`Warning`** | Non-blocking anomalies, quota threshold (>85%), dry-run divergence | Amber (`--color-warning`) | `.ds-badge-warning` | `ATTENTION REQUIRED`, `NEEDS ATTENTION`, `DEGRADED`, `WARN` |
| **`Critical`** | Unrecoverable error, process fault, connection failure, canary missing | Rose (`--color-danger`) | `.ds-badge-critical` | `FAULT`, `OFFLINE`, `ERROR`, `DANGER`, `FAILED` |
| **`Running`** | Active bidirectional sync pass, dry-run simulation, or file transfer | Indigo / Sky (`--color-info`) | `.ds-badge-running` | `SYNCING`, `SYNC IN PROGRESS`, `IN PROGRESS`, `ACTIVE` |
| **`Idle`** | System initialized, standby between scheduler intervals | Slate (`--color-text-secondary`) | `.ds-badge-idle` | `STANDBY`, `WAITING`, `SLEEPING` |
| **`Blocked`** | Stale lock detected, rate limit exponential backoff, or lock contention | Orange (`--color-blocked`) | `.ds-badge-blocked` | `LOCKED`, `STALE_LOCK`, `RATE_LIMIT_WAIT` |
| **`Review Required`** | Pending mass deletions exceeding safety ceiling or path conflict requiring operator decision | Purple (`--color-review`) | `.ds-badge-review` | `NEEDS REVIEW`, `MANUAL APPROVAL`, `PENDING APPROVAL`, `CONFLICT` |

### Normalization Engine

In `dashboard/frontend/js/utils.js`, `normalizeStatus(rawStatus)` normalizes any raw backend code (`OPERATIONAL`, `FAULT`, `SYNCING`, `DELETION_THRESHOLD_EXCEEDED`) into its canonical approved status label before rendering.

---

# Component Specifications

## Button

### Purpose

Primary action trigger.

### API

```yaml
variant:
  - primary
  - secondary
  - danger
  - ghost

size:
  - sm
  - md
  - lg

loading:
  - true
  - false

disabled:
  - true
  - false

icon:
  optional

label:
  required
```

### Examples

```text
Execute Sync
Save Configuration
Delete Profile
Cancel
```

---

## Badge

### Purpose

Operational status display.

### API

```yaml
type:
  - healthy
  - warning
  - critical
  - running
  - idle
  - review
```

---

## Alert

### Purpose

Display important information.

### API

```yaml
variant:
  - success
  - warning
  - danger
  - info

title:
  optional

message:
  required

dismissible:
  true | false
```

---

## Card

### Purpose

Primary content container.

### API

```yaml
title:
  optional

subtitle:
  optional

actions:
  optional

collapsible:
  true | false
```

---

## Modal

### Purpose

All confirmation workflows.

### API

```yaml
title:
  required

size:
  - sm
  - md
  - lg
  - xl

body:
  required

confirmLabel:
  optional

cancelLabel:
  optional

confirmVariant:
  - primary
  - warning
  - danger

onConfirm:
  required

onCancel:
  optional
```

### Approved Uses

```text
Delete Profile
Force Sync
OAuth Reconnect
Rollback
Conflict Approval
Deletion Approval
```

---

## Table

### Purpose

Operational data display.

### API

```yaml
columns:
  required

rows:
  required

sortable:
  true | false

filterable:
  true | false

searchable:
  true | false

loading:
  true | false

emptyState:
  optional
```

### Standard Column Order

```text
Status
Name
Description
Last Updated
Actions
```

---

## Drawer

### Purpose

Inspect entities without navigation.

### API

```yaml
title:
  required

width:
  - sm
  - md
  - lg

content:
  required
```

---

## Empty State

### Purpose

Display when no data exists.

### API

```yaml
icon:
  optional

title:
  required

message:
  required

action:
  optional
```

---

# Pattern Specifications

## Health Banner

### Purpose

System-wide health overview.

### Inputs

```yaml
healthyCount
warningCount
criticalCount
lastSync
```

---

## Sync Summary

### Purpose

Summarize synchronization outcomes.

### Inputs

```yaml
changes
conflicts
deletions
warnings
duration
```

---

## Review Queue

### Purpose

Guide user decisions.

### Inputs

```yaml
reviewType
items
```

Supported types:

```text
Conflict
Deletion
Approval
```

### Workflow

```text
Issue Detected
↓
Review Summary
↓
Inspect Details
↓
Take Action
↓
Confirm
↓
Execute
```

---

## Timeline

### Purpose

Chronological activity presentation.

### Inputs

```yaml
events
```

Supported uses:

```text
Audit History
Recovery History
Telemetry Events
Scheduler Events
```

---

## Entity Inspector

### Purpose

Drill into operational objects.

### Inputs

```yaml
entityType
entityId
```

Supported entities:

```text
Profile
Remote
File
Conflict
AuditEntry
```

---

# Layout & Navigation Architecture

## 2-Tier Global Header Architecture

Bisync Hub uses a persistent 2-tier navigation structure:

- **Tier 1: Global Context Bar**:
  - Left: Logo, Version pill (`v1.0`), Context Selector dropdown (`Switch Workspace` / active profile name).
  - Right:
    - **Polling Toggle** (`[LIVE]` vs `[PAUSED]`, shortcut `Alt+P`): Controls background telemetry fetching.
    - Global Settings cog button (`modal_global_settings.html`).
    - Daemon Live indicator (`LIVE` / `OFFLINE` + response latency in ms).
- **Tier 2: Sub-Navigation Bar (Context-Aware Workspace Tabs)**:
  - Tab 1: **Fleet** (Multi-workload status overview grid).
  - Tab 2: **Engine & Telemetry** (Selected profile cockpit + Unified Timeline Ledger).
  - Tab 3: **Remotes** (Cloud storage quota and OAuth connection health).
  - Tab 4: **Explorer** (Dual-pane path-jailed file explorer).
  - Tab 5: **Audit** (Cryptographically verified compliance ledger).

---

## Layout Specifications

### Dashboard Layout (`view_fleet.html`)
- Aggregate fleet view displaying the Health Banner pattern, search/filter bar, and responsive grid of workload cards (`.ds-card`).
- Empty states rendered using `component_empty_state`.

### Workspace Layout (`view_telemetry.html`, `view_remotes.html`, `view_explorer.html`)
- Sticky operational toolbar at the top with primary action triggers (`Start Sync`, `Dry Run`, `Force Deletions`, `Stop`, `Resync`, `Clear Lock`).
- Execution Contract banner specifying local and cloud remote paths.
- Timeline pattern for event ledger.

### Detail & Audit Layout (`view_audit.html`)
- Cryptographic verification continuity banner.
- Operations data table (`.ds-table`) with SHA-256 fingerprint previews.
- Slide-over Entity Inspector Drawer (`component_entity_drawer`) for forensic payload analysis.

---

# Operational Patterns

Operational patterns represent composite UI workflows assembled from Design System components to solve key systems administration needs.

## Health Banner Pattern (`pattern_health_banner`)

Provides instant situational awareness of fleet health across all configured synchronization profiles.
- **Placement**: Top of the Fleet View (`view_fleet.html`) and Engine Overview.
- **Metrics Displayed**: Global Daemon Ping (`LIVE` vs `OFFLINE`), Active Engine Status (`Running`, `Idle`, `Blocked`), and aggregate counts (`Healthy: N`, `Warning: N`, `Critical: N`).
- **Interactions**: Clicking filter badges filters the profile grid below to that status subset.

## Sync Summary Pattern (`pattern_sync_summary`)

Real-time synchronization progress strip rendered whenever background or manual file sync is active.
- **Trigger**: `x-show="telemetry.is_syncing && liveTransferStats"`.
- **Metrics**: Transfer speed (MB/s), bytes transferred / total bytes, percentage progress bar, ETA calculation, and currently active transferring file name with monospace truncation.

## Timeline Ledger Pattern (`pattern_timeline_ledger`)

Chronological unified event stream combining bisync engine milestones, rclone operations, and immutable cryptographic audit events.
- **50-Line On-Demand Pagination**: Default view requests 50 entries (`page=1&limit=50`). Subsequent pages load on demand, reducing browser memory usage by >80%.
- **Inline Log Deduplication & Folding**: Consecutive identical notices (e.g. `Duplicate directory found`) are grouped into collapsible clusters with `×N repeated` counters. Keeps chronological timeline order intact without opening nested modals. Includes global `Fold Repeated` toggle and `Expand / Collapse All` master button.
- **Filter Pills**: `All` (unified stream), `Sync` (engine operations only), and `Audit` (cryptographic ledger events only).

## Entity Inspector Pattern (Slide-Over Drawer)

Replaces modal-on-modal overlays for entity inspection.
- When an operator clicks `Ledger →` on an audit event in the timeline, the **Entity Drawer** slides in from the right edge.
- Displays full SHA-256 block hash, previous anchor hash, target object path, and formatted JSON parameters.
- Context remains visible on the left; pressing `Esc` dismisses the drawer immediately.

---

# Accessibility Requirements

All components must support:

- Keyboard navigation
- Visible focus indicators
- Screen-reader compatible labels
- Sufficient color contrast
- Non-color based status indicators

---

# Definition of Done

A feature is UI-complete when:

- Approved layout is used
- Approved components are used
- Approved terminology is used
- Semantic design tokens are used
- Dark mode is supported
- Loading state exists
- Empty state exists
- Accessibility requirements pass
- No duplicate component patterns introduced

---

# Long-Term Architecture Goal

Future features should be assembled from:

```text
Layouts
+
Patterns
+
Components
+
Data
```

rather than introducing new custom UI implementations.

This enables Bisync Hub to scale feature development while minimizing design and maintenance debt.