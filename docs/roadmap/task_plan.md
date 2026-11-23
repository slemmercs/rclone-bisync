# Task Plan: GitHub Issues Realignment & Multi-Profile Review Isolation Bugfix

## Goal
Reconcile and update GitHub feature request issues (#98, #99, #100, #101), deliver the per-profile review isolation bugfix so pending reviews only block their owning profile, run verification, and merge `refactor/frontend-component-decomposition` to `main`.

## Tasks

- [x] Task 1: Realign and update GitHub feature request issues (#98, #99, #100, #101) via `gh` CLI
  - **Scope**: GitHub issue tracker (#98, #99, #100, #101, and new split issue for remote quota/trash)
  - **Verification**: `gh issue list --state all`

- [x] Task 2: Implement multi-profile review isolation in backend services and API
  - **Scope**: `internal/models/review.go`, `internal/services/review_service.go`, `internal/services/sync_actions.go`, `internal/server/handlers_review.go`, `internal/services/telemetry_service.go`
  - **Verification**: `go test -count=1 ./internal/services -run TestReview` and `go test -count=1 ./internal/server -run TestReview`

- [x] Task 3: Align frontend stores with profile-scoped review queries
  - **Scope**: `dashboard/frontend/src/stores/review.ts`, `dashboard/frontend/src/stores/telemetry.ts`, `dashboard/frontend/src/types/models.ts`, `dashboard/frontend/src/types/models.gen.ts`, `dashboard/frontend/src/components/ConflictReviewModal.vue`
  - **Verification**: `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build`

- [x] Task 4: Full pre-flight quality verification and merge to main
  - **Scope**: Pre-flight verification across all 7 stages and branch merge
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-11-23: Completed Task 4. Ran full pre-flight verification `./scripts/check.sh all` (all 7/7 checks passed in ~35s). Committed changes on `refactor/frontend-component-decomposition` and merged branch into `main`. Clean working tree and all tests passing on `main`.

- 2026-11-23: Completed Task 3. Aligned frontend stores (`review.ts`, `telemetry.ts`, `profiles.ts`) to query review sessions scoped by active profile, prevent spurious `REVIEW_REQUIRED` state transitions for unaffected profiles, and display profile badges in `ConflictReviewModal.vue`. Verified via `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build`.
- 2026-11-23: Completed Task 2. Implemented per-profile scoping for `ReviewService.HasBlockingReview(profileID ...string)` and `GetActiveSession(ctx, profileID ...string)`. Tagged review sessions and pending deletion events with `profile_id`. Extended telemetry response with `review_profile_id`. Verified with `go test -count=1 ./internal/services -run TestReview` and `go test -count=1 ./internal/server -run TestReview`.
- 2026-11-23: Completed Task 1. Reconciled and updated GitHub issues: closed #98 (delivered in-tree), updated #99 (container stack), updated #100 (control-plane safety gates, marked DynamicProviderForm delivered), updated #101 (pure-RC storage toolbox), and created #102 (remote quota & empty trash). Verified with `gh issue list`.
- 2026-11-23: Initialized task plan for GitHub issues realignment and per-profile review isolation bugfix.









