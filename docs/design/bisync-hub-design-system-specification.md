# File: docs/design/design-specification.md

# Bisync Hub UI Design System Specification

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

# Approved Status Vocabulary

Only these labels are permitted:

```text
Healthy
Warning
Critical
Running
Idle
Blocked
Review Required
```

Disallowed examples:

```text
Error
Danger
Needs Attention
Manual Approval
Attention Required
```

when equivalent approved terminology exists.

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

# Layout Specifications

## Dashboard Layout

```text
Page Title
↓
Health Banner
↓
Summary Cards
↓
Detailed Content
```

---

## Workspace Layout

```text
Page Title
↓
Workspace Context
↓
Operational Content
```

---

## Detail Layout

```text
Page Title
↓
Summary
↓
Detail Sections
```

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