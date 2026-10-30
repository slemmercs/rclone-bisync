# Task Plan: Architecture Decision Records Review & Registration

## Goal
Review implemented features across releases v1.0.0-beta.1 through v1.0.0-beta.12, update ADR-004 to capture execution log decoupling, and create missing ADRs (ADR-006 through ADR-010) following the ADR-000 template.

## Tasks

- [x] Task 1: Update ADR-004 with Decoupled Execution Log Streaming & Bounded Schema
  - **Scope**: `docs/architecture/adr/ADR-004-Immutable-Audit-Ledger.md`
  - **Verification**: `./scripts/check.sh all`

- [x] Task 2: Create ADR-006: Multi-Layered Safety Circuit Breakers & Path Confinement
  - **Scope**: `docs/architecture/adr/ADR-006-Multi-Layered-Safety-Circuit-Breakers.md`
  - **Verification**: `test -f docs/architecture/adr/ADR-006-Multi-Layered-Safety-Circuit-Breakers.md`

- [x] Task 3: Create ADR-007: 3-Tier Cascading Operational Flags Hierarchy
  - **Scope**: `docs/architecture/adr/ADR-007-Cascading-Operational-Flags-Hierarchy.md`
  - **Verification**: `test -f docs/architecture/adr/ADR-007-Cascading-Operational-Flags-Hierarchy.md`

- [x] Task 4: Create ADR-008: Write-Ahead Log (WAL) Resilient Interactive Conflict Review
  - **Scope**: `docs/architecture/adr/ADR-008-WAL-Resilient-Interactive-Conflict-Review.md`
  - **Verification**: `test -f docs/architecture/adr/ADR-008-WAL-Resilient-Interactive-Conflict-Review.md`

- [ ] Task 5: Create ADR-009: Remote Hash Deduplication with Provider Safeguards and Baseline Resynchronization
  - **Scope**: `docs/architecture/adr/ADR-009-Remote-Deduplication-With-Provider-Safeguards.md`
  - **Verification**: `test -f docs/architecture/adr/ADR-009-Remote-Deduplication-With-Provider-Safeguards.md`

- [ ] Task 6: Create ADR-010: Zero-Build Component Design System & Responsive Telemetry Architecture
  - **Scope**: `docs/architecture/adr/ADR-010-Zero-Build-Design-System-And-Responsive-Telemetry.md`
  - **Verification**: `test -f docs/architecture/adr/ADR-010-Zero-Build-Design-System-And-Responsive-Telemetry.md`

- [ ] Task 7: Update ADR Registry Table in docs/architecture/adr/README.md & Full Pre-Flight Validation
  - **Scope**: `docs/architecture/adr/README.md`
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-10-30: Initialized plan from implementation_plan.md.

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





