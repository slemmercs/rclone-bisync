# Task Plan: Profile Manager, Dedupe Workspace, Audit Ledger Invariant, and Daemon Log Clustering

## Goal
Resolve Profile Manager save failures and option importing, restore Dedupe modal dual match modes and in-built explorer navigation, fix queue 404s from stale daemon processes, add daemon log Notices filtering, and enforce that the audit ledger records executions only when actual file operations occur.

## Tasks

- [x] Task 1: Audit Ledger Mutation Invariant & Clean Action Filter Dropdown
  - **Scope**: `internal/engine/runner/runner_handlers.go`, `internal/engine/runner/runner_test.go`, `dashboard/frontend/src/views/ViewAudit.vue`, `dashboard/frontend/partials/view_audit.html`
  - **Verification**: `go test -v -count=1 ./internal/engine/runner -run "TestSyncRunner_AuditExecuted_And_Aborted"`

- [x] Task 2: Go Backend Flexible Unmarshaling (`ProviderOptionsMap` & `SafetySettings`)
  - **Scope**: `internal/profiles/model.go`, `internal/profiles/model_test.go`
  - **Verification**: `go test -v -count=1 ./internal/profiles -run "TestProfileFlags_ProviderOptionsFlexibleUnmarshal"`

- [x] Task 3: Profile Manager Form Binding, Remote Preselection & Option Importing
  - **Scope**: `dashboard/frontend/src/components/ProfileManagerModal.vue`
  - **Verification**: `npm --prefix dashboard/frontend run build && go test -v -count=1 ./internal/server -run "TestProfiles_CRUD_Endpoints"`

- [x] Task 4: Dedupe Modal Dual Match Modes & Backend-Aware Strategy Differentiation
  - **Scope**: `dashboard/frontend/src/components/ModalDedupe.vue`, `dashboard/frontend/src/stores/dedupe.ts`
  - **Verification**: `npm --prefix dashboard/frontend run build`

- [x] Task 5: Dedupe Scan Result Caching (`sessionStorage` & In-Memory Retention)
  - **Scope**: `dashboard/frontend/src/stores/dedupe.ts`, `dashboard/frontend/src/components/ModalDedupe.vue`
  - **Verification**: `npm --prefix dashboard/frontend run build`

- [x] Task 6: In-Built Dedupe Explorer Navigation
  - **Scope**: `dashboard/frontend/src/components/ModalDedupe.vue`
  - **Verification**: `npm --prefix dashboard/frontend run build`

- [x] Task 7: Dedicated "Notices" Tab in Daemon Logs
  - **Scope**: `dashboard/frontend/src/components/ModalDaemonLogs.vue`
  - **Verification**: `npm --prefix dashboard/frontend run build`

- [ ] Task 8: End-to-End Suite Verification, Single Binary Build & Service Restart
  - **Scope**: `tests/e2e/cockpit_spa_smoke_test.go`, `/home/csaba/rclone-bisync/bin/rclone-bisync`
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-11-18: Initialized plan from implementation_plan.md.
- 2026-11-18: Completed Task 1 - Enforced audit ledger invariant in `runner_handlers.go` requiring `!dryRun && (FilesSynced > 0 || resync)` before appending `SYNC_EXECUTED` records. Suppressed 0-transfer dashboard/CLI runs and dry-runs from writing empty blocks to `audit.db`. Removed non-file actions `SECRET_ROTATED` and `CONFIG_MUTATED` from `ViewAudit.vue` and `view_audit.html` action filter dropdowns. Verified via `TestSyncRunner_AuditExecuted_And_Aborted`.
- 2026-11-18: Completed Task 2 - Implemented `ProviderOptionsMap` custom unmarshaler in `internal/profiles/model.go` to coerce booleans, numbers, and strings into strings. Implemented flexible `SafetySettings.UnmarshalJSON` accepting both `delete_threshold_pct` and `max_delete_percent` (as float, int, or percentage string) and string/bool safety thresholds. Verified via `TestProfileFlags_ProviderOptionsFlexibleUnmarshal` and `TestSafetySettings_FlexibleUnmarshal`.
- 2026-11-18: Completed Task 3 - Updated `ProfileManagerModal.vue` with `normalizedRemotes` (stripping trailing colons), auto-fetching remotes upon modal open, auto-populating provider options schema defaults on remote selection, normalizing `delete_threshold_pct` / `max_delete_percent`, and coercing provider options to strings on profile save. Verified via `npm --prefix dashboard/frontend run build` and `TestProfiles_CRUD_Endpoints`.
- 2026-11-18: Completed Task 4 - Implemented dual match modes in `ModalDedupe.vue` ("Same-Folder Duplicate Names (Safe Default)" vs "Cross-Folder Hash (--by-hash)") with dynamic badge indicator. Implemented provider-aware strategy differentiation in `stores/dedupe.ts` defaulting to `--by-hash` for single-filename backends (OneDrive, S3, B2, local, Swift, AzureBlob, SFTP, FTP) and same-folder matching for Google Drive. Verified via `npm --prefix dashboard/frontend run build` and `internal/server` tests.
- 2026-11-18: Completed Task 5 - Implemented dedupe scan result caching in `stores/dedupe.ts` using `sessionStorage` and reactive memory state with cache key scoping (`remote` + `path`). Scans are restored upon opening the modal, an informative cached badge with relative time and clear button is shown in `ModalDedupe.vue`, and cache is invalidated upon successful batch execution or explicit cache clear. Verified via `npm --prefix dashboard/frontend run build` and `TestRemoteDedupe`.
- 2026-11-18: Completed Task 6 - Fixed "Inspect in Explorer" in `ModalDedupe.vue` to compute the target parent folder and invoke `dedupeStore.loadExplorer(dirPath)` directly into the in-modal `DedupeExplorerTab.vue` without dismissing the dedupe modal or discarding candidate decisions. Retained an explicit external-link button (`jumpToMainExplorer`) for users wanting to transition out into the root Dual-Pane File Explorer. Verified via `npm --prefix dashboard/frontend run build` and `internal/server` tests.
- 2026-11-18: Completed Task 7 - Added dedicated "Notices" log filter tab and counter pill in `ModalDaemonLogs.vue` (and aligned `modal_daemon.html` / `logs_daemon.js`) to isolate Rclone duplicate object notices, file transfers, and deletions from standard Info logs. Added filter reset helper in empty state. Verified via `npm --prefix dashboard/frontend run build` and server unit tests.
