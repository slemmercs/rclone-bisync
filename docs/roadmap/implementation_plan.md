# Implementation Plan: Alpine Boolean Attribute Defense-in-Depth (Linting, Helper Standardization & Headless Chrome E2E)

## Proposed Changes

Implement comprehensive defense-in-depth prevention against Alpine.js v3 boolean attribute binding bugs through:
1. Static template linting (`scripts/lint-templates.sh`) integrated into `./scripts/check.sh`.
2. Standardization of Alpine component helper methods across all partials, eliminating dotted expressions from all boolean attribute bindings (`:disabled`, `:checked`, `:readonly`, etc.).
3. Automated Headless Chrome E2E smoke testing (`tests/e2e/dashboard_smoke_test.go`) auditing rendered button interactive states and asserting zero runtime console errors.

### Affected Components & Packages

- `scripts/lint-templates.sh`:
  - Standalone POSIX script scanning all HTML files in `dashboard/frontend/`.
  - Forbids any `.` inside boolean attribute bindings (`:(disabled|checked|readonly|hidden|required|selected)="[^"]*\.[^"]*"`).
- `scripts/check.sh`:
  - Integrated into Stage 2 (Static Analysis & Security).
- `dashboard/frontend/js/explorer.js`:
  - Add `isExplorerLoading()`, `isItemCopying(item)`, `isCanaryItem(item)`, `isDeleteModalDeleting()`.
- `dashboard/frontend/js/remotes.js`:
  - Add `isRemotesLoading()`, `isRemoteValidating(remote)`, `isMigrationLoading()`, `cannotSaveMigration()`.
- `dashboard/frontend/js/review.js`:
  - Add `isReviewDismissing()`, `isReviewRollingBack()`, `cannotApplyReview()`.
- `dashboard/frontend/js/audit.js`:
  - Add `isAuditVerifying()`, `isFirstAuditPage()`, `isLastAuditPage()`.
- `dashboard/frontend/partials/`:
  - Update `view_explorer.html`, `view_remotes.html`, `view_audit.html`, `modals/modal_delete.html`, `modals/modal_oauth_migration.html`, `modals/modal_review.html` to call helper methods with no dots.
- `dashboard/frontend/index.html`:
  - Increment script cache-busting query parameter to `?v=19`.
- `tests/e2e/dashboard_smoke_test.go`:
  - Automated headless Chrome smoke tests with `chromedp`.
  - Audits default button interactive states across tabs and verifies zero console errors/warnings.

### Technical Strategy & Data Flow

- Eliminating dotted property access from boolean bindings ensures Alpine's internal quirk (`c === void 0 && n.match(/\./) && (c = "")`) can never trigger.
- Dedicated component helper methods encapsulate business logic in clean, testable JavaScript functions returning strict booleans.
- The static linter enforces this contract at pre-flight, failing any PR or build that introduces a dotted boolean attribute binding.
- The `chromedp` E2E test runs against a mock backend, testing the actual rendered DOM in real Chrome without live cloud network dependencies.

### Safety & Error Handling

- Test gracefully skips when Chrome/Chromium is unavailable in minimal CI environments.
- Console error trapping catches any unhandled exceptions or syntax errors in frontend JavaScript during tests.
- Path confinement, canary safety, and advisory locks remain intact.

## Verification Plan

- `scripts/lint-templates.sh`: verify 0 template violations.
- `go test -v -count=1 ./tests/e2e/...`: verify headless Chrome smoke test passes cleanly and reports 0 disabled interactive buttons.
- `./scripts/check.sh`: verify all stages (fmt, lint, test, build) pass cleanly.




