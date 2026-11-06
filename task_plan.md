# Task Plan: Guided Flag Catalog & Rclone Reference Documentation Integration

## Goal
Expand the in-app interactive reference manual in Bisync Hub to document all 18+ guided operational flags from the Sync Profiles Manager dropdown with Bisync-specific guidance and direct outbound links to official `rclone.org` documentation, modularizing catalog data into `help-data.js` to respect the <400 LOC boundary.

## Tasks

- [x] Task 1: Create `help-data.js` Catalog & Script Registration
  - **Scope**: `dashboard/frontend/js/help-data.js`, `dashboard/frontend/index.html`
  - **Verification**: `./scripts/check.sh templates` and `./scripts/check.sh all`

- [ ] Task 2: Refactor `help.js` to Consume `window.HelpCatalog`
  - **Scope**: `dashboard/frontend/js/help.js`
  - **Verification**: `go test -count=1 ./tests/e2e/...` and `./scripts/check.sh all`

- [ ] Task 3: Render Outbound `rclone.org` Links in Drawer Template
  - **Scope**: `dashboard/frontend/partials/components/component_help_drawer.html`
  - **Verification**: `./scripts/check.sh templates` and `./scripts/check.sh all`

- [ ] Task 4: Headless Chrome E2E Verification for Expanded Catalog
  - **Scope**: `tests/e2e/dashboard_smoke_test.go`
  - **Verification**: `go test -count=1 -v ./tests/e2e/...` and `./scripts/check.sh all`

- [ ] Task 5: Rebuild Dev Binary & Live Verification on Port 8085
  - **Scope**: `/tmp/rclone-bisync-dev`
  - **Verification**: `curl -s http://127.0.0.1:8085/api/diagnostics | jq .status`

## Status Updates

- 2026-11-06: Initialized plan from implementation_plan.md on branch feat/in-app-user-guidance.
- 2026-11-06: Completed Task 1: Created static reference catalog `dashboard/frontend/js/help-data.js` documenting all 18+ guided operational flags with `rclone.org` URLs, safety policies, and runbooks (340 LOC), registered script tag in `index.html`. Verified with `./scripts/check.sh all`.

