# Task Plan: Frontend God Object Decomposition & Store Decoupling (Issue #103)

## Goal
Deconstruct oversized frontend components (`AppHeader.vue`, `ViewFleet.vue`), decouple `dedupeStore` from stale remote wizard bridges, and extract a unified log presentation composable across `ModalDaemonLogs`, `LogDock`, and `ViewTelemetryLedger`.

## Tasks

- [x] Task 1: Clean stores/dedupe.ts & enforce strict deduplication domain boundary
  - **Scope**: `dashboard/frontend/src/stores/dedupe.ts`
  - **Verification**: `cd dashboard/frontend && npm run type-check`

- [x] Task 2: Create useLogFilteringAndExport composable & refactor log viewers
  - **Scope**: `dashboard/frontend/src/composables/useLogFilteringAndExport.ts`, `dashboard/frontend/src/components/ModalDaemonLogs.vue`, `dashboard/frontend/src/components/LogDock.vue`, `dashboard/frontend/src/views/ViewTelemetryLedger.vue`
  - **Verification**: `cd dashboard/frontend && npm run type-check`

- [x] Task 3: Decompose ViewFleet.vue into Fleet Subcomponents
  - **Scope**: `dashboard/frontend/src/types/fleet.ts`, `dashboard/frontend/src/components/fleet/FleetKpiRibbon.vue`, `dashboard/frontend/src/components/fleet/FleetToolbar.vue`, `dashboard/frontend/src/components/fleet/FleetProfileCard.vue`, `dashboard/frontend/src/views/ViewFleet.vue`
  - **Verification**: `cd dashboard/frontend && npm run type-check && npm run build`

- [x] Task 4: Decompose AppHeader.vue into Header Subcomponents
  - **Scope**: `dashboard/frontend/src/components/header/HeaderBrandStatus.vue`, `dashboard/frontend/src/components/header/HeaderNavigation.vue`, `dashboard/frontend/src/components/header/HeaderProfileControl.vue`, `dashboard/frontend/src/components/header/HeaderActionToolbar.vue`, `dashboard/frontend/src/components/AppHeader.vue`
  - **Verification**: `cd dashboard/frontend && npm run type-check && npm run build`

- [ ] Task 5: Full pre-flight quality verification & binary asset sync
  - **Scope**: `npm run build`, `./scripts/check.sh all`, `go test -race ./...`
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-11-23: Completed Task 4. Decomposed `AppHeader.vue` (from 418 LOC down to 47 LOC shell) by extracting `HeaderBrandStatus.vue`, `HeaderNavigation.vue`, `HeaderProfileControl.vue`, and `HeaderActionToolbar.vue`. Verified with `npm run type-check` and `npm run build`.
- 2026-11-23: Completed Task 3. Created `types/fleet.ts` and decomposed `ViewFleet.vue` into `FleetKpiRibbon.vue`, `FleetToolbar.vue`, and `FleetProfileCard.vue`. Reduced `ViewFleet.vue` LOC and verified with `npm run type-check` and `npm run build`.
- 2026-11-23: Completed Task 2. Created `useLogFilteringAndExport.ts` composable (severity counts, clipboard export, badge styling) and refactored `ModalDaemonLogs.vue`, `LogDock.vue`, and `ViewTelemetryLedger.vue`. Verified with `npm --prefix dashboard/frontend run type-check` and `npm --prefix dashboard/frontend run build`.
- 2026-11-23: Completed Task 1. Removed dead wizard state bridges (`wizardOpen`, `providers`, `fetchProviders`, `createRemote`, etc.) and unused catalog types from `dashboard/frontend/src/stores/dedupe.ts`. Verified with `npm --prefix dashboard/frontend run type-check` and `npm --prefix dashboard/frontend run build`.
- 2026-11-23: Initialized task plan from implementation_plan.md for Issue #103.









