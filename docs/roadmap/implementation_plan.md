# Implementation Plan: MCP Inspector Modal & Environment Read-Only Configuration

## Proposed Changes

Provide complete end-to-end operational visibility and configuration control for the native Model Context Protocol (MCP) server.

1. **Environment Configuration for MCP Read-Only Mode (`MCP_READ_ONLY`)**:
   - Allow operators to enforce read-only safety mode directly in `bisync.env` without altering systemd unit files or CLI arguments.
   - Command-line flag `--read-only` remains an override (`*readOnly || cfg.MCPReadOnly`).

2. **Dashboard MCP Inspector Modal (`modal_mcp.html`)**:
   - Transform the header status pill (`MCP: ACTIVE [read/write]`) into an interactive element with cursor feedback and click handler (`openMcpModal()`).
   - Create a dedicated modal displaying:
     - Real-time connection status, transport (`sse` / `stdio`), host, port, process ID (PID), and uptime.
     - Security mode badge (`Read/Write` with dynamic confirmation tokens vs `Read-Only`).
     - Interactive catalog of all 11 MCP tools partitioned into Inspection and Mutating tools with live permission tags.
     - 1-click copyable AI assistant integration configuration snippet (for Claude Desktop, Cline, Cursor, Windsurf).
     - Configuration guide explaining how to toggle `MCP_READ_ONLY` in `bisync.env` or systemd.

### Affected Components & Packages

- `internal/config/config.go`: Add `MCPReadOnly bool` to `Config` struct, loaded via `getEnvBool("MCP_READ_ONLY", false)`.
- `internal/config/config_test.go`: Add test asserting `MCP_READ_ONLY` parsing.
- `main.go`: Update `runMCP` call in `mcp` CLI subcommand to incorporate `cfg.MCPReadOnly`.
- `dashboard/frontend/partials/modals/modal_mcp.html`: New modal partial with design-system styling.
- `dashboard/frontend/index.html`: Include `{{ template "modal_mcp" . }}` and make the header MCP pill clickable.
- `dashboard/frontend/app.js`: Add `showMcpModal`, `openMcpModal()`, `closeMcpModal()`, `copyMcpConfig()`, and `copyMcpEndpoint()` state helpers.

### Safety & Error Handling

- Read-only protection: If `cfg.MCPReadOnly` is true or `--read-only` flag is given, mutating tools (`bisync_trigger_sync`, `bisync_resolve_review`, `bisync_dedupe_by_hash`) are rejected with an explicit security error.
- Dynamic confirmation tickets: High blast-radius mutating tools require a cryptographically verified `confirmation_token` with 5-minute TTL.
- Zero-leak safety: No environment secrets or tokens are rendered in the MCP modal; only the local loopback SSE endpoint is displayed.

## Verification Plan

- `go test -v -count=1 ./internal/config/... -run TestLoadConfig_MCPReadOnly`
- `./scripts/check.sh all` (all 7 quality gates including HTML scanner, Go vet, linter, tests, and LOC budgets)
- Browser manual inspection / verification of modal rendering and clipboard copy actions.

