# Implementation Plan: Curate OneDrive Target Provider Options (Delta & Chunk Size)

## Proposed Changes

Add first-class curated controls for OneDrive in **Target Provider Options** (Tier 1) for `--onedrive-delta`, alongside `--onedrive-no-versions` and `--onedrive-chunk-size`, and ensure CLI command synthesis and backend argument compilation properly reflect all OneDrive provider options.

### Affected Components & Packages

- `dashboard/frontend/partials/modals/modal_provider_options.html`:
  - Enhance OneDrive section with two toggle options: `No Versions (Prevent duplicates)` and `Delta Enumeration (--onedrive-delta)`.
  - Provide `Upload Chunk Size` input in dedicated grid row with helper text `(e.g. 10M, 50M)`.
  - Remove `--onedrive-delta` from the custom flags `<select>` dropdown now that it is curated in Tier 1.
- `dashboard/frontend/js/flags_builder.js`:
  - Remove `--onedrive-delta` from `RCLONE_FLAGS_CATALOG` (curated in Target Provider Options).
  - Add `onedrive_delta` serialization to `synthesizeBisyncCommand()`.
  - Ensure `onedrive_chunk_size` and `onedrive_no_versions` are properly serialized into synthesized CLI command when active.
- `internal/profiles/builder.go` & `internal/profiles/builder_test.go`:
  - Add `onedrive_delta` handling in `appendProviderArgs()`.
  - Add unit test coverage in `builder_test.go`.
- `dashboard/frontend/js/help-data-flags.js`:
  - Update caveat for `--onedrive-delta` indicating it is configured directly in Target Provider Options.
- `tests/e2e/dashboard_smoke_test.go`:
  - Update E2E test to verify OneDrive Target Provider Options controls and absence of duplicate custom flag options.

## Verification Plan

- `go test -v -count=1 ./internal/profiles/...`: Verify `appendProviderArgs` with `onedrive_delta`.
- `go test -v -count=1 ./tests/e2e/...`: Verify headless Chrome testing of provider options and drawer.
- `./scripts/check.sh all`: Run all 7 quality stages.
- Rebuild binary and restart `rclone-dashboard.service`.
