---
layout: default
title: Architecture & Safety
nav_order: 7
description: "Core safety tenets, path confinement, POSIX locking, FSM orchestration, and cryptographic auditability."
---

# Architecture & Safety Principles

Bisync Hub is built on the assumption that networks drop, cloud APIs return unexpected responses, and distributed clocks drift. Every subsystem follows three core tenets:

1. **Visibility Before Action:** Never mutate storage silently without pre-flight inspection and an immutable audit trail.
2. **Safety Before Automation:** Halt immediately when an invariant is violated (such as a missing canary file or high deletion threshold) rather than guessing.
3. **Defense in Depth:** Isolate local paths, serialize concurrent executions, and scrub secrets at every boundary.

---

## 1. Safety & Isolation Boundaries

### Filesystem Path Confinement Jails
Local file browsing and sync operations enforce strict root containment so relative traversal (`../`) or escape paths cannot step outside configured directories:

```go
relPath, err := filepath.Rel(jailRoot, targetPath)
if err != nil || strings.HasPrefix(relPath, "..") || strings.HasPrefix(relPath, "/") {
    return ErrPathTraversalViolation
}
```

Any boundary violation is rejected immediately and recorded in the audit ledger.

### POSIX Advisory Process Locking
To prevent race conditions between scheduled background runs, manual CLI commands, and concurrent profile actions, Bisync Hub uses non-blocking POSIX advisory locks (`flock` with `LOCK_EX|LOCK_NB`) per lock target, with stale-lock cleanup after abnormal exits and fail-fast reporting on active contention.

### Secret Redaction
Inputs, configuration payloads, error messages, and logs pass through recursive sanitization (`security.RedactSecrets`), masking sensitive fields (`PASS`, `TOKEN`, `KEY`, `SECRET`, `CREDENTIAL`) as `[REDACTED]` before persistence or external emission.

### MCP Confirmation Ticket Gating
AI agents connecting over Model Context Protocol (`stdio` or embedded `SSE`) cannot trigger destructive operations (`trigger_sync` with `resync`/`force`, `resolve_review`, `dedupe_by_hash`) without passing a two-step confirmation flow:

* **Parameter Fingerprinting:** Tickets bind to `SHA256(canonical(params))`—any change to flags or arguments invalidates the ticket.
* **Short TTL & Read-Only Mode:** Tickets expire automatically after 5 minutes, and launching with `MCP_READ_ONLY=true` disables mutating tool registration completely.

---

## 2. Control Plane & Execution Engine

### Finite State Machine (FSM) & Queue Orchestrator
Workload execution is governed by an explicit state machine rather than ad-hoc shell scripts:

`idle` → `validating` → `running` → `review_required` / `completed` / `failed` / `aborted`

* **Invalid Transition Guarding:** Disallowed state transitions are rejected at the engine level.
* **Concurrency & Cancellation:** A bounded queue orchestrator prevents per-profile lock collisions while context cancellation cleanly stops active subprocesses and restores canary state on abort.

### Interactive Deduplication Workspace
Duplicate detection supports both content hashes (`MD5` / `SHA-256`) and size-plus-filename matching for remotes without native checksum support. Results group into collapsible clusters with staged actions (`keep-local`, `keep-remote`, `keep-both`, `newest`, `oldest`) backed by confirmation checks before deletion.

---

## 3. Auditability & Embedded Cockpit UI

### Cryptographic Immutable Audit Ledger
State-changing operations and safety events are written to an append-only SQLite ledger protected by:

* **Database Triggers:** `BEFORE UPDATE` and `BEFORE DELETE` triggers block in-place edits or deletions at the storage layer.
* **Hash Chaining:** Each entry links to its predecessor via `SHA-256(prev_hash || timestamp || action || profile_id || status || details)`.
* **Client-Side Verification:** The dashboard can verify chain integrity directly in the browser and export records as JSONL or CSV.

### Single-Binary Vue 3 + PrimeVue Cockpit
The management UI is compiled into static assets and embedded directly into the Go binary via `//go:embed`, requiring zero Node.js or external runtime dependencies on target hosts:

* **Modular State:** Pinia stores isolate domain state across profiles, remotes, file exploration, deduplication, audit logs, telemetry, and settings.
* **Operational Views:** Includes real-time status/progress updates, dual-pane local/remote file browsing, review/diff modals, and built-in diagnostic runbooks.
