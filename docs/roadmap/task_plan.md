
# Task Plan: Rebrand to Bisync Hub & Lucide Icon Standardization

## Goal
Transition all dashboard UI templates, documentation, deployment scripts, and service definitions to **Bisync Hub** (*"The safety-first dashboard for Rclone Bisync"*), and standardize all dashboard SVG icons with the [Lucide](https://lucide.dev) design system without adding external build or runtime dependencies.

## Tasks
- [x] Task 1: UI Header Branding & Tagline Updates
  - **Scope**: `dashboard/frontend/index.html` (title, h1, edition badge, subtitle, positioning) and `main.go` startup logs.
  - **Verification**: `go test -count=1 ./internal/server -run TestStaticAndIndexEmbedding`
- [x] Task 2: Standardize Header, Global Status & Action Controls (Lucide SVGs)
  - **Scope**: `dashboard/frontend/index.html` navigation pills and quick action buttons (`Play`, `Square`, `RefreshCw`, `ShieldAlert`, `Eye`, `Unlock`, `TimerReset`).
  - **Verification**: `go test -count=1 ./internal/server -run TestStaticAndIndexEmbedding`
- [x] Task 3: Standardize Engine & Telemetry Tab Icons (Lucide SVGs)
  - **Scope**: `dashboard/frontend/index.html` stat cards, parameter collapse disclosure (`ChevronDown`), charts header, and daemon log stream buttons.
  - **Verification**: `go test -count=1 ./internal/server -run TestStaticAndIndexEmbedding`
- [ ] Task 4: Standardize Storage Remotes & File Explorer Tab Icons (Lucide SVGs)
  - **Scope**: `dashboard/frontend/index.html` remote cards, OAuth health summary badges, breadcrumbs, folder/file glyphs, copy/delete buttons, and canary provisioning button.
  - **Verification**: `go test -count=1 ./internal/server -run TestStaticAndIndexEmbedding`
- [ ] Task 5: Standardize Modals & Log Viewer Icons (Lucide SVGs)
  - **Scope**: `dashboard/frontend/index.html` daemon log modal, raw log downloads, client ID migration wizard, reconnect progress spinner (`Loader2`), and audit trail modal.
  - **Verification**: `go test -count=1 ./internal/server -run TestStaticAndIndexEmbedding`
- [ ] Task 6: Systemd, Desktop Shortcut & Deployment Alignment
  - **Scope**: `deploy/desktop/rclone-dashboard.desktop`, `deploy/systemd/rclone-dashboard.service`, `deploy/install.sh`, `AI_DISCLOSURE.md`.
  - **Verification**: `./scripts/check.sh`
- [ ] Task 7: Full Verification Gate, Binary Build & Installation
  - **Scope**: Run `./scripts/check.sh`, verify coverage >80%, build `bin/rclone-bisync`, install to active environment, and test live dashboard endpoints.
  - **Verification**: `./scripts/check.sh` && `systemctl --user status rclone-dashboard.service`

## Status Updates
- 2026-10-22: Initialized plan on branch `feat/rebrand-bisync-hub`.
- 2026-10-22: Added `THIRD_PARTY_LICENSES.md` documenting Lucide (ISC/MIT), Alpine.js (MIT), Chart.js (MIT), Tailwind CSS (MIT), and Go sys (BSD 3-Clause).
- 2026-10-22: Completed Task 1 (Updated `dashboard/frontend/index.html` and `main.go` with Bisync Hub title, header, and vision tagline).
- 2026-10-23: Completed Task 2 (Standardized header navigation pills and quick action toolbar controls with Lucide SVGs).
- 2026-10-23: Completed Task 3 (Standardized Engine & Telemetry tab icons including status tiles, config accordion, performance chart, and bisync.log terminal controls with Lucide SVGs).











