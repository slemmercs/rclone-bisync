# Implementation Plan: UI Ergonomics, God Component Decomposition & Merge Documentation

## Proposed Changes

Address 4 architectural, ergonomic, and documentation requirements to prepare `feat/fsm-control-plane-and-vue-cockpit` for merging into `main`:
1. **Fleet Telemetry Chart Collapse Toggle**: Add a chevron collapse toggle to the Live Performance Telemetry card in `ViewFleet.vue` to allow users to hide/expand the canvas, preserving vertical space on compact displays while persisting preference in `localStorage`.
2. **Sticky Action Footers in Setup Modals**: Ensure `ProfileManagerModal.vue` and `ModalRemoteWizard.vue` pin their primary action buttons (`Save Configuration`, `Deploy Remote`, `Next Step`, `Back`, `Cancel`) to the bottom via sticky backdrop positioning so buttons never scroll off screen.
3. **Decompose Frontend "God Components"**:
   - Decompose `ModalDedupe.vue` (716 LOC) by extracting `DedupeGroupCluster.vue` and `DedupeActionBar.vue`.
   - Decompose `ModalGlobalSettings.vue` (686 LOC) by extracting `SettingsDefaultsTab.vue`, `SettingsEngineTab.vue`, `SettingsAlertsTab.vue`, and `SettingsAuditTab.vue`.
4. **Comprehensive Documentation & README Rewrite for Vue Cockpit Merge**: Overhaul `README.md`, `site/index.md`, and `site/architecture.md` to document the FSM Control Plane, Vue 3 + PrimeVue v4 Cockpit SPA, Dedupe Workspace dual match modes, and Audit Ledger cryptographic mutation invariants.

### Affected Components & Packages

- `dashboard/frontend/src/views/ViewFleet.vue`: Telemetry collapse toggle and state persistence.
- `dashboard/frontend/src/components/ProfileManagerModal.vue`: Sticky action footer.
- `dashboard/frontend/src/components/ModalRemoteWizard.vue`: Sticky navigation footer.
- `dashboard/frontend/src/components/ModalDedupe.vue`, `dashboard/frontend/src/components/dedupe/*`: Dedupe component decomposition.
- `dashboard/frontend/src/components/ModalGlobalSettings.vue`, `dashboard/frontend/src/components/settings/*`: Settings component decomposition.
- `README.md`, `site/index.md`, `site/architecture.md`: Full documentation overhaul for merge.

### Technical Strategy & Data Flow

- **Ergonomics**: Sticky positioning inside modal viewports uses `sticky bottom-0 bg-carbon-surface/95 backdrop-blur-sm z-10` with proper padding and borders to guarantee actionable controls remain accessible on all screen heights.
- **Component Decomposition**: Sub-components communicate via typed props and emitted events or direct Pinia store actions, ensuring single responsibility and component LOC <= 350.
- **Zero-Runtime Single Binary**: Pre-built Vue assets in `dist/` remain committed, and all Go tests continue to run without live external network dependencies.

### Safety & Error Handling

- All concurrency, canary protection (`RCLONE_TEST`), and jail confinement invariants remain unaffected.
- Headless Chrome E2E tests verify zero console errors and full SPA functionality across all tabs and modals.

## Verification Plan

- `npm --prefix dashboard/frontend run build`
- `go test -count=1 -v ./tests/e2e -run TestCockpitSPA_HeadlessChromeSmoke`
- `./scripts/check.sh all`
