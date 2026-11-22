# Task Plan: Storage Remote Virtual Overlay Linkage, Badging, Action Guardrails, and Plaintext Decryption Guidance

## Goal
Resolve virtual overlay linkage blind spots (`Base -> Crypt`, `Base -> Hasher`) across the storage remotes view and profile management: enrich backend telemetry with overlay relationships, show visual badges on base and overlay cards, prevent duplicate overlay creation collisions, provide clear plaintext decryption guidance, and render pipeline badges on sync profiles.

## Tasks

- [x] Task 1: Backend Overlay Dependency Graph in TelemetryService
  - **Scope**: `internal/services/telemetry_remotes.go`, `internal/services/telemetry_service_test.go`
  - **Verification**: `go test -count=1 ./internal/services -run TestTelemetry`

- [ ] Task 2: Storage Remote Tile Overlay Badges & Action Guardrails
  - **Scope**: `dashboard/frontend/src/types/remotes.ts`, `dashboard/frontend/src/components/remotes/RemoteCard.vue`
  - **Verification**: `npm --prefix dashboard/frontend run build`

- [ ] Task 3: ModalCryptWrap Overlay Validation & Plaintext Decryption Callouts
  - **Scope**: `dashboard/frontend/src/components/ModalCryptWrap.vue`, `dashboard/frontend/src/components/wizard/WizardStepConfigure.vue`
  - **Verification**: `npm --prefix dashboard/frontend run build`

- [ ] Task 4: Profile List & Edit Form Pipeline & Encryption Badges
  - **Scope**: `dashboard/frontend/src/components/profiles/ProfileListTab.vue`, `dashboard/frontend/src/components/profiles/ProfileFormTab.vue`
  - **Verification**: `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build`

- [ ] Task 5: Full Quality Verification Suite
  - **Scope**: Pre-flight checks across all stages
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-11-22: Initialized implementation plan and task plan for virtual overlay linkage, badging, guardrails, and plaintext decryption guidance.
- 2026-11-22: Completed Task 1. Added `remote` to safeKeys in RcloneClient `GetRemoteConfig`. Implemented `parseUnderlyingRemote` and bidirectional overlay linkage in `TelemetryService.GetRemotesSummaryCached`, populating `underlying_remote`, `underlying_path`, `crypt_overlay`, `hasher_overlay`, and `overlays`. Added unit test `TestTelemetryService_OverlayResolution`.



