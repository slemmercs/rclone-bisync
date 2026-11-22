# Task Plan: Storage Remote Virtual Overlay Linkage, Badging, Action Guardrails, and Plaintext Decryption Guidance

## Goal
Resolve virtual overlay linkage blind spots (`Base -> Crypt`, `Base -> Hasher`) across the storage remotes view and profile management: enrich backend telemetry with overlay relationships, show visual badges on base and overlay cards, prevent duplicate overlay creation collisions, provide clear plaintext decryption guidance, and render pipeline badges on sync profiles.

## Tasks

- [x] Task 1: Backend Overlay Dependency Graph in TelemetryService
  - **Scope**: `internal/services/telemetry_remotes.go`, `internal/services/telemetry_service_test.go`
  - **Verification**: `go test -count=1 ./internal/services -run TestTelemetry`

- [x] Task 2: Storage Remote Tile Overlay Badges & Action Guardrails
  - **Scope**: `dashboard/frontend/src/types/remotes.ts`, `dashboard/frontend/src/components/remotes/RemoteCard.vue`
  - **Verification**: `npm --prefix dashboard/frontend run build`

- [x] Task 3: ModalCryptWrap Overlay Validation & Plaintext Decryption Callouts
  - **Scope**: `dashboard/frontend/src/components/ModalCryptWrap.vue`, `dashboard/frontend/src/components/wizard/WizardStepConfigure.vue`
  - **Verification**: `npm --prefix dashboard/frontend run build`

- [x] Task 4: Profile List & Edit Form Pipeline & Encryption Badges
  - **Scope**: `dashboard/frontend/src/components/profiles/ProfileListTab.vue`, `dashboard/frontend/src/components/profiles/ProfileFormTab.vue`
  - **Verification**: `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build`

- [x] Task 5: Full Quality Verification Suite
  - **Scope**: Pre-flight checks across all stages
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-11-22: Initialized implementation plan and task plan for virtual overlay linkage, badging, guardrails, and plaintext decryption guidance.
- 2026-11-22: Completed Task 1. Added `remote` to safeKeys in RcloneClient `GetRemoteConfig`. Implemented `parseUnderlyingRemote` and bidirectional overlay linkage in `TelemetryService.GetRemotesSummaryCached`, populating `underlying_remote`, `underlying_path`, `crypt_overlay`, `hasher_overlay`, and `overlays`. Added unit test `TestTelemetryService_OverlayResolution`.
- 2026-11-22: Completed Task 2. Extended frontend RemoteEntry with overlay metadata, enhanced RemoteCard.vue with encrypted and accelerated badges on base and overlay cards, underlying storage breadcrumbs, duplicate wrap prevention, and added cloud ciphertext deletion safeguards in ViewRemotes.vue. Verified via `npm --prefix dashboard/frontend run build`.
- 2026-11-22: Completed Task 3. Enhanced ModalCryptWrap.vue with duplicate crypt overlay detection, duplicate remote name validation, and comprehensive plaintext sync/decryption guidance callouts across both ModalCryptWrap.vue and WizardStepConfigure.vue. Verified via `npm --prefix dashboard/frontend run build`.
- 2026-11-22: Completed Task 4. Added ENCRYPTED and ACCELERATED badges with pipeline topology breadcrumbs to ProfileListTab.vue. Added pipeline topology preview, encryption mode change resync warning, and overlay tip to ProfileFormTab.vue. Verified via `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build`.
- 2026-11-22: Completed Task 5. Verified full quality gate suite with `./scripts/check.sh all` (all 7/7 checks passed: Go format/shellcheck, Go vet, LOC audit, Go tests & coverage, race detector, Vue type-check, and Vite production build).







