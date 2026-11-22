# Task Plan: Frontend Component & Store Decomposition

## Goal
Decompose frontend God Objects into cohesive subcomponents, extract reusable composables, and decouple cross-domain Pinia stores across `dashboard/frontend/src/`.

## Tasks

- [x] Task 1: Decompose ProfileManagerModal into Profiles Subcomponents
  - **Scope**: Create `components/profiles/ProfileListTab.vue`, `components/profiles/ProfileFormTab.vue`, `components/profiles/ProfileHistoryTab.vue`, `components/profiles/ProfileCloneDialog.vue`, refactor `ProfileManagerModal.vue`.
  - **Verification**: `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build`

- [ ] Task 2: Decompose ViewRemotes with RemoteCard & RemotesKpiRibbon
  - **Scope**: Create `components/remotes/RemoteCard.vue` and `components/remotes/RemotesKpiRibbon.vue`, refactor `ViewRemotes.vue`.
  - **Verification**: `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build`

- [ ] Task 3: Decouple Remote Wizard State from DedupeStore to RemotesStore
  - **Scope**: `stores/remotes.ts`, `stores/dedupe.ts`, and relevant callers.
  - **Verification**: `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build`

- [ ] Task 4: Decompose ModalRemoteWizard into Wizard Step Subcomponents
  - **Scope**: Create `components/wizard/WizardStepProviderSelect.vue`, `components/wizard/WizardStepConfigure.vue`, `components/wizard/WizardStepReview.vue`, refactor `ModalRemoteWizard.vue`.
  - **Verification**: `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build`

- [ ] Task 5: Decompose ViewExplorer with useExplorerModals & useExplorerPanes
  - **Scope**: Create `composables/useExplorerModals.ts`, `composables/useExplorerPanes.ts`, refactor `views/ViewExplorer.vue`.
  - **Verification**: `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build`

- [ ] Task 6: Full Verification & Quality Gates
  - **Scope**: Run all 7 quality gate stages via `./scripts/check.sh all`.
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-11-22: Initialized implementation plan and task execution plan for frontend component & store decomposition.
- 2026-11-22: Completed Task 1. Decomposed `ProfileManagerModal.vue` (reduced from 648 LOC to ~220 LOC) into modular child components under `components/profiles/` (`ProfileListTab.vue`, `ProfileHistoryTab.vue`, `ProfileCloneDialog.vue`, `ProfileFormTab.vue`). Verified with `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build`.
