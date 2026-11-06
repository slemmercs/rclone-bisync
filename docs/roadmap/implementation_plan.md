# Implementation Plan: Deduplicate Target Provider Options from Custom Flag Builder

## Proposed Changes

Eliminate UI and configuration duplication between curated **Target Provider Options** (Tier 1) and **Operational & Custom CLI Flags** (Tier 2/3). Provider settings that have first-class checkboxes or input fields in "Target Provider Options" (`--drive-shared-with-me`, `--onedrive-no-versions`, and `--onedrive-chunk-size`) will no longer appear as redundant quick-add pills or `<select>` dropdown options in the Flag Builder. They will remain fully documented and scoped in the In-App Reference Manual (Help Drawer) with clear guidance that they are configured via Target Provider Options.

### Affected Components & Packages

- `dashboard/frontend/js/flags_builder.js`:
  - Remove `--drive-shared-with-me`, `--onedrive-no-versions`, and `--onedrive-chunk-size` from `RCLONE_FLAGS_CATALOG` so they are not offered in the custom flags builder.
  - Keep `--onedrive-delta` in `RCLONE_FLAGS_CATALOG` scoped to `onedrive` (as it has no dedicated Target Provider Options checkbox).
- `dashboard/frontend/partials/modals/modal_provider_options.html`:
  - Remove provider-specific quick-add buttons (`--onedrive-no-versions`, `--onedrive-chunk-size`, `--drive-shared-with-me`) from the Quick add row.
  - Remove redundant `<option>` entries for `--drive-shared-with-me`, `--onedrive-no-versions`, and `--onedrive-chunk-size` from the flag select dropdown.
- `dashboard/frontend/js/help-data-flags.js`:
  - Retain full catalog entries in the Reference Manual.
  - Note in the descriptions of `--drive-shared-with-me`, `--onedrive-no-versions`, and `--onedrive-chunk-size` that they are conveniently configured via Target Provider Options.
- `tests/e2e/dashboard_smoke_test.go`:
  - Update Sub-test 8 `ProviderAwareFlagScopingAndDrawerIsolation` to verify that:
    1. Dedicated provider settings (`--onedrive-no-versions`, `--drive-shared-with-me`) are NOT in the custom flag builder dropdown.
    2. Provider-scoped non-curated flags (e.g. `--onedrive-delta`) are applicable for OneDrive and excluded for Google Drive in the flag builder.
    3. The Help Drawer still documents and filters both provider flags correctly in context.

### Technical Strategy & Data Flow

- Clean separation of concerns:
  - **Tier 1 (Target Provider Options)**: Curated provider-specific options stored in `profile.flags.provider_options`.
  - **Tier 2 (Operational & Custom CLI Flags)**: Bisync operational parameters (`--track-renames`, `--fast-list`, `--checkers`, etc.) and un-curated provider flags (`--onedrive-delta`) stored in `profile.flags.custom_flags`.
  - **Reference Manual (Help Drawer)**: Complete reference documentation of all flags with backend badges.

## Verification Plan

- `tests/e2e/dashboard_smoke_test.go`: Automated test verifying clean separation, provider scoping in drawer, and absence of duplicate flag builder options.
- `./scripts/check.sh all`: Run all 7 static analysis, template, and test gates.
- Compile and restart `rclone-dashboard.service`, then verify UI.
