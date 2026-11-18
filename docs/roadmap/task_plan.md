# Task Plan: Vue 3 Cockpit Full Feature Parity, Modular Decomposition & UX Modernization

## Goal
Achieve 100% functional, operational, and visual parity between legacy Alpine.js and Vue 3 / PrimeVue cockpit while elevating modularity, performance, and operator ergonomics through composable extraction, toast subsystem integration, bandwidth throttling controls, pending deletions safety review, deduplication batch actions drawer, monolithic ViewExplorer decomposition, and global keyboard shortcuts.

## Tasks

- [x] Task 1: Core Composables Layer & PrimeVue Toast Notification Subsystem
  - **Scope**: `dashboard/frontend/src/composables/useFormatters.ts`, `dashboard/frontend/src/composables/useClipboard.ts`, `dashboard/frontend/src/composables/useToastNotification.ts`, `dashboard/frontend/src/main.ts`, `dashboard/frontend/src/App.vue`, and component consumers (`ViewFleet.vue`, `ViewTelemetryLedger.vue`, `ModalDedupe.vue`, `LogDock.vue`)
  - **Verification**: `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build && ./scripts/check.sh loc`

- [x] Task 2: Bandwidth Throttling & Live Core Stats Control Plane
  - **Scope**: `dashboard/frontend/src/types/models.ts`, `dashboard/frontend/src/stores/bandwidth.ts`, `dashboard/frontend/src/components/BandwidthPopover.vue`, `dashboard/frontend/src/components/settings/SettingsBandwidthTab.vue`, `dashboard/frontend/src/components/AppHeader.vue`, `dashboard/frontend/src/components/ModalGlobalSettings.vue`
  - **Verification**: `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build && ./scripts/check.sh loc`

- [x] Task 3: Pending Deletions Safety Review Modal
  - **Scope**: `dashboard/frontend/src/types/models.ts`, `dashboard/frontend/src/components/ModalPendingDeletions.vue`, `dashboard/frontend/src/stores/telemetry.ts`, `dashboard/frontend/src/components/AppHeader.vue`, `dashboard/frontend/src/App.vue`
  - **Verification**: `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build && ./scripts/check.sh loc`

- [x] Task 4: Dedupe Batch Staged Actions Drawer & Directory Explorer
  - **Scope**: `dashboard/frontend/src/types/dedupe.ts`, `dashboard/frontend/src/stores/dedupe.ts`, `dashboard/frontend/src/components/dedupe/DedupeStagedDrawer.vue`, `dashboard/frontend/src/components/dedupe/DedupeExplorerTab.vue`, `dashboard/frontend/src/components/ModalDedupe.vue`
  - **Verification**: `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build && ./scripts/check.sh loc`

- [ ] Task 5: Decompose Monolithic ViewExplorer into Cohesive Components
  - **Scope**: `dashboard/frontend/src/components/explorer/ExplorerBreadcrumbs.vue`, `dashboard/frontend/src/components/explorer/ExplorerToolbar.vue`, `dashboard/frontend/src/components/explorer/ExplorerTable.vue`, `dashboard/frontend/src/components/explorer/ExplorerPane.vue`, `dashboard/frontend/src/views/ViewExplorer.vue`
  - **Verification**: `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build && ./scripts/check.sh loc`

- [ ] Task 6: Global Keyboard Shortcuts Subsystem & Operator Polling Controls
  - **Scope**: `dashboard/frontend/src/composables/useKeyboardShortcuts.ts`, `dashboard/frontend/src/stores/app.ts`, `dashboard/frontend/src/App.vue`, `dashboard/frontend/src/components/AppHeader.vue`
  - **Verification**: `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build && ./scripts/check.sh loc`

- [ ] Task 7: Headless Chrome E2E Suite Expansion
  - **Scope**: `tests/e2e/cockpit_spa_smoke_test.go`
  - **Verification**: `go test -v -count=1 ./tests/e2e/...`

- [ ] Task 8: Full Pre-Flight Verification & All Quality Gates
  - **Scope**: Full repository quality gates across all 7 verification stages
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-11-18: Initialized plan from implementation_plan.md.
- 2026-11-18: Completed Task 1 - Registered PrimeVue ToastService in main.ts, mounted Toast in App.vue, created useFormatters, useClipboard, useToastNotification composables, and integrated into ViewFleet, ViewTelemetryLedger, ModalDedupe, and LogDock.
- 2026-11-18: Completed Task 2 - Implemented Bandwidth models & Pinia store, BandwidthPopover header trigger with presets and live stats, SettingsBandwidthTab in ModalGlobalSettings, and full test/LOC verification.
- 2026-11-18: Completed Task 3 - Implemented ModalPendingDeletions, pending deletions polling in useTelemetryStore, wired header & telemetry launch buttons, aligned TypeScript schema with Go backend, and resolved missing X-Dashboard-Request CSRF header on bandwidth mutations.
- 2026-11-18: Completed Task 4 - Implemented Dedupe staged actions and batch execution types/store methods, created DedupeStagedDrawer.vue and DedupeExplorerTab.vue, integrated Directory Explorer navigation & file previews in ModalDedupe.vue, verified with type-check, build, and LOC gate.
