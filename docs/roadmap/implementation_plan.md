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
- `go test -v -count=1 ./internal/server/...`
- `./scripts/check.sh all`
- `go test -v -run TestCockpitSPASmoke ./tests/e2e/...`

