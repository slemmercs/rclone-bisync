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

- [x] Task 3: Remote Creation Wizard Usability Overhaul (Tier-1 Spotlight, Clickable Steppers, OAuth/Custom Credential Toggle, Unstickable Reset)
  - **Scope**: `dashboard/frontend/partials/modals/modal_create_remote.html`, `dashboard/frontend/js/remotes_wizard.js`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh loc 400 && go test -count=1 ./tests/e2e/... -run TestDashboard_HeadlessChromeSmoke`

- [x] Task 4: Storage Remotes Control Bar Consolidation
  - **Scope**: `dashboard/frontend/partials/view_remotes.html`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh loc 400 && go test -count=1 ./tests/e2e/... -run TestDashboard_HeadlessChromeSmoke`

- [x] Task 5: Dual-Pane Directory Explorer Overhaul (Clear Action Labels, Single-Line Pane Header, Directional Action Bar, RCLONE_TEST Visual Guard)
  - **Scope**: `dashboard/frontend/partials/view_explorer.html`, `dashboard/frontend/partials/components/component_explorer_toolbar.html`, `dashboard/frontend/js/explorer_actions.js`, `dashboard/frontend/js/explorer.js`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh loc 400 && go test -count=1 ./tests/e2e/... -run TestDashboard_HeadlessChromeSmoke`

- [x] Task 6: E2E Headless Browser Journeys, Quality Gate & Documentation
  - **Scope**: `tests/e2e/dashboard_smoke_test.go`, `tests/e2e/sandbox_ui_journey_test.go`
  - **Verification**: `./scripts/check.sh all && go test -count=1 ./tests/e2e/...`

## Status Updates

- 2026-11-10: Initialized plan from implementation_plan.md.



- 2026-11-10: Task 1 verified: Implemented global `[ + Create ▾ ]` quick-create menu, consolidated System Health Chip `[ 🟢 Live 2ms • 🟣 MCP • ⚪ Idle ]`, and multi-tab unified System Health Modal (Daemon, MCP Server, and Scheduler & Poller). Registered `SystemHealthModule` and global keyboard shortcuts. Verified via templates, LOC audit, and Headless Chrome smoke test suite.

- 2026-11-10: Task 2 verified: Streamlined Fleet Overview header action bar by removing redundant Help/Settings controls, adding Connect Remote action, implementing 2-step first-launch onboarding hero card with live remote readiness status, and scoping search/empty states cleanly. Verified via templates, LOC audit, and Headless Chrome smoke test suite.

- 2026-11-10: Task 3 verified: Implemented Tier-1 curated cloud provider spotlight (Google Drive, OneDrive, S3, Dropbox, B2), clickable breadcrumb steppers with validation gates, segmented authentication toggling between Browser OAuth and Custom Client Credentials, and resilient unstickable error states with retry/modify paths. Verified via templates, LOC audit, and Headless Chrome smoke test suite.


- 2026-11-10: Task 4 verified: Unified Storage Remotes control bar into a single-line control row merging remote count indicators, profile scoping toggle, Add Remote trigger, and Quota refresh. Removed fragmented vertical banners. Verified via templates, LOC audit, and Headless Chrome smoke test suite.

- 2026-11-10: Task 5 verified: Overhauled Dual-Pane Directory Explorer by replacing ambiguous "Sync" labels with explicit directional transfers ("Transfer to Remote →" / "Transfer to Local ←"), collapsed multi-bar pane headers into a single high-density header with embedded interactive breadcrumbs and quick actions, integrated search directly into column sort headers, and safeguarded RCLONE_TEST canary rows with visual lock icons, CANARY badges, drag-prevention, and action restrictions. Verified via templates, LOC audit, and Headless Chrome smoke test suite.
- 2026-11-10: Task 6 verified: Added headless Chrome E2E verification assertions in `dashboard_smoke_test.go` covering top-bar quick-create menu toggling, multi-tab System Health Modal (Daemon, MCP Server, Scheduler & Poller), onboarding hero card empty-state condition, remote creation wizard segmented auth toggling (`oauth` vs `custom`), and dual-pane directory explorer directional transfer action buttons and canary drag-guards. Executed complete E2E test suite and full 7-stage quality gate (`./scripts/check.sh all`) with 100% pass rate.


- [x] Task 7: Expose `SupportsOAuth` in Backend Provider Catalog & Fix Step 2 Segmented Control
  - **Scope**: `internal/models/provider.go`, `internal/services/remote_wizard_service.go`, `internal/services/remote_wizard_service_test.go`, `dashboard/frontend/partials/modals/modal_create_remote.html`, `dashboard/frontend/js/remotes_wizard.js`
  - **Verification**: `go test -count=1 ./internal/services/... -run TestWizardService_GetProvidersCatalog` and `./scripts/check.sh templates && ./scripts/check.sh loc 400`

- [x] Task 8: Implement Cancellation & In-Flight Abort for Remote Creation Wizard Step 3
  - **Scope**: `dashboard/frontend/partials/modals/modal_create_remote.html`, `dashboard/frontend/js/remotes_wizard.js`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh loc 400 && go test -count=1 ./tests/e2e/... -run TestDashboard_HeadlessChromeSmoke`

- [x] Task 9: Interactive MCP Tool Registry Accordion in System Health Modal
  - **Scope**: `dashboard/frontend/partials/modals/modal_system_health.html`, `dashboard/frontend/js/system_health.js`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh loc 400 && go test -count=1 ./tests/e2e/... -run TestDashboard_HeadlessChromeSmoke`

- [x] Task 10: Fleet Onboarding Hero Card Ready Remote Badges & Navigation Linkout
  - **Scope**: `dashboard/frontend/partials/view_fleet.html`, `dashboard/frontend/js/fleet.js`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh loc 400 && go test -count=1 ./tests/e2e/... -run TestDashboard_HeadlessChromeSmoke`

- [x] Task 11: True Single-Bar Consolidation in Storage Remotes View
  - **Scope**: `dashboard/frontend/partials/view_remotes.html`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh loc 400 && go test -count=1 ./tests/e2e/... -run TestDashboard_HeadlessChromeSmoke`

- [ ] Task 12: E2E Smoke Tests & Full Quality Gate Check
  - **Scope**: `tests/e2e/dashboard_smoke_test.go`
  - **Verification**: `./scripts/check.sh all`

- 2026-11-10: Task 7 verified: Added `SupportsOAuth` boolean to `ProviderCatalogItem` in Go backend, populated via `IsOAuthProvider`, and implemented fallback helper `isOAuthProvider(p)` in `remotes_wizard.js` and `modal_create_remote.html` to reliably display the OAuth vs Custom segmented control.

- 2026-11-10: Task 8 verified: Added `cancelProvisioning()` in `remotes_wizard.js` with `AbortController` cancellation, OAuth stop endpoint dispatch, safe modal close hook, and interactive Cancel button during Step 3 provisioning. Verified via templates, LOC audit, and headless Chrome smoke test.

- 2026-11-10: Task 9 verified: Implemented interactive MCP Tool Registry inline accordion with 24 tool definitions, parameters schemas, safeguard risk profiles, search filtering, category pill toggles, and expanded accordion state in `modal_system_health.html` and `system_health.js`. Verified via headless Chrome smoke test, template validation, and LOC audit.

- 2026-11-10: Task 10 verified: Surfaced configured ready remote target badges/pills in Fleet onboarding Step 1 hero card with direct linkout navigation to the Remotes tab, backed by `hasConfiguredRemotes`, `readyRemotesList`, and `navigateToRemotes()` in `fleet.js`. Verified via headless Chrome smoke test, template validation, and LOC audit.

- 2026-11-10: Task 11 verified: Consolidated the Authentication Health Overview card and Configured Remotes control bar into a single unified sleek control deck in `view_remotes.html` containing remotes count, OAuth counts, connection status pills, profile scoping, audit log, quota refresh, and remote provisioning. Verified via templates, LOC audit, and headless Chrome smoke test.
