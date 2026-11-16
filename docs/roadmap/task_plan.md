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

- [x] Task 3: Runner FSM Integration & Data Plane Failure Contracts (`internal/engine/runner/`, `internal/services/`)
  - **Scope**: `internal/engine/runner/runner.go`, `runner_failure_test.go`, `internal/services/sync_service.go`
  - **Verification**: `go test -v -race -count=1 ./internal/engine/runner/...`

- [x] Task 4: PrimeVue Theming, TypeScript Contracts & Pinia Stores (`dashboard/frontend/src/`)
  - **Scope**: `src/theme/aura-preset.ts`, `src/types/fsm.ts`, `src/types/models.ts`, `src/stores/*.ts`
  - **Verification**: `npm run type-check`

- [x] Task 5: Cockpit Application Shell & High-Density Views (`dashboard/frontend/src/views/`)
  - **Scope**: `AppHeader.vue`, `FSMStateBadge.vue`, `ViewFleet.vue`, `ViewExplorer.vue` (Splitter), `ViewAudit.vue` (DataTable)
  - **Verification**: `npm run build`

- [x] Task 6: High-Performance Streaming Logs, Telemetry & Interactive Modals
  - **Scope**: `LogDock.vue`, `TelemetryChart.vue`, `ConflictReviewModal.vue` (PrimeVue Dialog)
  - **Verification**: `npm run build && ./scripts/check.sh loc`

- [x] Task 7: Go Server Cutover, Legacy Pruning & Final Quality Verification
  - **Scope**: `internal/server/routes.go`, `main.go`, `main_test.go`, prune legacy scripts and templates
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-11-15: Initialized plan for Unified FSM Storage Control Plane & Vue 3 + PrimeVue Cockpit Modernization from implementation_plan.md.
- 2026-11-16: Completed Task 1 - Backend FSM Core & Transition Matrix. Implemented thread-safe `SyncFSM` in `internal/engine/state_machine.go` with transition matrix validation (`CanTransition`), state change event listeners, and API DTO models in `internal/models/fsm.go`. Tested valid/invalid transitions and 50 concurrent reader/writer goroutines with race detector passing. Verified with `go test -v -race -count=1 ./internal/engine/...` and `./scripts/check.sh loc`.
- 2026-11-16: Completed Task 2 - Toolchain Setup & Project Scaffold. Configured NVM + Node 22 LTS (v22.23.5) in user space, scaffolded Vite + Vue 3 + PrimeVue v4 + Tailwind CSS + TypeScript in `dashboard/frontend/`, configured auto-rename of `cockpit.html` to `dist/index.html` to preserve test stability for legacy server routes, whitelisted `dist/` in `.gitignore`, and built production bundle. Verified with `node -v && npm run build`.
- 2026-11-16: Completed Task 3 - Runner FSM Integration & Data Plane Failure Contracts. Wired `SyncFSM` transitions into `SyncRunner` (`internal/engine/runner/runner.go` and `runner_handlers.go`), added `Reset()` to `SyncFSM`, and implemented failure contract test suite in `runner_failure_test.go` covering success paths, lock contention, rate limiting, canary trips, dispatch failures, conflict/deletion safety triggers (`REVIEW_REQUIRED`), and multi-run reset lifecycles. Verified with `go test -v -race -count=1 ./internal/engine/runner/...`.
- 2026-11-16: Completed Task 4 - PrimeVue Theming, TypeScript Contracts & Pinia Stores. Implemented custom `CarbonPreset` based on "Agnoster Carbon & Electric Blue" color palette in `src/theme/aura-preset.ts` and `tailwind.config.js` (`#090A0D` app canvas, `#111318` cards, `#191C24` elevated, `#262A36` hairline border, `#0EA5E9` electric blue accent). Defined strong TypeScript contracts in `src/types/fsm.ts` and `src/types/models.ts` strictly mirroring Go models. Implemented typed Pinia stores for FSM, Profiles, Telemetry, Logs, Review, and Explorer. Verified with `npm run type-check` and `npm run build`.
- 2026-11-16: Completed Task 5 - Cockpit Application Shell & High-Density Views. Implemented `FSMStateBadge.vue` with animated status indicators, `AppHeader.vue` with profile switcher, daemon status, and sync triggers, `ViewFleet.vue` with KPI telemetry ribbon and workload bindings cards, `ViewExplorer.vue` with PrimeVue `<Splitter>` dual-pane local/remote tree inspection and canary badges, and `ViewAudit.vue` with PrimeVue `<DataTable>` cryptographic hash chain validation and Steel Pill action tags (`#1E222B`). Verified with `npm run type-check` and `npm run build`.
- 2026-11-16: Completed Task 6 - High-Performance Streaming Logs, Telemetry & Interactive Modals. Implemented `LogDock.vue` with virtual log streaming, filter pills (`ALL`, `ERROR`, `NOTICE`), height presets (`collapsed`, `docked`, `expanded`), and auto-scroll; implemented `TelemetryChart.vue` using Chart.js with strict unproxied canvas lifecycle isolation (preventing Vue reactive proxy recursions) plotting real-time throughput and daemon latency with Agnoster Carbon gradients; implemented `ConflictReviewModal.vue` powered by PrimeVue `<Dialog>` with batch decision staging ("Keep All Local", "Keep All Remote", "Keep Both", "Restore All Deletions") and safety confirmation token gates. Verified with `npm run build && ./scripts/check.sh loc`.
- 2026-11-16: Completed Task 7 - Go Server Cutover, Legacy Pruning & Final Quality Verification. Updated `internal/server/routes.go` to serve precompiled SPA distribution assets (`/assets/`) and mount the modern Vue 3 + PrimeVue SPA at `/cockpit` with client-side fallback routing. Added unit tests in `internal/server/server_test.go` verifying the cockpit endpoint and mount point. Formatted backend Go files and verified the entire project with all 7 checks passing in `./scripts/check.sh all` and `npm run type-check && npm run build`.





















