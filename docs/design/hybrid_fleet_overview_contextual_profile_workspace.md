# Design Specification: Hybrid Fleet Overview & Contextual Profile Workspace

## Executive Summary

As **Bisync Hub** evolves from a single-tenancy dashboard to a multi-workload management platform, the user interface must transition from a resource-centric view (focused strictly on daemon metrics) to a **workload-centric hierarchy**.

This specification defines **Option 3: Hybrid Fleet Overview + Contextual Workspace**. This architecture provides a high-level **Fleet View (`/profiles`)** as the default landing page for fleet-wide monitoring, while retaining the deep **Scoped Workspace (`/profiles/:id`)** for telemetry, file exploration, and audit logs. A persistent header context switcher enables zero-click navigation between profiles without forcing users back to the landing page.

---

## 1. Information Architecture & Route Mapping

```
┌────────────────────────────────────────────────────────────────────────┐
│                        Bisync Hub UI Hierarchy                         │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │
         ┌──────────────────────────┴──────────────────────────┐
         ▼                                                     ▼
┌─────────────────────────────────┐           ┌─────────────────────────────────┐
│       Level 1: Fleet View       │           │    Level 2: Scoped Workspace    │
│           `/profiles`           │           │        `/profiles/:id`          │
├─────────────────────────────────┤           ├─────────────────────────────────┤
│ • System Health Summary Banner  │           │ • Header Global Profile Switcher│
│ • Interactive Profile Tile Grid │  ──────>  │ • Engine & Telemetry Panel      │
│ • Filter & Search Controls      │  Click    │ • Dual-Pane File Explorer       │
│ • Quick Actions (Sync, Review)  │   Tile    │ • Audit & Session Ledger        │
└─────────────────────────────────┘           └─────────────────────────────────┘
```

### Routing Table

| Path | View Mode | Purpose & Behavior |
| :--- | :--- | :--- |
| `/` or `/profiles` | **Fleet View** | Default root route. Renders high-level status cards for all configured profiles, aggregate fleet metrics, and safety alerts. |
| `/profiles/:id` | **Scoped Workspace** | Primary operational workspace for a single profile. Dynamically binds Telemetry, File Explorer, and Audit Ledger to `:id`. |
| `/profiles/new` | **Modal / Workflow** | Triggers the multi-step Profile Creation Wizard. |

---

## 2. Level 1 Specification: Fleet View (`/profiles`)

The Fleet View serves as the central command dashboard. It gives users immediate visibility into which synchronization workloads are healthy, which are currently running, and which require safety interventions.

### 2.1 UI Layout & Wireframe

```
┌───────────────────────────────────────────────────────────────────────────────────────────────┐
│  Bisync Hub  v1.0   [ Engine: LIVE | 1.09ms ]                 [ All Profiles ▾ ]  [ + Profile ]│
├───────────────────────────────────────────────────────────────────────────────────────────────┤
│  FLEET HEALTH SUMMARY                                                                         │
│  [ 4 Total Profiles ]   [ 2 Healthy ]   [ 1 Needs Review ]   [ 1 Failing ]   [ 0 Syncing ]      │
├───────────────────────────────────────────────────────────────────────────────────────────────┤
│  Search: [ Filter profiles...  ]   Sort by: [ Status ▾ ]   View: [ Grid ▦ | List ☰ ]          │
│                                                                                               │
│  ┌───────────────────────────────┐ ┌───────────────────────────────┐ ┌──────────────────────┐  │
│  │ Personal Google Drive         │ │ OneDrive Photos               │ │ Server S3 Archive    │  │
│  │ ● HEALTHY         [ 15m Auto ]│ │ ⚠ REVIEW REQUIRED  [ 1h Auto ]│ │ ✖ SYNC_FAILED [Manual]│  │
│  ├───────────────────────────────┤ ├───────────────────────────────┤ ├──────────────────────┤  │
│  │ Local: ~/gdrive               │ │ Local: ~/photos               │ │ Local: /mnt/data     │  │
│  │ Target: gdrive:backup         │ │ Target: onedrive:media        │ │ Target: s3:app-data  │  │
│  │                               │ │                               │ │                      │  │
│  │ Last: 10m ago (1.2s, 0 err)   │ │ Last: Hold (4 Deletions)      │ │ Last: 2h ago (E-500)│  │
│  │ Next: in 5 minutes            │ │ Next: Paused (Review Block)   │ │ Next: Manual Only    │  │
│  ├───────────────────────────────┤ ├───────────────────────────────┤ ├──────────────────────┤  │
│  │ [ ▶ Sync Now ] [ ⚙ Settings ] │ │ [ 🛡 Review Diff ] [ ⏸ Pause ]│ │ [ 📋 Logs ] [ ↻ Retry]│  │
│  └───────────────────────────────┘ └───────────────────────────────┘ └──────────────────────┘  │
└───────────────────────────────────────────────────────────────────────────────────────────────┘
```

### 2.2 Component Specifications

1. **System Health Summary Bar:**
   * Aggregate counters summarizing total profile states: `Total`, `Healthy`, `Needs Review`, `Failing`, and `Active Syncing`.
   * Clicking a health status pill filters the tile grid below (e.g., clicking `Needs Review` shows only blocked profiles).

2. **Profile Tile Cards:**
   * **Header Badge:** Displays profile name, status tag (`HEALTHY`, `SYNCING`, `REVIEW_REQUIRED`, `SYNC_FAILED`, `PAUSED`), and schedule type (`15m Auto`, `Manual`).
   * **Topology Preview:** Clean display of local path and remote target reference.
   * **Operational Telemetry:** Last run time, run duration, error count, and next scheduled execution timestamp.
   * **Action Bar:**
     * `HEALTHY` State: `[ Sync Now ]`, `[ Dry Run ]`, `[ Settings ]`.
     * `REVIEW_REQUIRED` State: Highlighting `[ Review Diff ]` in amber/accent color to prompt human authorization.
     * `SYNC_FAILED` State: Highlighting `[ View Logs ]` and `[ Retry ]` in red/warning tones.

3. **Interactivity:**
   * Clicking anywhere on the tile card body navigates directly to the **Scoped Workspace (`/profiles/:id`)**.
   * Action buttons execute quick commands asynchronously via API without triggering full page reloads.

---

## 3. Level 2 Specification: Scoped Workspace (`/profiles/:id`)

The Scoped Workspace provides deep diagnostic tools for a single selected profile. It retains the telemetry charts, dual-pane directory explorer, and audit ledgers established in the single-profile design, bound strictly to the active profile's context.

### 3.1 UI Header & Context Switcher Integration

```
┌───────────────────────────────────────────────────────────────────────────────────────────────┐
│  Bisync Hub / Profiles / Personal Google Drive         [ Profile: Personal GDrive ▾ ] [ LIVE ] │
├───────────────────────────────────────────────────────────────────────────────────────────────┤
│  [ Engine & Telemetry ]   [ Storage Remotes ]   [ File Explorer ]   [ Audit Ledger ]          │
└───────────────────────────────────────────────────────────────────────────────────────────────┘
```

1. **Breadcrumb Navigation:** Displays `Profiles / [Profile Name]` in the upper left header. Clicking `Profiles` returns the user to the Level 1 Fleet View.
2. **Global Profile Switcher Dropdown:** Replaces static profile tags in the top right.
   * Clicking the switcher opens a compact selector menu listing all profiles alongside micro-status indicators (`● Healthy`, `⚠ Review`, `✖ Failed`).
   * Selecting a different profile updates the active route to `/profiles/:new_id` and refreshes workspace data instantly without returning to `/profiles`.

---

## 4. Frontend State & Component Architecture

### 4.1 Alpine.js Store (`/dashboard/frontend/app.js`)

The frontend application state is expanded to manage both fleet-wide summary collections and active profile details.

```javascript
document.addEventListener('alpine:init', () => {
    Alpine.store('fleet', {
        profiles: [],
        summary: {
            total: 0,
            healthy: 0,
            review_required: 0,
            failing: 0,
            syncing: 0
        },
        activeProfileId: null,
        filterStatus: 'all',
        searchQuery: '',

        async fetchProfiles() {
            const res = await fetch('/api/profiles/summary');
            const data = await res.json();
            this.profiles = data.profiles;
            this.summary = data.summary;
        },

        setActiveProfile(id) {
            this.activeProfileId = id;
            window.location.hash = `/profiles/\${id}`;
            Alpine.store('workspace').loadProfile(id);
        }
    });
});
```

### 4.2 Endpoint Contracts

| Method | Path | Response / Purpose |
| :--- | :--- | :--- |
| `GET` | `/api/profiles/summary` | Returns list of all profiles with operational summary metrics and aggregate fleet counts. |
| `GET` | `/api/profiles/:id` | Fetches complete profile configuration, flags, and schedule settings. |
| `GET` | `/api/profiles/:id/telemetry` | Fetches telemetry metrics, latency graphs, and logs scoped to `:id`. |
| `POST` | `/api/profiles/:id/sync` | Triggers a sync run for the specified profile (supports `?dry_run=true` or `?resync=true`). |

---

## 5. UI/UX Design System & Theme Alignment

To ensure visual continuity with the dark-mode aesthetic of Bisync Hub:

* **Background Surface:** `#0d1117` (Main surface), `#161b22` (Tile card background).
* **Card Borders:** `1px solid #30363d` with smooth hover transition (`border-color: #58a6ff`).
* **Status Colors:**
  * **Healthy / Live:** `#3fb950` (Emerald Green)
  * **Review Required / Warning:** `#d29922` (Amber/Yellow)
  * **Failed / Error:** `#f85149` (Coral Red)
  * **Syncing / Active:** `#58a6ff` (Sky Blue)

---

## 6. Acceptance Criteria

- [ ] **Fleet Route (`/profiles`):** Displays tile grid containing all user profiles with status, topology paths, and execution metrics.
- [ ] **Fleet Health Banner:** Shows aggregate summary counts (`Total`, `Healthy`, `Needs Review`, `Failing`) and filters tiles on click.
- [ ] **Scoped Navigation (`/profiles/:id`):** Clicking a tile card opens the detailed workspace for that specific profile.
- [ ] **Header Profile Switcher:** Global dropdown in Level 2 workspace allows instant switching between profiles.
- [ ] **Breadcrumbs:** Breadcrumb header permits one-click return from Scoped Workspace to Fleet Overview.
- [ ] **Quick Actions:** `Sync Now`, `Review Diff`, and `View Logs` buttons on tile cards execute without breaking page context.