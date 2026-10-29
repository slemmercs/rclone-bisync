# Bisync Hub Design System: Page Layouts

## 1. 2-Tier Global Header Architecture

Bisync Hub uses a persistent 2-tier navigation structure:

- **Tier 1: Global Context Bar**:
  - Left: Logo, Version pill (`v1.0`), Context Selector dropdown (`Switch Workspace` / active profile name).
  - Right:
    - **Polling Toggle** (`[LIVE]` vs `[PAUSED]`, shortcut `Alt+P`): Controls background telemetry fetching.
    - Global Settings cog button (`modal_global_settings.html`).
    - Daemon Live indicator (`LIVE` / `OFFLINE` + response latency in ms).
- **Tier 2: Sub-Navigation Bar (Context-Aware Workspace Tabs)**:
  - Tab 1: **Fleet** (Multi-workload status overview grid).
  - Tab 2: **Engine & Telemetry** (Single-tenant or selected profile cockpit + Unified Timeline Ledger).
  - Tab 3: **Remotes** (Cloud storage quota and OAuth connection health).
  - Tab 4: **Explorer** (Dual-pane path-jailed file explorer).
  - Tab 5: **Audit** (Cryptographically verified compliance ledger).

---

## 2. Layout Types

### Dashboard Layout (`view_fleet.html`)
- Aggregate fleet view displaying the Health Banner pattern, search/filter bar, and responsive grid of workload cards (`.ds-card`).
- Empty states rendered using `component_empty_state`.

### Workspace Layout (`view_telemetry.html`, `view_remotes.html`, `view_explorer.html`)
- Sticky operational toolbar at the top with primary action triggers (`Start Sync`, `Dry Run`, `Force Deletions`, `Stop`, `Resync`, `Clear Lock`).
- Execution Contract banner specifying local and cloud remote paths.
- Timeline pattern for event ledger.

### Detail & Audit Layout (`view_audit.html`)
- Cryptographic verification continuity banner.
- Operations data table (`.ds-table`) with SHA-256 fingerprint previews.
- Slide-over Entity Inspector Drawer (`component_entity_drawer`) for forensic payload analysis.
