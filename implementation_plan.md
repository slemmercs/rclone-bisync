# Implementation Plan: Cockpit UI/UX Usability Polish & Bug Fixes

## Overview
Deliver a targeted usability and bug-fix overhaul addressing the 5 user-reported issues in the Bisync Hub cockpit:
1. Enable an inline expandable MCP Tool Registry accordion within the System Health modal's MCP Agent Server tab displaying all 24 registered tools and resources with tool search and safety mode badges.
2. Resolve missing OAuth vs Custom Credentials segmented toggle in Remote Creation Wizard Step 2 by exposing `SupportsOAuth` on backend `ProviderCatalogItem` and adding frontend provider prefix matching fallback.
3. Add full abort/cancellation capability to Remote Creation Wizard Step 3 during provisioning via an explicit "Cancel Provisioning" button, window close handler, and `POST /api/remotes/auth/oauth-stop`.
4. Surface active remote names and clickable linkouts (`switchTab('remotes')`) directly in the Fleet onboarding hero card's "Remotes Ready" badge and remote summary pills.
5. Consolidate the two stacked bars in the Storage Remotes view ("Authentication Health Overview" and "Configured Remotes") into a single cohesive control deck.

## Types
- `internal/models/provider.go`:
  - Add `SupportsOAuth bool `json:"supports_oauth"`` to `ProviderCatalogItem` struct.

## Files
- Existing files to modify:
  - `internal/models/provider.go`: Add `SupportsOAuth bool `json:"supports_oauth"`` to `ProviderCatalogItem`.
  - `internal/services/remote_wizard_service.go`: Set `SupportsOAuth` during catalog assembly using `remoteauth.IsOAuthProvider(p.Name) || remoteauth.IsOAuthProvider(p.Prefix)`.
  - `dashboard/frontend/partials/modals/modal_system_health.html`: Make the MCP Tool Registry card interactive (`cursor-pointer`, active highlight, chevron), embed the 24-tool catalog inline accordion with search and parameter details.
  - `dashboard/frontend/js/system_health.js`: Add `mcpToolsExpanded: false`, `mcpToolsSearch: ""`, and filtering helper `filteredMcpTools`.
  - `dashboard/frontend/partials/modals/modal_create_remote.html`: Add frontend OAuth check fallback `isOAuthProvider(createRemoteModal.selectedProvider)`, add Cancel button in Step 3 when `createRemoteModal.saving` is true, and enable Back/Cancel navigation.
  - `dashboard/frontend/js/remotes_wizard.js`: Add `isOAuthProvider(p)`, add `cancelProvisioning()` method calling `window.API.post("/api/remotes/auth/oauth-stop")` and aborting in-flight request, update `closeCreateRemoteWizard()` to abort if saving.
  - `dashboard/frontend/partials/view_fleet.html`: Make Step 1 readiness badge clickable (`@click="switchTab('remotes')"`), display active ready remote badges (e.g. `drive:`, `s3:`), and add linkout button.
  - `dashboard/frontend/partials/view_remotes.html`: Merge the Authentication Health Bar and Configured Remotes Control Bar into a single unified header bar.
  - `tests/e2e/dashboard_smoke_test.go`: Add E2E tests for the new interactions.

## Functions
- `internal/services/remote_wizard_service.go`:
  - `GetProvidersCatalog(ctx context.Context, forceRefresh bool)`: Enrich catalog items with `SupportsOAuth`.
- `dashboard/frontend/js/system_health.js`:
  - `toggleMcpToolCatalog()`: Toggles inline accordion visibility.
  - `filteredMcpTools`: Filter registered tools by name and description query.
- `dashboard/frontend/js/remotes_wizard.js`:
  - `isOAuthProvider(p)`: Helper checking `p.supports_oauth` or common OAuth provider prefixes (`drive`, `onedrive`, `dropbox`, `box`, `pcloud`, `yandex`, etc.).
  - `cancelProvisioning()`: Aborts in-flight create remote request and calls `/api/remotes/auth/oauth-stop`.

## Classes
- None (zero class inheritance; standard Alpine.js reactive objects and idiomatic Go structs).

## Dependencies
- None (native Go standard library and Alpine.js zero-build architecture).

## Testing
- `internal/services/remote_wizard_service_test.go`: Verify `SupportsOAuth` is populated correctly for `drive` and `s3`.
- `tests/e2e/dashboard_smoke_test.go`:
  - Verify clicking MCP Tool Registry expands the inline tool list.
  - Verify selecting Google Drive displays the OAuth vs Custom toggle in Step 2.
  - Verify Cancel button appears during Step 3 provisioning and halts the action.
  - Verify clicking Fleet onboarding ready badge navigates to `remotes` tab.
  - Verify Remotes view has a single consolidated control bar.
- Quality Gate: `./scripts/check.sh all` passing 7/7 stages.

## Implementation Order
1. Backend & Model: Update `internal/models/provider.go` and `internal/services/remote_wizard_service.go` to expose `supports_oauth`.
2. Wizard Step 2 & 3: Fix OAuth toggle detection in `modal_create_remote.html` and `remotes_wizard.js`; implement `cancelProvisioning()` with `stopOAuth()`.
3. System Health MCP Accordion: Implement inline expandable tool catalog in `modal_system_health.html` and `system_health.js`.
4. Fleet Hero Onboarding: Add ready remote tags and clickable tab switch in `view_fleet.html`.
5. Remotes Control Deck: Consolidate the two stacked bars into one unified bar in `view_remotes.html`.
6. E2E Tests & Pre-commit Quality Gate: Update headless Chrome assertions and verify `./scripts/check.sh all`.
