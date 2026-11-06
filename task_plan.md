# Task Plan: In-App User Guidance & Interactive Diagnostic Manual

## Goal
Embed contextual policy explanations, an interactive slide-over reference manual, and real-time self-diagnostics into the Bisync Hub dashboard to assist operators and enhance data safety understanding.

## Tasks

- [x] Task 1: Define Diagnostic Models & Structs
  - **Scope**: `internal/models/api.go`
  - **Verification**: `go test -count=1 ./internal/models/...`

- [x] Task 2: Implement Diagnostic Evaluation Engine & REST Handler
  - **Scope**: `internal/server/handlers_diagnostics.go`, `internal/server/server.go`
  - **Verification**: `go test -count=1 ./internal/server/...`

- [x] Task 3: Unit Testing Suite for Live Diagnostics
  - **Scope**: `internal/server/handlers_diagnostics_test.go`
  - **Verification**: `go test -race -v -count=1 ./internal/server/... -run TestHandleDiagnostics`

- [x] Task 4: In-Dashboard Reference Manual & Diagnostics Drawer Template
  - **Scope**: `dashboard/frontend/partials/components/component_help_drawer.html`
  - **Verification**: `go test -count=1 ./tests/e2e/...`

- [x] Task 5: Reusable Tooltip Components & Contextual Safety Explanations
  - **Scope**: `dashboard/frontend/partials/components/ui_components.html`, `dashboard/frontend/partials/modals/modal_profiles.html`, `dashboard/frontend/partials/modals/modal_global_settings.html`, `dashboard/frontend/partials/modals/modal_review.html`, `dashboard/frontend/partials/view_fleet.html`
  - **Verification**: `./scripts/check.sh all`

- [x] Task 6: Modular Frontend Controller, Keyboard Shortcuts & Header Trigger
  - **Scope**: `dashboard/frontend/js/help.js`, `dashboard/frontend/app.js`, `dashboard/frontend/index.html`
  - **Verification**: `go test -count=1 ./tests/e2e/...` and `./scripts/check.sh all`

- [x] Task 7: End-to-End Headless Chrome Testing & Final Verification
  - **Scope**: `tests/e2e/dashboard_smoke_test.go`
  - **Verification**: `go test -count=1 -v ./tests/e2e/... -run TestDashboard_HelpDrawerAndGuidance` and `./scripts/check.sh all`


- [x] Task 8: Context Tagging & Active Context Detection in `help.js`
  - **Scope**: `dashboard/frontend/js/help.js`
  - **Verification**: `go test -count=1 ./tests/e2e/...` and `./scripts/check.sh all`

- [x] Task 9: Drawer Template Context Indicator & Toggle in `component_help_drawer.html`
  - **Scope**: `dashboard/frontend/partials/components/component_help_drawer.html`
  - **Verification**: `./scripts/check.sh templates` and `./scripts/check.sh all`

- [x] Task 10: Modal Header Contextual Help Triggers
  - **Scope**: `dashboard/frontend/partials/modals/modal_profiles.html`, `dashboard/frontend/partials/modals/modal_review.html`, `dashboard/frontend/partials/modals/modal_global_settings.html`
  - **Verification**: `./scripts/check.sh templates` and `./scripts/check.sh all`

- [x] Task 11: E2E Headless Chrome Verification & Live Test
  - **Scope**: `tests/e2e/dashboard_smoke_test.go`
  - **Verification**: `go test -count=1 -v ./tests/e2e/...` and `./scripts/check.sh all`

## Status Updates

- 2026-11-06: Initialized plan from implementation_plan.md on branch feat/in-app-user-guidance.

- 2026-11-06: Completed Task 1: Defined `DiagnosticCheckStatus`, `DiagnosticCheck`, `DiagnosticSystemInfo`, and `DiagnosticsResponse` models in `internal/models/api.go` with schema tests in `internal/models/api_test.go`.
- 2026-11-06: Completed Task 8: Added `detectActiveContext`, `toggleContextFilter`, context metadata, and contextual scoping across policies, flags, runbooks, and diagnostic checks in `dashboard/frontend/js/help.js`.

- 2026-11-06: Completed Task 2: Implemented `handleDiagnostics` REST handler and operational check evaluators across daemon connectivity, storage accessibility, canary protection, remote token health, and host runtime environment in `internal/server/handlers_diagnostics.go`, wired to `GET /api/diagnostics` in `internal/server/server.go`.
- 2026-11-06: Completed Task 3: Built comprehensive unit testing suite in `internal/server/handlers_diagnostics_test.go` verifying offline failure states, healthy multi-remote passes, and elevated latency warning thresholds under `-race`.
- 2026-11-06: Completed Task 4: Created `dashboard/frontend/partials/components/component_help_drawer.html` featuring interactive slide-over tabs for System Diagnostics, Safety Policies, Flag Catalog, and Troubleshooting Runbooks. Verified with `main_test.go` and `tests/e2e/...`.
- 2026-11-06: Completed Task 9: Added scoped context indicator bar and interactive "Show All Topics / Show Scoped Only" toggle in `dashboard/frontend/partials/components/component_help_drawer.html`. Verified template safety with `./scripts/check.sh templates` and `./scripts/check.sh all`.


- 2026-11-06: Completed Task 5: Added `component_tooltip` macro to `ui_components.html`, embedded contextual safety policy explanations across `modal_profiles.html`, `modal_global_settings.html`, and `modal_review.html`, added Help & Manual quick link in `view_fleet.html`, and verified template compliance with `./scripts/check.sh all`.


- 2026-11-06: Completed Task 6: Implemented `dashboard/frontend/js/help.js`, integrated `window.HelpModule` into `app.js`, added header Help trigger and `component_help_drawer` template into `index.html`, and registered global `?` and `F1` keyboard shortcuts.

- 2026-11-06: Completed Task 10: Added contextual help triggers (`?` button with `openHelpDrawer(...)`) to headers of `modal_profiles.html`, `modal_review.html`, and `modal_global_settings.html`. Verified with `./scripts/check.sh templates` and `./scripts/check.sh all`.
- 2026-11-06: Completed Task 11: Added headless Chrome E2E test cases for context-aware detection, scoped filtering, context toggle, and modal header triggers in `tests/e2e/dashboard_smoke_test.go`. Verified with `go test -count=1 -v ./tests/e2e/...` and `./scripts/check.sh all` (7/7 passed).


- 2026-11-06: Completed Task 7: Implemented `TestDashboard_HelpDrawerAndGuidance` in `tests/e2e/dashboard_smoke_test.go` covering help drawer triggering, diagnostics fetch, tab switching, and keyboard shortcuts under headless Chrome, and verified all 7 quality gate stages passed.



