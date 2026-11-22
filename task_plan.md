<<<<<<< HEAD
# Task Plan: Universal Dynamic Remote Configuration Engine

## Goal
Replace static hardcoded forms in `ModalRemoteWizard.vue` with a reusable, dynamic schema-driven form engine powered by Rclone's `/config/providers` metadata, supporting S3 flavor filtering, option masking, hide bitmasks, and password obscuring.

## Tasks

- [x] Task 1: Go Model Schema Updates & Tests
  - **Scope**: `internal/models/provider.go`, `internal/models/provider_test.go`
  - **Verification**: `go test -count=1 ./internal/models/...`

- [ ] Task 2: TypeScript Contract Generation & Types Alignment
  - **Scope**: `cmd/gendocs/`, `dashboard/frontend/src/types/models.gen.ts`, `dashboard/frontend/src/types/dedupe.ts`
  - **Verification**: `make generate-ts && npm --prefix dashboard/frontend run type-check`

- [ ] Task 3: Build DynamicProviderForm Component
  - **Scope**: `dashboard/frontend/src/components/DynamicProviderForm.vue`
  - **Verification**: `npm --prefix dashboard/frontend run type-check`

- [ ] Task 4: Integrate DynamicProviderForm into ModalRemoteWizard
  - **Scope**: `dashboard/frontend/src/components/ModalRemoteWizard.vue`
  - **Verification**: `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build`

- [ ] Task 5: Pre-Flight Check & Quality Gates
  - **Scope**: Full repo verification and bundle check
=======
# Task Plan: Hasher Backend Integration (Checksum Acceleration & Emulation)

## Goal
Implement full-suite integration for Rclone's Hasher backend overlay across Bisync Hub's backend services, Cockpit UI, and MCP agent tool suite, unlocking checksum verification everywhere, instant dual-pane file comparisons, and cache pre-seeding.

## Tasks

- [x] Task 1: Add Rclone RC `BackendCommand` client and Hasher models
  - **Scope**: `internal/core/client/backend.go`, `internal/core/client/backend_test.go`, `internal/models/hasher.go`
  - **Verification**: `go test -count=1 ./internal/core/client -run TestBackendCommand`

- [ ] Task 2: Provider categorization and HasherService implementation
  - **Scope**: `internal/services/remote_wizard_service.go`, `internal/services/hasher_service.go`, `internal/services/hasher_service_test.go`, `internal/app/container.go`
  - **Verification**: `go test -count=1 ./internal/services -run TestHasher`

- [ ] Task 3: Hasher HTTP routes and server handlers
  - **Scope**: `internal/server/handlers_hasher.go`, `internal/server/handlers_hasher_test.go`, `internal/server/routes.go`
  - **Verification**: `go test -count=1 ./internal/server -run TestHasher`

- [ ] Task 4: MCP tool `bisync_hasher_command`
  - **Scope**: `internal/mcp/tools_remote.go`, `internal/mcp/tools_remote_test.go`
  - **Verification**: `go test -count=1 ./internal/mcp -run TestHasher`

- [ ] Task 5: Persistent cache configuration in deployment scripts
  - **Scope**: `deploy/install.sh`, `deploy/systemd/rclone-daemon.service`, `bisync.env.example`
  - **Verification**: `bash -n deploy/install.sh`

- [ ] Task 6: Frontend contracts, ModalHasherManager, and ViewRemotes actions
  - **Scope**: `dashboard/frontend/src/types/hasher.ts`, `dashboard/frontend/src/components/ModalHasherManager.vue`, `dashboard/frontend/src/views/ViewRemotes.vue`, `dashboard/frontend/src/components/ModalRemoteWizard.vue`
  - **Verification**: `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build`

- [ ] Task 7: Full test suite and pre-flight quality checks
  - **Scope**: Pre-flight validation across all 8 quality stages
>>>>>>> feat/hasher-backend-integration
  - **Verification**: `./scripts/check.sh all`

## Status Updates

<<<<<<< HEAD
- 2026-11-21: Completed Task 1. Added `Provider` and `Hide` fields to `ProviderOption` and `ProviderExample` in `internal/models/provider.go`. Added `IsHiddenInConfig()` helper and unit tests in `internal/models/provider_test.go`. Verified with `go test -count=1 ./internal/models/...` and `go test -count=1 ./internal/services -run TestRemoteWizard`.

- 2026-11-21: Initialized plan from implementation_plan.md on branch `feat/dynamic-provider-form`.

=======
- 2026-11-21: Completed Task 1. Added `BackendCommand` RC client method with unit tests in `internal/core/client/backend.go` and defined Hasher DTO models in `internal/models/hasher.go`. Verified with `go test -count=1 ./internal/core/client -run TestBackendCommand`.
- 2026-11-21: Initialized plan from implementation_plan.md and created GitHub issue #98 on branch `feat/hasher-backend-integration`.
>>>>>>> feat/hasher-backend-integration


