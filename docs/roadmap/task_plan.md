# Task Plan: Dual-Pane Explorer Jump to File Navigation

## Goal
Enable seamless deep navigation to files in the Dual-Pane Directory Explorer when clicking "Jump to file in Explorer" links across daemon logs, audit ledger, and dedupe modals.

## Tasks

- [x] Task 1: Update `explorerStore` and `ViewExplorer.vue` navigation logic
  - **Scope**: `dashboard/frontend/src/stores/explorer.ts`, `dashboard/frontend/src/views/ViewExplorer.vue`, `dashboard/frontend/src/components/explorer/ExplorerTable.vue`
  - **Verification**: `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build`

- [x] Task 2: Build bundle, execute tests, and run pre-commit quality gates
  - **Scope**: Pre-commit validation and end-to-end checks
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-11-20: Completed Task 2. Ran full quality gates via `./scripts/check.sh all` (all 7 stages: code formatting, template safety, license check, go vet, all unit/integration/E2E tests with -race, LOC audit, release binary build). All passed cleanly.
- 2026-11-20: Completed Task 1. Added `consumeTargetJumpPath` and updated `browse` in `explorerStore`. Added `applyTargetJump` in `ViewExplorer.vue` handling local/remote subpaths, pane swapping, and initial mount coordination. Added automatic element scroll in `ExplorerTable.vue`. Verified with `npm run type-check` and `npm run build`.
- 2026-11-20: Initialized plan from implementation_plan.md on branch `fix/explorer-jump-to-file-navigation`.


