# Task Plan: Phase 4 - Hybrid Fleet Overview & Operational Flags UI

## Goal
Implement the workload-centric dashboard hierarchy specified in `docs/design/hybrid_fleet_overview_contextual_profile_workspace.md` (Level 1 Fleet View, interactive profile tile grid, summary metrics banner) and the visual 3-tier operational flags editor with inheritance badges from `docs/design/bisync_operational_flags.md`.

## Tasks

- [x] Task 1: Level 1 Fleet View Component (`view_fleet.html`) & Template Registration
  - **Scope**: `dashboard/frontend/partials/view_fleet.html`, `internal/server/server.go`, `internal/server/server_test.go`
  - **Verification**: `go test -count=1 ./internal/server/...`

- [x] Task 2: Fleet Summary Counters, Filtering & Tile Action Handlers
  - **Scope**: `dashboard/frontend/js/profiles.js`, `dashboard/frontend/app.js`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh loc`

- [x] Task 3: Level 2 Header Context Switcher & Breadcrumb Navigation
  - **Scope**: `dashboard/frontend/index.html`, `dashboard/frontend/app.js`, `dashboard/frontend/js/profiles.js`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh loc`

- [x] Task 4: Visual 3-Tier Operational Flags Editor & Inheritance Badges
  - **Scope**: `dashboard/frontend/partials/modals/modal_profiles.html`, `dashboard/frontend/js/profiles.js`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh loc`

- [x] Task 5: End-to-End Quality Gate & Pre-Flight Verification
  - **Scope**: Full repository
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-10-28: Completed Phase 4 (Hybrid Fleet Overview & Operational Flags UI). Implemented Level 1 Fleet View (view_fleet.html) with real-time health summary counters, responsive tile grid, search/filter/sort, Level 2 header context switcher dropdown, breadcrumb navigation, and visual 3-tier operational flags editor with live inheritance badges and blocklist validation. Verified all 5 cockpit tabs (fleet, telemetry, remotes, explorer, audit) in headless Chrome smoke tests with zero console errors. Full pre-flight quality check passed clean (6/6 stages, 100%).
- 2026-10-28: Initialized Phase 4 Task Plan from implementation_plan.md and hybrid_fleet_overview_contextual_profile_workspace.md.
- 2026-10-27: Completed Phase 3 (Operational Flags & Profile Schema Alignment) with PR #42 merged and released in v1.0.0-beta.9.




