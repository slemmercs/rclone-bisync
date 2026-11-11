# Task Plan: Deep Path Navigation & Pinned Log Audit Dock

## Goal
Implement deep path navigation from log entries into the Dual-Pane File Explorer with synchronized navigation, highlighted target rows, and a persistent IDE-style collapsible bottom dock for rapid batch inspection.

## Tasks

- [x] Task 1: Contextual Tokenizer Core (`utils.js`)
  - **Scope**: `dashboard/frontend/js/utils.js`
  - **Verification**: `go test -count=1 -v ./tests/e2e -run TestDashboard_Smoke`

- [ ] Task 2: Concurrency & Async Navigation Engine (`profiles.js` & `logs.js`)
  - **Scope**: `dashboard/frontend/js/profiles.js`, `dashboard/frontend/js/logs.js`
  - **Verification**: `go test -count=1 -v ./tests/e2e -run TestDashboard_Smoke`

- [ ] Task 3: Visual Row Highlighting & Explorer Row Data Binding (`review_actions.js` & `view_explorer.html`)
  - **Scope**: `dashboard/frontend/js/review_actions.js`, `dashboard/frontend/partials/view_explorer.html`
  - **Verification**: `./scripts/check.sh templates`

- [ ] Task 4: Pinned Log Audit Dock Engine (`log_dock.js` & `component_log_dock.html`)
  - **Scope**: `dashboard/frontend/js/log_dock.js`, `dashboard/frontend/partials/components/component_log_dock.html`, `dashboard/frontend/index.html`, `dashboard/frontend/app.js`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh loc 400`

- [ ] Task 5: Ledger & Daemon Modal Integration (`pattern_timeline_ledger.html` & `modal_daemon.html`)
  - **Scope**: `dashboard/frontend/partials/patterns/pattern_timeline_ledger.html`, `dashboard/frontend/partials/modals/modal_daemon.html`
  - **Verification**: `./scripts/check.sh templates`

- [ ] Task 6: Roadmap Documentation, E2E Headless Smoke Tests & Pre-Commit Quality Gate
  - **Scope**: `docs/roadmap/backlog.md`, `docs/roadmap/ROADMAP.md`, `tests/e2e/dashboard_smoke_test.go`
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-11-11: Initialized task plan from implementation_plan.md.
- 2026-11-11: Completed Task 1 - Implemented contextual path tokenizer `extractPathFromLogMsg` and integrated with `parseJsonLogLine` in `dashboard/frontend/js/utils.js`. Verified with LOC gate (<400 LOC) and E2E headless smoke tests.


