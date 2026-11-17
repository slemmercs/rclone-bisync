# Implementation Plan: 100% Legacy Feature Parity Remediation (Alpine.js to Vue 3 Cockpit)

## Proposed Changes

Achieve complete visual and functional parity between the legacy Alpine.js cockpit templates/scripts and the modern Vue 3 + PrimeVue SPA. Based on an exhaustive side-by-side gap analysis of all 38 legacy partials/scripts against the Vue 3 source tree, the following critical capability domains require full-fidelity remediation:

### 1. Phase A: Profile Lifecycle, Dynamic Provider Options & History
- **Provider Options Engine**: Implement Tier 1 schema (`src/data/provider-options-schema.ts`) for Google Drive (`--drive-skip-shortcuts`, `--drive-use-trash`, `--drive-shared-with-me`, `--drive-skip-gdocs`, `--drive-acknowledge-abuse`, `--drive-export-formats`, `--drive-import-formats`), OneDrive (`--onedrive-no-versions`, `--onedrive-delta`, `--onedrive-chunk-size`), S3 (`--s3-storage-class`, `--s3-chunk-size`), B2 (`--b2-hard-delete`), and Dropbox (`--dropbox-batch-mode`).
- **Guided Flags Builder**: Implement `FlagsBuilder.vue` with flag catalog dropdown, boolean pills, text arguments, custom flag row adding/removal, and prohibited flags validation.
- **Synthesized Real-Time CLI Preview**: Live synthesized CLI string (`rclone bisync ...`) reflecting active configuration, flags, and provider options.
- **Profile Version History & Rollback**: History tab querying `GET /api/profiles/{id}/history`, displaying version snapshots, timestamps, diffs, and one-click rollback (`POST /api/profiles/{id}/rollback`).
- **Profile Cloning**: Clone workflow with unique ID generation and name copy (`POST /api/profiles/{id}/clone`).
- **Profile Deletion Blast Shield**: Safe confirmation modal (`ModalProfileDelete.vue` calling `DELETE /api/profiles/{id}`).

### 2. Phase B: Remote Authentication, OAuth Lifecycle & Quotas
- **Remote Reconnect & OAuth Flow**: Modal (`ModalRemoteReconnect.vue`) handling desktop/browser OAuth redirect, status spinner, auth URL copy, and cancellation (`/api/remotes/auth/oauth-stop`).
- **OAuth Migration Wizard**: Guided modal (`ModalOAuthMigration.vue`) for personal Google Drive & OneDrive Client ID/Secret configuration (`POST /api/remotes/{name}/oauth-credentials`).
- **Auth Audit Log**: Modal (`ModalAuthAudit.vue` querying `/api/remotes/auth/audit`) showing authentication events, credential updates, status badges, and error details.
- **Remote Validation Probing**: Interactive validation calling `POST /api/remotes/{name}/validate` with inline spinners and `Connected ✓` indicators.
- **App Credentials Tagging**: Badges for "Personal App" vs "Shared App ID" with migration alert banners.

### 3. Phase C: Cryptographic Audit Ledger & Compliance Cockpit
- **Cryptographic Continuity Header**: Header in `ViewAudit.vue` querying `GET /api/audit/verify` (emerald/rose shield, SHA-256 chained block count, HEAD hash snippet with copy action, verify chain trigger).
- **Ledger Reset**: Action calling `POST /api/audit/reset` with automated backup archive notice and confirmation dialog.
- **Action & Actor Filters**: Discrete dropdown filters for Action category (`SYNC_EXECUTED`, `DEDUPE_EXECUTED`, `CONFLICT_RESOLVED`, `DELETION_ACCEPTED`, `ROLLBACK_EXECUTED`, `WORKSPACE_ITEM_MOVED`, etc.) and Actor (`SYSTEM`, `USER`, `ADMIN`).
- **Forensic Block Inspection Drawer**: Modal/drawer (`ModalAuditDetail.vue`) on row click displaying previous hash, current hash, seq_id, actor, and formatted JSON payload.

### 4. Phase D: Dual Explorer Upload Dock & Transfer Queue
- **Upload Dock Modal**: Modal (`ModalUploadDock.vue` polling `GET /api/explorer/dock`).
- **Transfer Status**: Queued, uploading, completed, and failed status indicators with file sizes and destinations.
- **Sync Coordination**: "Sync in Progress (Deferred)" non-blocking lock notice and "Clear Completed" trigger (`DELETE /api/explorer/dock`).
- **Explorer Toolbar Integration**: Upload dock launcher button in `ViewExplorer.vue`.

### Affected Components & Packages

- `dashboard/frontend/src/data/`: `provider-options-schema.ts`, `flags-catalog.ts`
- `dashboard/frontend/src/types/`: `models.ts`, `explorer.ts`, `settings.ts`
- `dashboard/frontend/src/stores/`: `profiles.ts`, `remotes.ts`, `logs.ts`, `explorer.ts`
- `dashboard/frontend/src/components/`:
  - `ProfileManagerModal.vue`, `FlagsBuilder.vue`, `ModalProfileDelete.vue`
  - `ModalRemoteReconnect.vue`, `ModalOAuthMigration.vue`, `ModalAuthAudit.vue`
  - `ModalUploadDock.vue`, `ModalAuditDetail.vue`
- `dashboard/frontend/src/views/`: `ViewRemotes.vue`, `ViewAudit.vue`, `ViewExplorer.vue`, `ViewTelemetryLedger.vue`

### Technical Strategy & Data Flow

- Schema-driven provider options dynamically adapt based on remote provider backend.
- Flag builder serializes cleanly to and from POSIX-style CLI parameter strings.
- Audit ledger continuity is verified against backend SHA-256 chain verification endpoints.
- Upload dock tracks transfer queue non-blockingly during active sync runs.

### Safety & Error Handling

- Strict confirmation modals for destructive operations (profile deletion, ledger reset, remote deletion).
- Secret redaction for credentials and sensitive tokens.
- Canary file safeguards preserved across all workflows.

## Verification Plan

- `cd dashboard/frontend && npm run type-check && npm run build`

---

# Implementation Plan: Development Tooling, Rich Sandbox & Installer Build Modes

## Proposed Changes

Upgrade developer experience, installer flexibility, and test sandboxing across the repository:

### 1. Enterprise Installer (`deploy/install.sh`)
- Support flag-based frontend compilation (`--build-frontend` vs `--use-prebuilt`).
- `--build-frontend`: Verify Node.js/npm runtime, install dependencies (`npm ci` / `npm install`), execute `npm run build` as target user, and embed freshly compiled assets in Go binary.
- `--use-prebuilt` (default): Verify committed `dashboard/frontend/dist/index.html` and compile Go binary with zero external Node.js/npm dependencies.
- Clear error handling, validation, and usage help output.

### 2. Comprehensive Ephemeral Sandbox (`scripts/run-sandbox.sh`)
- Provide multi-remote configuration (`testremote`, `backup_remote`, `archive_remote`).
- Populate rich mock files across multiple formats (markdown, CSV, JSON, SVG diagrams, nested subfolders).
- Seed hash-identical duplicate files to test Deduplication by hash.
- Populate multi-profile state in `profiles.json` and historical version snapshots in `profiles_history.json`.
- Populate cryptographic SQLite `audit.db` with valid Goose migrations and verified SHA-256 hash chains (Genesis to HEAD) so Audit view is immediately green and interactive.
- Support `--build-frontend` and `--dev` flags.

### 3. All-in-One Development Toolchain (`scripts/update-tools.sh`)
- Broaden scope from Go-only tools to full project stack:
  - Go compiler (target: 1.26.11) and linters (`govulncheck`, `staticcheck`, `gosec`, `betterleaks`, `esbuild`).
  - Node.js and npm verification.
  - Frontend npm packages (`npm --prefix dashboard/frontend install`).
  - Shell script linter (`shellcheck`).
  - HTML linter (`tidy`).
  - Rclone binary (`curl https://rclone.org/install.sh | sudo bash`).
- Comprehensive `--status` diagnostic view and `--all` update mode.

### 4. Live Frontend Development Server (Vite HMR)
- Configure Vite dev server (`dashboard/frontend/vite.config.ts`) to serve `cockpit.html` on root `/` and route requests dynamically to configurable backend port (`BACKEND_PORT` / `VITE_BACKEND_PORT`).
- Provide seamless live dev script (`scripts/dev.sh` and `run-sandbox.sh --dev`) running backend + Vite HMR.

## Verification Plan
- `sudo bash deploy/install.sh --dry-run --use-prebuilt`
- `sudo bash deploy/install.sh --dry-run --build-frontend`
- `./scripts/run-sandbox.sh --dry-run`
- `./scripts/update-tools.sh --status`
- `./scripts/check.sh all`

- `go test -v -count=1 ./internal/server/...`
- `./scripts/check.sh all`
- `go test -v -run TestCockpitSPASmoke ./tests/e2e/...`


---

# Implementation Plan: Dynamic Engine Flag Optimization, Fair Scheduling Queue & Global Settings Flag Builder

## Proposed Changes

Rclone RC treats `--tpslimit`, `--tpslimit-burst`, and `--checkers` as process-global main options within `fs.Config.Main` rather than per-job parameters. To enable per-profile custom rate and checker tuning without corrupting shared token buckets, implement Compatibility Clustering (homogeneous grouping) with a unified FIFO fair execution queue and dynamic `/options/set` engine switching. Simultaneously, provide visual flag management in Global Fleet Settings.

### 1. Phase 1: Global Fleet Settings Flags Builder
- **Model & Configuration**: Add `CustomFlags string` to `GlobalDefaults` (`internal/models/api.go`, `internal/config/config.go`) so baseline CLI flags can be stored and retrieved via `/api/config`.
- **UI Integration (`ModalGlobalSettings.vue`)**: Embed `FlagsBuilder.vue` directly within `ModalGlobalSettings.vue` under a dedicated "Fleet Flags & Concurrency" tab, allowing visual configuration of baseline `--tpslimit`, `--checkers`, `--transfers`, `--buffer-size`, etc.
- **Process-Global Callout**: Clearly annotate process-global options in `FlagsBuilder.vue` with an informative badge and tooltip.

### 2. Phase 2: Rclone RC Dynamic Options Client
- **Rclone Client Methods**: Add `GetMainOptions(ctx)` (`POST /options/get`) and `SetMainOptions(ctx, opts)` (`POST /options/set`) in `internal/core/client/client.go`.
- **Mocking Support**: Extend mock RC server in tests to handle `/options/get` and `/options/set`.

### 3. Phase 3: Engine Signatures & Compatibility Clustering
- **Engine Signature Formulation**: Parse process-global flags (`--tpslimit`, `--tpslimit-burst`, `--checkers`) from profile custom flags + fleet defaults. Compute an `EngineSignature` (e.g. `SIG_DEFAULT`, `SIG_TPS4_CHK3`).
- **Compatibility Rule**: Any profiles sharing an identical `EngineSignature` are 100% compatible and execute concurrently in parallel up to the worker concurrency limit.

### 4. Phase 4: Unified FIFO Fair Queue & Engine Switcher
- **Unified Fair Queue**: Replace isolated per-worker triggers with a centralized FIFO run queue tracking `profile_id`, `opts`, `signature`, `enqueued_at`, and `priority`.
- **Opportunistic Compatibility Batching**: While the engine is active under `Signature A`, compatible jobs in the queue are dispatched immediately in parallel.
- **Starvation-Free Aging SLA**: When a queued job with a differing signature waits longer than `MaxQueueWait` (e.g. 60s), the dispatcher halts new admissions for the active signature, drains running jobs to zero, applies `POST /options/set` to switch to the waiting signature, and executes the waiting profile.
- **Baseline Restore**: When all custom-flag runs complete or the scheduler shuts down, restore the baseline engine options.

### Affected Components & Packages
- `internal/models/api.go`, `internal/config/config.go`, `internal/config/dto.go`: GlobalDefaults custom flags schema.
- `internal/core/client/client.go`: `GetMainOptions`, `SetMainOptions`.
- `internal/profiles/scheduler.go`, `internal/profiles/model.go`: Engine signature parser, fair queue, dispatcher loop.
- `dashboard/frontend/src/components/ModalGlobalSettings.vue`: Embed `FlagsBuilder.vue` and add "Fleet Flags" tab.
- `dashboard/frontend/src/data/flags-catalog.ts`, `dashboard/frontend/src/components/FlagsBuilder.vue`: Process-global badges.

### Safety & Error Handling
- Safe fallback: If `/options/set` returns an error, the scheduler logs a warning and falls back to running with current engine settings without crashing.
- Lock isolation: Process switches are protected by an advisory execution mutex.
- Non-blocking: Timers and UI sync triggers never block the HTTP request handlers.

## Verification Plan
- Unit tests:
  - `TestClient_GetSetMainOptions`: verifies RC API payload formatting.
  - `TestEngineSignature_Parsing`: verifies signature equivalence clustering.
  - `TestScheduler_FairQueueAndClustering`: verifies FIFO order, parallel execution of identical signatures, and sequential draining for divergent signatures.
- Frontend compilation: `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build`.
- Quality gates: `./scripts/check.sh all`.
- Headless Chrome E2E test verifying Global Settings flags builder interaction.


---

# Implementation Plan: Interactive Sync Queue Manager (Option A)

## Proposed Changes

Provide real-time visualization and administrative governance over the centralized FIFO sync execution queue and compatibility clustering dispatcher. Enable operators to view running cohorts and pending jobs, pause/resume dispatcher admissions, cancel individual queued syncs before execution, and purge backlogged queues in bulk.

### 1. Phase 1: Scheduler Queue Governance Engine (`internal/profiles/`)
- **Pause / Resume Dispatcher**: Add `isPaused atomic.Bool` to `Scheduler`. When paused, active jobs finish normally, but `popAdmissibleLocked` will not admit new items. Resuming notifies `queueCh` to immediately process eligible jobs.
- **Cancel Queued Job**: `CancelQueued(profileID string) error` searches `s.queue`, removes the matching entry, and sends `ErrSyncCancelled = errors.New("sync cancelled by operator")` to its `ErrCh`.
- **Clear Queue**: `ClearQueue() int` drains all pending items from `s.queue`, sends `ErrSyncCancelled` to all callers, and if no active jobs remain, restores the baseline engine options via `POST /options/set`.
- **Queue Snapshot**: `GetQueueSnapshot() QueueSnapshotDTO` returns dispatcher status (`IsPaused`, `ActiveSignature`, `MaxConcurrency`, `RunningCount`, `QueuedCount`), active executions (`[]QueueJobDTO`), and pending queued executions in FIFO order (`[]QueueJobDTO`).

### 2. Phase 2: REST API Endpoints (`internal/server/`)
- Define DTOs in `internal/models/api.go`: `QueueSnapshotResponse`, `QueueJobDTO`, `QueueActionResponse`.
- Add REST handlers in `internal/server/handlers_queue.go`:
  - `GET /api/queue`: Returns full snapshot of dispatcher state, running jobs, and queued jobs.
  - `POST /api/queue/pause`: Halts admission of queued jobs.
  - `POST /api/queue/resume`: Resumes admission of queued jobs.
  - `DELETE /api/queue/{id}`: Cancels and dequeues a specific pending profile.
  - `DELETE /api/queue`: Drains and cancels all pending jobs in bulk.
- Register routes in `internal/server/routes.go`.

### 3. Phase 3: Frontend Queue Store & Header Indicator (`dashboard/frontend/`)
- Define types in `dashboard/frontend/src/types/queue.ts`.
- Implement `useQueueStore` in `dashboard/frontend/src/stores/queue.ts` with actions: `fetchSnapshot()`, `pauseDispatcher()`, `resumeDispatcher()`, `cancelJob(id)`, `clearAll()`.
- Add reactive status pill in the top navigation header (`App.vue`):
  - Shows `Queue: N (M running)` or `Queue: Paused` with amber indicator.
  - Clicking opens `ModalQueueManager.vue`.

### 4. Phase 4: Queue Manager Modal (`ModalQueueManager.vue`)
- Component in `dashboard/frontend/src/components/ModalQueueManager.vue` using PrimeVue Dialog.
- **Dispatcher Toolbar**: Status chip (`Active: SIG_DEFAULT`, `Paused`), `Pause Queue` / `Resume Queue` toggle, `Clear All Pending` button with confirmation.
- **Active Cohort Panel**: Live card/table of running profiles with profile name, started time, elapsed timer, dry-run badge, and engine flags.
- **Queued Profiles Table**: Ordered list (#1, #2, ...) with profile name, origin (`manual`, `schedule`, `mcp`), queue wait timer, engine compatibility tag (`Compatible` vs `Requires Option Switch`), and single-click Cancel button.
- **Empty State**: Friendly visual when no jobs are active or waiting.

## Verification Plan
- Backend unit tests:
  - `TestScheduler_PauseResumeAdmissions`: Verifies no jobs admitted while paused, and immediate admission on resume.
  - `TestScheduler_CancelQueuedJob`: Verifies item removed, `ErrSyncCancelled` received on `ErrCh`.
  - `TestScheduler_ClearQueueAndRestoreBaseline`: Verifies all items drained, baseline options restored.
  - `TestHandlers_QueueAPI`: Verifies `GET /api/queue`, pause, resume, delete single, and delete all.
- Frontend build: `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build`.
- Quality gates: `./scripts/check.sh all`.
- E2E smoke tests: Headless Chrome SPA navigation and modal verification.

