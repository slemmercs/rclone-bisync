# Task Plan: Profile Configuration Redesign & Guided Flag Builder

## Goal
Implement Active Strategic Priority #1: Reclassify provider options, fix boolean checkbox persistence bugs in profile settings, deliver the interactive `+ Add Flag` row builder with real-time CLI preview, and ensure all JS/Go modules remain strictly below 400 LOC.

## Tasks

- [x] Task 1: Backend Boolean Serialization & Provider Flags Enhancement
  - **Scope**: `internal/profiles/builder.go`, `internal/profiles/builder_test.go`
  - **Verification**: `go test -v -count=1 ./internal/profiles/...`

- [x] Task 2: Modular Frontend Flags Builder Module
  - **Scope**: `dashboard/frontend/js/flags_builder.js`, `dashboard/frontend/index.html`
  - **Verification**: `./scripts/check-loc.sh 400` and `go test -count=1 ./cmd/check/...`

- [x] Task 3: Profile Modal State Integration & Checkbox Persistence Fix
  - **Scope**: `dashboard/frontend/js/profiles_modal.js`
  - **Verification**: Verify row state reactivity and `./scripts/check-loc.sh 400`

- [ ] Task 4: Provider Options & Guided Flag Builder UI Overhaul
  - **Scope**: `dashboard/frontend/partials/modals/modal_provider_options.html`, `dashboard/frontend/partials/modals/modal_profiles.html`
  - **Verification**: `./scripts/check.sh all` (HTML tokenizer, linters, tests)

- [ ] Task 5: End-to-End Verification & Full Quality Gate
  - **Scope**: `tests/e2e/dashboard_smoke_test.go`, `./scripts/check.sh all`
  - **Verification**: Complete 7-stage quality check suite passing 100%

## Status Updates

- 2026-11-01: Initialized implementation plan and task plan for Active Strategic Priority #1 (Profile Redesign & Guided Flag Builder).
- 2026-11-02: Completed Task 1 (Backend boolean serialization, explicit `=false` provider flags, `--drive-shared-with-me`, and RC compiler `=value` parser support). Passed all unit tests.
- 2026-11-02: Completed Task 2 (Modular Frontend Flags Builder module `flags_builder.js` with curated catalog, token parser, row serializer, and live CLI preview; registered in `index.html` and `app.js`). Passed all 7 quality gates.
- 2026-11-02: Completed Task 3 (Profile Modal State Integration & Checkbox Persistence Fix in `profiles_modal.js` with `getProviderBoolOption` / `setProviderBoolOption`, `initFlagsBuilder` on open, and `syncFlagsToCustomFlags` on save). Passed all 7 quality gates including headless Chrome E2E.











