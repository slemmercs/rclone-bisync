# Task Plan: Deduplicate Target Provider Options from Custom Flag Builder

## Goal
Remove redundant quick-add pills and flag builder select options for flags already managed by dedicated controls in Target Provider Options (`--drive-shared-with-me`, `--onedrive-no-versions`, `--onedrive-chunk-size`), while preserving full documentation and provider filtering in the Reference Manual Help Drawer.

## Tasks

- [x] Task 1: Deduplicate Flag Builder Catalog & Quick-Add Buttons
  - **Scope**: `dashboard/frontend/js/flags_builder.js`, `dashboard/frontend/partials/modals/modal_provider_options.html`, `dashboard/frontend/js/help-data-flags.js`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh all`

- [ ] Task 2: Update E2E Headless Chrome Tests & Live Service Verification
  - **Scope**: `tests/e2e/dashboard_smoke_test.go`, compilation & restart of `rclone-dashboard.service`
  - **Verification**: `go test -v -count=1 ./tests/e2e/... && ./scripts/check.sh all`

## Status Updates

- 2026-11-06: Initialized task plan based on user decision to remove dedicated provider options from Custom Flags & Quick Add.
- 2026-11-06: Completed Task 1: Removed `--drive-shared-with-me`, `--onedrive-no-versions`, and `--onedrive-chunk-size` from `flags_builder.js` and `modal_provider_options.html`. Updated caveats in `help-data-flags.js`. Passed all 7 quality gates.


