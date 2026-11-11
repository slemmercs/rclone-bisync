# Implementation Plan: Enforce UI-Aligned Enums & Validation for MCP & Profile Configurations

## Proposed Changes

Enforce strict validation across all profile configuration entry points—specifically MCP tools (`bisync_create_profile`, `bisync_update_profile`, `bisync_create_remote`) and backend domain models (`internal/profiles`)—so that MCP tools and API callers cannot set options outside what is defined and supported in the dashboard UI.

### 1. Domain Model Validation (`internal/profiles/model.go`)
- Define allowed sets as constants/slices:
  - **Schedules**: `manual`, `15m`, `30m`, `1h`, `4h`, `daily`
  - **Review Modes**: `disabled`, `deletions`, `conflicts`, `all`
  - **Conflict Resolve Modes**: `path1`, `path2`, `newer`, `older`, `larger`, `smaller`
  - **Compare Modes**: `size,modtime`, `size,modtime,checksum`, `checksum`, `modtime`
  - **Notification Preferences**: `all`, `failures-only`, `review-required`, `disabled`
- Add concrete error types: `ErrInvalidSchedule`, `ErrInvalidReviewMode`, `ErrInvalidConflictMode`, `ErrInvalidCompareMode`, `ErrInvalidNotificationPref`.
- In `(*Profile).Validate()`:
  - Validate `Schedule` against the allowed set (case-insensitive trim, default to `manual` if empty).
  - Validate `ReviewMode` against the allowed set (default to `disabled` if empty).
  - Validate `NotificationPref` against the allowed set (default to `failures-only` if empty).
  - Validate `Flags.ConflictResolve` if set against the allowed set.
  - Validate `Flags.CompareMode` if set against the allowed set.
  - Normalize `Remote` by stripping trailing colons (`strings.TrimRight(remote, ":")`) so remote identifiers are consistently stored without colon suffixes.

### 2. MCP Tools Schema & Handler Validation (`internal/mcp/tools_profile.go`, `internal/mcp/tools_remote.go`)
- Update `bisync_create_profile`, `bisync_update_profile`, and `bisync_create_remote` tool schemas:
  - `schedule`: Update description to explicitly document the allowed values: `'manual', '15m', '30m', '1h', '4h', 'daily'`.
  - `conflict_mode`: Update description to include `'path1', 'path2', 'newer', 'older', 'larger', 'smaller'`.
  - Expose `review_mode` parameter on `bisync_create_profile` and `bisync_update_profile` with allowed values `'disabled', 'deletions', 'conflicts', 'all'`.
  - Expose `compare_mode` parameter on `bisync_create_profile` and `bisync_update_profile` with allowed values `'size,modtime', 'size,modtime,checksum', 'checksum', 'modtime'`.
- In MCP handler execution (`handleCreateProfile`, `handleUpdateProfile`, `handleCreateRemote`):
  - Pre-validate inputs against the allowed sets with descriptive error messages before persisting.
  - Normalize remote name by trimming trailing colons.

### 3. Frontend Telemetry & Explorer Display Resilience (`dashboard/frontend/partials/view_telemetry.html`, `dashboard/frontend/js/system_health.js`)
- In `view_telemetry.html`, format remote target cleanly as `${p.remote.replace(/:+$/, '')}:${p.remote_path ? p.remote_path.replace(/^\/+/, '') : ''}` to prevent double colons (e.g., `local_backup::backup_store`).
- In `dashboard/frontend/js/system_health.js`, update the registered MCP tool catalog schema descriptions in `mcpTools` to match the exact allowed parameters and options.

### 4. Tests & Quality Gate
- Update unit tests in `internal/profiles/model_test.go`, `internal/mcp/tools_profile_test.go`, and `internal/services/remote_wizard_service_test.go` to use valid UI schedules (e.g., `"1h"`, `"15m"`) instead of arbitrary cron expressions (`"0 * * * *"`), and add negative tests asserting rejection of unauthorized options.
- Run full pre-commit gate (`./scripts/check.sh all`).

### Affected Components & Packages
- `internal/profiles/model.go`: Domain enum constants and validation logic.
- `internal/profiles/model_test.go`: Unit tests for profile validation.
- `internal/mcp/tools_profile.go`: Tool registration schemas and handlers for profile CRUD.
- `internal/mcp/tools_remote.go`: Remote creation bridge schedule documentation and validation.
- `internal/mcp/tools_profile_test.go`: MCP profile tool tests with valid UI selections.
- `internal/services/remote_wizard_service_test.go`: Bridge tests with valid schedules.
- `dashboard/frontend/partials/view_telemetry.html`: Contract banner remote path display formatting.
- `dashboard/frontend/js/system_health.js`: Registry schema descriptions.

### Safety & Error Handling
- Strict validation rejects invalid inputs at the API / MCP boundary before any filesystem, scheduler, or DB writes.
- Clean backward compatibility: empty values still default safely (`manual` schedule, `disabled` review mode, `failures-only` notifications).

## Verification Plan
1. Unit Tests:
   - `go test -count=1 ./internal/profiles/...`
   - `go test -count=1 ./internal/mcp/...`
   - `go test -count=1 ./internal/services/...`
2. Full Quality Gate:
   - `./scripts/check.sh all`
