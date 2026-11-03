# Task Plan: MCP Inspector Modal & Environment Read-Only Configuration

## Goal
Provide full operational visibility and configuration control for the native Model Context Protocol (MCP) server by adding `MCP_READ_ONLY` support in `bisync.env` and introducing an interactive MCP Inspector Modal accessed from the header status pill.

## Tasks

- [x] Task 1: Environment Configuration for MCP Read-Only Mode (`MCP_READ_ONLY`)
  - **Scope**: `internal/config/config.go`, `internal/config/config_test.go`, `main.go`
  - **Verification**: `go test -v -count=1 ./internal/config/... -run "TestLoadConfig_MCPReadOnly"` and `./scripts/check.sh all`

- [x] Task 2: Interactive MCP Inspector Modal (`modal_mcp.html`) & Header Integration
  - **Scope**: `dashboard/frontend/partials/modals/modal_mcp.html`, `dashboard/frontend/index.html`, `dashboard/frontend/app.js`
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-11-03: Initialized task plan from approved `implementation_plan.md` on branch `feat/mcp-inspector-modal-and-env-config`.
- 2026-11-03: Completed Task 1:
  - Added `MCPReadOnly bool` configuration setting in `internal/config/config.go` parsed via `getEnvBool("MCP_READ_ONLY", false)`.
  - Updated `runMCP` in `main.go` so `cfg.MCPReadOnly` sets default server read-only mode alongside `--read-only` flag.
  - Added unit test `TestLoadConfig_MCPReadOnly` in `internal/config/config_test.go`.
  - Verified with `go test` and all 7 stages of `./scripts/check.sh all`.
- 2026-11-03: Completed Task 2:
  - Created `dashboard/frontend/partials/modals/modal_mcp.html` with tabs for server status, client integration snippet (Claude Desktop / Cline / Cursor), and an interactive 11-tool catalog.
  - Created `dashboard/frontend/js/mcp.js` exposing `window.McpModule` with clipboard copy helpers and endpoint URL builders.
  - Transformed header MCP status pill in `dashboard/frontend/index.html` into an interactive button opening the MCP inspector modal.
  - Added headless Chrome smoke test assertion `TestDashboard_HeadlessChromeSmoke/MCPModalInteraction`.
  - Verified with all 7 quality gate checks in `./scripts/check.sh all`.


