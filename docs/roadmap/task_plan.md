# Task Plan: Audit UX Polish, Deduplication Audit Trail, Explorer Lock Setting & Dedupe Safeguards

## Goal
Improve Audit Ledger filter and forensic drawer UX, record deduplication events in the immutable audit ledger, add Global Fleet Setting to unlock/lock File Explorer remote with Lucide SVG icons, fix inner folded log ordering across midnight, and implement deduplication safeguards (same-folder name deduplication vs cross-folder `--by-hash` deduplication with root warning).

## Tasks

- [x] Task 1: Audit Profile Filter Dropdown & Forensic Slide-Over Drawer Redesign
  - **Scope**: `dashboard/frontend/partials/view_audit.html`, `dashboard/frontend/js/audit.js`, `dashboard/frontend/partials/components/ui_components.html`
  - **Verification**: `./scripts/check.sh templates`

- [x] Task 2: Deduplication Audit Trail Integration
  - **Scope**: `internal/audit/taxonomy.go`, `internal/server/handlers_auth.go`, `internal/server/handlers_audit_test.go`, `dashboard/frontend/partials/view_audit.html`
  - **Verification**: `go test -count=1 ./internal/server/... -run TestHandleRemoteDedupe && go test -count=1 ./internal/server/... -run TestHandleAuditRecords`

- [x] Task 3: Global Fleet Setting for Explorer Remote Lock & Lucide SVG Icons
  - **Scope**: `internal/config/dto.go`, `internal/config/dto_test.go`, `dashboard/frontend/partials/modals/modal_global_settings.html`, `dashboard/frontend/js/settings.js`, `dashboard/frontend/partials/view_explorer.html`
  - **Verification**: `go test -count=1 ./internal/config/... && ./scripts/check.sh templates`

- [x] Task 4: Fix Inner Folded Log Ordering & Midnight Header Timestamp
  - **Scope**: `dashboard/frontend/js/logs.js`
  - **Verification**: `./scripts/check.sh loc && ./scripts/check.sh templates`

- [x] Task 5: Dedupe Safeguards: Same-Folder vs Cross-Folder Hash Option & Root Path Warning
  - **Scope**: `internal/core/client/fs_ops.go`, `internal/core/client/client_test.go`, `internal/models/api.go`, `internal/server/handlers_auth.go`, `dashboard/frontend/partials/modals/modal_dedupe.html`, `dashboard/frontend/js/remotes.js`
  - **Verification**: `go test -count=1 ./internal/core/client/... && go test -count=1 ./internal/server/...`

- [x] Task 6: Pre-flight Validation, Build & Deployment on Port 8085
  - **Scope**: Recompile `bin/bisync-hub`, redeploy to port 8085, run pre-flight `./scripts/check.sh all`, checkpoint commit.
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-10-30: Initialized task plan from approved implementation_plan.md.
- 2026-10-30: Completed all 6 tasks:
  1. Converted Audit Ledger profile filter from `<input list>` to standard `<select>` displaying human-readable profile names bound to profile IDs.
  2. Redesigned forensic slide-over drawer into structured headers: Audit Verification & Hashes (SHA-256 fingerprint, previous chain hash with GENESIS badge, actor, profile) and Structured Action Metadata (scannable key-value cards with pills + collapsible raw JSON accordion).
  3. Registered `DEDUPE_EXECUTED` in audit taxonomy and wired immutable audit ledger logging whenever deduplication runs.
  4. Added `lock_explorer_remote` setting in `config.GlobalDefaults` and Global Fleet Settings modal; replaced unicode lock characters in File Explorer with native Lucide SVG icons (`lock` and `lock-open`) plus a 1-click quick-unlock toggle.
  5. Eliminated midnight string comparison in `clusterLogEntries()`, guaranteeing cluster headers and nested folded rows strictly display newest-first timestamps.
  6. Added Deduplication Match Mode (Same-Folder Names safe default vs Cross-Folder Hash) with root scope warning and File Explorer path auto-population.
  7. Compiled `bin/bisync-hub`, deployed to port 8085, and passed all 6 quality stages in `./scripts/check.sh all`.

  3. Added user-facing guidance callouts in `modal_profiles.html` and `modal_global_settings.html` noting that changing `Compare Mode` requires a one-time `--resync`.


- 2026-10-28: Completed Task 9:
  1. Fixed unclosed `<template x-if="getEditingRemoteType() === 'drive'">` in `modal_provider_options.html` and elevated Tier 3 Audited Custom CLI Flags into a dedicated, always-visible card.
  2. Expanded compare modes across UI and backend validation to include `size,modtime` (default: everyday sync) and `size,modtime,checksum` (thorough & reliable).
  3. Fixed falsy `||` coercion in `view_telemetry.html` and `modal_profiles.html` with nullish coalescing `??` so `max_delete: 0` displays as `0%`.
  4. Added `>= 0` condition in CLI builder so `--max-delete 0` is forwarded to rclone CLI.
  5. Added `Origin` propagation in `SyncOptions`, tagging dashboard triggers as `dashboard` (emerald badge) instead of falling back to `scheduler`.

- 2026-10-28: Completed Phase 5 (Tiered Cloud Provider Architecture, Provider Options UI & Global Fleet Settings):
  1. Decoupled engine defaults in internal/config/config.go and bisync.env by removing Google Drive flags, ensuring global defaults are strictly provider-agnostic.
  2. Registered Tier 1 provider flags in compiler.CLIToRCMap (OneDrive, S3, Dropbox, Google Drive) with full connection-string path parameter translation.
  3. Extended ProfileFlags and ResolvedFlags with ProviderOptions map[string]string and implemented typed CLI argument builder with test coverage.
  4. Introduced GET/PUT /api/config handlers with runtime validation and secrets redaction.
  5. Implemented unified Target Provider Configuration panel (modal_provider_options.html) in profile modal with Tier 1 first-class controls (GDrive, OneDrive, S3, Dropbox) and audited custom flags for both Tier 1 and Tier 2 generic providers.
  6. Replaced raw .env dump in view_telemetry.html with a concise Profile Execution Contract card displaying topology, schedule, provider badge, and effective flags pill cloud.
  7. Created Global Fleet Settings modal (modal_global_settings.html, settings.js) accessible via Tier 1 header and Fleet banner.
  8. Verified all 6 stages of ./scripts/check.sh all and headless Chrome E2E smoke tests passing with zero errors.
- 2026-10-28: Initialized Phase 5 Task Plan from approved architectural design.





