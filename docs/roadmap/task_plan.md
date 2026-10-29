# Task Plan: Design System Foundation, Telemetry Acceleration & Inline Log Folding

## Goal
Implement the complete Bisync Hub UI Design System, accelerate backend telemetry polling (<10ms), extract telemetry polling into `polling.js` with Dev-Mode network silence controls, implement 50-line on-demand pagination with inline log deduplication/folding, migrate all five frontend views and consolidate modals into the new design system, and author the design system documentation suite.

## Tasks

- [x] Task 1: Backend Telemetry Acceleration, Remote Quota TTL Caching & Paginated Log API
  - **Scope**: `internal/models/api.go`, `internal/services/filetail.go`, `internal/services/telemetry_service.go`, `internal/services/telemetry_remotes.go`, `internal/remoteauth/service.go`, `internal/server/handlers_telemetry.go`, `internal/server/handlers_auth.go`, `internal/server/server.go`, tests
  - **Verification**: `go test -count=1 ./internal/services/... ./internal/server/...`

- [x] Task 2: Dedicated Polling Engine (`polling.js`) with Dev-Mode Silence
  - **Scope**: `dashboard/frontend/js/polling.js`, `dashboard/frontend/app.js`, `dashboard/frontend/index.html`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh loc`

- [x] Task 3: Design System Foundations (`design-tokens.css` & Status Vocabulary)
  - **Scope**: `dashboard/frontend/css/design-tokens.css`, `dashboard/frontend/index.html`, `dashboard/frontend/app.js`, `dashboard/frontend/js/utils.js`, `dashboard/frontend/js/fleet.js`, `dashboard/frontend/js/profiles.js`, `dashboard/frontend/js/remotes.js`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh loc`

- [x] Task 4: Reusable Component Library, Patterns & Layouts
  - **Scope**: `internal/server/server.go`, `dashboard/frontend/partials/components/ui_components.html`, `dashboard/frontend/partials/patterns/ui_patterns.html`, `dashboard/frontend/partials/layouts/ui_layouts.html`
  - **Verification**: `go test -count=1 ./internal/server/... && ./scripts/check.sh templates`

- [x] Task 5: 50-Line On-Demand Pagination & Inline Log Folding
  - **Scope**: `dashboard/frontend/js/logs.js`, `dashboard/frontend/partials/patterns/pattern_timeline_ledger.html`, `dashboard/frontend/partials/modals/modal_daemon.html`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh loc`

- [x] Task 6: Migrate All Views & Consolidate Redundant Modals
  - **Scope**: `dashboard/frontend/partials/view_fleet.html`, `dashboard/frontend/partials/view_telemetry.html`, `dashboard/frontend/partials/view_explorer.html`, `dashboard/frontend/partials/view_remotes.html`, `dashboard/frontend/partials/view_audit.html`, `dashboard/frontend/partials/modals/modal_confirm.html`, `dashboard/frontend/partials/modals/modal_profile_delete.html`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh loc`

- [x] Task 7: Design System Documentation Suite & Full Quality Validation
  - **Scope**: `docs/design/status-language.md`, `docs/design/components.md`, `docs/design/patterns.md`, `docs/design/layouts.md`, `docs/design/daemon_log_deduplication_inline_folding.md`
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-10-29: Initialized plan from implementation_plan.md on branch feat/design-system-and-telemetry-perf.
- 2026-10-29: Completed Tasks 1-7:
  - Telemetry polling accelerated (<10ms) by decoupling rclone-daemon.log from GetTelemetryPayload and caching bisync.log reads by (mtime, size, limit).
  - 60s TTL caching implemented for cloud storage quotas and OAuth auth status with ?refresh=true bypass.
  - On-demand 50-line pagination implemented via GET /api/logs/bisync?page=1&limit=50.
  - Extracted telemetry polling into dashboard/frontend/js/polling.js (window.PollingModule) with ?dev=true / ?poll=false auto-pause, Alt+P shortcut, and header toggle pill.
  - Bisync Hub Design System foundations established (design-tokens.css, 7 Approved Status Vocabulary states, reusable components, operational patterns, and layout headers).
  - Inline log deduplication & folding implemented for both bisync.log and correlated daemon logs with O(N) clustering and Expand/Collapse All controls.
  - All views migrated to design system tokens and vocabulary; audit inspection converted to slide-over Entity Inspector Drawer.
  - All 8 quality stages in ./scripts/check.sh all and headless Chrome E2E smoke tests verified passing.

  1. Fixed Edit Profile cog icon and New Profile button on Fleet tab by setting `this.profileModal.show = true` and invoking `fetchRemotes()` in `startEditProfile` and `startNewProfile`.
  2. Fixed profile tiles and header pills dynamically jumping/reordering by adding deterministic sorting in `Store.List()` (sorted by `CreatedAt` then `Name`) and client-side sorting in `fetchProfiles()`.
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





