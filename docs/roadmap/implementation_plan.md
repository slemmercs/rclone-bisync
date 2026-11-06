# Implementation Plan: Guided Flag Catalog & Rclone Reference Documentation Integration

## Proposed Changes

Expand the in-app interactive reference manual in Bisync Hub to document all guided operational flags offered in the Sync Profiles Manager dropdown (`--track-renames`, `--track-renames-strategy`, `--fast-list`, `--drive-shared-with-me`, `--checksum`, `--ignore-checksum`, `--size-only`, `--resilient`, `--recover`, `--checkers`, `--transfers`, `--bwlimit`, `--tpslimit`, `--tpslimit-burst`, `--buffer-size`, `--retries`, `--retries-sleep`, `--low-level-retries`), with Bisync-specific operational guidance and outbound links to official `rclone.org` documentation.

To prevent bloating `dashboard/frontend/js/help.js` (currently 396 LOC, approaching the 400 LOC ceiling) and ensure architectural modularity, static reference data (policies, flags, runbooks, documentation URLs) will be extracted into a dedicated `dashboard/frontend/js/help-data.js` module. `help.js` will serve exclusively as the reactive Alpine controller, dropping to ~150 LOC.

### Affected Components & Packages

- `dashboard/frontend/js/help-data.js`: New static catalog module registering `window.HelpCatalog` containing comprehensive safety policies, complete guided flags catalog (with categories, Bisync context tags, safe defaults, and official `rclone.org` URLs), and troubleshooting runbooks.
- `dashboard/frontend/js/help.js`: Slimmed-down reactive controller referencing `window.HelpCatalog`, maintaining backward-compatible methods (`filteredFlags`, `filteredPolicies`, `filteredRunbooks`, `openHelpDrawer`).
- `dashboard/frontend/index.html`: Load `<script src="/static/js/help-data.js"></script>` before `help.js`.
- `dashboard/frontend/partials/components/component_help_drawer.html`: Render external documentation links (`rclone.org ↗`) on flag cards with safe `rel="noopener noreferrer"` attributes.
- `tests/e2e/dashboard_smoke_test.go`: E2E test assertions validating catalog extraction, flag coverage, searchability of new flags, and doc links.

### Technical Strategy & Data Flow

1. **Catalog Separation**:
   - `window.HelpCatalog.flags`: Array of flag objects with fields:
     - `flag`: CLI argument name (e.g., `--track-renames`).
     - `name`: Human-readable title.
     - `category`: Categorization (`Detection`, `Performance`, `Safety`, `Concurrency`, `Rate Limiting`, `Resilience`, `Provider`, `Verification`).
     - `contexts`: Context tags (`modal_profiles`, `view_remotes`, etc.).
     - `description`: Bisync-specific explanation and operational behavior.
     - `example`: Typical CLI snippet.
     - `docUrl`: Direct URL to upstream rclone documentation (e.g. `https://rclone.org/bisync/#track-renames`, `https://rclone.org/flags/`).
2. **Controller Wiring**:
   - `help.js` initializes `flagsList`, `policiesList`, `runbooksList` from `window.HelpCatalog`.
3. **Template Enhancement**:
   - `component_help_drawer.html` displays an external link badge next to the flag name when `flag.docUrl` is present.

### Safety & Error Handling

- Zero build dependencies: Vanilla JS single binary distribution.
- Target `_blank` links strictly enforce `rel="noopener noreferrer"` to prevent tab-nabbing vulnerabilities.
- Safe fallbacks: If `window.HelpCatalog` fails to load, `help.js` defaults gracefully to empty arrays without runtime JS exceptions.
- Line-of-code ceiling: Both `help-data.js` and `help.js` adhere to the <400 LOC boundary.

## Verification Plan

- `tests/e2e/dashboard_smoke_test.go`: Automated headless Chrome test verifying flag catalog expansion, flag searches, and external documentation link rendering.
- Quality gates: `./scripts/check.sh templates` and `./scripts/check.sh all`.
- Dev binary verification on port 8085.
