# Task Plan: Provider-Aware "Open on Remote" Capability Filtering

## Goal
Filter out "Open on Remote" from the File Explorer context menu when the active storage provider does not support public sharing links (`operations/publiclink`), avoiding runtime errors on unsupported storage tiers.

## Tasks

- [x] Task 1: Backend Remote Capability Detection (`operations/fsinfo` & `can_public_link`)
  - **Scope**: `internal/core/client/remotes.go`, `internal/services/telemetry_remotes.go`, `internal/services/telemetry_service_test.go`
  - **Verification**: `go test -v -count=1 ./internal/services/... -run TestGetRemotesSummary`

- [ ] Task 2: Frontend Context Menu Filtering & Dynamic Capability Degradation
  - **Scope**: `dashboard/frontend/js/explorer_operations.js`, `dashboard/frontend/partials/components/component_context_menu.html`, `tests/e2e/dashboard_smoke_test.go`
  - **Verification**: `go test -v -count=1 ./tests/e2e/... -run TestDashboard_HeadlessChromeSmoke`

- [ ] Task 3: Quality Gate Verification & Pre-Commit Check
  - **Scope**: Full quality check suite
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-11-08: Initialized 3-task plan for provider-aware "Open on Remote" capability filtering.

