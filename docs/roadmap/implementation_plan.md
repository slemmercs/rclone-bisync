# Implementation Plan: Bisync Hub Model Context Protocol (MCP) Server

## Proposed Changes

Implement a native Go Model Context Protocol (MCP) server directly inside the `rclone-bisync` binary (`rclone-bisync mcp`), integrating AI agent workflows (Claude Desktop, Cursor, Cline, Ollama) with Bisync Hub.

### Affected Components & Packages

- `go.mod` & `THIRD_PARTY_LICENSES.md`:
  - Add `github.com/mark3labs/mcp-go` (MIT license) as direct dependency.
  - Document license and terms in `THIRD_PARTY_LICENSES.md`.
- `internal/mcp/`:
  - `server.go`: Core MCP server creation, tool/resource registry, transport dispatch (`stdio` or `sse`).
  - `confirmation.go`: High-risk mutation ticket engine (TTL-backed confirmation token generator and validator).
  - `tools_read.go`: Read-only MCP tools (`bisync_get_health`, `bisync_list_profiles`, `bisync_review_queue`, `bisync_audit_inspect`).
  - `tools_write.go`: Mutating MCP tools (`bisync_trigger_sync`, `bisync_resolve_review`, `bisync_dedupe_by_hash`) with read-only guards and confirmation gating.
  - `resources.go`: Read-only MCP resources (`bisync://telemetry`, `bisync://profiles`, `bisync://audit`).
  - `server_test.go` & `tools_test.go`: Comprehensive unit tests covering tool invocations, resources, confirmation tickets, and read-only protections.
- `main.go`:
  - Add `case "mcp":` CLI subcommand with `--read-only`, `--require-confirmation`, `--transport`, `--host`, `--port`, `--env-file`, and `--debug` flags.
  - Add `runMCP(...)` runner function managing container lifecycle and MCP transport execution.

### Technical Strategy & Data Flow

1. **Protocol Engine**:
   - Utilize `github.com/mark3labs/mcp-go` (MIT, zero external runtime dependencies) targeting latest MCP specification.
   - Dual transport: default to standard I/O (`stdio`) for local agent CLI sessions, support streamable HTTP / Server-Sent Events (`sse`) for networked installations.

2. **Caller Context & Telemetry Tracking**:
   - All operations initiated via MCP specify execution origin as `"mcp"`.
   - The Unified Ledger timeline and telemetry will log and render the violet `"MCP"` badge.

3. **Read-Only / Read-Write Operational Modes**:
   - Flag `--read-only` enforces strict read-only operation: mutating tools return error `server is operating in read-only mode`.

4. **Confirmation Tickets for Destructive Operations**:
   - Flag `--require-confirmation` (enabled by default for mutating actions or configurable):
   - Operations with high blast radius (such as `resync=true`, `force=true`, `bisync_dedupe_by_hash`, `bisync_resolve_review` delete actions) generate a unique cryptographic token (TTL: 5m) with action fingerprint.
   - The agent is instructed to present the confirmation ticket in a follow-up call to confirm intent.

### Safety & Error Handling

- Respect existing canary files (`RCLONE_TEST`) and circuit breaker limits (`TOO_MANY_DELETES`, `MANDATORY_SUBDIR`).
- Sanitize sensitive credentials (passwords, tokens, secret keys) via `security.RedactSecrets` before returning JSON outputs in MCP tools or resources.
- Strict path confinement and read-only mode guarantees.


### LLM Token Efficiency & Ergonomics Optimizations

1. **Parameter Fingerprint Validation**:
   - `ValidateTicket` and `ConsumeTicket` strictly enforce parameter equality (`params map[string]string`) so tokens cannot be reused with altered flags (e.g., escalating from dry-run to force resync).

2. **Status Primitives vs. Monolithic Telemetry**:
   - `bisync_get_status` returns concise, high-signal primitives by default. Verbose Rclone logs, historical metrics, and runtime configs are opt-in via `include_logs` and `include_config`.

3. **Paginated & Projected Explorations**:
   - `bisync_explore_path` implements `limit`, `offset`, and filename `filter`, projecting only essential fields (`name`, `path`, `is_dir`, `size`, `mod_time`) with summary counts (`total_items`, `returned_items`, `has_more`).

4. **Bounded Dedupe & Review Engine Alignment**:
   - `bisync_dedupe_by_hash` defaults to safe mode `"list"`, truncates CLI terminal outputs for LLM presentation, and complies with ADR-004 by omitting raw terminal output from SQLite audit payloads.
   - `bisync_resolve_review` aligns parameter schema with actual domain decisions (`keep-local`, `keep-remote`, `keep-both`, `restore-item`, `accept-delete`, `skip`, `dismiss`).

## Verification Plan

- `go test -v -count=1 ./internal/mcp/...`
- `./scripts/check.sh licenses`
- `./scripts/check.sh loc 400`
- `./scripts/check.sh all`
- CLI test: `rclone-bisync mcp --help`







