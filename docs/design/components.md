# Bisync Hub Design System: Components Specification

## 1. Component Architecture

Bisync Hub utilizes a **two-tier component model** optimized for Go `html/template` and Alpine.js:

- **Tier 1 — Atomic Primitives as Semantic CSS Classes (`design-tokens.css`)**:
  - `Card`, `Button`, `Badge`, `Alert`, `Table`, `Input` are implemented as zero-overhead CSS utility classes.
  - This allows direct declarative binding of Alpine directives (`@click`, `:disabled`, `x-model`, `:class`) without cumbersome template argument passing.
- **Tier 2 — Structural Composites as Go Templates (`partials/components/`)**:
  - `Drawer`, `EmptyState`, `Modal` provide reusable HTML shells driven by shared Alpine state objects.

---

## 2. Component Reference

### Card (`.ds-card`)
Operational container for workload metrics, configurations, and data grids.
- `.ds-card`: Outer container (`bg-slate-900 border border-slate-800 rounded-2xl shadow-xl`).
- `.ds-card-header`: Header strip with bottom border and title/actions layout.
- `.ds-card-body`: Content body with 16px (`--space-4`) padding.
- `.ds-card-footer`: Metadata and pagination strip with top border.

### Button (`.ds-btn`)
Action trigger with loading, focus, and disabled states.
- Variants:
  - `.ds-btn-primary`: Indigo accent (`bg-indigo-600 hover:bg-indigo-500`).
  - `.ds-btn-secondary`: Slate muted (`bg-slate-800 hover:bg-slate-700`).
  - `.ds-btn-danger`: Rose destructive (`bg-rose-600 hover:bg-rose-500`).
  - `.ds-btn-ghost`: Borderless hover button (`hover:bg-slate-800`).
- Sizes:
  - `.ds-btn-sm`: 4px 10px, 11px font size.
  - `.ds-btn-md`: 8px 14px, 12px font size.

### Badge (`.ds-badge`)
Uppercase compact status and origin indicator.
- Classes:
  - `.ds-badge-healthy`, `.ds-badge-warning`, `.ds-badge-critical`
  - `.ds-badge-running`, `.ds-badge-idle`, `.ds-badge-blocked`, `.ds-badge-review`

### Alert (`.ds-alert`)
Operational notice banner for transient states or sync progress.
- `.ds-alert-success`: Green confirmation banner.
- `.ds-alert-warning`: Amber threshold notice.
- `.ds-alert-danger`: Red integrity fault or rate-limit notice.
- `.ds-alert-info`: Indigo live sync progress strip.

### Slide-Over Drawer (`component_entity_drawer`)
Right slide-over panel for read-only forensic inspection (cryptographic audit records, correlated daemon logs) without losing context or stacking modals.
- Driven by `entityDrawer` in Alpine state:
  ```javascript
  openAuditDrawer(record) // Sets entityDrawer.open = true
  ```
- Closes via `Esc` or clicking the backdrop overlay.

### Empty State (`component_empty_state`)
Standard empty data feedback with dashed border and icon slot.
