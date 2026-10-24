# Implementation Plan - Frontend Template Modularization (`html/template`)

[Overview]
Decompose the monolithic 1,500+ LOC `dashboard/frontend/index.html` into clean, maintainable Go standard library `html/template` partials (`partials/view_*.html`, `partials/modals/*.html`) assembled server-side with zero npm dependencies and zero DOM pop-in, establishing a clean foundation for upcoming feature workspaces such as the Immutable Audit Ledger.

Bisync Hub prioritizes single-binary distribution without external runtime dependencies. As the UI expanded with Cockpit Telemetry, Storage Remotes, Dual-Pane File Explorer, Interactive Sync Review, and Correlated Daemon Logs, `index.html` grew beyond 1,500 LOC. Server-side template assembly parses modular partials at application boot and renders the unified page upon HTTP requests. This eliminates DOM layout shift, preserves instant responsiveness, and keeps individual HTML components under 300 LOC.

[Types]
No Go type additions required. Go `html/template` utilizes `nil` or existing view models.

[Files]
New files:
- `/home/csaba/repos/rclone-bisync/dashboard/frontend/partials/view_telemetry.html`: Cockpit cards, speed charts, orchestrator status, and `bisync.log` terminal (~270 LOC).
- `/home/csaba/repos/rclone-bisync/dashboard/frontend/partials/view_remotes.html`: Storage remotes capacity summary, provider grid, and remote actions (~220 LOC).
- `/home/csaba/repos/rclone-bisync/dashboard/frontend/partials/view_explorer.html`: Dual-pane file explorer workspace, breadcrumbs, search, and canary status (~230 LOC).
- `/home/csaba/repos/rclone-bisync/dashboard/frontend/partials/modals/modal_review.html`: Interactive Sync Review modal, recommendation badges, and batch trigger (~450 LOC).
- `/home/csaba/repos/rclone-bisync/dashboard/frontend/partials/modals/modal_daemon.html`: Correlated daemon log viewer modal (~160 LOC).
- `/home/csaba/repos/rclone-bisync/dashboard/frontend/partials/modals/modal_delete.html`: Explorer file/folder deletion confirmation modal (~70 LOC).
- `/home/csaba/repos/rclone-bisync/dashboard/frontend/partials/modals/modal_remote_reconnect.html`: Storage remote OAuth re-authentication modal (~60 LOC).
- `/home/csaba/repos/rclone-bisync/dashboard/frontend/partials/modals/modal_remote_credentials.html`: Remote credentials and token viewer modal (~60 LOC).

Modified files:
- `/home/csaba/repos/rclone-bisync/dashboard/frontend/index.html`: Streamlined master shell containing `<head>`, top navigation pills, container `<main>`, template injection tags, and script tags (~120 LOC).
- `/home/csaba/repos/rclone-bisync/internal/server/server.go`: Pre-parse templates on boot via `template.ParseFS` and render dynamic template on `GET /{$}`.
- `/home/csaba/repos/rclone-bisync/internal/server/server_test.go`: Verify `GET /` serves assembled HTML containing all sub-views and modals.

[Functions]
New functions:
- None required.

Modified functions:
- `(*Server).routes()` (`internal/server/server.go`): Pre-compile templates on startup and update `GET /{$}` handler to execute the compiled template.

[Classes]
(In Go, structs with methods):
- None required.

[Dependencies]
- Standard Go `html/template` and `io/fs`.
- Zero new external libraries; zero npm or Node.js dependencies.

[Testing]
- `internal/server/server_test.go`:
  - `TestServer_RootHandler_TemplateAssembly`: Verify that `GET /` returns HTTP 200 and the rendered HTML includes markers from each partial view and modal.
- Full verification:
  - `./scripts/check.sh` ensuring all existing tests pass with statement coverage >80% and zero static analysis warnings.

[Implementation Order]
1. Task 1: Create `dashboard/frontend/partials/` and extract `view_telemetry.html`, `view_remotes.html`, `view_explorer.html`.
2. Task 2: Create `dashboard/frontend/partials/modals/` and extract `modal_review.html`, `modal_daemon.html`, `modal_delete.html`, `modal_remote_reconnect.html`, `modal_remote_credentials.html`.
3. Task 3: Streamline `index.html` down to the master shell with `{{ template "..." . }}` directives and update `internal/server/server.go` to parse and render templates.
4. Task 4: Run `./scripts/check.sh` to ensure full linting, cross-compilation, and test coverage.



