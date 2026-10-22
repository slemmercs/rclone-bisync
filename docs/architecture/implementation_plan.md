# Implementation Plan: Rebrand to Bisync Hub & Lucide SVG Standard

## Overview
Elevate the project identity, dashboard UI, and deployment assets to **Bisync Hub**, adopting the core positioning:
> *"The safety-first dashboard for Rclone Bisync"*  
> **Vision:** *"Review. Decide. Sync."*

Standardize and replace all dashboard UI icons with clean, inline SVG icons from the [Lucide](https://lucide.dev) design system for Vanilla JS. This enforces uniform stroke widths, crisp 24x24 viewBox geometry, and Tailwind slate/indigo styling without introducing any npm, Node, or external CDN runtime dependencies.

---

## Brand Architecture & System Specifications

### 1. Visual Identity & Copy Tokens
- **Application Title**: `Bisync Hub`
- **Version & Edition Badge**: `v2.4 Enterprise`
- **Positioning**: *"The safety-first dashboard for Rclone Bisync"*
- **Vision Tagline**: *"Review. Decide. Sync."*
- **Primary Accent Theme**: Slate 950 (`bg-slate-950`), Indigo 600 (`bg-indigo-600`), Emerald 400 (`text-emerald-400`), Rose 400 (`text-rose-400`), Amber 400 (`text-amber-400`).

### 2. Zero-Build Lucide SVG Specification
All UI icons will be standardized as standalone, offline inline SVGs conforming strictly to:
- `viewBox="0 0 24 24"`
- `fill="none"`
- `stroke="currentColor"`
- `stroke-width="2"` (or `1.75` for dense item listings)
- `stroke-linecap="round"`
- `stroke-linejoin="round"`
- Utility classes: `w-3.5 h-3.5`, `w-4 h-4`, `w-5 h-5`

### 3. Icon Mapping Catalog

| UI Component | Current Glyph | Lucide Standard Replacement |
| :--- | :--- | :--- |
| **Telemetry Tab** | Lightning bolt | `Activity` (`<path d="M22 12h-4l-3 9L9 3l-3 9H2"/>`) |
| **Storage Remotes Tab** | Custom cloud | `Cloud` (`<path d="M17.5 19H9a7 7 0 1 1 6.71-9h1.79a4.5 4.5 0 1 1 0 9Z"/>`) |
| **File Explorer Tab** | Legacy folder | `Folder` (`<path d="M20 20a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.9a2 2 0 0 1-1.69-.9L9.6 3.9A2 2 0 0 0 8 3H4a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2Z"/>`) |
| **Start Sync Action** | Simple triangle | `Play` (`<polygon points="6 3 20 12 6 21 6 3"/>`) |
| **Stop Sync Action** | Stop square | `Square` (`<rect width="18" height="18" x="3" y="3" rx="2"/>`) |
| **Force Resync Action** | Basic reload | `RefreshCw` (`<path d="M21 12a9 9 0 0 0-9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/><path d="M3 3v5h5"/><path d="M3 12a9 9 0 0 0 9 9 9.75 9.75 0 0 0 6.74-2.74L21 16"/><path d="M16 21h5v-5"/>`) |
| **Force Sync Action** | Generic shield | `ShieldAlert` (`<path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10"/><line x1="12" x2="12" y1="8" y2="12"/><line x1="12" x2="12.01" y1="16" y2="16"/>`) |
| **Dry Run Action** | Basic eye | `Eye` (`<path d="M2 12s3-7 10-7 10 7 10 7-3 7-10 7-10-7-10-7Z"/><circle cx="12" cy="12" r="3"/>`) |
| **Remove Lock Action** | Generic lock | `Unlock` (`<rect width="18" height="11" x="3" y="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 9.9-1"/>`) |
| **Clear Rate Limit** | Generic clock | `TimerReset` (`<path d="M10 2h4"/><path d="M12 14v-4"/><path d="M4 13a8 8 0 0 1 8-7 8 8 0 1 1-5.3 14L4 17.6"/><path d="M9 17H4v5"/>`) |
| **Canary Provision** | Plus glyph | `ShieldPlus` (`<path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10"/><line x1="12" x2="12" y1="9" y2="15"/><line x1="9" x2="15" y1="12" y2="12"/>`) |
| **File Delete** | Simple trash | `Trash2` (`<path d="3 6h18"/><path d="19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"/><path d="8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/><line x1="10" x2="10" y1="11" y2="17"/><line x1="14" x2="14" y1="11" y2="17"/>`) |
| **Copy Action** | Overlapping squares | `Copy` (`<rect width="14" height="14" x="8" y="8" rx="2" ry="2"/><path d="M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2"/>`) |
| **Spinning Loader** | Legacy arrows | `Loader2` (`<path d="21 12a9 9 0 1 1-6.219-8.56"/>`) |
| **Close Modal** | Manual cross | `X` (`<path d="18 6 6 18"/><path d="m6 6 12 12"/>`) |

---

## Touchpoint Manifest
1. `dashboard/frontend/index.html`: Header branding, nav pills, action toolbar, telemetry cards, remotes grid, explorer rows, and modals.
2. `main.go`: Console startup banner log (`Bisync Hub dashboard running at http://...`).
3. `deploy/desktop/rclone-dashboard.desktop`: Application title (`Name=Bisync Hub`) and comment (`Comment=The safety-first dashboard for Rclone Bisync`).
4. `deploy/systemd/rclone-dashboard.service`: Service description (`Description=Bisync Hub - Dashboard & Sync Orchestrator`).
5. `deploy/install.sh`: Installer heading, progress logs, and completion message.
6. `AI_DISCLOSURE.md`: Reference to `Bisync Hub`.

---

## Verification & Quality Gates
- **Shell & Git Hygiene**: Pre-flight checks (`./scripts/check.sh`).
- **Test Suite**: Automated tests pass cleanly (`go test -count=1 ./...`).
- **Static Asset Tests**: Embedded frontend assets validated (`go test -count=1 ./internal/server -run TestStaticAndIndexEmbedding`).
- **Binary Build & Subcommands**: `go build -o bin/rclone-bisync main.go` && `./bin/rclone-bisync -h`.


