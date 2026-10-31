# Task Plan: Design Specs Consolidation, README Modernization, and Notification Polish

## Goal
Consolidate UI design specifications in `docs/design/`, link them to controlling ADRs (ADR-001 through ADR-010), create `docs/design/README.md` registry, refresh root `README.md` with user-centric overview, and enhance desktop notifications with `favicon.svg` branding and profile name prefixing.

## Tasks

- [x] Task 1: Consolidate Design System Micro-Specs into `bisync-hub-design-system-specification.md` and Clean Up
  - **Scope**: `docs/design/bisync-hub-design-system-specification.md`, remove `components.md`, `layouts.md`, `patterns.md`, `status-language.md`
  - **Verification**: `test ! -f docs/design/components.md && test -f docs/design/bisync-hub-design-system-specification.md`

- [x] Task 2: Create Master Specifications Registry in `docs/design/README.md` & Add ADR Cross-Links
  - **Scope**: `docs/design/README.md`, metadata headers in `docs/design/*.md`
  - **Verification**: `test -f docs/design/README.md && ./scripts/check.sh all`

- [ ] Task 3: Refresh Root `README.md` with User-Centric Value Proposition & Remove `README_v2.md`
  - **Scope**: `README.md`, delete `README_v2.md`
  - **Verification**: `test ! -f README_v2.md && grep -q "Review. Decide. Sync." README.md`

- [ ] Task 4: Add `favicon.svg` Desktop Notification Support & Resolution
  - **Scope**: `internal/core/notifier/notifier.go`, `icons/favicon.svg`, `internal/core/notifier/notifier_test.go`
  - **Verification**: `go test -count=1 ./internal/core/notifier/...`

- [ ] Task 5: Implement Multi-Profile Attribution in Desktop Notifications
  - **Scope**: `internal/config/config.go`, `internal/app/container.go`, `internal/engine/runner/runner.go`, `internal/engine/runner/runner_handlers.go`, `internal/engine/runner/runner_test.go`
  - **Verification**: `go test -count=1 ./internal/engine/runner/... && go test -count=1 ./internal/app/...`

- [ ] Task 6: Full Pre-Flight Quality Gate & Release Readiness Validation
  - **Scope**: Full test suite and linting
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-10-31: Initialized plan from implementation_plan.md.
- 2026-10-31: Completed Task 1: Consolidated UI design system micro-specs (components, layouts, patterns, status-language) into `bisync-hub-design-system-specification.md` and pruned obsolete partials.
- 2026-10-31: Completed Task 2: Created Master Specifications Registry in `docs/design/README.md` and added metadata headers with governing ADR cross-links across all design specifications.





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





