# Task Plan: Bisync Hub Model Context Protocol (MCP) Server

## Goal
Implement a native Go Model Context Protocol (MCP) server directly inside the single `rclone-bisync` binary, exposing read-only inspection tools/resources, safely gated mutation tools with confirmation tickets, dual stdio/SSE transports, and CLI subcommand integration.

## Tasks

- [x] Task 1: Dependency Integration & License Compliance Guard
  - **Scope**: `go.mod`, `go.sum`, `THIRD_PARTY_LICENSES.md`
  - **Verification**: `./scripts/check.sh licenses`

- [x] Task 2: Confirmation Ticket Engine & MCP Server Scaffold
  - **Scope**: `internal/mcp/confirmation.go`, `internal/mcp/server.go`, `internal/mcp/confirmation_test.go`
  - **Verification**: `go test -v -count=1 ./internal/mcp/...`

- [x] Task 3: Read-Only MCP Tools & Resources
  - **Scope**: `internal/mcp/tools_read.go`, `internal/mcp/resources.go`, `internal/mcp/tools_read_test.go`
  - **Verification**: `go test -v -count=1 -run "TestReadTools|TestResources" ./internal/mcp/...`

- [ ] Task 4: Mutating MCP Tools with Read-Only Guard & Confirmation
  - **Scope**: `internal/mcp/tools_write.go`, `internal/mcp/write_test.go`
  - **Verification**: `go test -v -count=1 -run "TestWriteTools|TestConfirmationFlow" ./internal/mcp/...`

- [ ] Task 5: CLI Subcommand `rclone-bisync mcp` & Quality Gate Validation
  - **Scope**: `main.go`, `internal/mcp/server_test.go`, documentation updates
  - **Verification**: `go test -count=1 ./...` and `./scripts/check.sh all`

## Status Updates

- 2026-11-01: Initialized task plan for Model Context Protocol (MCP) server implementation on branch feat/mcp-server.
- 2026-11-01: Completed Task 1: Added github.com/mark3labs/mcp-go (v1.5.0, MIT) to go.mod, documented in THIRD_PARTY_LICENSES.md, and verified zero copyleft violations via ./scripts/check.sh licenses.
- 2026-11-01: Completed Task 2: Implemented confirmation ticket engine (internal/mcp/confirmation.go) and MCP server scaffolding with dual transport (stdio and streamable SSE) in internal/mcp/server.go. Verified via internal/mcp unit tests and LOC check.
- 2026-11-01: Completed Task 3: Implemented 6 read-only MCP inspection tools (bisync_list_profiles, bisync_get_status, bisync_list_remotes, bisync_get_sync_history, bisync_inspect_conflict_session, bisync_explore_path with jail confinement) and 5 MCP resources (bisync://profiles, bisync://status, bisync://profile/{id}, bisync://history/{id}, bisync://conflicts/{session_id}). Verified via comprehensive unit tests in internal/mcp/tools_read_test.go and LOC check.









