# Implementation Plan

[Overview]
Deliver Active Strategic Priority #1 ("In-App User Guidance & Interactive Diagnostic Manual") by embedding contextual policy explanations, an interactive slide-over reference manual, and real-time self-diagnostics into the Bisync Hub dashboard.

Bisync Hub prioritizes safety and human-in-the-loop decision making, but complex synchronization concepts like `TOO_MANY_DELETES` thresholds, `MANDATORY_SUBDIR` path safety, `RCLONE_TEST` canary tokens, compare modes, and heuristic risk scoring require clear, accessible explanations for operators. This implementation introduces an integrated guidance system consisting of contextual inline tooltips, a comprehensive slide-over Reference Manual with copyable runbooks and flag catalogs, and an interactive Diagnostics engine providing live health validation across background daemons, storage permissions, and remote authentication.

The architecture maintains Bisync Hub's zero-build, offline-first philosophy: all documentation, runbooks, and diagnostic templates are embedded into the single binary distribution via Go's standard `embed.FS` and modular Alpine.js components, ensuring 100% availability even when internet connectivity or remote daemons are offline.

[Types]
Type system additions in `internal/models/api.go` for the live system diagnostics API:

```go
// DiagnosticCheckStatus represents the outcome of a diagnostic evaluation.
type DiagnosticCheckStatus string

const (
	DiagnosticStatusPass    DiagnosticCheckStatus = "pass"
	DiagnosticStatusWarning DiagnosticCheckStatus = "warning"
	DiagnosticStatusFail    DiagnosticCheckStatus = "fail"
)

// DiagnosticCheck represents an individual operational health evaluation.
type DiagnosticCheck struct {
	ID          string                `json:"id"`
	Category    string                `json:"category"` // "daemon", "storage", "remotes", "safety", "system"
	Name        string                `json:"name"`
	Status      DiagnosticCheckStatus `json:"status"`
	Message     string                `json:"message"`
	Detail      string                `json:"detail,omitempty"`
	ActionTitle string                `json:"action_title,omitempty"`
	ActionURL   string                `json:"action_url,omitempty"`
}

// DiagnosticSystemInfo summarizes host environment and runtime metadata.
type DiagnosticSystemInfo struct {
	Version     string `json:"version"`
	GoVersion   string `json:"go_version"`
	OS          string `json:"os"`
	Arch        string `json:"arch"`
	PID         int    `json:"pid"`
	UptimeSec   int64  `json:"uptime_sec"`
	BaseDir     string `json:"base_dir"`
	LogFilePath string `json:"log_file_path"`
	IsSystemd   bool   `json:"is_systemd"`
}

// DiagnosticsResponse represents the response payload for GET /api/diagnostics.
type DiagnosticsResponse struct {
	Status     string               `json:"status"` // "ok", "warning", "error"
	Summary    map[string]int       `json:"summary"` // counts of pass, warning, fail
	Checks     []DiagnosticCheck    `json:"checks"`
	SystemInfo DiagnosticSystemInfo `json:"system_info"`
	Timestamp  string               `json:"timestamp"`
}
```

[Files]
File modifications:

1. **New Files**:
   - `internal/server/handlers_diagnostics.go`: Implements the `GET /api/diagnostics` HTTP handler.
   - `internal/server/handlers_diagnostics_test.go`: Unit test suite verifying `GET /api/diagnostics` scenarios.
   - `dashboard/frontend/js/help.js`: Zero-build Alpine.js module coordinating the Help Drawer, search indices, and API queries.
   - `dashboard/frontend/partials/components/component_help_drawer.html`: Go template partial defining the slide-over manual and diagnostics drawer.

2. **Existing Files to Modify**:
   - `internal/models/api.go`: Append diagnostic model structs.
   - `internal/server/server.go`: Register route `GET /api/diagnostics`.
   - `dashboard/frontend/index.html`: Add Help Drawer header trigger, keyboard shortcut, script tag, and drawer include.
   - `dashboard/frontend/app.js`: Compose `HelpModule` into root Alpine dashboard component.
   - `dashboard/frontend/partials/components/ui_components.html`: Add inline tooltip macros.
   - `dashboard/frontend/partials/modals/modal_profiles.html`: Add contextual policy tooltips.
   - `dashboard/frontend/partials/modals/modal_global_settings.html`: Add contextual tooltips to global settings.
   - `dashboard/frontend/partials/modals/modal_review.html`: Add contextual guidance on heuristic risk scores.
   - `dashboard/frontend/partials/view_fleet.html`: Add contextual help triggers.
   - `tests/e2e/dashboard_smoke_test.go`: Add headless Chrome smoke validation for the Help Drawer.

3. **Files to Delete or Move**: None.
4. **Configuration Updates**: None.

[Functions]
Function modifications:

1. **New Functions**:
   - `handleDiagnostics(w http.ResponseWriter, r *http.Request)` (`internal/server/handlers_diagnostics.go`): Collects diagnostic checks across daemons, base directories, profile canary states, and remotes auth, returning a structured JSON response.
   - `evaluateSystemChecks(ctx context.Context, s *Server) []models.DiagnosticCheck` (`internal/server/handlers_diagnostics.go`): Helper function performing non-blocking health audits against configured subsystems.
   - `initHelp()` (`dashboard/frontend/js/help.js`): Initializes the help drawer state, registers keyboard shortcuts, and pre-populates search indices.
   - `openHelpDrawer(tab, topic)` (`dashboard/frontend/js/help.js`): Opens the slide-over drawer to a specific tab (`manual`, `policies`, `diagnostics`, `runbook`) and optionally highlights a topic.
   - `closeHelpDrawer()` (`dashboard/frontend/js/help.js`): Closes the slide-over drawer with animation and clears search filters.
   - `fetchDiagnostics()` (`dashboard/frontend/js/help.js`): Queries `GET /api/diagnostics` and populates reactive diagnostic cards.
   - `copyCodeSnippet(text)` (`dashboard/frontend/js/help.js`): Copies CLI commands to the clipboard and triggers temporary copied feedback badge.

2. **Modified Functions**:
   - `(s *Server) routes()` (`internal/server/server.go`): Add registration for `s.Mux.HandleFunc("GET /api/diagnostics", s.handleDiagnostics)`.
   - `composeDashboard(...)` (`dashboard/frontend/app.js`): Include `window.HelpModule` in the composed dashboard object.

3. **Removed Functions**: None.

[Classes]
Class / Module modifications:

1. **New Modules / Structs**:
   - `DiagnosticCheck`: Data model representing a single evaluation check.
   - `DiagnosticsResponse`: Root payload model for diagnostic telemetry.
   - `HelpModule` (`dashboard/frontend/js/help.js`): Alpine.js module object managing drawer visibility, active tab, quick search query, filtered flag catalog, and diagnostic state.

2. **Modified Structs**:
   - `Server` (`internal/server/server.go`): Uses existing services (`TelemetryService`, `ProfileService`, `AuthService`, `Config`) without adding new fields or bloating container contracts.

3. **Removed Classes**: None.

[Dependencies]
No new third-party dependencies or external packages are introduced.
- Backend: Standard Go library packages (`context`, `encoding/json`, `net/http`, `os`, `runtime`, `time`).
- Frontend: Existing embedded Alpine.js, Tailwind CSS, and embedded SVG icons. Zero-build maintained.

[Testing]
Testing approach:

1. **Unit Testing (`internal/server/handlers_diagnostics_test.go`)**:
   - `TestHandleDiagnostics_AllHealthy`: Asserts HTTP 200 and all checks pass when daemon is online and directories are writable.
   - `TestHandleDiagnostics_DaemonDown`: Asserts proper classification and warning/fail status when Rclone RC daemon is unreachable.
   - `TestHandleDiagnostics_RemotesAuthWarning`: Asserts warning checks raised when a remote has an expired or missing OAuth grant.
   - `TestHandleDiagnostics_CanaryMissing`: Asserts warning checks raised when active profile has missing canary files.
2. **End-to-End Testing (`tests/e2e/dashboard_smoke_test.go`)**:
   - `TestDashboard_HelpDrawerAndGuidance`: Launches headless Chromium, triggers the help drawer button, verifies drawer rendering, cycles through `manual`, `policies`, `diagnostics`, and `runbook` tabs, and confirms zero console errors or uncaught exceptions.
3. **Full Quality Gate Verification**:
   - Targeted unit tests: `go test -race -v -count=1 ./internal/server/... -run TestHandleDiagnostics`
   - Full package tests: `go test -race ./...`
   - Quality gate script: `./scripts/check.sh all` (all 7 quality check stages).

[Implementation Order]
1. Define diagnostic data models in `internal/models/api.go`.
2. Implement `handleDiagnostics` and system evaluation checks in `internal/server/handlers_diagnostics.go`.
3. Write unit tests in `internal/server/handlers_diagnostics_test.go` and verify with `go test -race`.
4. Register `GET /api/diagnostics` in `internal/server/server.go`.
5. Create frontend module `dashboard/frontend/js/help.js` providing drawer control, search filtering, and diagnostic API integration.
6. Create `dashboard/frontend/partials/components/component_help_drawer.html` containing the Reference Manual, Safety Policies, Diagnostics, and Runbook UI tabs.
7. Update `dashboard/frontend/partials/components/ui_components.html` with inline tooltip patterns.
8. Embed contextual tooltips into `modal_profiles.html`, `modal_global_settings.html`, and `modal_review.html`.
9. Update `dashboard/frontend/index.html` to add the header trigger button, keyboard shortcut handler, script tag, and drawer template include.
10. Wire `HelpModule` into `dashboard/frontend/app.js`.
11. Add E2E tests in `tests/e2e/dashboard_smoke_test.go` and execute headless Chrome smoke test.
12. Run full verification (`go test -race ./...` and `./scripts/check.sh all`).




