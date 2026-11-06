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

- [ ] Task 4: In-Dashboard Reference Manual & Diagnostics Drawer Template
  - **Scope**: `dashboard/frontend/partials/components/component_help_drawer.html`
  - **Verification**: `go test -count=1 ./tests/e2e/...`

- [ ] Task 5: Reusable Tooltip Components & Contextual Safety Explanations
  - **Scope**: `dashboard/frontend/partials/components/ui_components.html`, `dashboard/frontend/partials/modals/modal_profiles.html`, `dashboard/frontend/partials/modals/modal_global_settings.html`, `dashboard/frontend/partials/modals/modal_review.html`, `dashboard/frontend/partials/view_fleet.html`
  - **Verification**: `./scripts/check.sh all`

- [ ] Task 6: Modular Frontend Controller, Keyboard Shortcuts & Header Trigger
  - **Scope**: `dashboard/frontend/js/help.js`, `dashboard/frontend/app.js`, `dashboard/frontend/index.html`
  - **Verification**: `go test -count=1 ./tests/e2e/...`

- [ ] Task 7: End-to-End Headless Chrome Testing & Final Verification
  - **Scope**: `tests/e2e/dashboard_smoke_test.go`
  - **Verification**: `go test -count=1 -v ./tests/e2e/... -run TestDashboard_HelpDrawerAndGuidance` and `./scripts/check.sh all`

## Status Updates

- 2026-11-06: Initialized plan from implementation_plan.md on branch feat/in-app-user-guidance.

- 2026-11-06: Completed Task 1: Defined `DiagnosticCheckStatus`, `DiagnosticCheck`, `DiagnosticSystemInfo`, and `DiagnosticsResponse` models in `internal/models/api.go` with schema tests in `internal/models/api_test.go`.
- 2026-11-06: Completed Task 2: Implemented `handleDiagnostics` REST handler and operational check evaluators across daemon connectivity, storage accessibility, canary protection, remote token health, and host runtime environment in `internal/server/handlers_diagnostics.go`, wired to `GET /api/diagnostics` in `internal/server/server.go`.
- 2026-11-06: Completed Task 3: Built comprehensive unit testing suite in `internal/server/handlers_diagnostics_test.go` verifying offline failure states, healthy multi-remote passes, and elevated latency warning thresholds under `-race`.



