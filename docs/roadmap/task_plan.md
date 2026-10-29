# Task Plan: UI Polish, Network Silence, Historical Deletion Bug Fix, Hash Deduplication & MCP Roadmap

## Goal
Consolidate UI/UX fixes across modals and logs, optimize network polling, eliminate the historical log scraper bug surfacing non-existent pending deletions, implement cloud remote hash deduplication (`rclone dedupe --by-hash`), and add Model Context Protocol (MCP) server support to the roadmap.

## Tasks

- [x] Task 1: Fix Historical Deletion Scraper Bug in Deletion Service
  - **Scope**: `internal/services/deletion_service.go`, `internal/services/deletion_test.go`, `dashboard/frontend/partials/modals/modal_confirm.html`
  - **Verification**: `go test -count=1 ./internal/services/... -run TestDeletion`

- [x] Task 2: Network Silence & Polling Optimization
  - **Scope**: `dashboard/frontend/js/polling.js`, `dashboard/frontend/app.js`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh loc`

- [x] Task 3: Correlated Daemon Log Modal & Alpine Fixes
  - **Scope**: `dashboard/frontend/partials/modals/modal_daemon.html`
  - **Verification**: `./scripts/check.sh templates`

- [x] Task 4: Reverse Unified Ledger & Folding Timestamp Update
  - **Scope**: `dashboard/frontend/js/logs.js`, `dashboard/frontend/partials/patterns/pattern_timeline_ledger.html`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh loc`

- [x] Task 5: Slide-Over Drawer Stacking Context Fix
  - **Scope**: `dashboard/frontend/css/design-tokens.css`, `dashboard/frontend/partials/components/ui_components.html`
  - **Verification**: `./scripts/check.sh templates`

- [x] Task 6: Storage Remotes Scoped Quota & File Explorer Isolation
  - **Scope**: `internal/server/handlers_auth.go`, `dashboard/frontend/partials/view_explorer.html`, `dashboard/frontend/partials/modals/modal_confirm.html`
  - **Verification**: `go test -count=1 ./internal/server/... && ./scripts/check.sh templates`

- [x] Task 7: Remote Deduplication by Hash & MCP Server Roadmap Specifications
  - **Scope**: `dashboard/frontend/partials/view_remotes.html`, `dashboard/frontend/partials/modals/modal_dedupe.html`, `dashboard/frontend/js/remotes.js`, `docs/roadmap/ROADMAP.md`, `docs/roadmap/backlog.md`
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-10-29: Initialized task plan from approved implementation_plan.md.
- 2026-10-29: Completed all 7 tasks:
  1. Remedied historical log scraper bug in `deletion_service.go` by verifying active circuit breaker aborts and resetting upon sync success, preventing 153 completed deletions from surfacing as pending.
  2. Optimized polling loop in `polling.js` to single lightweight `/api/telemetry?limit=0` query with adaptive intervals (8s idle / 2s sync) and conditional review/profile fetches.
  3. Fixed `modal_daemon.html` Alpine reference errors, restored comparison settings grid, added Error/Warning/Info filter counts, and hidden Go source line noise unless in dev mode.
  4. Reversed unified ledger order (newest on top) and updated clusterLogEntries to use latest timestamp for folded rows.
  5. Corrected drawer stacking context (`overlay: z-40`, `panel: z-50`) to eliminate backdrop blur over the panel.
  6. Added scoped remote quota validation (`POST /api/remotes/auth/check?remote=...`), locked Explorer remote dropdown to active profile, excluded `.review_backups` from Explorer listings, and added visual `[LOCAL]` ➔ `[REMOTE]` transfer preview to directory copy confirmation.
  7. Implemented native Remote Hash Deduplication (`modal_dedupe.html`, `/api/remotes/dedupe`) with dry-run preview, and formalized Model Context Protocol (MCP) AI server specifications in `ROADMAP.md` and `backlog.md`.


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





