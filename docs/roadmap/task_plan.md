# Task Plan: Deduplication Output UX, OneDrive Backend Safeguards & Restored Item Visibility

## Goal
Improve deduplication output formatting across the Audit Ledger drawer and the Deduplication modal, add OneDrive-specific deduplication safeguards (auto-detecting name-vs-hash support and excluding locked Personal Vault), and expose concrete file/folder paths on expanded `DELETION_RESTORED` timeline log rows.

## Tasks

- [x] Task 1: Backend Enrichment for Audit Log Object & OneDrive Personal Vault Exclusion
  - **Scope**: `internal/models/api.go`, `internal/services/audit_service.go`, `internal/core/client/fs_ops.go`, `internal/core/client/client_test.go`, `internal/server/handlers_auth.go`, `internal/server/handlers_audit_test.go`
  - **Verification**: `go test -count=1 ./internal/core/client/... && go test -count=1 ./internal/services/... && go test -count=1 ./internal/server/...`

- [x] Task 2: Dedupe Output Parser & Remote-Specific Mode Selector
  - **Scope**: `dashboard/frontend/js/utils.js`, `dashboard/frontend/js/remotes.js`
  - **Verification**: `./scripts/check.sh loc && ./scripts/check.sh templates`

- [x] Task 3: Dedupe Modal Title, Remote Type Guidance & Structured Execution Output
  - **Scope**: `dashboard/frontend/partials/modals/modal_dedupe.html`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh loc`

- [x] Task 4: Structured Dedupe Rendering in Audit Slide-Over Drawer
  - **Scope**: `dashboard/frontend/partials/components/ui_components.html`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh loc`

- [x] Task 5: Expanded Log Line Item Path / Object Visibility in Timeline Ledger
  - **Scope**: `dashboard/frontend/partials/patterns/pattern_timeline_ledger.html`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh loc`

- [ ] Task 6: Pre-flight Validation, Build & Deployment on Port 8085
  - **Scope**: Recompile `bin/bisync-hub`, deploy to port 8085, run pre-flight `./scripts/check.sh all`, checkpoint commit.
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-10-30: Completed Task 5:
  - Added object path badge (`subEntry.object` / `cluster.entry.object`) to single entry rows and nested cluster rows in `dashboard/frontend/partials/patterns/pattern_timeline_ledger.html`.
  - Added action badge (`subEntry.action`) and contextual direct navigation button (`Ledger →` for audit records, `Daemon` for daemon logs) in expanded cluster rows.
  - Verified with `./scripts/check.sh templates && ./scripts/check.sh loc`.
- 2026-10-30: Completed Task 4:
  - Added dedicated Deduplication Run Summary card in `dashboard/frontend/partials/components/ui_components.html` (`component_entity_drawer`) displaying collisions detected, duplicates removed, and direct "View in Correlated Logs" link button.
  - Added deep linking button to correlated daemon logs whenever `run_id` or `job_id` appears in structured action metadata.
  - Protected against massive stdout dumps in legacy records by displaying `output` inside a bounded, scrollable monospace container.
  - Verified with `./scripts/check.sh templates && ./scripts/check.sh loc`.
- 2026-10-30: Completed Task 3:
  - Updated `dashboard/frontend/partials/modals/modal_dedupe.html` modal title to "Remote Deduplication" with subtitle clarifying file & directory collisions.
  - Added OneDrive backend guidance notice explaining duplicate name constraints and default Cross-Folder Hash mode.
  - Added structured cards view displaying collisions found and removed, notice/error alerts, and toggle to raw console.
  - Added deep link button to jump straight to correlated daemon run logs when `dedupeModal.runId` is returned.
  - Verified with `./scripts/check.sh templates && ./scripts/check.sh loc`.
- 2026-10-30: Completed Task 2:
  - Added `parseDedupeOutput()` in `dashboard/frontend/js/utils.js` extracting collision counts, deleted counts, error messages, and per-item duplication statistics.
  - Added auto-detection in `openDedupeModal()` (`dashboard/frontend/js/remotes.js`) defaulting OneDrive remotes to `--by-hash` mode and tracking `runId` from backend responses with cross-link navigation support.
  - Streamlined `dashboard/frontend/js/remotes.js` to 386 LOC adhering to the 400 LOC ceiling.
  - Verified with `./scripts/check.sh loc && ./scripts/check.sh templates`.
- 2026-10-30: Completed Task 1:
  - Enriched `AuditLogRecord` with `Object string` and wired `RecordEvent()` to extract `item_path`/`path`/`object`/`file` for timeline logs.
  - Added OneDrive `--exclude "Personal Vault/**"` safeguard to `Dedupe()` in `internal/core/client/fs_ops.go` with unit test verification.
  - Decoupled audit decisions from log output in `internal/server/handlers_auth.go`: generated unique execution `run_id`, recorded clean structured event in `audit_ledger.db` without storing raw multiline output strings, streamed individual dedupe output lines directly to `bisync.log` via `s.AuditService.Logger.RecordSyncRun()`, and exposed `run_id` in `models.DedupeResponse`.
  - Verified with `go test -count=1 ./internal/core/client/... && go test -count=1 ./internal/services/... && go test -count=1 ./internal/server/...`.
- 2026-10-30: Initialized task plan for Deduplication Output UX, OneDrive Backend Safeguards & Restored Item Visibility.


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





