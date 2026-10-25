# Implementation Plan: Alpine.js Runtime Syntax Fixes, Cloak FOUC Prevention & Modal Keyboard Accessibility

## Proposed Changes

Address Alpine.js runtime syntax limitations, eliminate initial page load FOUC, and complete WCAG-compliant keyboard escape navigation across all modal templates.

### Affected Components & Packages

- `dashboard/frontend/index.html`: Inject `<style>[x-cloak] { display: none !important; }</style>` in `<head>`.
- `dashboard/frontend/partials/modals/modal_daemon.html`: Fix array destructuring in `x-for`, add `x-cloak`, add `@keydown.escape.window="closeDaemonModal()"`.
- `dashboard/frontend/partials/view_telemetry.html`: Fix array destructuring in `filteredConfigEntries` `x-for`, add `x-cloak` to root view container.
- `dashboard/frontend/partials/modals/modal_review.html`: Add `x-cloak` and `@keydown.escape.window="closeReviewModal()"`.
- `dashboard/frontend/partials/modals/modal_deletions.html`: Add `x-cloak` and `@keydown.escape.window="showDeletionsModal = false"`.
- `dashboard/frontend/partials/modals/modal_reconnect.html`: Add `x-cloak` and `@keydown.escape.window="reconnectModal.show = false"`.
- `dashboard/frontend/partials/modals/modal_oauth_migration.html`: Add `x-cloak` and `@keydown.escape.window="closeMigrationWizard()"`.
- `dashboard/frontend/partials/modals/modal_auth_audit.html`: Add `x-cloak` and `@keydown.escape.window="auditModal.show = false"`.
- `dashboard/frontend/partials/modals/modal_delete.html`: Add `@keydown.escape.window="cancelDelete()"`.
- `dashboard/frontend/partials/view_explorer.html`: Add `x-cloak` to root container.
- `dashboard/frontend/partials/view_audit.html`: Add `x-cloak` to root container.
- `dashboard/frontend/partials/view_remotes.html`: Add `x-cloak` to root container.
- `dashboard/frontend/partials/toast.html`: Add `x-cloak` to notification container.
- `internal/server/server_test.go`: Update string assertion for modal_daemon details iteration.

### Technical Strategy & Data Flow

- Alpine.js `x-for` regex parser does not support array destructuring `[key, val] in collection`. Iterations are refactored to `(item, idx)` pairs, referencing index `item[0]` (key) and `item[1]` (val).
- `[x-cloak]` CSS rule hides all inactive views and modals until Alpine initializes and strips `x-cloak`.
- Escape key listeners (`@keydown.escape.window`) provide standard keyboard dismissal for all overlay modals.

### Safety & Error Handling

- All changes preserve HTML formatting, valid Go html/template syntax, and strict LOC under 400 lines per file.
- All template assembly tests in `server_test.go` and static validation (`check.sh`) verified clean.

## Verification Plan

- `go test -count=1 ./internal/server -run TestDashboardHandler_HTMLStructure`
- `./scripts/check-loc.sh 400`
- `./scripts/check.sh`
- Binary build & UI server verification: Launch `bisync-hub` and curl `/` to verify rendered HTML, style block, and lack of unparsed template directives.

