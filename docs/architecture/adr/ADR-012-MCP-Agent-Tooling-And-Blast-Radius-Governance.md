# ADR-012: Model Context Protocol (MCP) AI Agent Governance & Blast-Radius Circuit Breakers

**Status:** Accepted

**Date:** 2026-11-23

---

## Context

Modern AI coding and DevOps agents (such as Cline, Claude Desktop, Cursor, and automated assistants) increasingly operate directly on backend infrastructure, orchestrating storage profiles, triggering synchronization passes, and managing files across local and cloud environments.

Providing AI agents with raw shell access or unconstrained API keys poses severe operational risks:
1. **Catastrophic Data Deletion**: An agent misinterpreting a prompt could delete entire cloud buckets or local directories without confirmation.
2. **Synchronization Loops & Lock Collisions**: Spawning parallel or uncoordinated bisync runs can corrupt lock state or trigger emergency aborts.
3. **Path Traversal & Host Compromise**: An unconstrained tool could access or overwrite arbitrary files outside the synchronization root (e.g., `/etc/`, `~/.ssh/`).
4. **Secret Leakage**: Configuration payloads, access tokens, and cloud credentials could be returned unredacted in agent context buffers.

Bisync Hub requires a standardized, sandboxed, and governance-driven interface that enables AI agents to inspect, manage, and execute synchronization workflows safely.

---

## Decision

Bisync Hub adopts the **Model Context Protocol (MCP)** specification via a dedicated subsystem (`internal/mcp/`) that exposes controlled tools and resources over standard I/O (stdio) and Server-Sent Events (SSE) transports.

All MCP interactions are governed by the following architectural safeguards:

### 1. Dual Transport Architecture (`internal/mcp/server.go`)
- **CLI Stdio Mode**: Run via `rclone-bisync mcp` for direct invocation by local agent runners (e.g., Claude Desktop, Cline).
- **HTTP SSE Mode**: Integrated into the daemon web server (`/api/mcp/sse`) for networked agent clients.

### 2. Tiered Tooling & Blast-Radius Circuit Breakers (`internal/mcp/confirmation.go`)
MCP tools are partitioned into two distinct security tiers:
- **Tier 1: Read-Only Inspection Tools (Zero Confirmation)**:
  - `bisync_get_status`, `bisync_get_sync_history`, `bisync_get_logs`, `bisync_list_profiles`, `bisync_list_remotes`, `bisync_list_providers`, `bisync_explore_path`, `bisync_preview_file`, `bisync_validate_flags`, `bisync_inspect_conflict_session`.
  - These return sanitized, redacting-safe telemetry and file previews without modifying state.
- **Tier 2: Destructive / Mutating Operations (Mandatory Confirmation Token)**:
  - `bisync_delete_item`, `bisync_purge_dir`, `bisync_delete_remote`, `bisync_delete_profile`, `bisync_rollback_profile`, `bisync_resolve_review` (`execute=true`), `bisync_dedupe_by_hash` (`mode != 'list'`), and `bisync_trigger_sync` (`dry_run=false` or `resync=true`).
  - High-impact operations enforce a **two-phase token exchange**:
    1. First invocation without a token performs a blast-radius calculation, dry-run simulation, and returns an ephemeral cryptographic `confirmation_token` (valid for 5 minutes) describing exact impacted paths and counts.
    2. Execution is rejected unless the client re-invokes the tool supplying the valid `confirmation_token`.

### 3. Local Filesystem Jail Containment
- Any MCP file exploration or write operation targeting `local` storage resolves relative to the active profile's root path or an enforced jail root.
- Strict path confinement (`filepath.Rel` and `..` prefix rejection) blocks path traversal attempts.

### 4. Canary & Infrastructure Protection
- MCP file operations (`delete_item`, `purge_dir`, `move_item`) strictly refuse actions targeting canary files (`RCLONE_TEST`) or system root directories.
- Remote deletion safeguards (`bisync_delete_remote`) prevent deleting remotes linked to active sync profiles unless explicitly overridden.

### 5. Autonomous Secret Redaction
- All outputs returned to AI agents pass through recursive secret sanitization (`security.RedactSecrets`), masking passwords, tokens, client secrets, and webhooks.

---

## Rationale

1. **Industry Alignment**: MCP is an open, widely adopted standard for agent-to-environment communication, removing the need for proprietary plugin schemas.
2. **Deterministic Safety (Human-in-the-Loop & Circuit Breaker)**: The confirmation token pattern forces agents to pause, examine the blast radius, and present the action to the human user (or deliberate before executing), preventing accidental mass deletions.
3. **Defense-in-Depth**: Jail checks and canary guards in the MCP layer provide safety guarantees even if underlying API services or user prompts lack bounds.

---

## Consequences

### Positive
- AI agents can autonomously diagnose sync failures, explore file states, preview conflicts, and compile CLI flags with complete safety.
- Two-phase confirmation tokens prevent runaway deletion loops or unintended resync triggers.
- Secret sanitization guarantees credentials are never ingested into remote LLM training sets or context logs.
- Consistent developer experience across both terminal agents (stdio) and web dashboard agents (SSE).

### Negative / Trade-offs
- Multi-step confirmation requires agents to handle two-phase conversational turns for destructive operations.
- Token state must be maintained in memory with TTL expiry, requiring agent retries if expired.

---

## Compliance & Verification

- MCP handlers must be comprehensively unit tested in `internal/mcp/*_test.go` with mock confirmations, invalid tokens, and boundary violations.
- Secret redaction on MCP outputs is verified by test suites ensuring no tokens or credentials leak in tool payloads.
- Local filesystem traversal and canary protections are tested against malicious input payloads.