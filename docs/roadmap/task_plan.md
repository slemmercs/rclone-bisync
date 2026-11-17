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

- [x] Task 12: System Health, Fleet Settings, and MCP Inspector Modals
  - **Scope**: `src/components/ModalSystemHealth.vue`, `src/components/ModalGlobalSettings.vue`, `src/components/ModalMcp.vue`, `src/stores/settings.ts`, `src/components/AppHeader.vue`, `src/App.vue`
  - **Verification**: `./scripts/check.sh all && npm run type-check && npm run build`

- [x] Task 13: Profile Lifecycle, Dynamic Provider Options & History (Phase A)
  - **Scope**: `src/data/provider-options-schema.ts`, `src/data/flags-catalog.ts`, `src/components/FlagsBuilder.vue`, `src/components/ModalProfileDelete.vue`, `src/components/ProfileManagerModal.vue`, `src/stores/profiles.ts`
  - **Verification**: `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build && go test -v -count=1 ./internal/server/...`

- [x] Task 14: Remote Authentication, OAuth Lifecycle & Quotas (Phase B)
  - **Scope**: `src/components/ModalRemoteReconnect.vue`, `src/components/ModalOAuthMigration.vue`, `src/components/ModalAuthAudit.vue`, `src/views/ViewRemotes.vue`, `src/stores/remotes.ts`
  - **Verification**: `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build && go test -v -count=1 ./internal/server/...`

- [x] Task 15: Cryptographic Audit Ledger & Compliance Cockpit (Phase C)
  - **Scope**: `src/views/ViewAudit.vue`, `src/components/ModalAuditDetails.vue`, `src/components/ModalAuditReset.vue`, `src/stores/audit.ts`, `src/types/models.ts`
  - **Verification**: `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build && go test -v -count=1 ./internal/server/...`

- [x] Task 16: Dual Explorer Upload Dock & Transfer Queue (Phase D)
  - **Scope**: `src/components/ModalUploadDock.vue`, `src/views/ViewExplorer.vue`, `src/stores/explorer.ts`
  - **Verification**: `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build && go test -v -count=1 ./internal/server/...`

- [x] Task 17: Full Verification & E2E Validation (Phase E)
  - **Scope**: `tests/e2e/cockpit_spa_smoke_test.go`, `./scripts/check.sh all`
  - **Verification**: `./scripts/check.sh all && go test -v -run TestCockpitSPASmoke ./tests/e2e/...`

- [x] Task 18: Installer Frontend Build Modes (`deploy/install.sh`)
  - **Scope**: `deploy/install.sh`
  - **Verification**: `sudo bash deploy/install.sh --dry-run --use-prebuilt && sudo bash deploy/install.sh --dry-run --build-frontend`

- [x] Task 19: Sandbox Rich Test Data & Seeded Cryptographic Ledger (`scripts/run-sandbox.sh`)
  - **Scope**: `scripts/run-sandbox.sh`
  - **Verification**: `./scripts/run-sandbox.sh --dry-run` and sandbox launch verification

- [x] Task 20: Comprehensive Dev Toolchain Maintenance (`scripts/update-tools.sh`)
  - **Scope**: `scripts/update-tools.sh`
  - **Verification**: `./scripts/update-tools.sh --status && ./scripts/update-tools.sh --dry-run`

- [x] Task 21: Live Frontend Dev Server & Sandbox Live Integration (`dashboard/frontend/vite.config.ts`, `scripts/dev.sh`, `scripts/run-sandbox.sh`)
  - **Scope**: `dashboard/frontend/vite.config.ts`, `scripts/dev.sh`, `scripts/run-sandbox.sh`
  - **Verification**: `./scripts/check.sh all`


- [x] Task 22: Global Fleet Settings Flags Builder (`internal/models/api.go`, `internal/config/`, `dashboard/frontend/src/components/ModalGlobalSettings.vue`, `FlagsBuilder.vue`)
  - **Scope**: `internal/models/api.go`, `internal/config/config.go`, `internal/config/dto.go`, `dashboard/frontend/src/components/ModalGlobalSettings.vue`, `FlagsBuilder.vue`, `src/data/flags-catalog.ts`
  - **Verification**: `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build && go test -count=1 ./internal/server/...`

- [x] Task 23: Rclone RC Dynamic Options Client (`internal/core/client/client.go`, `client_test.go`)
  - **Scope**: `internal/core/client/client.go`, `internal/core/client/client_test.go`
  - **Verification**: `go test -count=1 -race ./internal/core/client/...`

- [x] Task 24: Engine Signature Formulation & Compatibility Clustering (`internal/profiles/engine_signature.go`, `engine_signature_test.go`)
  - **Scope**: `internal/profiles/engine_signature.go`, `internal/profiles/engine_signature_test.go`
  - **Verification**: `go test -count=1 -race ./internal/profiles/...`

- [ ] Task 25: Unified FIFO Fair Queue & Dynamic Engine Switcher (`internal/profiles/scheduler.go`, `scheduler_test.go`)
  - **Scope**: `internal/profiles/scheduler.go`, `internal/profiles/scheduler_test.go`, `internal/profiles/service.go`
  - **Verification**: `go test -count=1 -race ./internal/profiles/...`

- [ ] Task 26: Full Integration & Pre-Commit Verification
  - **Scope**: Entire repository
  - **Verification**: `./scripts/check.sh all && go test -v -count=1 -tags=e2e ./tests/e2e -run TestCockpitSPA`


## Status Updates

- 2026-11-17: Completed Task 22 - Global Fleet Settings Flags Builder. Added `CustomFlags` to `GlobalDefaults` Go model and frontend TypeScript definitions; marked process-global options (`--tpslimit`, `--tpslimit-burst`, `--checkers`) with `isProcessGlobal: true` and rendered distinct amber `[Global]` tags in `FlagsBuilder.vue`; embedded `FlagsBuilder` into `ModalGlobalSettings.vue` under a dedicated "Fleet Flags" tab for visual parameter tuning; updated server handler tests in `handlers_config_test.go` and verified with `npm run type-check`, `npm run build`, and `go test ./internal/server/...`.
- 2026-11-17: Completed Task 23 - Rclone RC Dynamic Options Client. Added `MainOptions` struct with pointer fields for zero-preserving serialization, pointer helper functions (`IntPtr`, `Float64Ptr`), and safe fallback getters; implemented `GetMainOptions` (`POST /options/get`), `SetMainOptions` (`POST /options/set`), and arbitrary `SetOptions` on `RcloneClient` in `internal/core/client/client.go`; added comprehensive test coverage in `internal/core/client/client_test.go` covering successful gets/sets, explicit zero preservation for unlimited TPS, empty payload fallbacks, malformed JSON, and server error handling. Verified with `go test -count=1 -race ./internal/core/client/...`.
- 2026-11-17: Completed Task 24 - Engine Signature Formulation & Compatibility Clustering. Implemented `ProcessGlobalFlags` (`--checkers`, `--tpslimit`, `--tpslimit-burst`), deterministic `EngineSignature` generation (`SIG_DEFAULT`, `SIG_CHK=8_TPS=4`, etc.), `IsCompatible` equivalence rules, flag inheritance/merging (`Merge`), round-trip conversions to/from `client.MainOptions`, `StripProcessGlobalFlags`, `FilterProcessGlobalTokens`, `ResolveEngineSignature`, and profile grouping via `ClusterProfiles` in `internal/profiles/engine_signature.go`. Added comprehensive test coverage in `internal/profiles/engine_signature_test.go` verifying flag parsing, space/equals formatting, explicit zero TPS handling, signature equivalence across flag permutations, token filtering, inheritance from fleet defaults, and clustering. Verified with `go test -count=1 -race ./internal/profiles/...`.



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
- 2026-11-17: Completed Task 18 - Installer Frontend Build Modes (`deploy/install.sh`). Added `-b` / `--build-frontend` and `--use-prebuilt` flags. Configured automated pre-flight validation checking Node.js and npm when fresh builds are selected, fallback to pre-built dist/index.html with zero Node dependencies, mutually exclusive flag validation, and automated frontend compilation before compiling the static Go binary. Verified dry-runs and conflict handling.

- 2026-11-16: Completed Task 12 - System Health, Fleet Settings, and MCP Inspector Modals. Implemented `useSettingsStore` in `src/stores/settings.ts` managing `/api/config`, `/api/config/encryption`, and `/api/mcp/status`. Implemented `ModalGlobalSettings.vue` with 4 operational tabs (Defaults, Engine & Polling, Alert Webhooks, and Config Encryption with passphrase management). Implemented `ModalSystemHealth.vue` featuring daemon state, engine latency, correlated daemon logs quick-launcher, MCP status, and remote connectivity probe. Implemented `ModalMcp.vue` featuring server status, ready-to-copy client configuration snippet, and full categorized catalog of all 24 MCP tools (10 read-only inspection tools, 14 guarded mutation tools with confirmation ticket badges). Integrated interactive launchers in `src/components/AppHeader.vue` (daemon health status button, MCP status pill, Settings cog button) and mounted all modals in `src/App.vue`. Verified with `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build` and `./scripts/check.sh all` (all 7 quality gates passed).

- 2026-11-17: Completed Task 13 (Phase A) - Profile Lifecycle, Dynamic Provider Options & History. Added Tier 1 provider options schema (`src/data/provider-options-schema.ts`) for Google Drive, OneDrive, S3, B2, Dropbox; added flags catalog (`src/data/flags-catalog.ts`) and safety validation against prohibited engine flags (`validateCustomFlagsString`); implemented `FlagsBuilder.vue` component with flag selection pills and reactive CLI parameter builder; implemented `ModalProfileDelete.vue` with confirmation safeguards; enhanced `src/stores/profiles.ts` with `fetchProfileHistory`, `cloneProfile`, and `rollbackProfile`; enhanced `ProfileManagerModal.vue` with version history rollback tab, clone dialog, dynamic provider options, guided flags builder, and real-time synthesized CLI preview (`rclone bisync ...`). Fixed headless Chrome SPA smoke test (`tests/e2e/cockpit_spa_smoke_test.go`). Verified with `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build && go test -v -count=1 ./internal/server/...` and `go test -v -run "TestCockpitSPA_HeadlessChromeSmoke" ./tests/e2e/...`.
- 2026-11-17: Completed Task 19 - Sandbox Rich Test Data & Seeded Cryptographic Ledger (`scripts/run-sandbox.sh`). Added CLI flags (`--help`, `--dry-run`, `--build-frontend`, `--port`, `--mcp-port`, `--rc-port`). Added native CLI subcommand `audit seed` / `audit verify` in `main.go`. Seeded 3 isolated remotes (`testremote:`, `backup_remote:`, `archive_remote:`), rich test data across multiple directories and formats (markdown, CSV, JSON, SVG, conflict candidate, cryptographic hash duplicate target), 3 sync profiles in `profiles.json`, version rollback snapshots in `profiles_history.json`, and 5 cryptographically-chained blocks in `audit.db`. Verified dry-run execution, live `/api/audit/verify` verification (HTTP 200, valid=true), and clean sandbox cleanup on termination.
- 2026-11-17: Completed Task 20 - Comprehensive Dev Toolchain Maintenance (`scripts/update-tools.sh`). Upgraded `--status` diagnostics to inspect Node.js/npm, frontend `node_modules` availability, Rclone runtime engine version, ShellCheck, HTML Tidy, Headless Chrome/Chromium browser presence, Go security tools, PATH validation, and backup retention. Added `--tools-only` (skip Go binary download for unprivileged dev tool/npm maintenance) and `--install-frontend` flags. Tested and verified `--help`, `--status`, and `--tools-only --dry-run`.

- 2026-11-17: Completed Task 21 - Live Frontend Dev Server & Sandbox Live Integration (`dashboard/frontend/vite.config.ts`, `scripts/dev.sh`, `scripts/run-sandbox.sh`). Configured Vite reverse proxy for `/api` with support for `VITE_BACKEND_URL` and `VITE_PORT` plus dev middleware rewriting `/` and `/index.html` to `/cockpit.html`. Created dedicated `scripts/dev.sh` launcher supporting direct Vite dev mode and automated `--sandbox` launching. Upgraded `scripts/run-sandbox.sh` with `--dev` and `--vite-port` options to run Vite HMR concurrently with clean trap termination. Extracted audit CLI handling to `internal/audit/cli.go` to strictly adhere to backend LOC guidelines and eliminated gosec taint warnings. Verified with `./scripts/check.sh all` (7/7 passed in 46s) and Headless Chrome E2E SPA smoke suite.



- 2026-11-17: Completed Task 14 (Phase B) - Remote Authentication, OAuth Lifecycle & Quotas. Added remote auth models (`RemoteAuthInfo`, `AuthHealthSummary`, `OAuthAuditEvent`, `ReconnectResponse`, `ValidateResponse`) in `src/types/models.ts`. Extended `useRemotesStore` in `src/stores/remotes.ts` with parallel `api.get('/api/remotes')` and `api.get('/api/remotes/auth')` fetching, reactive validating/validated states, `validateRemote(name)`, `reconnectRemote(name)`, `cancelReconnect(name)`, `stopOAuth()`, `openMigrationWizard(remote)`, `saveMigrationCredentials()`, and `viewAuthAudit()`. Implemented `ModalRemoteReconnect.vue` with status icons, fallback clickable browser authorization URLs, and cancellation. Implemented `ModalOAuthMigration.vue` with guidance callouts, collapsible step-by-step setup guides for Google Drive and Microsoft OneDrive, secret visibility toggle, and client validation. Implemented `ModalAuthAudit.vue` displaying live historical authentication audit logs. Enhanced `ViewRemotes.vue` with multi-segment storage quota bars and breakdown metrics (Used, Free, Trash), migration-required warning banners, inline connection testing (`Test` / `Connected ✓`), OAuth reconnect buttons, and toolbar audit launcher. Verified with `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build && go test -v -count=1 ./internal/server/...` and headless Chrome smoke test.

- 2026-11-17: Completed Task 15 (Phase C) - Cryptographic Audit Ledger & Compliance Cockpit. Added audit models (`AuditRecord`, `AuditVerification`, `AuditRecordsResponse`, `AuditResetResponse`) in `src/types/models.ts`. Implemented dedicated Pinia store `src/stores/audit.ts` managing real server-side paginated queries, live cryptographic chain verification (`/api/audit/verify`), head hash copying, and destructive reset with automated backup archive (`/api/audit/reset`). Implemented `ModalAuditReset.vue` safety modal for ledger archiving and genesis reinitialization. Implemented `ModalAuditDetails.vue` for inspecting individual audit blocks with SHA-256 hash chaining verification, actor/profile metadata, and structured key-value vs raw JSON payload viewer. Completely re-engineered `ViewAudit.vue` with cryptographic continuity banner, HEAD hash badge, Verify Now trigger, Reset Ledger trigger, JSONL/CSV download links, 3-tier filter controls (Profile, Action, Actor, free-text search), hash chaining visualization (`hash[0:10]... ← prev_hash[0:6]...`), and paginated ledger table. Verified with `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build && go test -v -count=1 ./internal/server/...` and headless Chrome SPA smoke test.

- 2026-11-17: Completed Task 16 (Phase D) - Dual Explorer Upload Dock & Transfer Queue. Added `UploadQueueItem` and `UploadDockStatus` models to `src/types/explorer.ts`. Extended `useExplorerStore` in `src/stores/explorer.ts` with `uploadDock` state, `uploadDockModalOpen`, `fetchUploadDockStatus(profileId)`, `clearCompletedUploads(profileId)`, `removeDockItem(id, profileId)`, `openUploadDockModal(profileId)`, and `closeUploadDockModal()`. Implemented `ModalUploadDock.vue` modal displaying real-time upload queue items, file size formatting, status pills (`queued`, `waiting_for_sync`, `uploading`, `completed`, `failed`), sync-deferred alert, per-item deletion, and clear completed action. Integrated `Uploads (N)` launcher button with deferred badge in `ViewExplorer.vue` toolbar, mounted `ModalUploadDock.vue`, and synced upload dock status across profile changes and file uploads. Added `UploadDockModalInteraction` headless chrome test in `tests/e2e/cockpit_spa_smoke_test.go`. Verified with `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build && go test -v -count=1 ./internal/server/...` and `go test -v -run TestCockpitSPA_HeadlessChromeSmoke ./tests/e2e/...`.

- 2026-11-17: Completed Task 17 (Phase E) - Full Verification & E2E Validation. Executed full repository quality gate `./scripts/check.sh all` passing all 7/7 verification stages (Go formatting & shell check, Alpine template safety, LOC audit, license compliance & copyleft guard, static analysis & gosec, unit tests & coverage >80%, and multi-platform build verification). Validated end-to-end headless Chrome browser suite in `tests/e2e/cockpit_spa_smoke_test.go` covering SPA mount, header controls, all 5 core tabs (Fleet, Engine & Telemetry, Remotes, Dual Explorer, Audit Trail), Log Dock, Help Drawer, Global Settings, System Health, MCP Server, and Upload Dock with 0 console errors and clean shutdown. Completed 100% feature parity migration across Phases A through E.



