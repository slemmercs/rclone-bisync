# Task Plan: Architecture Modularization & Unified Log Ledger Run-Level Consolidation

## Goal
Decompose 10 priority files approaching the 400 LOC ceiling across backend and frontend into focused single-responsibility modules, while consolidating Unified Log Ledger entries so sync runs display as a single unified action row with interactive deep-linking badges.

## Tasks

- [x] Task 1: Backend Go Models & Profile Store Decomposition (`internal/models/`, `internal/profiles/`)
  - **Scope**: `internal/models/api_explorer.go`, `internal/models/api_diagnostics.go`, `internal/models/api_logs.go`, `internal/models/api.go`, `internal/profiles/store_history.go`, `internal/profiles/store.go`
  - **Verification**: `go test -count=1 ./internal/models/... && go test -count=1 ./internal/profiles/...`

- [x] Task 2: Backend Handlers, Config, Services & MCP Tools Decomposition (`internal/server/`, `internal/config/`, `internal/services/`, `internal/mcp/`)
  - **Scope**: `internal/server/handlers_dedupe.go`, `internal/server/handlers_auth.go`, `internal/config/env.go`, `internal/config/config.go`, `internal/services/explorer_download.go`, `internal/services/mime_types.go`, `internal/services/explorer_preview.go`, `internal/mcp/tools_review.go`, `internal/mcp/tools_dedupe.go`, `internal/mcp/tools_write.go`
  - **Verification**: `go test -count=1 ./internal/server/... && go test -count=1 ./internal/config/... && go test -count=1 ./internal/services/... && go test -count=1 ./internal/mcp/...`

- [x] Task 3: Backend Logging Consolidation & Run Action Metadata (`internal/engine/runner/`, `internal/core/client/`)
  - **Scope**: `internal/engine/runner/runner.go`, `internal/engine/runner/runner_audit.go`, `internal/engine/runner/runner_handlers.go`, `internal/models/api.go`, `internal/core/client/monitor.go`
  - **Verification**: `go test -count=1 ./internal/engine/runner/...`

- [x] Task 4: Frontend Utility & Parser Modularization (`dashboard/frontend/js/`)
  - **Scope**: `dashboard/frontend/js/dedupe_parser.js`, `dashboard/frontend/js/utils.js`, `dashboard/frontend/index.html`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh loc 400`

- [ ] Task 5: Frontend Action Consolidation & Log Clustering (`dashboard/frontend/js/`)
  - **Scope**: `dashboard/frontend/js/logs_clustering.js`, `dashboard/frontend/js/logs_daemon.js`, `dashboard/frontend/js/logs.js`, `dashboard/frontend/index.html`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh loc 400`

- [ ] Task 6: Frontend Explorer & App State Modularization (`dashboard/frontend/js/`)
  - **Scope**: `dashboard/frontend/js/explorer_preview.js`, `dashboard/frontend/js/explorer_actions.js`, `dashboard/frontend/js/app_state.js`, `dashboard/frontend/app.js`, `dashboard/frontend/index.html`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh loc 400`

- [ ] Task 7: Interactive Ledger Badges & Full Pre-Commit Gate (`pattern_timeline_ledger.html`, Repository-Wide)
  - **Scope**: `dashboard/frontend/partials/patterns/pattern_timeline_ledger.html`, full test & check suite
  - **Verification**: `./scripts/check.sh all && ./scripts/check.sh loc 400`

## Status Updates

- 2026-11-14: Initialized task plan based on approved implementation plan.
- 2026-11-14: Completed Task 1 - Decomposed `internal/models/api.go` (from 396 to 174 LOC) into `api_explorer.go`, `api_diagnostics.go`, `api_logs.go`, and decomposed `internal/profiles/store.go` (from 395 to 320 LOC) by extracting `store_history.go`. Verified with unit tests.
- 2026-11-14: Completed Task 2 - Decomposed `internal/server/handlers_auth.go` (366 to 200 LOC) by extracting `handlers_dedupe.go`, `internal/config/config.go` (378 to 266 LOC) by extracting `env.go`, `internal/services/explorer_preview.go` (377 to 217 LOC) by extracting `explorer_download.go` and `mime_types.go`, and `internal/mcp/tools_write.go` (386 to 130 LOC) by extracting `tools_review.go` and `tools_dedupe.go`. Verified with unit tests across all four packages.
- 2026-11-14: Completed Task 3 - Enhanced sync run logging with run-level action metadata (`Action: "SYNC_EXECUTED"`, `Action: "SYNC_STARTED"`, `Action: "SYNC_FAILED"`, `Action: "SYNC_ABORTED"`), and linked audit milestone metadata (`SeqID`, `Actor`) directly into `BisyncLogEntry`. Verified with `go test -count=1 ./internal/engine/runner/...`.
- 2026-11-14: Completed Task 4 - Extracted `dashboard/frontend/js/dedupe_parser.js` (69 LOC) to slim `dashboard/frontend/js/utils.js` from 391 to 350 LOC, unpacked nested audit telemetry details into top-level properties in `parseJsonLogLine`, registered the new module in `index.html`. Verified with `./scripts/check.sh templates && ./scripts/check.sh loc 400` and e2e smoke tests.






