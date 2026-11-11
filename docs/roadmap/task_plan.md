# Task Plan: Enforce UI-Aligned Enums & Validation for MCP & Profile Configurations

## Goal
Enforce strict validation across all profile configuration entry points (MCP tools `bisync_create_profile`, `bisync_update_profile`, `bisync_create_remote` and backend domain models `internal/profiles`) so that MCP tools and API callers can only use options defined and supported in the dashboard UI (`manual`, `15m`, `30m`, `1h`, `4h`, `daily` for schedule; defined review, conflict, and compare modes; normalized remote names without colons).

## Tasks

- [x] Task 1: Domain Enums & Invariant Validation in `internal/profiles/model.go`
  - **Scope**: `internal/profiles/model.go`, `internal/profiles/model_test.go`
  - **Verification**: `go test -count=1 ./internal/profiles/...`

- [x] Task 2: MCP Profile Tools Schema & Strict Parameter Validation
  - **Scope**: `internal/mcp/tools_profile.go`, `internal/mcp/tools_remote.go`, `internal/mcp/tools_profile_test.go`, `internal/services/remote_wizard_service_test.go`
  - **Verification**: `go test -count=1 ./internal/mcp/... ./internal/services/...`

- [x] Task 3: Frontend Resilience (Remote Target Colon Normalization & MCP Registry Schema Alignment)
  - **Scope**: `dashboard/frontend/partials/view_telemetry.html`, `dashboard/frontend/partials/view_fleet.html`, `dashboard/frontend/js/system_health.js`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh loc 400 && go test -count=1 ./tests/e2e/... -run TestDashboard_HeadlessChromeSmoke`

- [ ] Task 4: Full Quality Gate Verification & Pre-commit Check
  - **Scope**: Full repository
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-11-11: Initialized plan from implementation_plan.md.
- 2026-11-11: Completed Task 1 - Domain enums and invariant validation added to internal/profiles/model.go with trailing colon stripping and full test suite passing in internal/profiles.
- 2026-11-11: Completed Task 2 - MCP profile and remote tools schemas updated with UI preset enums, early parameter validation, and remote trailing colon stripping. All MCP and services unit tests passing.
- 2026-11-11: Completed Task 3 - Frontend profile target formatting normalized without double colons, and MCP Tool Registry parameter schema descriptions aligned with backend presets. Verified via templates, LOC, and E2E headless Chrome smoke test.
