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






















---

# Task Plan: 100% Feature Parity Migration (Legacy Alpine.js to Vue 3 SPA Cockpit)

## Goal
Achieve 100% functional, operational, and visual feature parity with the legacy Alpine.js cockpit across all 5 primary feature areas: Dual Explorer file operations & previews, Unified Run-Aggregated Ledger & correlated daemon logs, remote deduplication workspaces, slide-over help manual & diagnostics, and system settings modals.

## Tasks

- [x] Task 8: Dual Explorer File Operations, Previews & Context Menus
  - **Scope**: `src/types/explorer.ts`, `src/components/ModalFilePreview.vue`, `src/components/ModalCompare.vue`, `src/components/ModalExplorerMkdir.vue`, `src/components/ModalExplorerMove.vue`, `src/components/ModalConfirm.vue`, `src/components/ExplorerContextMenu.vue`, `src/views/ViewExplorer.vue`
  - **Verification**: `npm run type-check && npm run build`

- [x] Task 9: Unified Run-Aggregated Ledger & Correlated Daemon Logs
  - **Scope**: `src/views/ViewTelemetryLedger.vue`, `src/views/ViewTelemetry.vue`, `src/components/ModalDaemonLogs.vue`, `src/components/LogDock.vue`
  - **Verification**: `npm run type-check && npm run build`

- [x] Task 10: Remote Deduplication Workspace & Remote Creation Wizard
  - **Scope**: `src/types/dedupe.ts`, `src/stores/dedupe.ts`, `src/components/ModalDedupe.vue`, `src/components/ModalRemoteWizard.vue`, `src/views/ViewRemotes.vue`
  - **Verification**: `npm run type-check && npm run build`

- [x] Task 11: Slide-Over Help & Diagnostics Drawer
  - **Scope**: `src/data/help-policies.ts`, `src/data/help-flags.ts`, `src/components/HelpDrawer.vue`, `src/components/AppHeader.vue`, `src/App.vue`
  - **Verification**: `npm run type-check && npm run build`

- [ ] Task 12: System Health, Fleet Settings, and MCP Inspector Modals
  - **Scope**: `src/components/ModalSystemHealth.vue`, `src/components/ModalGlobalSettings.vue`, `src/components/ModalMcp.vue`, `src/stores/settings.ts`, `src/components/AppHeader.vue`, `src/App.vue`
  - **Verification**: `./scripts/check.sh all && npm run type-check && npm run build`

## Status Updates

- 2026-11-16: Initialized 100% Feature Parity Migration plan from approved implementation plan covering all 5 phases.

- 2026-11-16: Completed Task 8 (Phase 1) - Dual Explorer File Operations, Previews & Context Menus. Implemented typed explorer contracts in src/types/explorer.ts, updated useExplorerStore with preview, compare, mkdir, move, delete, and download operations, implemented ModalFilePreview.vue (multi-format preview with text, markdown, image data URLs, PDF iframe, and binary warnings), ModalCompare.vue (cryptographic hash and equivalence clustering), ModalExplorerMkdir.vue (subfolder creation), ModalExplorerMove.vue (rename and move), ModalConfirm.vue (blast-shield confirmation), and ExplorerContextMenu.vue (right-click action menu). Built and verified production bundle with npm run type-check && npm run build.
- 2026-11-16: Enhanced Task 8 Dual Explorer with multi-column sorting (Name, Type, Modified, Size with folders-first ordering and asc/desc toggles), inline quick-filtering per pane, short-formatted modification timestamps, local file MIME-type sniffing via `services.DetectMimeType`, and updated `formatBytes` to return `--` for non-sized Google Doc entries (`Size: -1`), resolving the 0 B discrepancy on Google Drive native spreadsheets. All tests and 7/7 quality checks passing.
- 2026-11-16: Completed Task 9 - Unified Run-Aggregated Ledger & Correlated Daemon Logs. Implemented `ViewTelemetryLedger.vue` with run-level event consolidation, multi-category filter pills (`All`, `File Ops`, `Sync`, `Audit`), profile dropdown filtering, repetitive log folding, and pagination; implemented `ModalDaemonLogs.vue` with PrimeVue Dialog, correlation by `run_id`/`job_id`/timestamps, level filters with badge counters, live search, fold repeated toggle, and raw/JSON copy; enhanced `src/stores/logs.ts` with bisync and daemon log actions; ported message normalization and run clustering in `src/utils/logClustering.ts`; integrated `ModalDaemonLogs` globally in `App.vue` and `LogDock.vue` with interactive `run:` badges. Verified with `npm run type-check && npm run build` and `go test ./...`.

- 2026-11-16: Refactored Vue 3 telemetry, log ecosystem, and Dual Explorer navigation to achieve complete parity with legacy Alpine cockpit:
  - Fixed missing daemon log lines, diff lists, and comparison settings grid by porting `unwrapAndParseLogLines` to unroll multi-line embedded JSON inside Rclone RC notice wrappers.
  - Rendered structured details grid for comparison settings (`Modtime`, `Size`, `Checksum`, `HashType1`, `HashType2`, `SlowHashDetected`, etc.) and lockfiles in `ModalDaemonLogs.vue`.
  - Added interactive file jump buttons (`→ path`) on all file-referencing log entries across `ModalDaemonLogs.vue`, `ViewTelemetryLedger.vue`, and `LogDock.vue`, seamlessly switching to the Dual Explorer tab and pulsing the target file row via `explorerStore.jumpToPath`.
  - Upgraded `LogDock.vue` from raw string dumping to a structured table layout with timestamp, level badge, action badge, profile badge, interactive file path chips, keyboard navigation (`N` / `P` step), pagination, auto-scroll, and category filter pills (`All`, `File Ops`, `Sync`, `Audit`).
  - Resolved duplicate run entries in the Unified Ledger by reversing the backend chronological tail order on fetch and consolidating `sync` execution progress, audit events, and terminal completion records into unified cards with expandable file drawers.
  - All 7/7 check.sh quality gates passed and frontend built successfully.

- 2026-11-16: Completed Task 10 - Remote Deduplication Workspace & Remote Creation Wizard. Implemented typed deduplication and remote wizard contracts in `src/types/dedupe.ts`, created `useDedupeStore` in `src/stores/dedupe.ts` with output parsing and staged batch decisions, created `src/utils/dedupeParser.ts` for structured CLI collision extraction, implemented `ModalDedupe.vue` featuring a 3-section workspace with dedupe strategy selection (`list`, `oldest`, `newest`, `rename`, `largest`, `smallest`), `--by-hash` toggle, paired pass options, collision candidates list, inline keep/delete staging, raw CLI logs tab, staged actions drawer with resync toggle, and daemon log links (`run:dedupe-...`). Implemented `ModalRemoteWizard.vue` with 3-step guided flow (provider selection with categories & search, remote configuration & profile bridging, deployment review). Enhanced `ViewRemotes.vue` with "Add Remote" wizard launcher, per-card "Deduplicate" launcher, category filtering (`All`, `Cloud Drive`, `Object Storage`, `Protocols`), search, and delete safeguards. Verified with `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build` and `./scripts/check.sh all` (all 7 quality gates passed).

- 2026-11-16: Completed Task 11 - Slide-Over Help & Diagnostics Drawer. Implemented typed reference catalogs `src/data/help-policies.ts` (safety policies, operational runbooks) and `src/data/help-flags.ts` (curated CLI flags with descriptions, caveats, examples, and doc links). Added diagnostic contracts to `src/types/models.ts` and created `useHelpStore` in `src/stores/help.ts` supporting context-aware scoping, live self-diagnostic polling (`/api/diagnostics`), and tab navigation. Created `src/components/HelpDrawer.vue` powered by PrimeVue `Drawer` featuring 4 tabs: Diagnostics (system health summary, pass/warning/fail checks, uptime/system info), Safety Policies (canary, POSIX locks, mandatory subdirs, circuit breakers, jails, risk scores), Flag Catalog (CLI flags with copy buttons and docs links), and Runbooks (step-by-step troubleshooting guides with remediation commands). Connected drawer trigger button in `AppHeader.vue` and mounted drawer in `App.vue`. Verified with `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build` and `./scripts/check.sh all` (all 7 quality gates passed).


