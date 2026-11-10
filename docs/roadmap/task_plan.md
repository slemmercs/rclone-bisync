# Task Plan: Bisync Hub Cockpit UI/UX Consolidation

## Goal
Consolidate Bisync Hub cockpit UI/UX across top navigation, fleet onboarding, remote creation wizard, and dual-pane file explorer to eliminate visual clutter, remove dangerous/ambiguous nomenclature, and streamline operator workflows with Lucide SVG icons and sub-400 LOC modularity.

## Tasks

- [x] Task 1: Header Quick-Create Menu, Consolidated System Health Chip & Unified System Health Modal
  - **Scope**: `dashboard/frontend/partials/modals/modal_system_health.html`, `dashboard/frontend/js/system_health.js`, `dashboard/frontend/index.html`, `dashboard/frontend/app.js`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh loc 400 && go test -count=1 ./tests/e2e/... -run TestDashboard_HeadlessChromeSmoke`

- [x] Task 2: Fleet Overview Header Cleanup, Status Metric Filters & 2-Step Onboarding Hero Card
  - **Scope**: `dashboard/frontend/partials/view_fleet.html`, `dashboard/frontend/js/fleet.js`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh loc 400 && go test -count=1 ./tests/e2e/... -run TestDashboard_HeadlessChromeSmoke`

- [ ] Task 3: Remote Creation Wizard Usability Overhaul (Tier-1 Spotlight, Clickable Steppers, OAuth/Custom Credential Toggle, Unstickable Reset)
  - **Scope**: `dashboard/frontend/partials/modals/modal_create_remote.html`, `dashboard/frontend/js/remotes_wizard.js`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh loc 400 && go test -count=1 ./tests/e2e/... -run TestDashboard_HeadlessChromeSmoke`

- [ ] Task 4: Storage Remotes Control Bar Consolidation
  - **Scope**: `dashboard/frontend/partials/view_remotes.html`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh loc 400 && go test -count=1 ./tests/e2e/... -run TestDashboard_HeadlessChromeSmoke`

- [ ] Task 5: Dual-Pane Directory Explorer Overhaul (Clear Action Labels, Single-Line Pane Header, Directional Action Bar, RCLONE_TEST Visual Guard)
  - **Scope**: `dashboard/frontend/partials/view_explorer.html`, `dashboard/frontend/partials/components/component_explorer_toolbar.html`, `dashboard/frontend/js/explorer_actions.js`, `dashboard/frontend/js/explorer.js`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh loc 400 && go test -count=1 ./tests/e2e/... -run TestDashboard_HeadlessChromeSmoke`

- [ ] Task 6: E2E Headless Browser Journeys, Quality Gate & Documentation
  - **Scope**: `tests/e2e/dashboard_smoke_test.go`, `tests/e2e/sandbox_ui_journey_test.go`
  - **Verification**: `./scripts/check.sh all && go test -count=1 ./tests/e2e/...`

## Status Updates

- 2026-11-10: Initialized plan from implementation_plan.md.



- 2026-11-10: Task 1 verified: Implemented global `[ + Create ▾ ]` quick-create menu, consolidated System Health Chip `[ 🟢 Live 2ms • 🟣 MCP • ⚪ Idle ]`, and multi-tab unified System Health Modal (Daemon, MCP Server, and Scheduler & Poller). Registered `SystemHealthModule` and global keyboard shortcuts. Verified via templates, LOC audit, and Headless Chrome smoke test suite.

- 2026-11-10: Task 2 verified: Streamlined Fleet Overview header action bar by removing redundant Help/Settings controls, adding Connect Remote action, implementing 2-step first-launch onboarding hero card with live remote readiness status, and scoping search/empty states cleanly. Verified via templates, LOC audit, and Headless Chrome smoke test suite.

