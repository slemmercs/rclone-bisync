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
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-11-21: Completed Task 1. Added `Provider` and `Hide` fields to `ProviderOption` and `ProviderExample` in `internal/models/provider.go`. Added `IsHiddenInConfig()` helper and unit tests in `internal/models/provider_test.go`. Verified with `go test -count=1 ./internal/models/...` and `go test -count=1 ./internal/services -run TestRemoteWizard`.

- 2026-11-21: Initialized plan from implementation_plan.md on branch `feat/dynamic-provider-form`.



