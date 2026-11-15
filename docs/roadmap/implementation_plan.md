# Implementation Plan: Google Drive Deduplication Fixes, In-App Modals & Streaming Resilience

## Proposed Changes

Deliver complete resolution for live Google Drive deduplication failures: resolve the path-truncation batch rename failure, prevent erroneous `--resync` triggers when dedupe batches fail, surface granular failure reasons, replace native browser `prompt()` with in-app dark-themed dialogs, fix blank dedupe directory explorer caused by Alpine `:key` collisions, lift streaming size caps (>2MB) on previews and downloads, and cache scan results across modal operations.

### 1. Dedupe Batch Engine Path Concatenation, Resync Safety & Error Reporting
- In `internal/services/dedupe_service.go`:
  - Combine target directory `req.Path` with candidate `action.Path` when not already prefixed:
    ```go
    cleanPath := strings.Trim(strings.TrimSpace(action.Path), "/")
    reqPath := strings.Trim(strings.TrimSpace(req.Path), "/")
    if reqPath != "" && !strings.HasPrefix(cleanPath, reqPath+"/") && cleanPath != reqPath {
        cleanPath = fmt.Sprintf("%s/%s", reqPath, cleanPath)
    }
    ```
  - Derive `dstPath` relative to the resolved directory:
    ```go
    dstDir := filepath.Dir(cleanPath)
    dstPath := newName
    if dstDir != "." && dstDir != "" {
        dstPath = fmt.Sprintf("%s/%s", strings.Trim(dstDir, "/"), newName)
    }
    ```
  - Guard resync execution: only trigger `TriggerSyncWithOptions` if `req.TriggerResync && resp.SuccessCount > 0`.
  - Provide concrete error messages: if all actions fail, populate `resp.Error` with `resp.Errors[0]` instead of generic `"All staged dedupe actions failed"`.
  - Record dedupe attempt in audit ledger (`DEDUPE_EXECUTED`) even on partial failure or failure with complete diagnostic metadata.

### 2. Frontend In-App Styled Prompt Modal & Detailed Batch Error Diagnostics
- In `dashboard/frontend/partials/modals/modal_confirm.html`:
  - Add conditional text input field (`confirmModal.showInput`) with label, placeholder, and two-way binding (`confirmModal.inputValue`).
  - Style with Tailwind dark theme (Slate-950 input, Amber border focus, Enter to submit, Escape to dismiss).
- In `dashboard/frontend/js/app_state.js`:
  - Implement `askPrompt({ title, message, inputLabel, inputValue, placeholder, confirmText, confirmClass, onConfirm })`.
- In `dashboard/frontend/js/dedupe_batch.js`:
  - Replace `prompt(...)` with `this.askPrompt(...)`.
  - Surface granular errors from `data.errors` in UI toasts and append error traces into `this.dedupeModal.output`.

### 3. Dedupe Workspace Directory Explorer Resilience & Inline Candidate Actions
- In `dashboard/frontend/partials/modals/modal_dedupe.html`:
  - Replace `:key="item.Path || item.Name"` in the directory explorer loop with `:key="getDedupeExplorerItemRowKey(item, idx)"` to prevent DOM key collisions on duplicate files.
  - Add amber `DUPLICATE` badges and inline quick actions (**Keep**, **Rename**, **Delete**, **Preview**) on colliding rows.
- In `dashboard/frontend/js/dedupe_workspace.js`:
  - Implement `getDedupeExplorerItemRowKey(item, idx)`.
  - Add dedupe directory inline action handlers.

### 4. Large File (>2MB) Preview & Download Support
- In `internal/core/client/fs_read.go`:
  - Forward explicit `maxSize` to `/operations/getfile` instead of defaulting to Rclone's 2MB limit.
- In `internal/services/explorer_preview.go`:
  - Forward `MaxImagePreviewSize` as `maxSize` for images.
  - Catch `"exceeds maximum limit"` gracefully and return HTTP 200 with `Truncated: true` and direct download guidance.
- In `internal/services/explorer_download.go`:
  - Pass file size as `maxSize` to `CatStream` and support direct streaming when `--rc-serve` is enabled.
- In `deploy/systemd/rclone-daemon.service`:
  - Add `--rc-serve` flag to daemon command.

### 5. Dedupe Scan Session Caching & Quality Pre-Flight
- In `dashboard/frontend/js/dedupe_workspace.js`:
  - Cache scan results in `this._dedupeCache` by `${remote}:${path}` with a 15-minute TTL.
- Validate through `./scripts/check.sh all` and `./scripts/check-loc.sh 400`.

## Verification Plan

- Unit Tests:
  - `go test -count=1 ./internal/services/... -run "Test.*DedupeBatch.*|Test.*Preview.*|Test.*Download.*"`
  - `go test -count=1 ./internal/server/...`
- E2E and Quality gates:
  - `./scripts/check.sh templates`
  - `./scripts/check-loc.sh 400`
  - `./scripts/check.sh all`




