# Implementation Plan: Coverage Buffer (>86%), Multi-Upload, File Ops Filter, & File-Level Deep Nav

## Overview
Address the nightly deep quality GitHub Actions failure (`test coverage 84.9% is below required threshold 85.0%`) by boosting coverage to >86.0%. Concurrently resolve the three user-reported Explorer and Log friction points:
1. Multi-file batch upload & desktop drag-and-drop in Directory Explorer.
2. File-level deep navigation and pulsing from audit logs for files uploaded into subdirectories.
3. Dedicated "File Ops" log filtering in both the Live Sync Log Dock and Telemetry Timeline Ledger.

---

## Proposed Changes

### Phase 1: Elevate Test Coverage Buffer (>86.0%)
- `internal/models/provider_test.go`:
  - Unit tests for `IsOAuthProvider()` across supported provider types (`drive`, `dropbox`, `onedrive`, `s3`, `b2`, `sftp`, `local`). Elevate `internal/models` from 50% to 100%.
- `internal/services/explorer_ops_test.go`:
  - Targeted unit tests for `UploadFile()` error branches: jail violations, nil streams, directory target paths, and Rclone RC remote upload mocking. Elevate `UploadFile()` from 45.8% to >85%.
- `internal/server/handlers_explorer_test.go`:
  - Test `resolveExcludePatterns()` with nil profiles, missing profiles, and custom exclude lists. Elevate from 45.5% to >90%.
- `internal/services/explorer_preview_test.go`:
  - Test `DetectMimeType()` edge cases (svg, json, markdown, zip, binary, empty files). Elevate from 37.9% to >85%.

### Phase 2: Multi-File Selection & Desktop Drag-and-Drop
- `dashboard/frontend/partials/view_explorer.html`:
  - Add `multiple` attribute to `#explorer-upload-left` and `#explorer-upload-right` file inputs.
- `dashboard/frontend/js/explorer_actions.js`:
  - Update `handleFileUpload(paneKey, event)` to iterate over `Array.from(files)` sequentially.
  - Display progress toasts (`Uploading 3 files...` -> `Uploaded 3 files successfully` / warning on partial failure).
  - Automatically set `this.explorer.highlightName = files[0].name` to highlight the newly uploaded item(s).
  - Update `onDrop(targetPaneKey, event)` to check `event.dataTransfer?.files` and trigger `handleFileUpload` directly when external desktop files are dropped onto an explorer pane.
  - Maintain LOC < 400.

### Phase 3: File-Level Deep Navigation & Audit Path Resolution
- `internal/server/handlers_explorer.go`:
  - In `handleExplorerUpload`, include `"item_path": path.Join(targetPath, header.Filename)` in the `ActionWorkspaceFileUploaded` audit details map.
- `dashboard/frontend/js/utils.js`:
  - In `parseJsonLogLine`, prioritize `item_path` or combined `path + "/" + filename` when determining `entry.object` so uploaded files in subfolders resolve to their full relative path (`subfolder/file.ext`) rather than the directory path alone.
  - Maintain LOC < 400.
- `dashboard/frontend/js/logs.js`:
  - In `jumpToExplorerPath(targetPath, profileId)`:
    - Sanitize `profileId`: parse string ID if passed as string, or extract `profileId?.profile_id` if passed as an entry object.
    - Navigate to directory `dir` and pulse `filename` row via `scrollToHighlightedRow(filename)`.
  - Maintain LOC < 400.

### Phase 4: Strict File/Folder Operation Log Filtering
- `dashboard/frontend/js/log_dock.js`:
  - Update `getFilteredDockLogs()` to implement strict `isFileOperationLog(entry)` logic:
    - Matches entries with an explicit `object` or `getEntryPath(entry)` (relative file/dir path).
    - Matches workspace staging and file mutation audit actions (`WORKSPACE_FILE_UPLOADED`, `WORKSPACE_ITEM_COPIED`, `WORKSPACE_ITEM_MOVED`, `WORKSPACE_ITEM_DELETED`, `WORKSPACE_DIR_CREATED`, `WORKSPACE_DIR_PURGED`, `ITEM_RESTORED`, `CONFLICT_RESOLVED`, `FILE_DELETED`, `FILE_TRANSFER`, `FILE_DIFF`).
    - Discards false positives: generic `NOTICE`/`ERROR` daemon startup messages, dry-run banners, and summary statements like `(0 file(s) transferred)`.
  - Change dock filter tab from `mutations` to `file_ops` with label `File Ops`.
- `dashboard/frontend/partials/components/component_log_dock.html`:
  - Update dock filter buttons: `[All] [File Ops] [Errors]` with explanatory tooltip.
- `dashboard/frontend/partials/patterns/pattern_timeline_ledger.html` & `dashboard/frontend/js/logs.js`:
  - Add `file_ops` filter pill to the Telemetry Timeline ledger so users can filter the full audit stream to file/folder operations.

---

## Safety & Error Handling

- **Coverage Safety**: Elevate test coverage to >86.0% (comfortably above the 85.0% gate) under `-race -covermode=atomic` across all Go packages.
- **Local Jail Enforcement**: Ensure multi-file upload strictly verifies directory confinement for every file before creating disk handles.
- **LOC Boundaries**: Every modified Go and JS file must remain strictly under the 400 LOC project limit (`scripts/check.sh loc 400`).
- **Canary File Protection**: Preserve canary file safeguards against overwrite or deletion during batch uploads.

---

## Verification Plan

- `MIN_COVERAGE=85.0 RUN_RACE=true ./scripts/check.sh test` (Must pass with >86.0% coverage)
- `go test -count=1 -v ./internal/models -run TestIsOAuthProvider`
- `go test -count=1 -v ./internal/services -run TestUploadFile`
- `go test -count=1 -v ./internal/server -run TestHandleExplorerUpload`
- `go test -count=1 -v ./tests/e2e -run TestDashboard_HeadlessChromeSmoke`
- `./scripts/check.sh templates`
- `./scripts/check.sh loc 400`
- `./scripts/check.sh all` (all 7 quality gates passing)

