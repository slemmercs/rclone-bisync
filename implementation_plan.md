# Implementation Plan: Frontend God Object Decomposition & Store Decoupling

Deconstruct oversized Vue 3 components and decouple stores across `dashboard/frontend/src/` to enforce single responsibility, strict bounded contexts, and high testability.

Following the initial decomposition of `ViewExplorer`, `ModalRemoteWizard`, `ProfileManagerModal`, and `ViewRemotes`, this plan addresses the remaining frontend structural hotspots identified in Issue #103: `AppHeader.vue` (418 LOC orchestrating 9 stores), `ViewFleet.vue` (407 LOC monolithic template), `stores/dedupe.ts` (retaining stale wizard bridges), and duplicated log filtering/export routines across `ModalDaemonLogs.vue`, `LogDock.vue`, and `ViewTelemetryLedger.vue`.

The implementation splits monolithic views into cohesive subcomponents under `components/header/` and `components/fleet/`, cleans cross-domain leakage from `dedupeStore`, and introduces a shared `useLogFilteringAndExport.ts` composable. All refactoring preserves 100% of existing behavior, props, emits, keyboard shortcuts, and visual design while enabling zero-dependency build and type-checking verification via `npm run type-check` and `npm run build`.

---

## Types

### 1. Fleet Component Contracts (`dashboard/frontend/src/types/fleet.ts`)
Create a dedicated typing contract for fleet filtering and status metrics:
```typescript
import type { SyncFSMState } from './fsm'
import type { Profile } from './models'

export type StatusFilterType = 'ALL' | 'ACTIVE' | 'IDLE' | 'FAILED' | 'REVIEW_REQUIRED'

export interface FleetStatusPill {
  label: string
  value: StatusFilterType
  count: number
}

export interface FleetKpiData {
  totalProfiles: number
  activeWorkloads: number
  engineState: string
  pendingReview: boolean
}

export interface FleetProfileSyncEvent {
  profile: Profile
  dryRun: boolean
}
```

### 2. Log Export Contracts (`dashboard/frontend/src/types/logs.ts`)
Standardize export formats and severity statistics:
```typescript
export interface LogSeverityCounts {
  total: number
  errors: number
  warnings: number
  notices: number
  info: number
}

export type LogCategoryFilter = 'all' | 'file_ops' | 'sync' | 'audit'
```

---

## Files

### New Files
1. `dashboard/frontend/src/types/fleet.ts`: Shared TypeScript interfaces for fleet filter types, KPI metrics, and card events.
2. `dashboard/frontend/src/components/header/HeaderBrandStatus.vue`: Control plane identity, daemon health indicator, MCP inspector launcher, and live telemetry poller toggle.
3. `dashboard/frontend/src/components/header/HeaderNavigation.vue`: Workspace navigation tab bar (`Fleet`, `Telemetry`, `Remotes`, `Explorer`, `Audit`).
4. `dashboard/frontend/src/components/header/HeaderProfileControl.vue`: Profile dropdown switcher, `FSMStateBadge`, review alert trigger, pending deletions trigger, and primary `Sync Now` button.
5. `dashboard/frontend/src/components/header/HeaderActionToolbar.vue`: Toolbar launchers (Log Dock toggle, Queue manager pill, Bandwidth throttle popover, Help drawer, Global settings, Shortcuts modal, Quick-create dropdown).
6. `dashboard/frontend/src/components/fleet/FleetKpiRibbon.vue`: 4 top KPI metrics cards (Total Profiles, Active Workloads, Engine State, Safety Circuit).
7. `dashboard/frontend/src/components/fleet/FleetToolbar.vue`: Search input with clear button and segmented status filter pills.
8. `dashboard/frontend/src/components/fleet/FleetProfileCard.vue`: Encapsulated profile card with paths, schedule/review badges, last run formatting, error callouts, and action buttons.
9. `dashboard/frontend/src/composables/useLogFilteringAndExport.ts`: Shared composable for log severity counting, clipboard export, and action badge styling.

### Modified Files
1. `dashboard/frontend/src/components/AppHeader.vue`: Refactored into a ~45 LOC shell delegating to `HeaderBrandStatus`, `HeaderNavigation`, `HeaderProfileControl`, and `HeaderActionToolbar`.
2. `dashboard/frontend/src/views/ViewFleet.vue`: Refactored into a ~130 LOC container delegating to `FleetKpiRibbon`, `FleetToolbar`, and `FleetProfileCard`.
3. `dashboard/frontend/src/stores/dedupe.ts`: Purge legacy remote creation wizard state (`wizardOpen`, `providers`, `fetchProviders`, `createRemote`, etc.) and remove dependency on `useRemotesStore`.
4. `dashboard/frontend/src/components/ModalDaemonLogs.vue`: Adopt `useLogFilteringAndExport` for severity counting and clipboard exports.
5. `dashboard/frontend/src/components/LogDock.vue`: Adopt `useLogFilteringAndExport` for action badge classes and copy operations.
6. `dashboard/frontend/src/views/ViewTelemetryLedger.vue`: Adopt `useLogFilteringAndExport` for audit badge classes and copy operations.

### Files to Delete or Move
- None.

---

## Functions

### New Functions
- `useLogFilteringAndExport.ts`:
  - `computeLogSeverityCounts(logs: Ref<LogEntry[]> | ComputedRef<LogEntry[]>): ComputedRef<LogSeverityCounts>`
  - `useLogClipboardExport()`: Provides `copyLogs(logs: LogEntry[], asJSON: boolean)`, `copiedRaw`, and `copiedJSON`
  - `getAuditActionBadgeClass(action?: string): string`: Unified Tailwind badge styling
- `FleetProfileCard.vue`:
  - `onSelect(id: string)`, `onSync(profile: Profile, dryRun: boolean)`, `onOpenExplorer(profile: Profile)`, `onOpenTelemetry(profile: Profile)`

### Modified Functions
- `stores/dedupe.ts`:
  - Remove exported properties: `wizardOpen`, `providers`, `providersLoading`, `selectedProvider`, `fetchProviders`, `openWizard`, `closeWizard`, `createRemote`.
  - Retain all deduplication methods (`runDedupe`, `stageCandidateAction`, `applyBatch`, `loadExplorer`, etc.).
- `ModalDaemonLogs.vue`:
  - Replace local `copyLogs` and `counts` computation with `useLogFilteringAndExport`.
- `ViewTelemetryLedger.vue`:
  - Replace local `getAuditActionBadgeClass` with imported helper.
- `LogDock.vue`:
  - Replace local `getActionBadgeClass` with imported helper.

---

## Classes

*Not applicable (standard Vue 3 Composition API `<script setup lang="ts">` components and Pinia store composables).*

---

## Dependencies

- No new NPM packages or runtime dependencies required.
- Toolchain: Native Vue 3, Pinia 2.2+, PrimeVue 4.2+, Tailwind CSS 3.4+.

---

## Testing

### Validation Strategy
- Static Analysis & TypeScript: Run `npm run type-check` (`vue-tsc --noEmit`) to verify zero type errors across all new and refactored components.
- Frontend Production Bundler: Run `npm run build` (`vite build`) to ensure all SFC templates, CSS bindings, and chunks compile cleanly.
- Full System Integration: Run `./scripts/check.sh all` to verify all 8 quality stages (Go unit tests, race detector, security scans, linters, and frontend asset verification).

---

## Implementation Order

1. **Step 1: Clean `stores/dedupe.ts` & Isolate Bounded Context**
   - Remove unused wizard compatibility bridges and unused type imports.
   - Verify with `npm run type-check`.

2. **Step 2: Create Shared Log Filtering Composable & Refactor Log Viewers**
   - Create `dashboard/frontend/src/composables/useLogFilteringAndExport.ts`.
   - Update `ModalDaemonLogs.vue`, `LogDock.vue`, and `ViewTelemetryLedger.vue`.
   - Verify with `npm run type-check`.

3. **Step 3: Decompose `ViewFleet.vue` into Fleet Subcomponents**
   - Create `dashboard/frontend/src/types/fleet.ts`.
   - Create `components/fleet/FleetKpiRibbon.vue`, `components/fleet/FleetToolbar.vue`, and `components/fleet/FleetProfileCard.vue`.
   - Refactor `ViewFleet.vue` to compose the new subcomponents.
   - Verify with `npm run type-check` and `npm run build`.

4. **Step 4: Decompose `AppHeader.vue` into Header Subcomponents**
   - Create `components/header/HeaderBrandStatus.vue`.
   - Create `components/header/HeaderNavigation.vue`.
   - Create `components/header/HeaderProfileControl.vue`.
   - Create `components/header/HeaderActionToolbar.vue`.
   - Refactor `AppHeader.vue` into the root shell orchestrating the subcomponents.
   - Verify with `npm run type-check` and `npm run build`.

5. **Step 5: Full Suite Verification & Build Artifacts**
   - Execute `npm run build` to update `dashboard/frontend/dist/`.
   - Run `./scripts/check.sh all` and `go test -race ./...`.



