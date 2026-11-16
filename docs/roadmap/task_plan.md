# Task Plan: Unified FSM Storage Control Plane & Vue 3 + PrimeVue Cockpit Modernization

## Goal
Transform Bisync Hub into an autonomous, deterministic Storage Control Plane by implementing a thread-safe Go Finite State Machine (`SyncFSM`) with data-plane failure contract tests, co-designed with a desktop-grade cockpit SPA (Vue 3, PrimeVue v4, Tailwind CSS, TypeScript).

## Tasks

- [x] Task 1: Backend FSM Core & Transition Matrix (`internal/engine/`)
  - **Scope**: `internal/engine/state_machine.go`, `internal/engine/state_machine_test.go`, `internal/models/fsm.go`
  - **Verification**: `go test -v -race -count=1 ./internal/engine/...`

- [x] Task 2: Toolchain Setup & Project Scaffold (`~/.nvm`, `dashboard/frontend/`)
  - **Scope**: NVM, Node 22 LTS, `dashboard/frontend/package.json`, `tsconfig.json`, `vite.config.ts`, `tailwind.config.js`
  - **Verification**: `node -v && npm run build`

- [ ] Task 3: Runner FSM Integration & Data Plane Failure Contracts (`internal/engine/runner/`, `internal/services/`)
  - **Scope**: `internal/engine/runner/runner.go`, `runner_contract_test.go`, `internal/services/sync_service.go`
  - **Verification**: `go test -v -race -count=1 ./internal/engine/runner/... -run "TestContract_"`

- [ ] Task 4: PrimeVue Theming, TypeScript Contracts & Pinia Stores (`dashboard/frontend/src/`)
  - **Scope**: `src/theme/aura-preset.ts`, `src/types/fsm.ts`, `src/types/models.ts`, `src/stores/*.ts`
  - **Verification**: `npm run type-check`

- [ ] Task 5: Cockpit Application Shell & High-Density Views (`dashboard/frontend/src/views/`)
  - **Scope**: `AppHeader.vue`, `FSMStateBadge.vue`, `ViewFleet.vue`, `ViewExplorer.vue` (Splitter), `ViewAudit.vue` (DataTable)
  - **Verification**: `npm run build`

- [ ] Task 6: High-Performance Streaming Logs, Telemetry & Interactive Modals
  - **Scope**: `ViewLogs.vue` (VirtualScroller), `ViewTelemetry.vue`, `ModalReview.vue`, `ModalCreateRemote.vue` (Stepper)
  - **Verification**: `npm run build && npm run type-check`

- [ ] Task 7: Go Server Cutover, Legacy Pruning & Final Quality Verification
  - **Scope**: `internal/server/routes.go`, `main.go`, `main_test.go`, prune legacy scripts and templates
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-11-15: Initialized plan for Unified FSM Storage Control Plane & Vue 3 + PrimeVue Cockpit Modernization from implementation_plan.md.
- 2026-11-16: Completed Task 1 - Backend FSM Core & Transition Matrix. Implemented thread-safe `SyncFSM` in `internal/engine/state_machine.go` with transition matrix validation (`CanTransition`), state change event listeners, and API DTO models in `internal/models/fsm.go`. Tested valid/invalid transitions and 50 concurrent reader/writer goroutines with race detector passing. Verified with `go test -v -race -count=1 ./internal/engine/...` and `./scripts/check.sh loc`.
- 2026-11-16: Completed Task 2 - Toolchain Setup & Project Scaffold. Configured NVM + Node 22 LTS (v22.23.5) in user space, scaffolded Vite + Vue 3 + PrimeVue v4 + Tailwind CSS + TypeScript in `dashboard/frontend/`, configured auto-rename of `cockpit.html` to `dist/index.html` to preserve test stability for legacy server routes, whitelisted `dist/` in `.gitignore`, and built production bundle. Verified with `node -v && npm run build`.





















