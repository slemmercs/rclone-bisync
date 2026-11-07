# Task Plan: Automated Rclone Config Passphrase Persistence & Daemon Safety

## Goal
Implement automatic persistence of `RCLONE_CONFIG_PASS` to `rc-secrets.env` when encrypting `rclone.conf` via the UI, detect missing daemon passphrase on startup/polling for externally encrypted configurations, and display a prominent warning banner with guided resolution.

## Tasks

- [x] Task 1: Environment Secret Management Utilities & Unit Tests
  - **Scope**: `internal/config/secrets.go`, `internal/config/secrets_test.go`, `internal/config/config.go`
  - **Verification**: `go test -v -count=1 ./internal/config/... -run TestUpdateRemoveSecret`

- [x] Task 2: Service Layer Passphrase Synchronization & Validation
  - **Scope**: `internal/models/encryption.go`, `internal/services/config_encryption_service.go`, `internal/services/config_encryption_service_test.go`
  - **Verification**: `go test -v -count=1 ./internal/services/... -run TestConfigEncryptionService`

- [x] Task 3: API Endpoints for Passphrase Persistence & Verification
  - **Scope**: `internal/server/handlers_encryption.go`, `internal/server/handlers_encryption_test.go`, `internal/server/server.go`
  - **Verification**: `go test -v -count=1 ./internal/server/... -run TestHandleSaveDaemonPassphrase`

- [x] Task 4: Frontend Cockpit Warning Banner & Passphrase Persistence UI
  - **Scope**: `dashboard/frontend/index.html`, `dashboard/frontend/partials/modals/modal_global_settings.html`, `dashboard/frontend/js/settings.js`, `dashboard/frontend/app.js`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh loc`

- [ ] Task 5: Full Quality Gates & E2E Smoke Testing
  - **Scope**: `tests/e2e/dashboard_smoke_test.go`
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-11-07: Initialized task plan from approved implementation_plan.md.
- 2026-11-07: Completed Task 1 - Environment Secret Management Utilities & Unit Tests. Verified with unit tests (`go test -v -count=1 ./internal/config/... -run TestUpdateRemoveSecret`).
- 2026-11-07: Completed Task 2 - Service Layer Passphrase Synchronization & Validation. Verified with unit tests (`go test -v -count=1 ./internal/services/... -run TestConfigEncryptionService`).
- 2026-11-07: Completed Task 3 - API Endpoints for Passphrase Persistence & Verification. Verified with unit tests (`go test -v -count=1 ./internal/server/... -run TestHandleSaveDaemonPassphrase`).
- 2026-11-07: Completed Task 4 - Frontend Cockpit Warning Banner & Passphrase Persistence UI. Verified with template safety and LOC checks (`./scripts/check.sh templates && ./scripts/check.sh loc`).

