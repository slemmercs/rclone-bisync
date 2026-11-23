# Task Plan: Storage Remote Delete Safeguard Modal & Cascade Warning UI

## Goal
Replace the native browser `window.confirm()` dialog with a styled Cockpit safeguard modal (`ModalRemoteDelete.vue`) that presents remote endpoint details, dependent overlay cascade warnings, and crypt key deletion alerts.

## Tasks

- [x] Task 1: Create `dashboard/frontend/src/components/ModalRemoteDelete.vue`
  - **Scope**: `dashboard/frontend/src/components/ModalRemoteDelete.vue`
  - **Verification**: `npm --prefix dashboard/frontend run type-check`

- [ ] Task 2: Integrate `ModalRemoteDelete` into `ViewRemotes.vue`
  - **Scope**: `dashboard/frontend/src/views/ViewRemotes.vue`
  - **Verification**: `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build`

- [ ] Task 3: Full Quality Verification Suite
  - **Scope**: Pre-flight verification across all 7 stages
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-11-23: Completed Task 1. Created `ModalRemoteDelete.vue` with dark-mode styling, remote endpoint details card, dependent overlay cascade warning, and crypt key safeguard banner. Verified with `npm --prefix dashboard/frontend run type-check`.
- 2026-11-23: Initialized task plan based on approved implementation plan for `ModalRemoteDelete.vue` with cascade warning UI.








