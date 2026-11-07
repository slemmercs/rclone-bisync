# Task Plan: Explorer Ergonomics, PDF Preview, Config Encryption Fix, and Layout Optimization

## Goal
Resolve operational ergonomics, safety barriers, and layout issues from testing: fix config encryption `--password-command` execution, restrict preview icons to supported formats, enable native PDF preview, move bandwidth controls into Settings modal, clean up duplicate telemetry cards, and add Explorer Upload & Context Menu.

## Tasks

- [x] Task 1: Fix Rclone Config Encryption `--password-command` Execution
  - **Scope**: `internal/services/config_encryption_service.go`, `internal/services/config_encryption_service_test.go`
  - **Verification**: `go test -v -count=1 ./internal/services/... -run TestConfigEncryptionService`

- [x] Task 2: File Preview Format Classification & Native In-Browser PDF Streaming
  - **Scope**: `internal/models/preview.go`, `internal/services/explorer_preview.go`, `internal/server/handlers_preview.go`, `dashboard/frontend/partials/modals/modal_file_preview.html`, tests
  - **Verification**: `go test -v -count=1 ./internal/services/... -run 'TestExplorerPreview|TestDetectMimeType'` and `go test -v -count=1 ./internal/server/... -run 'TestHandlePreview|TestHandleDownload'`

- [x] Task 3: Header & Telemetry Layout Cleanup (Bandwidth in Settings Modal)
  - **Scope**: `dashboard/frontend/partials/components/component_header.html`, `dashboard/frontend/index.html`, `dashboard/frontend/partials/modals/modal_global_settings.html`, `dashboard/frontend/partials/view_telemetry.html`, `dashboard/frontend/js/bandwidth.js`
  - **Verification**: `./scripts/check.sh templates` and `go test -v -count=1 ./tests/e2e/... -run TestDashboard_HeadlessChromeSmoke/GlobalSettingsModalTabs`

- [x] Task 4: File Explorer Upload & Right-Click Context Menu
  - **Scope**: `internal/server/handlers_explorer.go`, `dashboard/frontend/partials/view_explorer.html`, `dashboard/frontend/js/explorer_actions.js`, `internal/server/handlers_explorer_test.go`
  - **Verification**: `go test -v -count=1 ./internal/server/ -run TestHandleExplorerUpload` and `./scripts/check.sh templates`

- [x] Task 5: End-to-End Smoke Tests & Full Quality Gate Verification
  - **Scope**: `tests/e2e/dashboard_smoke_test.go`, all updated components
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-11-07: Completed all Milestone 1 tasks and quality gates. All 7 checks in `./scripts/check.sh all` and Chrome E2E tests passed.
