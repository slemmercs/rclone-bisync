# Implementation Plan: Bisync Hub Cockpit UI/UX Consolidation

## Proposed Changes

Consolidate Bisync Hub cockpit UI/UX across top navigation, fleet onboarding, remote creation wizard, and dual-pane file explorer to eliminate visual clutter, remove dangerous/ambiguous nomenclature, and streamline operator workflows. Standardize on high-contrast Lucide SVG icons across all newly introduced and refreshed UI elements.

### Affected Components & Packages

- `dashboard/frontend/index.html`: Header layout consolidation, global `[ + Create ▾ ]` dropdown, consolidated System Health Chip, modal template registration, script registration.
- `dashboard/frontend/partials/modals/modal_system_health.html` (*new*): Unified modal covering Daemon Engine & Logs, MCP Agent Server, and Scheduler & Poller tabs with sub-system alerts.
- `dashboard/frontend/js/system_health.js` (*new*): Alpine.js module managing System Health Modal state, tab transitions, log tailing, and polling controls.
- `dashboard/frontend/app.js`: Register `SystemHealthModule`, global keyboard shortcuts (`c`/`Ctrl+K` for Create, `n` for Profile, `r` for Remote, `?` for Help), and consolidated health chip computed state.
- `dashboard/frontend/partials/view_fleet.html`: Fleet header cleanup, click-to-filter profile metric cards, and 2-step first-launch onboarding hero card (`profilesData.list.length === 0`).
- `dashboard/frontend/partials/modals/modal_create_remote.html`: Tier-1 provider spotlight (Google Drive, OneDrive, S3, Dropbox, B2), clickable breadcrumb steppers, OAuth vs Custom credentials segmented toggle, unstickable error states.
- `dashboard/frontend/js/remotes_wizard.js`: Wizard state handling for provider spotlight, credential toggling, and resilient reset.
- `dashboard/frontend/partials/view_remotes.html`: Single-line unified control bar merging remote count, status badges, profile scoping, and action buttons.
- `dashboard/frontend/partials/view_explorer.html` & `component_explorer_toolbar.html`: Explorer toolbar cleanup, removing ambiguous "Sync" labels, single-line pane toolbar, directional transfer bar, and locked `RCLONE_TEST` canary visual presentation.
- `tests/e2e/dashboard_smoke_test.go` & `tests/e2e/sandbox_ui_journey_test.go`: E2E headless Chrome assertions for header health chip, quick create menu, onboarding card, and explorer operations.

### Technical Strategy & Data Flow

1. **Header & Consolidated Health Chip**:
   - Merge `POLL`, `MCP`, and `LIVE` status pills into a single high-density chip: `[ 🟢 Live 2ms • 🟣 MCP • ⚪ Idle ]`.
   - Preserve backward-compatible attribute hooks (`[x-text="mcpStatusText"]`) within the chip or modal to maintain test resilience.
   - Primary `[ + Create ▾ ]` CTA dropdown provides immediate global access to `+ New Sync Profile` (`startNewProfile()`) and `+ Connect Storage Remote` (`openCreateRemoteWizard()`).
2. **Unified System Health Modal**:
   - Tab 1: Daemon Engine & Correlated Logs (integrating log stream, latency ping, error counters, fold repetitive clusters).
   - Tab 2: MCP Agent Server (transport inspection, 24-tool catalog, client config snippet with copy button).
   - Tab 3: Scheduler & Poller (polling frequency, pause/resume toggle, job runner status).
   - Live badge alert dots on tabs if any sub-system is degraded.
3. **Fleet Overview & Onboarding**:
   - Metric cards become interactive filters: clicking Healthy, Review, Critical, Running, or Total filters the profile list.
   - 0-State: A 2-step visual onboarding card guides the operator through Remote Connection -> Profile Creation -> Bidirectional Sync.
4. **Remote Wizard Usability**:
   - Step 1 displays Tier-1 curated provider spotlight cards with standard Lucide SVGs, with searchable catalog below.
   - Clickable stepper allows jumping back to completed steps.
   - Step 2 adds an explicit segmented toggle between Automated Browser OAuth and Custom Client Credentials.
5. **Explorer Directional Clarification & Canary Safety**:
   - Eliminate misleading `Sync Local to Remote` / `Sync Remote to Local` buttons; replace with explicit item-level directional transfers (`Transfer to Remote →`, `Copy Selection`).
   - Collapse 3 pane header lines into 1 line (~130px vertical reclamation).
   - `RCLONE_TEST` canary rows styled with a lock icon, disabled deletion/renaming, and informative tooltip.

### Safety & Error Handling

- **Lucide SVG Icons**: All icons adhere to Lucide 24×24 stroke-based SVGs.
- **Canary Guard**: Strict visual lock and disabled interactive controls on `RCLONE_TEST` to prevent accidental deletion.
- **Unstickable Wizard State**: Cancel and Back buttons remain responsive during errors, and spinner reverts automatically upon request failure.
- **Zero LOC Regressions**: All files strictly kept below 400 LOC per `./scripts/check.sh loc 400`.
- **Zero Unsafe Dotted Alpine Bindings**: Validated via `./scripts/check.sh templates`.

## Verification Plan

- `./scripts/check.sh templates`: Validate Alpine.js binding safety.
- `./scripts/check.sh loc 400`: Enforce file length constraints.
- `go test -count=1 ./tests/e2e/...`: Verify headless Chrome E2E test suite.
- `./scripts/check.sh all`: Full 7-stage pre-commit quality gate.


