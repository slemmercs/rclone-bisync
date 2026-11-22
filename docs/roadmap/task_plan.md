# Task Plan: Epic 4 — Wizard Virtual Remote Guardrails & Hasher Acceleration

## Goal
Implement contextual safeguards, canonical overlay chaining (Base -> Crypt -> Hasher), bucket validation, and union/combine bisync guardrails across backend services and the frontend remote wizard.

## Tasks

- [x] Task 1: Backend Model & Service Extensions (Chaining, Hasher injection, Bucket validation, Union/Combine guards)
  - **Scope**: `internal/models/provider.go`, `internal/services/remote_wizard_service.go`, `internal/app/container.go`
  - **Verification**: `go test -count=1 ./internal/services -run TestRemoteWizardService`

- [ ] Task 2: Backend Unit Tests & Gendocs Synchronization
  - **Scope**: `internal/services/remote_wizard_service_test.go`, `cmd/gendocs/`
  - **Verification**: `go test -count=1 ./internal/services/...`

- [ ] Task 3: Frontend Type Contracts & Helper Functions
  - **Scope**: `dashboard/frontend/src/types/remotes.ts`, `dashboard/frontend/src/stores/remotes.ts`
  - **Verification**: `npm --prefix dashboard/frontend run type-check`

- [ ] Task 4: Wizard Step 2 & 3 UI Guardrails & Hasher Acceleration
  - **Scope**: `dashboard/frontend/src/components/wizard/WizardStepConfigure.vue`, `dashboard/frontend/src/components/wizard/WizardStepReview.vue`, `dashboard/frontend/src/components/ModalRemoteWizard.vue`
  - **Verification**: `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build`

- [ ] Task 5: Full Quality Suite Verification
  - **Scope**: All quality gate stages via `./scripts/check.sh all`
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-11-22: Initialized implementation plan and task plan for Epic 4.
- 2026-11-22: Completed Task 1. Extended `CreateRemoteBridgeRequest` with `AccelerateHasher` and `HasherHashes`, injected `HasherService` into `RemoteWizardService`, implemented overlay chaining (`Base -> Crypt -> Hasher`), baked subpath into inner layers to prevent double-nesting, added bucket-based subpath validation, and blocked `union`/`combine` profile creation. Verified with `go test -count=1 ./internal/services -run TestRemoteWizard` and `go test -count=1 ./internal/app`.
