---
layout: default
title: "6. Model Context Protocol (MCP) Server"
parent: Core Pillars
nav_order: 6
---

# Native Model Context Protocol (MCP) Server

Bisync Hub exposes a built-in [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) server directly from the `bisync-hub` binary—no Python wrappers, Node bridges, or log scrapers required.

---

## Core Capabilities

* **Dual Transport (`stdio` & Embedded `SSE`):** Local agents (Cline, Claude Desktop, Cursor) can spawn `bisync-hub mcp` over `stdio`, while remote or multi-agent setups can connect to the embedded `SSE` endpoint served alongside the dashboard daemon.
* **Parameter-Fingerprinted Confirmation Tickets:** Destructive or state-altering calls (`bisync_trigger_sync` with `resync`/`force`, `bisync_resolve_review` with `execute=true`, or mutating `bisync_dedupe_by_hash` modes) require a two-step confirmation flow. Tickets carry a short TTL and bind to a `SHA-256` hash of the exact parameters—changing arguments invalidates the ticket.
* **Read-Only Mode & Guardrails:** Running with read-only mode enabled (`MCP_READ_ONLY=true` / `--read-only`) blocks mutating tools, while path confinement jails and recursive secret redaction remain active on every call.
* **Context-Window Discipline:** `bisync_get_status` returns a compact summary by default, `bisync_get_logs` supports level filtering, and `bisync_explore_path` paginates results so agents don't burn context on noisy output.

---

## Agent Configuration Examples

### Cline (`~/.cline/data/settings/cline_mcp_settings.json`)
```json
{
  "mcpServers": {
    "bisync-hub": {
      "command": "/usr/local/bin/bisync-hub",
      "args": ["mcp"],
      "disabled": false,
      "autoApprove": [
        "bisync_get_status",
        "bisync_get_logs",
        "bisync_list_providers",
        "bisync_list_profiles",
        "bisync_list_remotes",
        "bisync_get_sync_history",
        "bisync_inspect_conflict_session",
        "bisync_explore_path",
        "bisync_preview_file",
        "bisync_validate_flags"
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
      "command": "/usr/local/bin/bisync-hub",
      "args": ["mcp"]
    }
  }
}
```
