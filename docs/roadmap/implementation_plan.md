# Implementation Plan: RC Argument Compilation, Rename Tracking Flags & Review Session Cache Fix

## Proposed Changes

Fix Rclone RC argument compilation for profile flags (specifically `--track-renames`, `--track-renames-strategy`, and `--max-delete` percent vs count), resolve ghost barrier review sessions after file deletion, and fix the HTML `<select>` catalog options rendering bug in the profile flags builder.

### Affected Components & Packages

- `internal/core/compiler/compiler.go`:
  - Add `TypeParamMaxDelete` to handle both percentage values (`"5%"`) and numeric counts (`"5"` or `50`). Strip `%` and parse as integer `intVal`, populating `bisyncParams["maxDelete"] = intVal`.
  - Reclassify `--track-renames` from `TypeParamBool` to `TypeConfigBool` with target `"TrackRenames"` in `_config`.
  - Add `--track-renames-strategy` as `TypeConfigStr` with target `"TrackRenamesStrategy"` in `_config`.
- `internal/core/compiler/compiler_test.go`:
  - Unit tests validating `--max-delete 5%`, `--max-delete=10`, `--track-renames`, and `--track-renames-strategy modtime,leaf` compilation.
- `internal/services/review_execution.go`:
  - Clear in-memory session cache (`s.session = nil`) when session file on disk does not exist (`os.IsNotExist(err)`).
- `internal/services/review_service_test.go`:
  - Regression test verifying that deleting the session file on disk clears `GetActiveSession()` to `nil`.
- `dashboard/frontend/partials/modals/modal_provider_options.html`:
  - Replace `<template x-for>` inside `<select>` with standard `<option>` tags to prevent Chromium HTML parser from stripping catalog options.
- `dashboard/frontend/js/flags_builder.js`:
  - Add `--track-renames` (`isBool: true`) and `--track-renames-strategy` (`isBool: false`, placeholder `modtime,leaf`) to `RCLONE_FLAGS_CATALOG`.
- `tests/e2e/dashboard_smoke_test.go`:
  - E2E test verifying that profile modal dropdown renders all curated flags and enables boolean switch without stripping.

### Technical Strategy & Data Flow

- **MaxDelete**: Bisync RC endpoint expects `maxDelete: int` representing count or percent number. CLI flags `--max-delete 5%` or `--max-delete=5%` have trailing `%` stripped and converted to integer before sending in JSON payload.
- **_config vs params**: Rclone 1.68+ bisync handles global config options (`TrackRenames`, `TrackRenamesStrategy`) in `_config` dictionary (`fs.ConfigInfo`).
- **Session Cache**: `ensureSessionLoadedLocked()` checks file existence; if missing, `s.session = nil` ensures API returns `null` rather than a deleted ghost session.

### Safety & Error Handling

- **Backward Compatibility**: Existing profiles continue working without migration.
- **Fail-Safe Compilation**: Invalid integer formats in `--max-delete` fallback gracefully without crashing.
- **POSIX Advisory Locking**: Review service preserves existing lock isolation during session loads.

## Verification Plan

- `go test -v -count=1 ./internal/core/compiler/...`
- `go test -v -count=1 ./internal/services/... -run "TestReview"`
- `go test -v -count=1 ./tests/e2e/...`
- Full check suite: `./scripts/check.sh all`









