# Implementation Plan: Automated Code Documentation Generator & Sandbox Screenshot Capture

## Proposed Changes

Implement two automated documentation capabilities for Bisync Hub’s GitHub Pages showcase platform:
1. **Native Go Documentation Generator (`cmd/gendocs`)**:
   - A standalone, dependency-free Go CLI utility parsing Go AST from `internal/models/`, `internal/profiles/`, and `internal/mcp/`.
   - Generates a comprehensive, search-indexed `site/technical-reference.md` documentation page for the Jekyll `just-the-docs` platform.
   - Integrates into `Makefile` (`make docs-generate`) and `.github/workflows/pages.yml` to automatically regenerate documentation on push.
2. **Automated Sandbox Screenshot Engine (`tests/e2e/screenshots_test.go`)**:
   - An automated test harness using `chromedp` against an ephemeral sandbox backend (zero touch of live app, live config, or live storage remotes).
   - Generates updated, high-definition (1440x900) PNG screenshots of the Vue 3 + PrimeVue Cockpit SPA:
     - `bisync-hub_fleet-overview.png` (Fleet view with telemetry and profile cards)
     - `bisync-hub_file-explorer.png` (Dual-pane explorer with local/remote replicas)
     - `bisync-hub_storage-remotes.png` (Storage remotes management)
     - `bisync-hub_sync-review-resolution.png` (Active conflict review session & diffs)
     - `bisync-hub_audit-ledger.png` (Cryptographic audit ledger & hash chain)
     - `bisync-hub_profile-settings.png` (Profile settings modal)
     - `bisync-hub_engine-telemetry.png` (Telemetry cockpit)
   - Updates `site/gallery.md` to showcase the new Vue Cockpit SPA instead of legacy Alpine.js screenshots.

### Affected Components & Packages

- `cmd/gendocs/main.go`: Native AST doc generator.
- `site/technical-reference.md`: Generated technical documentation page.
- `site/_config.yml`: Navigation order and auxiliary links.
- `site/gallery.md`: Refreshed UI gallery copy and screenshot references.
- `.github/workflows/pages.yml`: GitHub Actions Pages workflow with automated Go doc generation.
- `Makefile`: `docs-generate` and `docs-screenshots` targets.
- `tests/e2e/screenshots_test.go`: Automated headless Chrome sandbox screenshot capture suite.
- `site/assets/images/screenshots/*`: Fresh sandbox screenshots.

### Technical Strategy & Data Flow

- **AST Parsing**: Uses Go standard library (`go/parser`, `go/ast`, `go/token`) to inspect struct declarations and field comments, extracting JSON tags and types deterministically without reflection panics.
- **MCP Reflection**: Imports the MCP server tool definitions to dynamically emit schema parameter tables and blast-radius classifications.
- **Zero-Touch Sandbox**: Screenshot generation leverages `sandboxHarness` with an isolated temp directory (`t.TempDir()`), mock Rclone RC daemon, and pre-seeded review sessions, guaranteeing zero modification to the host environment.
- **File Confinement**: All Go files strictly observe the <= 400 LOC ceiling.

### Safety & Error Handling

- Generator validates input file presence and fails fast with non-zero exit codes.
- Screenshot suite skips gracefully if Chrome/Chromium binary is missing from PATH.
- Pre-flight quality checks (`./scripts/check.sh all`) enforce formatting, coverage, and LOC boundaries.

## Verification Plan

- `go run ./cmd/gendocs --out site/technical-reference.md`
- `go test -count=1 -v ./tests/e2e -run TestCaptureSandboxScreenshots`
- `ls -lh site/assets/images/screenshots/`
- `./scripts/check.sh all`

