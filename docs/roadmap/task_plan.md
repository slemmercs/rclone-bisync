# Task Plan: UI Ergonomics, God Component Decomposition & Merge Documentation

## Goal
Implement telemetry collapse toggle, sticky modal action footers, decompose frontend God Components (`ModalDedupe`, `ModalGlobalSettings`), and rewrite merge documentation (`README.md`, `site/`).

## Tasks

- [x] Task 1: Telemetry Chart Collapse Toggle on Fleet View
  - **Scope**: `dashboard/frontend/src/views/ViewFleet.vue`
  - **Verification**: `npm --prefix dashboard/frontend run build && go test -count=1 ./tests/e2e -run TestCockpitSPA_HeadlessChromeSmoke`

- [x] Task 2: Sticky Action Footers in Setup Modals
  - **Scope**: `dashboard/frontend/src/components/ProfileManagerModal.vue`, `dashboard/frontend/src/components/ModalRemoteWizard.vue`
  - **Verification**: `npm --prefix dashboard/frontend run build && go test -count=1 ./tests/e2e -run TestCockpitSPA_HeadlessChromeSmoke`

- [x] Task 3: Vue "God Component" Decomposition - Dedupe Workspace
  - **Scope**: `dashboard/frontend/src/components/ModalDedupe.vue`, `dashboard/frontend/src/components/dedupe/`
  - **Verification**: `npm --prefix dashboard/frontend run build && go test -count=1 ./tests/e2e -run TestCockpitSPA_HeadlessChromeSmoke/DedupeModalInteraction`

- [ ] Task 4: Vue "God Component" Decomposition - Global Settings
  - **Scope**: `dashboard/frontend/src/components/ModalGlobalSettings.vue`, `dashboard/frontend/src/components/settings/`
  - **Verification**: `npm --prefix dashboard/frontend run build && go test -count=1 ./tests/e2e -run TestCockpitSPA_HeadlessChromeSmoke/GlobalSettingsModalInteraction`

- [ ] Task 5: Documentation & README Rewrite for Vue Cockpit Merge
  - **Scope**: `README.md`, `site/index.md`, `site/architecture.md`
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-11-18: Initialized plan from implementation_plan.md for UI ergonomics, God Component decomposition, and merge documentation.
- 2026-11-18: Completed Task 1 - Added collapse toggle chevron button to Live Performance Telemetry card in `ViewFleet.vue`, persisted collapse preference via `localStorage.getItem('fleet_telemetry_collapsed')`, conditionally collapsed canvas while keeping inline latency/speed/health stats visible. Verified via frontend build and Chrome E2E smoke tests.
- 2026-11-18: Completed Task 2 - Implemented sticky bottom action footers in `ProfileManagerModal.vue` and `ModalRemoteWizard.vue` with backdrop blur, borders, and elevated z-index so submit, deploy, and navigation buttons remain anchored and visible across scrolling form contents. Verified via frontend build and headless Chrome E2E smoke suite.
- 2026-11-18: Completed Task 3 - Decomposed oversized `ModalDedupe.vue` (716 LOC -> 343 LOC) by extracting `DedupeGroupCluster.vue` (335 LOC) for collision search/filters, collapse controls, duplicate groups, and candidate actions, and `DedupeActionBar.vue` (62 LOC) for staged batch decisions and execution. Verified via frontend build and headless Chrome E2E suite (`DedupeModalInteraction` + all 22 tests passing).
