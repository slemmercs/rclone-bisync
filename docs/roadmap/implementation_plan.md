# Implementation Plan: Vue 3 Cockpit Full Feature Parity & Architecture Optimization

## Executive Summary & Expert Review Panel

A cross-disciplinary panel consisting of a **Software Architect & System Designer**, a **Senior Full-Stack & Vue 3 Developer**, and a **UI/UX Specialist** conducted an architectural and feature audit of the legacy Alpine.js cockpit vs. the modern Vue 3 / PrimeVue implementation.

### Key Audit Findings

1. **Parity Gaps in Vue 3 Frontend**:
   - **Bandwidth Throttling & Live Core Stats (`/api/core/bwlimit`, `/api/core/stats`)**: Fully implemented in Alpine (`bandwidth.js`, `component_header.html` popover, `modal_global_settings.html` tab), but missing in Vue.
   - **Pending Deletions Review Modal (`/api/deletions/pending`)**: Implemented in Alpine (`modal_deletions.html`), allowing operators to inspect files queued for deletion before triggering `--force` or aborting. Missing in Vue.
   - **Dedupe Batch Review Drawer & Directory Explorer (`/api/dedupe/batch`)**: Alpine provides `modal_dedupe_drawer.html` (slide-up staged decisions review drawer) and `modal_dedupe_explorer.html` (interactive directory explorer tab). Vue's `ModalDedupe.vue` lacks the interactive explorer tab and the slide-up batch application review drawer.

2. **Architecture & Vue Optimization Opportunities**:
   - **God Object Decomposition**: `ViewExplorer.vue` has grown to 1,416 LOC. It contains dual-pane rendering, toolbars, breadcrumbs, search filters, context menus, and drag-and-drop mechanics in a single monolithic SFC. It should be decomposed into cohesive standalone components (`ExplorerPane.vue`, `ExplorerToolbar.vue`, `ExplorerBreadcrumbs.vue`, `ExplorerTable.vue`).
   - **Composable Extraction (`src/composables/`)**: Currently zero composables exist in `src/`. Reusable logic is duplicated across multiple views and modals:
     - `useFormatters.ts`: `formatBytes`, `formatSpeed`, `formatRelativeTime`, `formatDuration`.
     - `useClipboard.ts`: Text copying with auto-resetting `copied` feedback state.
   - **Settings Tab Modularity**: Bandwidth throttling controls should be embedded into `ModalGlobalSettings.vue` via a standalone `SettingsBandwidthTab.vue` component matching `SettingsEncryptionTab.vue`.

---

## Proposed Changes

### 1. Composables Layer (`dashboard/frontend/src/composables/`)
- Extract `useFormatters.ts`: Centralize numeric and temporal formatting across the cockpit.
- Extract `useClipboard.ts`: Encapsulate reactive clipboard writing and notification badges.

### 2. Bandwidth & Core Stats Subsystem
- **Store**: Create `dashboard/frontend/src/stores/bandwidth.ts` managing `/api/core/bwlimit` and `/api/core/stats` polling.
- **Header Popover**: Create `dashboard/frontend/src/components/BandwidthPopover.vue` integrated into `AppHeader.vue`.
- **Global Settings Tab**: Create `dashboard/frontend/src/components/settings/SettingsBandwidthTab.vue` integrated into `ModalGlobalSettings.vue`.

### 3. Pending Deletions Review Subsystem
- **Component**: Create `dashboard/frontend/src/components/ModalPendingDeletions.vue` querying `/api/deletions/pending` with search filtering and "Cancel & Protect" vs "Authorize & Delete with --force" actions.
- **Integration**: Mount in `App.vue` and wire into `useTelemetryStore` / `AppHeader.vue` alert notifications.

### 4. Deduplication Batch Staged Drawer & Directory Explorer
- **Component**: Create `dashboard/frontend/src/components/dedupe/DedupeStagedDrawer.vue` to inspect and execute staged Keep/Delete/Rename decisions via `/api/dedupe/batch`.
- **Component**: Create `dashboard/frontend/src/components/dedupe/DedupeExplorerTab.vue` to navigate directories and trigger scoped duplicate scans.
- **Refactor**: Integrate both into `ModalDedupe.vue`.

### 5. File Explorer Decomposition (`ViewExplorer.vue`)
- Decompose the 1,416-LOC monolith into:
  - `src/components/explorer/ExplorerBreadcrumbs.vue`
  - `src/components/explorer/ExplorerToolbar.vue`
  - `src/components/explorer/ExplorerTable.vue`
  - `src/components/explorer/ExplorerPane.vue`
- Retain `ViewExplorer.vue` as the thin layout orchestrator managing dual-pane layout, drag-and-drop state, and modal mounts.

---

## Technical Strategy & Data Flow

```
[ Backend REST API ]
  ├── /api/core/bwlimit  <---> [ useBandwidthStore ] <---> [ BandwidthPopover / SettingsBandwidthTab ]
  ├── /api/core/stats    <---> [ useBandwidthStore ]
  ├── /api/deletions/pending <-> [ useTelemetryStore ] <-> [ ModalPendingDeletions ]
  └── /api/dedupe/batch  <---> [ useDedupeStore ]    <---> [ DedupeStagedDrawer ]

[ File Explorer Architecture ]
  [ ViewExplorer.vue ] (Orchestrator: Dual Pane State & Drag/Drop)
    ├── Pane Left:  [ ExplorerPane.vue ]
    │     ├── [ ExplorerToolbar.vue ]
    │     ├── [ ExplorerBreadcrumbs.vue ]
    │     └── [ ExplorerTable.vue ]
    └── Pane Right: [ ExplorerPane.vue ]
          ├── [ ExplorerToolbar.vue ]
          ├── [ ExplorerBreadcrumbs.vue ]
          └── [ ExplorerTable.vue ]
```

---

## Safety & Error Handling
- **Canary File Protection**: Deletion guards for `RCLONE_TEST` tokens remain strictly enforced.
- **Advisory Locks & Confirmations**: High blast-radius operations require confirmation tokens and FSM safety checks.
- **Reactivity Isolation**: Polling intervals and event listeners must be properly torn down on component unmount (`onUnmounted`).

---

## Verification Plan

### Automated Testing
- Frontend TypeScript type-checking and production build:
  `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build`
- Architecture Line of Code verification:
  `./scripts/check.sh loc`
- Full pre-flight quality checks:
  `./scripts/check.sh all`
- Headless Chrome E2E test suite:
  `go test -v -count=1 ./tests/e2e/...`


