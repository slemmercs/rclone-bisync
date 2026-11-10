# Implementation Plan: Milestone 3 — Unified Cloud Remote & Sync Profile Provisioning

## Proposed Changes

Deliver a unified operational journey for creating, configuring, and managing Rclone remotes and sync profiles—accessible both through a browser-native UI wizard and programmatically through the native Model Context Protocol (MCP) server. Concurrently, bridge the MCP tool suite to feature parity with the Dual-Pane File Commander (Milestone 2) by adding dedicated tools for file preview, mkdir, copy, move, and purge.

### Affected Components & Packages

- `internal/models/provider.go`: DTOs for remote creation payloads, provider catalog categorization, and profile bridge requests.
- `internal/services/remote_wizard_service.go` (*new*): Domain service managing provider catalog discovery (`/config/providers`), categorized presets, and remote creation with audit logging.
- `internal/services/remote_wizard_service_test.go` (*new*): Unit tests for catalog caching, provider categorization, and creation validation.
- `internal/server/handlers_remotes_mgmt.go` (*new*): HTTP endpoints for `/api/remotes/providers`, `POST /api/remotes/create`, and `DELETE /api/remotes/{name}` with profile dependency guards.
- `internal/server/handlers_remotes_mgmt_test.go` (*new*): REST test suite with mocked RC responses.
- `internal/server/routes.go`: Route registration for new remote management endpoints.
- `internal/mcp/tools_remote.go` (*new*): Remote lifecycle MCP tools (`bisync_list_providers`, `bisync_create_remote`, `bisync_delete_remote`).
- `internal/mcp/tools_profile.go` (*new*): Profile lifecycle MCP tools (`bisync_create_profile`, `bisync_update_profile`, `bisync_delete_profile`, `bisync_rollback_profile`).
- `internal/mcp/tools_explorer_ops.go` (*new*): File explorer MCP tools (`bisync_file_preview`, `bisync_file_mkdir`, `bisync_file_copy`, `bisync_file_move`, `bisync_file_purge`).
- `internal/mcp/server.go`: Wire new tool registrations into `NewServer()`.
- `dashboard/frontend/partials/modals/modal_create_remote.html` (*new*): Step-by-step remote creation wizard with provider categorization and profile bridge.
- `dashboard/frontend/js/remotes_wizard.js` (*new*): Standalone Alpine.js wizard state controller.
- `dashboard/frontend/partials/view_remotes.html`: Add `[ + New Remote ]` button and empty-state onboarding trigger.
- `dashboard/frontend/partials/modals/modal_profiles.html`: Add `[ + Add New Remote ]` link in profile creation remote picker.
- `dashboard/frontend/partials/modals/modal_mcp.html`: Update MCP inspector modal to showcase all newly exposed tools.
- `dashboard/frontend/app.js`: Register `remotes_wizard` module.
- `site/usps/06-mcp-agent-server.md`: Document expanded MCP tool registry and parameters.

### Technical Strategy & Data Flow

1. **Provider Discovery & Remote Provisioning**:
   - `RemoteWizardService` queries Rclone `/config/providers`, categorizing providers into `cloud`, `object`, `protocol`, and `virtual`, while highlighting curated presets (Google Drive, OneDrive, AWS S3, Backblaze B2, Dropbox, SFTP, WebDAV, Alias).
   - Validates parameters, invokes Rclone `/config/create`, flushes remotes cache, and logs `REMOTE_CREATED` to the audit ledger.
   - Profile Provisioning Bridge: If requested, immediately creates the corresponding profile (`remote` + `local_path` + `schedule`).

2. **Profile Lifecycle via MCP**:
   - Exposes `bisync_create_profile`, `bisync_update_profile`, `bisync_delete_profile`, and `bisync_rollback_profile`.
   - Reuses `profiles.Service` with flag validation, version ledger, and scheduler hot-reloading.

3. **Dual-Pane Explorer Actions via MCP**:
   - Bridges Milestone 2 capabilities: `bisync_file_preview`, `bisync_file_mkdir`, `bisync_file_copy`, `bisync_file_move`, and `bisync_file_purge`.

### Safety & Error Handling

- **Path Jails**: Path confinement on `local` storage (`filepath.Rel` and `..` check).
- **Canary Protection**: Guard `RCLONE_TEST` against deletion/modification across all UI and MCP file actions.
- **Confirmation Tickets (`TicketManager`)**: Two-step confirmation for destructive actions (`bisync_file_purge`, `bisync_delete_remote`, `bisync_delete_profile`, `bisync_rollback_profile`).
- **Remote Deletion Conflict Guard**: Prevents deleting remotes active in existing profiles without explicit confirmation.
- **Read-Only Mode**: Write operations reject immediately when MCP `--read-only` is active.
- **LOC Discipline**: Strict <400 LOC ceiling per file.

## Verification Plan

- Unit tests for `RemoteWizardService`, REST endpoints, and all new MCP tools.
- Pre-commit quality gate `./scripts/check.sh all`.
- E2E journey test in `tests/e2e/sandbox_ui_journey_test.go`.
