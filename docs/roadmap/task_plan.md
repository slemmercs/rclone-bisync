# Task Plan: Provider-Aware Flag Builder, Scoped Catalog & Guidance

## Goal
Make the Profile Flag Builder and In-App Reference Manual provider-aware so that flags not applicable to a profile's cloud storage provider (e.g. Google Drive flags on a OneDrive remote) are strictly filtered out of the Flag Builder dropdown and catalog views, while modularizing `help-data.js` and `help.js` to ensure zero files approach the 400 LOC ceiling.

## Tasks

- [x] Task 1: Modularize Help Data & Diagnostics Subsystems
  - **Scope**: Create `dashboard/frontend/js/help-data-flags.js` & `dashboard/frontend/js/help_diagnostics.js`, trim `help-data.js` & `help.js`, register in `index.html`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh all`

- [x] Task 2: Provider Metadata & OneDrive Guidance in Flags Catalog
  - **Scope**: `dashboard/frontend/js/help-data-flags.js`
  - **Verification**: `go test -count=1 ./tests/e2e/...`

- [x] Task 3: Strictly Provider-Scoped Flag Builder & Quick-Add Pills
  - **Scope**: `dashboard/frontend/js/flags_builder.js`, `dashboard/frontend/partials/modals/modal_provider_options.html`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh all`

- [ ] Task 4: Provider-Aware Filtering & Badges in Help Drawer
  - **Scope**: `dashboard/frontend/js/help.js`, `dashboard/frontend/partials/components/component_help_drawer.html`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh all`

- [ ] Task 5: End-to-End Automated & Live Verification
  - **Scope**: `tests/e2e/dashboard_smoke_test.go`, `/tmp/rclone-bisync-dev`
  - **Verification**: `go test -v -count=1 ./tests/e2e/... && ./scripts/check.sh all`

## Status Updates

- 2026-11-06: Initialized plan from implementation_plan.md on branch feat/in-app-user-guidance.
- 2026-11-06: Completed Task 1: Modularized `help-data-flags.js` (269 LOC) and `help_diagnostics.js` (54 LOC), trimming `help-data.js` to 137 LOC and `help.js` to 188 LOC (all well below 400 LOC ceiling). Registered modules in `index.html` and `app.js`. Passed all 7 quality gate checks.
- 2026-11-06: Completed Task 2: Enriched `help-data-flags.js` with provider backend scopes (`all`, `onedrive`, `drive`), added 3 OneDrive-specific flags (`--onedrive-no-versions`, `--onedrive-chunk-size`, `--onedrive-delta`), and added operational caveats for OneDrive fast-list and Graph API HTTP 429 rate limiting (file size: 304 LOC). Verified with E2E suite.
- 2026-11-06: Completed Task 3: Updated `flags_builder.js` (276 LOC) with backend scope metadata and provider filtering helpers (`getAvailableFlagsForEditingRemote()`, `isFlagApplicableForEditingRemote()`). Updated `modal_provider_options.html` with provider-scoped `<option :hidden>` and dynamic quick-add pills for OneDrive and Google Drive. Passed all 7 quality gate checks.

