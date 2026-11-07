# Task Plan: Rclone-UI Feature Suite - Milestone 1 (Operational Controls & Inspection)

## Goal
Implement Milestone 1 of the Rclone-UI Feature Suite natively in Bisync Hub: Dynamic Bandwidth Throttling, Transfer Stats, Safe File Preview/Download, and Configuration Encryption Status.

## Tasks

- [x] Task 1: Bandwidth - [ ] Task 1: Bandwidth & Core Stats Models and Client RC Methods Core Stats Models and Client RC Methods
  - **Scope**: `internal/models/core.go`, `internal/core/client/client.go`, `internal/core/client/client_test.go`
  - **Verification**: `go test -v -count=1 ./internal/core/client/...`

- [x] Task 2: Core Service and HTTP API Handlers
  - **Scope**: `internal/services/core_service.go`, `internal/services/core_service_test.go`, `internal/server/handlers_core.go`, `internal/server/server.go`, `internal/app/container.go`
  - **Verification**: `go test -v -count=1 ./internal/services/... -run TestCoreService`

- [ ] Task 3: File Preview & Streaming Service
  - **Scope**: `internal/models/preview.go`, `internal/services/explorer_preview.go`, `internal/services/explorer_preview_test.go`
  - **Verification**: `go test -v -count=1 ./internal/services/... -run TestPreview`

- [ ] Task 4: File Preview & Download HTTP Endpoints and Explorer UI
  - **Scope**: `internal/server/handlers_preview.go`, `dashboard/frontend/partials/modals/modal_file_preview.html`, `dashboard/frontend/partials/view_explorer.html`
  - **Verification**: `go test -v -count=1 ./internal/server/...`

- [ ] Task 5: Rclone Config Encryption Service and Settings UI
  - **Scope**: `internal/models/encryption.go`, `internal/services/config_encryption_service.go`, `internal/services/config_encryption_service_test.go`, `internal/server/handlers_encryption.go`, `dashboard/frontend/partials/modals/modal_global_settings.html`
  - **Verification**: `go test -v -count=1 ./internal/services/... -run TestEncryption`

- [ ] Task 6: Header Bandwidth Controller and Real-Time Stats UI
  - **Scope**: `dashboard/frontend/app.js`, `dashboard/frontend/js/bandwidth.js`, `dashboard/frontend/partials/components/component_header.html`, `dashboard/frontend/partials/view_telemetry.html`
  - **Verification**: `./scripts/check.sh templates`

- [ ] Task 7: Milestone 1 Verification, E2E Smoke Tests, and Full Quality Gates
  - **Scope**: `tests/e2e/dashboard_smoke_test.go`, all Milestone 1 components
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-11-07: Initialized task plan from implementation_plan.md for Milestone 1.
