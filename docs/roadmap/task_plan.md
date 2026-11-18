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

- [ ] Task 3: Profile Manager Form Binding, Remote Preselection & Option Importing
  - **Scope**: `dashboard/frontend/src/components/ProfileManagerModal.vue`
  - **Verification**: `npm --prefix dashboard/frontend run build && go test -v -count=1 ./internal/server -run "TestProfiles_CRUD_Endpoints"`

- [ ] Task 4: Dedupe Modal Dual Match Modes & Backend-Aware Strategy Differentiation
  - **Scope**: `dashboard/frontend/src/components/ModalDedupe.vue`, `dashboard/frontend/src/stores/dedupe.ts`
  - **Verification**: `npm --prefix dashboard/frontend run build`

- [ ] Task 5: Dedupe Scan Result Caching (`sessionStorage` & In-Memory Retention)
  - **Scope**: `dashboard/frontend/src/stores/dedupe.ts`, `dashboard/frontend/src/components/ModalDedupe.vue`
  - **Verification**: `npm --prefix dashboard/frontend run build`

- [ ] Task 6: In-Built Dedupe Explorer Navigation
  - **Scope**: `dashboard/frontend/src/components/ModalDedupe.vue`
  - **Verification**: `npm --prefix dashboard/frontend run build`

- [ ] Task 7: Dedicated "Notices" Tab in Daemon Logs
  - **Scope**: `dashboard/frontend/src/components/ModalDaemonLogs.vue`
  - **Verification**: `npm --prefix dashboard/frontend run build`

- [ ] Task 8: End-to-End Suite Verification, Single Binary Build & Service Restart
  - **Scope**: `tests/e2e/cockpit_spa_smoke_test.go`, `/home/csaba/rclone-bisync/bin/rclone-bisync`
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-11-18: Initialized plan from implementation_plan.md.
- 2026-11-18: Completed Task 1 - Enforced audit ledger invariant in `runner_handlers.go` requiring `!dryRun && (FilesSynced > 0 || resync)` before appending `SYNC_EXECUTED` records. Suppressed 0-transfer dashboard/CLI runs and dry-runs from writing empty blocks to `audit.db`. Removed non-file actions `SECRET_ROTATED` and `CONFIG_MUTATED` from `ViewAudit.vue` and `view_audit.html` action filter dropdowns. Verified via `TestSyncRunner_AuditExecuted_And_Aborted`.
- 2026-11-18: Completed Task 2 - Implemented `ProviderOptionsMap` custom unmarshaler in `internal/profiles/model.go` to coerce booleans, numbers, and strings into strings. Implemented flexible `SafetySettings.UnmarshalJSON` accepting both `delete_threshold_pct` and `max_delete_percent` (as float, int, or percentage string) and string/bool safety thresholds. Verified via `TestProfileFlags_ProviderOptionsFlexibleUnmarshal` and `TestSafetySettings_FlexibleUnmarshal`.
