
# Task Plan: Frontend Template Modularization (`html/template`)

## Goal
Decompose the monolithic 1,500+ LOC `dashboard/frontend/index.html` into clean, maintainable Go standard library `html/template` partials (`partials/view_*.html`, `partials/modals/*.html`) assembled server-side with zero npm dependencies and zero DOM pop-in, establishing a clean foundation for upcoming feature workspaces such as the Immutable Audit Ledger.

## Tasks
- [x] Task 1: Directory Setup & Sub-View Partials Extraction
  - **Scope**: Create `dashboard/frontend/partials/` and extract `view_telemetry.html`, `view_remotes.html`, `view_explorer.html`.
  - **Verification**: Files exist and contain valid HTML structures matching the respective views.
- [x] Task 2: Modals Partials Extraction
  - **Scope**: Create `dashboard/frontend/partials/modals/` and extract `modal_review.html`, `modal_daemon.html`, `modal_delete.html`, `modal_deletions.html`, `modal_oauth_migration.html`, `modal_reconnect.html`, `modal_auth_audit.html`.
  - **Verification**: Files exist and contain valid modal HTML structures.
- [x] Task 3: Master Shell Streamlining & Server Template Assembly
  - **Scope**: Streamline `dashboard/frontend/index.html` into a ~95 LOC master shell utilizing `{{ template "name" . }}`. Update `internal/server/server.go` to parse all templates at server boot with `template.ParseFS` and render dynamically on `GET /{$}`.
  - **Verification**: `go test -v -count=1 ./internal/server -run "TestServer_RootHandler_"`
- [x] Task 4: Full Quality Gate & Pre-Flight Validation
  - **Scope**: Run `./scripts/check.sh` ensuring all tests pass, formatting is clean, static analysis is clean, and binaries compile successfully.
  - **Verification**: `./scripts/check.sh`

## Next Up
- **Milestone Complete**: Frontend Template Modularization verified and clean.
- **Upcoming Milestone**: Immutable Audit Ledger & Unified Log Architecture (`internal/audit`, `internal/core/logger`, and `partials/view_audit.html`).

## Status Updates
- 2026-10-24: Created branch `refactor/modularize-frontend-templates`. Updated `docs/design/immutable_audit_ledger_design_specification.md`, `ROADMAP.md`, and `backlog.md`.
- 2026-10-24: Completed Tasks 1 & 2: Carved out all sub-views and modals into `dashboard/frontend/partials/` and `dashboard/frontend/partials/modals/` wrapped in named `{{ define "..." }}` blocks.
- 2026-10-24: Completed Task 3: Streamlined `dashboard/frontend/index.html` to a 95 LOC master shell, updated `internal/server/server.go` to parse templates at startup with fallback, and added comprehensive unit tests in `internal/server/server_test.go` verifying template assembly with both mock and real filesystems.
- 2026-10-24: Completed Task 4: Ran `./scripts/check.sh` with 8/8 quality checks passing, zero vet warnings, and 83.2% statement coverage.
- 2026-10-24: Fixed UI daemon modal crash (object iteration in Alpine `modal_daemon.html`), corrected SVG arc flag syntax in `view_telemetry.html`, silenced offline vendor Tailwind console notice, wired click handlers to daemon status tiles, and updated active binary in `/home/csaba/rclone-bisync/bin/rclone-bisync`.



