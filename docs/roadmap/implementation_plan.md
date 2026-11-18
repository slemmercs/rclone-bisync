# Implementation Plan: Profile Manager, Dedupe Workspace, Audit Ledger Invariant, and Daemon Log Clustering

## Proposed Changes

Address 5 operational and architectural requirements:
1. **Audit Ledger Mutation Invariant & Action Filter Cleanup**: Enforce that the cryptographically verified audit ledger records `SYNC_EXECUTED` only when actual file operations/mutations occur (`FilesSynced > 0 && !dryRun`), removing empty 0-transfer records caused by dashboard/cli origin overrides. Remove non-file actions (`SECRET_ROTATED`, `CONFIG_MUTATED`) from the frontend audit filter dropdowns.
2. **Go Backend Type Resilience**: Allow `ProfileFlags.ProviderOptions` to unmarshal flexibly from arbitrary JSON types (strings, booleans, numbers) into `map[string]string` representation so frontend boolean toggles do not cause 400 Bad Request deserialization failures. Accept `delete_threshold_pct` and `max_delete_percent` interchangeably.
3. **Profile Manager Remote Selection & Option Importing**: Automatically strip trailing colons on remote names, preselect configured remotes, import tier-1 options schemas when selecting providers, and cleanly coerce form values on save.
4. **Dedupe Workspace Dual Match Modes & In-Built Explorer Navigation**: Restore separate match strategies: safe same-folder duplicate detection (default for Google Drive) vs. cross-folder cryptographic hash detection (`--by-hash`). Retain scan output and collision candidate lists in memory and `sessionStorage`. Fix "Inspect in Explorer" to navigate within the modal without dismissing it.
5. **Daemon Logs "Notices" Tab**: Add a dedicated "Notices" filter pill and counter in `ModalDaemonLogs.vue` so Rclone duplicate object notices, file transfers, and deletions stand out distinctly.

### Affected Components & Packages

- `internal/engine/runner/runner_handlers.go`: Guard `recordAuditExecuted` to require `monitorRes.FilesSynced > 0 && !dryRun`.
- `internal/engine/runner/runner_test.go`: Verify 0-transfer syncs and dry-runs do not emit `SYNC_EXECUTED` to audit ledger.
- `dashboard/frontend/src/views/ViewAudit.vue`, `dashboard/frontend/partials/view_audit.html`: Remove `SECRET_ROTATED` and `CONFIG_MUTATED` from the action `<select>` dropdown.
- `internal/profiles/model.go`, `internal/profiles/model_test.go`: Flexible unmarshaling for `ProviderOptionsMap`.
- `dashboard/frontend/src/components/ProfileManagerModal.vue`: Trailing colon normalization, schema import, boolean-to-string coercion.
- `dashboard/frontend/src/components/ModalDedupe.vue`, `dashboard/frontend/src/stores/dedupe.ts`: Match mode dropdown, scan result retention, internal explorer tab navigation.
- `dashboard/frontend/src/components/ModalDaemonLogs.vue`: Dedicated "Notices" log filter tab.

### Technical Strategy & Data Flow

- **Audit Ledger**: ADR-004 dictates that the audit trail is an append-only ledger of verified disk mutations. Runs resulting in 0 file changes, dry-runs, and configuration actions that do not alter target filesystems must not emit `SYNC_EXECUTED`.
- **Flexible Deserialization**: A custom `UnmarshalJSON` method on `ProviderOptionsMap` converts booleans (`true` -> `"true"`, `false` -> `"false"`) and numbers (`123` -> `"123"`) into strings, providing frictionless interoperability between Vue forms and Go backend flags compiler.

### Safety & Error Handling

- Concurrency and lock isolation strictly preserved.
- No live RC API calls during unit testing.
- Canary files (`RCLONE_TEST`) remain untouched.

## Verification Plan

- `go test -v -count=1 ./internal/engine/runner -run "TestSyncRunner_RunSuccessWithAuditAndLogs"`
- `go test -v -count=1 ./internal/profiles -run "TestProfileFlags_ProviderOptionsFlexibleUnmarshal"`
- `npm --prefix dashboard/frontend run build`
- `./scripts/check.sh all`
