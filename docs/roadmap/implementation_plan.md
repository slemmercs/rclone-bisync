# Implementation Plan: Vue 3 Cockpit Full Feature Parity, Modular Decomposition & UX Modernization

## Proposed Changes

Achieve 100% functional and architectural feature parity between legacy Alpine.js and Vue 3 / PrimeVue cockpit while elevating modularity and operator ergonomics:

1. **Foundations & Reactive Feedback Subsystem**:
   - Register PrimeVue `ToastService` and mount `<Toast />` in `App.vue` with typed `useToastNotification` composable.
   - Core composables layer: `useFormatters` (bytes, speed, duration, relative times) and `useClipboard` (reactive copy with auto-expiring status). Refactor consumers (`ViewFleet`, `ViewTelemetryLedger`, `ModalDedupe`, `LogDock`).

2. **Bandwidth Throttling & Live Core Stats Control Plane**:
   - Establish `stores/bandwidth.ts` integrating `/api/core/bwlimit` and `/api/core/stats`.
   - Build `BandwidthPopover.vue` in `AppHeader.vue` for live speed, pause/resume, and presets (512k, 1M, 5M, 10M, Off).
   - Build `SettingsBandwidthTab.vue` inside `ModalGlobalSettings.vue` matching split-pane design.

3. **Pending Deletions Safety Review Subsystem**:
   - Add `PendingDeletionsData` typing to `types/models.ts`.
   - Build `ModalPendingDeletions.vue` connecting to `/api/deletions/pending` with search, "Cancel & Protect", and "Authorize & Delete with --force".
   - Mount in `App.vue` and link alert badge in `AppHeader.vue` and `telemetryStore`.

4. **Deduplication Batch Review Drawer & Directory Explorer**:
   - Add `DedupeBatchApplyRequest` and `DedupeBatchApplyResponse` types to `types/dedupe.ts`.
   - Build `components/dedupe/DedupeStagedDrawer.vue` for reviewing staged decisions before atomic execution via `/api/dedupe/batch`.
   - Build `components/dedupe/DedupeExplorerTab.vue` (directory tree navigator for targeted scans).
   - Integrate both into `ModalDedupe.vue`.

5. **Monolithic ViewExplorer Decomposition**:
   - Decompose `ViewExplorer.vue` (1,416 LOC) into:
     - `components/explorer/ExplorerBreadcrumbs.vue`
     - `components/explorer/ExplorerToolbar.vue`
     - `components/explorer/ExplorerTable.vue`
     - `components/explorer/ExplorerPane.vue`
   - Slim down `ViewExplorer.vue` to < 250 LOC as dual-pane layout orchestrator.

6. **Global Keyboard Shortcuts & Operator Polling Controls**:
   - Build `composables/useKeyboardShortcuts.ts` with input guard rails (`?`/`F1`, `Alt+N`, `Alt+R`, `Alt+S`, `Alt+H`, `Alt+L`, `Alt+P`).
   - Add polling pause/resume controls in `stores/app.ts` and reflect state in footer.

7. **Verification & Quality Gate Audits**:
   - Expand E2E suite (`tests/e2e/cockpit_spa_smoke_test.go`) covering bandwidth popover, pending deletions, dedupe drawer, shortcuts.
   - Run complete repository quality checks (`./scripts/check.sh all`).

### Affected Components & Packages

- `dashboard/frontend/src/main.ts`: Register PrimeVue `ToastService`.
- `dashboard/frontend/src/App.vue`: Mount `<Toast />`, wire keyboard shortcuts, register polling controls.
- `dashboard/frontend/src/composables/`: `useFormatters.ts`, `useClipboard.ts`, `useToastNotification.ts`, `useKeyboardShortcuts.ts`.
- `dashboard/frontend/src/types/`: `models.ts`, `dedupe.ts`.
- `dashboard/frontend/src/stores/`: `bandwidth.ts`, `app.ts`, `telemetry.ts`, `dedupe.ts`.
- `dashboard/frontend/src/components/`:
  - `BandwidthPopover.vue`, `ModalPendingDeletions.vue`, `AppHeader.vue`, `ModalGlobalSettings.vue`
  - `settings/SettingsBandwidthTab.vue`
  - `dedupe/DedupeStagedDrawer.vue`, `dedupe/DedupeExplorerTab.vue`, `ModalDedupe.vue`
  - `explorer/ExplorerBreadcrumbs.vue`, `explorer/ExplorerToolbar.vue`, `explorer/ExplorerTable.vue`, `explorer/ExplorerPane.vue`
- `dashboard/frontend/src/views/`: `ViewExplorer.vue`, `ViewFleet.vue`, `ViewTelemetryLedger.vue`, `LogDock.vue`.
- `tests/e2e/cockpit_spa_smoke_test.go`: Comprehensive E2E UI journey tests.

### Technical Strategy & Data Flow

```
[ Backend REST API ]
  ├── /api/core/bwlimit      <---> [ useBandwidthStore ]     <---> [ BandwidthPopover & SettingsBandwidthTab ]
  ├── /api/core/stats        <---> [ useBandwidthStore ]
  ├── /api/deletions/pending <---> [ useTelemetryStore ]     <---> [ ModalPendingDeletions ]
  └── /api/remotes/dedupe/batch <-> [ useDedupeStore ]       <---> [ DedupeStagedDrawer ]

[ Reactive Event & Toast Subsystem ]
  [ API / User Actions ] ---> [ useToastNotification() ] ---> PrimeVue ToastService ---> [ <Toast /> in App.vue ]

[ Modular File Explorer ]
  [ ViewExplorer.vue ] (Dual Pane Layout & Cross-Pane Drag/Drop Coordinator)
    ├── Pane Left:  [ ExplorerPane.vue ]  --> [ Toolbar, Breadcrumbs, Table ]
    └── Pane Right: [ ExplorerPane.vue ] --> [ Toolbar, Breadcrumbs, Table ]
```

### Safety & Error Handling

- **Canary File Protection**: Deletion guards for `RCLONE_TEST` remain unconditionally active across all actions.
- **Blast-Radius Safeguards**: Pending deletions and deduplication batch execution require affirmative user confirmation.
- **Event Listener Cleanup**: All keyboard shortcut listeners and polling intervals clean up on unmount (`onUnmounted`).
- **Input Guard Rails**: Global shortcuts ignore keystrokes inside text inputs, textareas, and selects.
- **Single Static Binary**: Precompiled assets in `dist/` remain committed for zero-Node Go backend builds.

## Verification Plan

- TypeScript type checking: `npm --prefix dashboard/frontend run type-check`
- Production SPA bundling: `npm --prefix dashboard/frontend run build`
- Architecture LOC enforcement: `./scripts/check.sh loc`
- Headless Chrome E2E suite: `go test -v -count=1 ./tests/e2e/...`
- Full pre-flight quality verification: `./scripts/check.sh all` (all 7 quality stages)



