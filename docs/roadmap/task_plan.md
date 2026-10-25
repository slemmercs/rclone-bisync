# Task Plan: Alpine Boolean Attribute Defense-in-Depth

## Goal
Implement static template linting, standardized helper methods for all boolean attribute bindings, and automated headless Chrome E2E smoke tests.

## Tasks

- [x] Task 1: Add helper methods to frontend JS modules
  - **Scope**: `dashboard/frontend/js/explorer.js`, `dashboard/frontend/js/remotes.js`, `dashboard/frontend/js/review.js`, `dashboard/frontend/js/audit.js`
  - **Verification**: `./scripts/check-loc.sh 400`

- [x] Task 2: Refactor HTML partials to use dot-free helper methods
  - **Scope**: `dashboard/frontend/partials/` (`view_explorer.html`, `view_remotes.html`, `view_audit.html`, `modals/*.html`), bump `index.html` asset version to `?v=19`
  - **Verification**: `./scripts/check-loc.sh 400`

- [x] Task 3: Implement static template linting script and integrate into `check.sh`
  - **Scope**: `scripts/lint-templates.sh`, `scripts/check.sh`
  - **Verification**: `./scripts/lint-templates.sh` && `./scripts/check.sh lint`

- [x] Task 4: Add `chromedp` dependency and implement Headless Chrome E2E Smoke Test
  - **Scope**: `go.mod`, `go.sum`, `tests/e2e/dashboard_smoke_test.go`
  - **Verification**: `go test -v -count=1 ./tests/e2e`

- [ ] Task 5: Full verification & pre-flight commit
  - **Scope**: `./scripts/check.sh`
  - **Verification**: All stages pass cleanly

## Status Updates

- 2026-10-25: Initialized plan from implementation_plan.md on branch `feat/alpine-boolean-lint-and-e2e`.
- 2026-10-25: Completed Task 1: Added standardized helper methods (`isExplorerLoading`, `isItemCopying`, `isCanaryItem`, `isDeleteModalDeleting`, `isRemotesLoading`, `isRemoteValidating`, `isMigrationLoading`, `cannotSaveMigration`, `isReviewDismissing`, `isReviewRollingBack`, `cannotApplyReview`, `isAuditVerifying`, `isFirstAuditPage`, `isLastAuditPage`) across frontend JS modules. Verified all under 400 LOC.
- 2026-10-25: Completed Task 2: Refactored all partials to eliminate dotted boolean attribute expressions and bumped asset version to `?v=19` in `index.html`.
- 2026-10-25: Completed Task 3: Created `scripts/lint-templates.sh` forbidding dotted boolean attributes and integrated into `scripts/check.sh` Stage 2. Verified 0 template violations.
- 2026-10-25: Completed Task 4: Implemented Headless Chrome E2E Smoke Test in `tests/e2e/dashboard_smoke_test.go` verifying dashboard rendering, tab switching, and button interactive states with zero runtime JavaScript errors. Verified with `go test -v -count=1 ./tests/e2e/...`.






