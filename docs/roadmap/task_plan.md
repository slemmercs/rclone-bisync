# Task Plan: Alpine.js Runtime Syntax Fixes, Cloak FOUC Prevention & Modal Keyboard Accessibility

## Goal
Fix Alpine.js destructuring syntax in `modal_daemon.html` and `view_telemetry.html`, inject global `x-cloak` style in `index.html`, add `x-cloak` to root view/modal containers, and attach `@keydown.escape.window` listeners across modals.

## Tasks

- [x] Task 1: Fix Alpine.js `x-for` array destructuring
  - **Scope**: `dashboard/frontend/partials/modals/modal_daemon.html`, `dashboard/frontend/partials/view_telemetry.html`, and `internal/server/server_test.go`
  - **Verification**: `go test -count=1 ./internal/server -run TestServer_RootHandler_RealFileSystem`

- [x] Task 2: Inject global `x-cloak` CSS and add `x-cloak` to root containers
  - **Scope**: `dashboard/frontend/index.html`, all partial views (`view_telemetry.html`, `view_explorer.html`, `view_audit.html`, `view_remotes.html`), `toast.html`, and modals
  - **Verification**: `./scripts/check.sh`

- [x] Task 3: Add `@keydown.escape.window` keyboard escape handlers across modals
  - **Scope**: `modal_review.html`, `modal_deletions.html`, `modal_daemon.html`, `modal_reconnect.html`, `modal_oauth_migration.html`, `modal_auth_audit.html`, `modal_delete.html`
  - **Verification**: `./scripts/check.sh`

- [x] Task 4: Build binary and run UI verification
  - **Scope**: Build single binary `bisync-hub` and verify UI rendering and templates
  - **Verification**: `go build -o bisync-hub . && ./scripts/check.sh && ./scripts/check-loc.sh 400`

## Status Updates

- 2026-10-25: Created plan for Alpine.js runtime syntax fixes, x-cloak FOUC prevention, and modal escape handlers.
- 2026-10-25: Fixed Alpine.js `x-for` destructuring in `modal_daemon.html` and `view_telemetry.html`.
- 2026-10-25: Added global `<style>[x-cloak] { display: none !important; }</style>` to `index.html`.
- 2026-10-25: Added `x-cloak` to all 7 modals, 4 tab views, and toast container.
- 2026-10-25: Attached `@keydown.escape.window` handlers to all 7 modals for accessible keyboard dismissal.
- 2026-10-25: Built new single binary `bisync-hub` and verified live server HTML rendering, style injection, and escape key event attributes.


