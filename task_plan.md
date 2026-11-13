# Task Plan: Deep Path Navigation & Pinned Log Audit Dock

## Goal
Implement deep path navigation from log entries into the Dual-Pane File Explorer with synchronized navigation, highlighted target rows, and a persistent IDE-style collapsible bottom dock for rapid batch inspection.

## Tasks

- [x] Task 1: Contextual Tokenizer Core (`utils.js`)
  - **Scope**: `dashboard/frontend/js/utils.js`
  - **Verification**: `go test -count=1 -v ./tests/e2e -run TestDashboard_Smoke`

- [x] Task 2: Concurrency & Async Navigation Engine (`profiles.js` & `logs.js`)
  - **Scope**: `dashboard/frontend/js/profiles.js`, `dashboard/frontend/js/logs.js`
  - **Verification**: `go test -count=1 -v ./tests/e2e -run TestDashboard_Smoke`

- [x] Task 3: Visual Row Highlighting & Explorer Row Data Binding (`review_actions.js` & `view_explorer.html`)
  - **Scope**: `dashboard/frontend/js/review_actions.js`, `dashboard/frontend/partials/view_explorer.html`
  - **Verification**: `./scripts/check.sh templates`

- [x] Task 4: Pinned Log Audit Dock Engine (`log_dock.js` & `component_log_dock.html`)
  - **Scope**: `dashboard/frontend/js/log_dock.js`, `dashboard/frontend/partials/components/component_log_dock.html`, `dashboard/frontend/index.html`, `dashboard/frontend/app.js`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh loc 400`

- [x] Task 5: Ledger & Daemon Modal Integration (`pattern_timeline_ledger.html` & `modal_daemon.html`)
  - **Scope**: `dashboard/frontend/partials/patterns/pattern_timeline_ledger.html`, `dashboard/frontend/partials/modals/modal_daemon.html`
  - **Verification**: `./scripts/check.sh templates`

- [x] Task 6: Roadmap Documentation, E2E Headless Smoke Tests & Pre-Commit Quality Gate
  - **Scope**: `docs/roadmap/backlog.md`, `docs/roadmap/ROADMAP.md`, `tests/e2e/dashboard_smoke_test.go`
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-11-11: Initialized task plan from implementation_plan.md.
- 2026-11-11: Completed Task 1 - Implemented contextual path tokenizer `extractPathFromLogMsg` and integrated with `parseJsonLogLine` in `dashboard/frontend/js/utils.js`. Verified with LOC gate (<400 LOC) and E2E headless smoke tests.
- 2026-11-11: Completed Task 2 - Implemented async `selectProfile` with `overrideDir` bypass in `dashboard/frontend/js/profiles.js` and dual-pane `jumpToExplorerPath` & `scrollToHighlightedRow` in `dashboard/frontend/js/logs.js`. Verified with LOC gate and E2E smoke tests.
- 2026-11-11: Completed Task 3 - Bound `:data-item-name="item.Name"` attributes to explorer left and right pane rows, updated `getItemReviewRowClass` with pulse animation, and added `@keyframes rowPulse` to `design-tokens.css`. Verified with template validator, LOC gate, and E2E smoke tests.
- 2026-11-11: Completed Task 4 - Created `dashboard/frontend/js/log_dock.js` and `component_log_dock.html`, registered in `index.html` and `app.js`, added dock button to `component_explorer_toolbar.html`, and embedded in `view_explorer.html`. Verified with template validator, LOC gate, and E2E smoke tests.
- 2026-11-11: Completed Task 5 - Integrated interactive path chips and `jumpToExplorerPath` deep navigation triggers into `pattern_timeline_ledger.html` and `modal_daemon.html` across single and clustered log rows. Verified with template validator, LOC gate, and E2E smoke tests.
- 2026-11-11: Completed Task 6 - Added headless Chrome E2E test `DeepPathNavigationAndLogDock` verifying contextual path tokenization, log dock toggle/filtering/stepping, and row pulse highlights. Updated `backlog.md` and `ROADMAP.md` for `v1.0.0-beta.32` delivery. Passed all 7 pre-commit quality gate stages (`./scripts/check.sh all`).


