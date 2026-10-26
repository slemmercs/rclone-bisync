# Implementation Plan: Profile-Driven Dual-Pane Explorer Integration, Safeguarded Local Path Resolution & Custom Deletion Modal

## Proposed Changes

Integrate workspace profiles seamlessly with the dual-pane file explorer, safeguard local directory provisioning with home directory confinement, decouple configuration lifecycle events from file audit records, and enhance the dashboard UI with exclude-pattern visual dimming, pane orientation swapping, and a styled custom profile deletion confirmation modal.

### Affected Components & Packages

- `internal/profiles/model.go`: Add `ExcludePatterns []string` to `Profile` model and `ProfileHistorySnapshot`.
- `internal/profiles/store.go`: Support `ExcludePatterns` in JSON serialization and history persistence.
- `internal/profiles/service.go`:
  - Decouple audit domains: profile lifecycle operations record strictly into `profiles_history.json` with discriminator actions (`CREATED`, `UPDATED`, `CLONED`, `ROLLED_BACK`), reserving file audit ledger exclusively for file transfer operations.
  - Implement path expansion (`~`) and home directory auto-provisioning (`os.MkdirAll` for paths within `$HOME` or temporary directory).
  - Enforce safeguarding against OS permission errors by requiring non-home paths to exist prior to profile creation/update.
- `internal/server/handlers_profiles.go`: Return detected user `home_dir` in `GET /api/profiles` payload.
- `dashboard/frontend/js/profiles.js`:
  - Store `homeDir` and compute initial profile local path defaults (`${homeDir}/${remoteName || 'sync'}`).
  - Provide `isProfileDeleting()` helper for Alpine safety compliance.
  - Replace native browser `confirm()` with custom deletion confirmation modal state and methods (`promptDeleteProfile`, `cancelDeleteProfile`, `executeDeleteProfile`).
  - Support comma-separated exclude patterns in editing state.
- `dashboard/frontend/partials/modals/modal_profiles.html`:
  - Replace broken trash icon with complete Lucide trash SVG.
  - Add comma-separated exclude pattern input under directory settings.
  - Render action badges (`CREATED`, `UPDATED`, `CLONED`, `ROLLED_BACK`) in history view.
  - Embed native dark-themed profile deletion confirmation dialog.
- `dashboard/frontend/js/explorer.js`:
  - Bind left/right panes to active profile target and paths.
  - Implement `swapExplorerPanes()` toggle (`⇄`) and track swapped state.
  - Implement `isItemExcluded(item)` matching profile glob patterns.
  - Refactor into modular helpers to adhere to the 400 LOC ceiling.
- `dashboard/frontend/partials/view_explorer.html`:
  - Add active profile indicator and `⇄ Swap Panes` button in cockpit toolbar.
  - Dynamically display `Local` vs `Remote` pane badges and paths based on swapped state.
  - Apply `isItemExcluded(item)` visual dimming and `EXCLUDED` badge to filtered files/folders.
  - Modularize partials if needed to stay below the 400 LOC ceiling.
- `dashboard/frontend/js/review.js`: Integrate `isItemExcluded(item)` into row classification.

### Technical Strategy & Data Flow

- **Path Resolution & Safeguarding**:
  - `filepath.Clean` and `os.UserHomeDir` normalize paths with `~`.
  - Path confinement check tests if path has prefix of `homeDir` or `os.TempDir()`. If yes, auto-create directory with `0755`. If outside, check if directory exists; if not, return actionable error message instructing manual creation.
- **Audit Domain Decoupling**:
  - Profile mutations write strictly to `profiles_history.json` with an explicit `Action` discriminator. The `AuditLedger` remains strictly dedicated to file synchronization actions.
- **Dual-Pane Binding & Swapping**:
  - When profile changes, explorer rebinds Pane A and Pane B to the active profile's local and remote targets.
  - Swapping panes switches target, current path, and items between left and right without reloading.
- **LOC Compliance**:
  - Strictly maintain Go, JS, and HTML files below 400 lines by modularizing code or extracting sub-partials if necessary.

### Safety & Error Handling

- **Alpine Template Safety**: Ensure all `:disabled` and `:class` bindings avoid unsafe optional chaining on dotted properties by using Alpine component methods (e.g., `isProfileDeleting()`).
- **Canary File Preservation**: Never treat canary file `RCLONE_TEST` as an excluded file; preserve canary touch and inspection operations.
- **No Path Traversal**: Restrict local file exploration within permitted directories and prevent accidental root filesystem modifications.

## Verification Plan

- `go test -count=1 ./internal/profiles/...`: Verify path resolution, auto-provisioning, history actions, and validation logic.
- `go test -count=1 ./internal/server/...`: Verify `GET /api/profiles` returns `home_dir` and exclude patterns.
- `./scripts/check.sh all`: Run full pre-flight verification across all stages (vet, test, loc, template safety, formatting).
- Browser / smoke check: Test profile creation, deletion modal, pane swap, and exclude dimming.

