# Implementation Plan: Provider-Aware Flag Builder, Scoped Catalog & Guidance

## Proposed Changes

Implement strict provider-aware filtering in the Profile Flag Builder and the In-App Reference Manual. When editing a sync profile, operational flags not applicable to that profile's cloud storage provider (e.g. Google Drive flags on a OneDrive remote) are strictly filtered out of the Flag Builder dropdown and quick-add pills. Furthermore, the In-App Reference Manual reflects provider-specific caveats and automatically scopes its catalog to relevant flags.

To adhere strictly to engineering standards (<400 LOC ceiling per file) and keep the frontend modular and maintainable:
1. `dashboard/frontend/js/help-data-flags.js`: Dedicated flags catalog module (~230 LOC) containing complete guided flags with backend scopes (`backends: ["all"]`, `["onedrive"]`, `["drive"]`), Bisync operational notes, provider-specific caveats, and outbound `rclone.org` documentation links.
2. `dashboard/frontend/js/help-data.js`: Modularized policies and runbooks catalog (~140 LOC).
3. `dashboard/frontend/js/help_diagnostics.js`: Dedicated diagnostics execution and check filtering module (~80 LOC).
4. `dashboard/frontend/js/help.js`: Reactive controller managing drawer state, tab switching, and context/provider filtering (~160 LOC).
5. `dashboard/frontend/js/flags_builder.js` & `dashboard/frontend/partials/modals/modal_provider_options.html`: Strictly filter dropdown options and quick-add pills by `getEditingRemoteType()`.

### Affected Components & Packages

- `dashboard/frontend/js/help-data-flags.js`: New static module registering `window.HelpCatalog.flags` with backend classifications and provider caveats.
- `dashboard/frontend/js/help-data.js`: Refactored to focus on policies, runbooks, and base registry initialization.
- `dashboard/frontend/js/help_diagnostics.js`: New module encapsulating diagnostics API fetching and check filtering.
- `dashboard/frontend/js/help.js`: Slimmed controller integrating diagnostics module and backend-aware catalog filtering.
- `dashboard/frontend/js/flags_builder.js`: Implements `getAvailableFlagsForEditingRemote()` and provider-scoped quick pills.
- `dashboard/frontend/partials/modals/modal_provider_options.html`: Dynamic `<select>` rendering and provider-specific quick-add pills.
- `dashboard/frontend/partials/components/component_help_drawer.html`: Backend filter badges and provider caveat rendering.
- `dashboard/frontend/index.html`: Load new modular script files in proper dependency order.
- `tests/e2e/dashboard_smoke_test.go`: End-to-end headless Chrome tests validating provider-scoped dropdown options, quick-add pills, and help drawer provider filtering.

### Technical Strategy & Data Flow

1. **Provider Mapping & Scoping**:
   - Universal flags: Included on all remotes (`--track-renames`, `--checkers`, `--transfers`, `--bwlimit`, `--tpslimit`, `--buffer-size`, `--resilient`, `--recover`, etc.).
   - OneDrive: Universal + `--onedrive-no-versions`, `--onedrive-chunk-size`, `--onedrive-delta`. Excludes `--drive-shared-with-me`.
   - Google Drive: Universal + `--drive-shared-with-me`, `--fast-list`. Excludes `--onedrive-...`.
   - Local / Generic: Universal flags only.
2. **Flag Builder Dynamic Scoping**:
   - `getAvailableFlagsForEditingRemote()` inspects `getEditingRemoteType()`.
   - The `<select>` options in `modal_provider_options.html` use this filtered list.
3. **Help Drawer Provider Awareness**:
   - `detectActiveContext()` extracts active profile remote backend when editing a profile.
   - Flag catalog highlights provider badges and renders specific operational caveats (e.g. OneDrive `--fast-list` requiring `--onedrive-delta`, symlink limitations, checksum compatibility).

### Safety & Error Handling

- Zero build step: Vanilla JS distribution.
- Safe fallbacks: If remote type is unknown or unset, defaults to universal flags.
- Custom flag option remains available in all modes.
- Every JS file kept strictly under 300 LOC (well below 400 LOC ceiling).

## Verification Plan

- `./scripts/check-loc.sh 400`: Verify no file exceeds 400 LOC.
- `./scripts/check.sh all`: Full lint, template scan, security check, and test suite.
- `tests/e2e/dashboard_smoke_test.go`: Automated test verifying OneDrive/GDrive provider scoping.
- Dev binary verification on port 8085.
