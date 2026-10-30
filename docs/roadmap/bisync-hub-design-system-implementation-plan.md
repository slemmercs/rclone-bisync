# File: docs/design/implementation-plan.md

# Bisync Hub UI Design System - Implementation Plan

## Objective

Establish a lightweight, reusable design system that improves consistency, reduces UI duplication, and enables rapid feature development while preserving the existing:

- Go Templates
- Alpine.js
- Tailwind CSS
- Single Binary Deployment Model

The goal is **not** to introduce a frontend framework.

The goal is to create a reusable operational UI platform that supports the Bisync Hub philosophy:

> Review. Decide. Sync.

---

# Phase 1 - Design Foundations

## Goal

Create a single source of truth for colors, spacing, typography, and status terminology.

## Deliverables

### Design Tokens

Create:

```text
dashboard/frontend/css/design-tokens.css
```

Define:

```css
--color-success
--color-warning
--color-danger
--color-info

--surface-primary
--surface-secondary

--space-xs
--space-sm
--space-md
--space-lg
--space-xl

--radius-sm
--radius-md
--radius-lg
```

### Status Vocabulary

Create:

```text
docs/design/status-language.md
```

Approved states:

```text
Healthy
Warning
Critical
Running
Idle
Blocked
Review Required
```

### Acceptance Criteria

- No new raw Tailwind color usage in feature code
- All new screens consume semantic tokens
- Status terminology is standardized
- Global search contains no duplicate state labels

---

# Phase 2 - Component Library

## Goal

Replace duplicated UI implementations with reusable components.

## Folder Structure

```text
dashboard/frontend/partials/components/
```

## Components

### Card

Primary content container.

### Button

Variants:

```text
Primary
Secondary
Danger
Ghost
```

### Badge

Variants:

```text
Healthy
Warning
Critical
Running
Idle
Review
```

### Alert

Variants:

```text
Success
Warning
Danger
Info
```

### Modal

Single implementation for all confirmation dialogs.

### Table

Standard data presentation component.

### Drawer

Side-panel detail viewer.

### Empty State

Reusable zero-data pattern.

### Acceptance Criteria

- No duplicate modal implementations
- No duplicate table implementations
- No duplicate card implementations
- All components support dark mode
- All components support loading states where applicable

---

# Phase 3 - Product Patterns

## Goal

Standardize operational workflows.

## Folder Structure

```text
dashboard/frontend/partials/patterns/
```

## Patterns

### Health Banner

Displays:

```text
Healthy
Warning
Critical
```

and summary counts.

### Sync Summary

Displays:

```text
Changes
Conflicts
Deletes
Warnings
Duration
```

### Review Queue

Supports:

```text
Conflict Review
Deletion Review
Approval Review
```

### Timeline

Supports:

```text
Audit History
Recovery Events
Scheduler Activity
Telemetry Events
```

### Entity Inspector

Supports:

```text
Profile
Remote
File
Conflict
Audit Entry
```

### Acceptance Criteria

- All review workflows use Review Queue
- All status overviews use Health Banner
- All synchronization results use Sync Summary
- All activity histories use Timeline
- No custom workflow implementation duplicates existing patterns

---

# Phase 4 - Layout System

## Goal

Create predictable page structures.

## Folder Structure

```text
dashboard/frontend/partials/layouts/
```

## Layouts

### Dashboard Layout

```text
Page Title
↓
Health Banner
↓
Summary Cards
↓
Working Area
```

### Workspace Layout

Used for:

```text
Profiles
Synchronization
Operations
```

### Detail Layout

Used for:

```text
Profile Details
Conflict Details
Audit Details
```

### Acceptance Criteria

- All primary pages conform to approved layouts
- Navigation remains consistent
- Page hierarchy is predictable

---

# Phase 5 - UI Refactoring

## Goal

Reduce existing design debt.

## Tasks

Replace:

```text
DeleteModal
ConfirmModal
OAuthModal
RollbackModal
ReviewModal
```

with:

```text
Modal
```

Replace:

```text
ConflictTable
AuditTable
ReviewTable
ProfileTable
```

with:

```text
Table
```

Replace:

```text
Multiple Card Variants
```

with:

```text
Card
```

### Acceptance Criteria

- Modal duplication eliminated
- Table duplication eliminated
- Shared styling extracted
- UI behavior standardized

---

# Phase 6 - Documentation

## Goal

Document the design system for future contributors and AI coding agents.

## Structure

```text
docs/design/

implementation-plan.md
design-specification.md
status-language.md
components.md
patterns.md
layouts.md
```

### Acceptance Criteria

- Every component documented
- Every pattern documented
- API contracts documented
- Examples provided

---

# Success Metrics

## Consistency

- Common interactions behave identically

---

## Implementation Status (Completed in v1.0.0-beta.12)

- [x] **Phase 1: Design Foundations**: `dashboard/frontend/css/design-tokens.css` with semantic color tokens, spacing scale, border radius, monospace path truncation, `tabular-nums`, and 7 Approved Status Vocabulary (`Healthy`, `Warning`, `Critical`, `Running`, `Idle`, `Blocked`, `Review Required`).
- [x] **Phase 2: Core Components**: Semantic CSS classes for `Card`, `Button`, `Badge`, `Alert`, `Table`, `Input` and Go template partials for `Empty State` and `Drawer` in `dashboard/frontend/partials/components/`.
- [x] **Phase 3: Operational Patterns**: `Health Banner`, `Sync Summary`, and 50-line paginated + inline-folded `Timeline Ledger` in `dashboard/frontend/partials/patterns/`.
- [x] **Phase 4: Layout System**: Dynamic template discovery for `partials/components/`, `partials/patterns/`, `partials/layouts/` in `internal/server/server.go`.
- [x] **Phase 5: Migration & Consolidation**: All 5 views (`view_fleet`, `view_telemetry`, `view_explorer`, `view_remotes`, `view_audit`) migrated to design system tokens and vocabulary. Audit inspection migrated to slide-over Entity Inspector Drawer (`component_entity_drawer`).
- [x] **Phase 6: Documentation**: Complete documentation suite authored in `docs/design/` (`status-language.md`, `components.md`, `patterns.md`, `layouts.md`, `daemon_log_deduplication_inline_folding.md`).

- Common workflows look identical

## Maintainability

- Reduced duplicated markup
- Reduced duplicated Alpine logic

## Development Speed

New functionality assembled from:

```text
Layouts
+
Patterns
+
Components
+
Data
```

instead of custom implementations.

## Architecture Stability

Continue using:

```text
Go Templates
+
Alpine.js
+
Tailwind CSS
```

No SPA migration required.
