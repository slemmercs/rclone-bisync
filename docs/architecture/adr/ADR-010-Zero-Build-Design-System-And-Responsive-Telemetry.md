# ADR-010: Zero-Build Component Design System & Responsive Telemetry Architecture

**Status:** Accepted

**Date:** 2026-10-30

---

## Context

Bisync Hub provides a rich web dashboard delivering multi-profile fleet metrics, directory tree exploration, live streaming daemon logs, conflict review workflows, and cryptographic audit ledger inspection.

Under ADR-005 (Local-First Single Binary Deployment Model), the application strictly enforces a zero-dependency architecture: zero Node.js runtimes, zero npm package managers, and zero external frontend build steps or bundlers. All UI assets are embedded directly into the Go executable via `//go:embed`.

However, building complex dashboards without modern compile-time toolchains introduces severe UI/UX and runtime challenges:
1. **Design System Drift**: Uncoordinated CSS classes produce visual inconsistencies, broken contrast ratios, and haphazard modal layering.
2. **Modal Hierarchy Hell**: Stacking modal dialogs (e.g. audit inspector over review queues) leads to z-index clipping, broken keyboard traps, and poor responsive usability.
3. **Telemetry CPU Thrashing**: Indiscriminate `setInterval` polling loops cause browser tab freezes, excessive DOM re-renders, and battery drain when running in background tabs.
4. **Log Stream Flooding**: High-frequency sync logs generate thousands of repetitive lines, causing DOM bloat, UI stutter, and browser memory exhaustion.
5. **Reactivity Proxy Recursion**: Wrapping complex third-party DOM libraries (such as Chart.js) inside Alpine.js reactive proxies causes circular mutation loops and UI crashes.

---

## Decision

Bisync Hub implements a **Zero-Build Component Design System & Responsive Telemetry Architecture**:

### 1. Pure CSS Variable Design Tokens (`design-tokens.css`)
- Centralized CSS custom properties establish a unified dark theme based on `slate-950` backgrounds, `indigo-600` primary accents, elevated surface tiers (`--ds-surface-0` through `--ds-surface-3`), and typography tokens (`--ds-font-mono`, `--ds-font-sans`).
- Employs vanilla CSS loaded directly by the browser without preprocessors or build steps.
- Embedded Lucide SVGs replace unstable unicode glyphs across all interactive controls.

### 2. Slide-Over Entity Inspector Drawer
- Eliminates nested modal dialogs in favor of an un-nested, full-height slide-over drawer (`dashboard/frontend/partials/components/ui_components.html`).
- Inspects cryptographic audit blocks, profile schemas, and deduplication summaries without disrupting the user's active dashboard context.

### 3. Adaptive Tab-Aware Polling Coordinator (`polling.js`)
- Replaces uncoordinated polling intervals with a centralized, tab-aware scheduler.
- Executes an immediate **10ms first-load burst** on boot for instant UI interactivity.
- Dynamically adjusts polling frequencies based on page visibility (`document.hidden`) and the active dashboard tab, reducing background CPU usage and network round-trips by up to 90%.

### 4. Client-Side Inline Log Folding (`logs.js`)
- Groups repetitive consecutive daemon and bisync log messages into clustered runs using signature normalization.
- Clustered rows display count badges, target object paths (`subEntry.object`), action badges, and inline expand/collapse toggles, rendered newest-first without DOM thrashing.

### 5. Chart & Reactivity Isolation
- Third-party Chart.js instances are stored in plain JavaScript references outside Alpine.js reactive proxies, preventing circular observation loops and memory leaks.

---

## Rationale

1. **Zero External Toolchain Burden**: Developers and operators can modify frontend assets without installing Node.js, npm, or webpack.
2. **Instant Perceived Performance**: The 10ms first-load telemetry burst ensures the UI renders live state immediately upon launch.
3. **Forensic Clarity**: Slide-over entity drawers and folded log clusters make large-scale audit ledgers and execution logs easily navigable on desktop and mobile.

---

## Consequences

### Positive
- Sub-second dashboard load time with zero layout shift.
- Drastic reduction in client browser memory and CPU consumption during high-frequency sync runs.
- Clean visual hierarchy with zero nested modal collisions.
- 100% adherence to single-binary Go distribution invariants.

### Negative / Trade-offs
- Frontend modules must be kept modularized (`utils.js`, `polling.js`, `logs.js`, `remotes.js`) to adhere strictly to the project's sub-400 LOC budget without bundlers.

---

## Related Principles

- [Product Principle 3: Explainability Over Black Boxes](../PRODUCT_PRINCIPLES.md#3-explainability-over-black-boxes)
- [Engineering Principle 1: Service-Oriented Architecture](../ENGINEERING_PRINCIPLES.md#1-service-oriented-architecture)
- [Engineering Principle 3: Small Cohesive Components](../ENGINEERING_PRINCIPLES.md#3-small-cohesive-components)
- [Architecture Principles: Core Invariants](../ARCHITECTURE_PRINCIPLES.md#core-system-invariants)

---

## Related ADRs

- [ADR-004: Cryptographic Append-Only SQLite Audit Ledger](ADR-004-Immutable-Audit-Ledger.md)
- [ADR-005: Local-First Single Binary Deployment Model](ADR-005-Local-First-Architecture.md)

---

## Metadata

**Owner:** @maintainer

**Tags:** frontend, design-system, alpinejs, telemetry, polling, logs, performance
