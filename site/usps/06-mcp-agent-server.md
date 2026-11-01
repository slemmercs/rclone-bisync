---
layout: default
title: "6. Model Context Protocol (MCP) Server"
parent: Core Pillars
nav_order: 6
---

# Native Model Context Protocol (MCP) Server

Bisync Hub integrates directly into AI agent workflows (Claude Desktop, Cursor, Cline, OpenCode) through a native, zero-dependency implementation of the [Model Context Protocol (MCP)](https://modelcontextprotocol.io/).

Instead of running external Python bridges, Node.js wrappers, or custom subprocess scrapers, **Bisync Hub exposes operational inspection tools, safe mutation primitives, and live telemetry resources directly from the single Go binary**.

```text
┌────────────────────────────────────────────────────────────────────────┐
│                   AI Agent (Cline / Claude / Cursor)                   │
└────────────────────────────────────┬───────────────────────────────────┘
                                     │ JSON-RPC (stdio / SSE)
                                     ▼
┌────────────────────────────────────────────────────────────────────────┐
│                        Bisync Hub MCP Server                           │
│                                                                        │
│  [ Confirmation Ticket Engine ]  ──►  Exact Parameter Fingerprint      │
│  [ Read-Only Enforcement ]       ──►  Strict Mutation Guard            │
│  [ Jail-Confinement Layer ]      ──►  Path Traversal Protection        │
│                                                                        │
│  Tools:                                Resources:                      │
│   • bisync_get_status (Token-Optimized) • bisync://profiles            │
│   • bisync_get_logs (Level-Filtered)    • bisync://status              │
│   • bisync_list_profiles                • bisync://profile/{id}        │
│   • bisync_list_remotes                 • bisync://history/{id}        │
│   • bisync_get_sync_history             • bisync://conflicts/{session} │
│   • bisync_inspect_conflict_session                                    │
│   • bisync_explore_path (Paginated)                                    │
│   • bisync_trigger_sync (Confirmation Gated)                           │
│   • bisync_resolve_review (Backup Staged)                              │
│   • bisync_dedupe_by_hash (Safe Preview Default)                       │
└────────────────────────────────────┬───────────────────────────────────┘
                                     │
                                     ▼
                     [ Bisync Hub Core Engine & Jails ]
```

---

## Architectural Principles

### 1. Dual Transport Architecture (`stdio` & `sse`)
The MCP server operates out of the box with zero runtime dependencies:
- **Standard I/O (`stdio`)**: Ideal for local agent sessions in Cline, Cursor, or Claude Desktop. The agent spawns the `rclone-bisync mcp` command as a managed subprocess.
- **Server-Sent Events (`sse`)**: Streamable HTTP transport for remote installations, Docker containers, or centralized orchestration servers (`rclone-bisync mcp --transport sse --host 0.0.0.0 --port 8081`).

### 2. High-Blast-Radius Confirmation Tickets
Mutating tools that could alter files or rebuild sync state enforce a cryptographic, two-step confirmation loop when `--require-confirmation` is enabled (default):
1. **Initial Call**: The agent invokes an action requiring elevated confirmation (e.g. `bisync_trigger_sync` with `resync=true` or `force=true`, `bisync_resolve_review` with `execute=true`, or `bisync_dedupe_by_hash` with `mode != "list"`).
2. **Ticket Generation**: Bisync Hub generates a temporary, cryptographically random confirmation ticket (5-minute TTL) with an exact SHA-256 parameter fingerprint.
3. **Execution Verification**: The action executes only if the agent resubmits the valid `confirmation_token` with the exact same parameters. Any attempt to alter flags between stages immediately invalidates the ticket.

### 3. Deep Defense-in-Depth & Read-Only Safety
- **Strict Read-Only Mode**: Launching with `--read-only` disables all write tools (`bisync_trigger_sync`, `bisync_resolve_review`, `bisync_dedupe_by_hash`), returning an instant security error if called.
- **Path Confinement Jails**: `bisync_explore_path` guarantees strict isolation via `filepath.Rel` and prevents directory traversal attacks outside the configured local root.
- **Secret Redaction**: Every response automatically strips credential patterns (`PASS`, `KEY`, `TOKEN`, `SECRET`) via recursive tree sanitization.

### 4. LLM Token Optimization & Ergonomics
To prevent agent context windows from being overwhelmed:
- **Compact Default Status**: `bisync_get_status` outputs a 195-character summary by default (**~98.5% token reduction**), deferring heavy logs and configs to explicit opt-in flags.
- **Dedicated Log Filtering**: `bisync_get_logs` retrieves tail-limited logs with severity filtering (`ERROR`, `NOTICE`, `INFO`).
- **Paginated Explorer**: `bisync_explore_path` provides windowed pagination (`limit`, `offset`) and filename substring filtering.
- **Bounded Dedupe Output**: `bisync_dedupe_by_hash` truncates terminal output to 2,000 characters and avoids writing empty audit records for zero-deletion dry runs per ADR-004.

---

## Agent Configuration

### Cline (`~/.cline/data/settings/cline_mcp_settings.json`)
```json
{
  "mcpServers": {
    "rclone-bisync": {
      "command": "/usr/local/bin/rclone-bisync",
      "args": ["mcp"],
      "disabled": false,
      "autoApprove": [
        "bisync_get_status",
        "bisync_get_logs",
        "bisync_list_profiles",
        "bisync_list_remotes",
        "bisync_get_sync_history",
        "bisync_inspect_conflict_session",
        "bisync_explore_path"
      ]
    }
  }
}
```

### Claude Desktop (`claude_desktop_config.json`)
```json
{
  "mcpServers": {
    "bisync-hub": {
      "command": "/usr/local/bin/rclone-bisync",
      "args": ["mcp"]
    }
  }
}
```
