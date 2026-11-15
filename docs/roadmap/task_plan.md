# Task Plan: Deduplication Workspace Redesign & Link/Whitespace Edge-Case Resolution

## Goal
Resolve Google Drive link/shortcut and trailing-space path navigation edge cases, and redesign the Deduplication modal into an expansive 3-section workspace (Configuration & Scope rail, Collision queue ledger, and Inbuilt File Explorer with duplicate candidate inspector) strictly adhering to the <400 LOC ceiling.

## Tasks

- [x] Task 1: Backend Trailing Whitespace & Transparent Link/Shortcut Auto-Recovery (`internal/services/`, `internal/models/`)
  - **Scope**: `internal/services/explorer_service.go`, `internal/services/explorer_service_test.go`, `internal/models/api_explorer.go`
  - **Verification**: `go test -count=1 ./internal/services/... -run TestBrowsePath`

- [x] Task 2: Deduplication Parser Detail Extraction (`dashboard/frontend/js/dedupe_parser.js`)
  - **Scope**: `dashboard/frontend/js/dedupe_parser.js`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh loc 400`

- [ ] Task 3: Deduplication Workspace Module Extraction (`dashboard/frontend/js/dedupe_workspace.js`, `remotes_modals.js`)
  - **Scope**: `dashboard/frontend/js/dedupe_workspace.js`, `dashboard/frontend/js/remotes_modals.js`, `dashboard/frontend/index.html`, `dashboard/frontend/app.js`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh loc 400`

- [ ] Task 4: Deduplication Modal 3-Section UI/UX Redesign (`dashboard/frontend/partials/modals/modal_dedupe.html`)
  - **Scope**: `dashboard/frontend/partials/modals/modal_dedupe.html`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh loc 400`

- [ ] Task 5: End-to-End Headless Chrome Testing & Full Quality Gate (`tests/e2e/dashboard_smoke_test.go`, full gate)
  - **Scope**: `tests/e2e/dashboard_smoke_test.go`, repository pre-commit checks
  - **Verification**: `./scripts/check.sh all && ./scripts/check.sh loc 400`

## Status Updates

- 2026-11-15: Initialized plan from approved `implementation_plan.md` on branch `fix/dedupe-workspace-redesign-and-link-edge-case`.
- 2026-11-15: Completed Task 1 - Preserved folder segment trailing whitespace across `ExplorerService.BrowsePath` and dedupe handlers. Implemented transparent auto-resolution for 404 remote and local directories with trailing spaces or shortcut aliases. Added shortcut detection (`IsShortcut: true`) for composite Google Drive IDs and symlinks. Verified with unit tests (`TestExplorerService_BrowsePath`).
- 2026-11-15: Completed Task 2 - Enhanced `parseDedupeOutput` in `dashboard/frontend/js/dedupe_parser.js` to extract candidate item paths, clean names, parent directories, and candidate detail records (`size`, `modtime`, `id`, `index`). Exported both for browser and CommonJS module environments. Verified via `./scripts/check.sh templates && ./scripts/check.sh loc 400` and `./...` tests.







