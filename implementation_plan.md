# Implementation Plan: Frontend Component & Store Decomposition

## Proposed Changes

Refactor bloated Vue 3 Single-File Components and Pinia stores across `dashboard/frontend/src/` to eliminate "God Objects", enforce single responsibility, improve component testability, and isolate UI presentation from domain orchestration.

### Affected Components & Packages

1. **Profile Management (`dashboard/frontend/src/components/profiles/`)**:
   - `ProfileListTab.vue`: Active profiles overview, status badges, sync triggers, action bar.
   - `ProfileFormTab.vue`: Profile identity, schedule, review mode, comparison mode, safety limits, exclude patterns, dynamic provider options, guided flags builder, and synthesized live CLI preview.
   - `ProfileHistoryTab.vue`: Historical version snapshot list and rollback execution.
   - `ProfileCloneDialog.vue`: Modal dialog for duplicating workload configuration.
   - `ProfileManagerModal.vue`: Streamlined dialog container coordinating active tab state.

2. **Remotes Workspace (`dashboard/frontend/src/components/remotes/`)**:
   - `RemotesKpiRibbon.vue`: Top storage metric summary tiles (Total, Used, Free, Count).
   - `RemoteCard.vue`: Card representing a storage remote with quota visualization, auth badges, migration banner, and action buttons.
   - `ViewRemotes.vue`: Streamlined view orchestrating toolbar filters, search, and remote grid.

3. **Remote Wizard Store Decoupling (`dashboard/frontend/src/stores/`)**:
   - `stores/remotes.ts`: Incorporate wizard state (`wizardOpen`, `providers`, `providersLoading`, `selectedProvider`, `fetchProviders`, `openWizard`, `closeWizard`, `createRemote`).
   - `stores/dedupe.ts`: Retain compatibility alias / delegate to `remotesStore` or eliminate wizard state from dedupe store so dedupe concerns remain cleanly isolated.

4. **Remote Provisioning Wizard Stepper (`dashboard/frontend/src/components/wizard/`)**:
   - `WizardStepProviderSelect.vue`: Step 1 provider catalog grid, category pills, and search filtering.
   - `WizardStepConfigure.vue`: Step 2 remote naming, OAuth toggle / `DynamicProviderForm`, Crypt wrap settings, and profile bridging.
   - `WizardStepReview.vue`: Step 3 pre-deployment summary checklist, deployment action, and OAuth redirect polling.
   - `ModalRemoteWizard.vue`: Streamlined stepper coordinator.

5. **File Explorer Decomposition (`dashboard/frontend/src/composables/`)**:
   - `composables/useExplorerModals.ts`: Centralizes state and helpers for preview, compare, mkdir, move, transfer, and delete confirmation modals.
   - `composables/useExplorerPanes.ts`: Encapsulates dual pane state, pane swapping, path navigation, and cross-pane drag/drop coordination.
   - `views/ViewExplorer.vue`: Clean orchestrator delegating to composables and child components.

### Technical Strategy & Data Flow

- Decompose UI templates without altering any existing TypeScript API contracts or Pinia store getters.
- Preserve all existing event semantics, accessibility attributes, keyboard navigation, and PrimeVue integration.
- Ensure strict type safety with Vue 3 `defineProps` and `defineEmits`.

### Safety & Error Handling

- Props and Emits contracts maintain 100% parity with parent components.
- Reactive refs and computeds retain referential integrity across decomposed composables.
- Zero client-side bundle regression: run `vue-tsc --noEmit` and `vite build` to guarantee compilation integrity.

## Verification Plan

- `npm --prefix dashboard/frontend run type-check` after every micro-task.
- `npm --prefix dashboard/frontend run build` to verify production assets.
- `./scripts/check.sh all` for final full-system quality gates.
