---
layout: default
title: Architecture & Safety
nav_order: 7
description: "Core safety tenets, POSIX advisory locks, path confinement, and defense-in-depth architecture."
---

# Architecture & Safety Principles

Bisync Hub is designed under the assumption that networks drop, cloud APIs produce unexpected HTTP responses, and distributed clocks drift. 

Every design decision adheres to three foundational tenets:

1. **Visibility Before Action**: Never mutate remote storage silently without an immutable record and pre-flight inspection.
2. **Safety Before Automation**: If an environmental invariant is violated (e.g. missing canary, unresolvable conflict), halt immediately rather than guess.
3. **Defense in Depth**: Protect the host filesystem with path confinement jails, POSIX process flocking, and secret scrubbing.

---

## Safety Mechanisms

### 1. Filesystem Path Confinement Jails
When browsing files or preparing sync operations on local storage endpoints, Bisync Hub prevents arbitrary filesystem access by enforcing strict jail boundaries:

```go
// Enforcing local path isolation
relPath, err := filepath.Rel(jailRoot, targetPath)
if err != nil || strings.HasPrefix(relPath, "..") || strings.HasPrefix(relPath, "/") {
    return ErrPathTraversalViolation
}
```
Any request attempting directory traversal (`../`) outside the configured root returns an immediate authorization violation and is recorded in the audit log.

### 2. POSIX Flock Process Isolation
To avoid lock races between scheduled background timers, cron jobs, and interactive operator CLI requests, Bisync Hub acquires advisory file locks using native POSIX `flock(fd, LOCK_EX|LOCK_NB)`.
- Concurrent executions fail fast with a clear error rather than piling up or corrupting state.
- Stale lock detection cleans up safely if a previous process terminated abnormally.

### 3. Secret Redaction at Every Boundary
All external inputs, configuration files, and API logs pass through recursive sanitization via `security.RedactSecrets`.
- Keywords matching `PASS`, `TOKEN`, `KEY`, `SECRET`, and `CREDENTIAL` are dynamically masked (`[REDACTED]`).
- Raw tokens are never persisted in SQLite ledger records, JSON error messages, or telemetry streams.

### 4. Zero-Exposure CI/CD Security
Bisync Hub's GitHub Actions pipelines are deployed 100% on cloud-hosted ephemeral runners (`ubuntu-latest`) with zero self-hosted runner exposure. Every commit is validated with:
- Static analysis via `go vet` and `staticcheck`
- Automated leak scanning via `betterleaks`
- Vulnerability auditing via `govulncheck`
- Strict branch protection with sole review governance (`@slemmercs`)

### 5. High-Performance Go Quality Gate (`cmd/check`)
To catch regressions early during local development, Bisync Hub includes a dedicated native Go audit engine (`cmd/check`) executing check stages in parallel:
- **True HTML Tokenization**: Multi-line attribute tokenization preventing template boolean attribute coercion bugs (`:disabled`, `:readonly`).
- **Architectural LOC Budgets**: Single-pass line-count enforcement ensuring services remain modular and stay under ~400 LOC.
- **Copyleft License Auditing**: Native dependency license validator (`cmd/check/licenses.go`) enforcing permissive-only licensing (MIT, Apache-2.0, BSD) across all Go modules.
- **Coverage Assurance**: Pre-commit validation enforcing >80% statement coverage across all packages (currently at 88.3%).

### 6. MCP Confirmation Tickets & Parameter Fingerprinting
To safely allow autonomous AI agents (Claude, Cursor, Cline) to interact with synchronization workloads without risking accidental mass deletions or unexpected resyncs, the Model Context Protocol engine implements cryptographic ticket gating:
- **Parameter Fingerprinting**: Every issued ticket binds to the exact parameter hash (`SHA256(canonical(params))`). Re-submitting a ticket with altered flags (e.g. escalating from dry-run to force resync) immediately rejects execution.
- **5-Minute Non-Renewable TTL**: Confirmation tickets expire automatically after 300 seconds.
- **Read-Only Process Isolation**: When started with `--read-only`, mutating tools are strictly disabled at the registration level.

---

## Control Plane & Modern Cockpit Architecture

### 7. Finite State Machine (FSM) Control Plane & Queue Orchestrator
Bisync Hub replaces ad-hoc shell execution with an event-driven Finite State Machine governing sync workload lifecycles:
- **Deterministic State Graph**: Workloads transition through strictly validated states: `idle` → `validating` → `running` → `review_required` / `completed` / `failed` / `aborted`. Invalid transitions are rejected at the engine level.
- **Queue Engine & Profile Isolation**: Synchronization jobs are dispatched through a prioritized concurrency manager ensuring no two jobs for the same profile execute simultaneously or compete for advisory file locks.
- **Cooperative Subprocess Cancellation**: Sync runs respect context cancellation tokens. When aborted by an operator or circuit breaker, process trees are cleanly terminated and canary files are safely restored.

### 8. Vue 3 + PrimeVue v4 Modern Cockpit SPA
The operator cockpit is built as a reactive Single Page Application (SPA) embedded directly into the Go executable:
- **Zero-Runtime Single Binary**: Built with Vue 3.4 SFCs and PrimeVue v4, compiled into static bundles (`dashboard/frontend/dist/`) via Vite, and compiled into the Go binary with native `//go:embed`. Zero Node.js or npm runtime required on target machines.
- **Decoupled Pinia State Management**: Cross-cutting domain concerns live in dedicated, strongly-typed stores (`profiles`, `remotes`, `explorer`, `audit`, `dedupe`, `telemetry`, `settings`), eliminating global `window.*` mutations.
- **Operational Ergonomics**:
  - Live Performance Telemetry with persistent collapse states via `localStorage`.
  - Sticky backdrop-blurred modal action footers ensuring save, next, and deploy buttons stay anchored across long scrolling views.
  - Dual-pane local/remote filesystem explorer with drag-and-drop transfers and instant preview drawers.

### 9. Interactive Deduplication Workspace
Bisync Hub provides a visual deduplication engine supporting both local and remote storage targets:
- **Dual Matching Strategies**: Deduplication can match duplicates by cryptographic hash (`SHA-256`/`MD5`) or by filename and file size for backends that lack native hash support.
- **Cluster Hierarchy**: Duplicate groups are organized into collapsible clusters showing size distributions, modified timestamps, and replica paths.
- **Staged Review Drawer**: Decisions (`keep-local`, `keep-remote`, `keep-both`, `newest`, `oldest`) are staged in memory before execution, guarded by confirmation tokens to avoid accidental bulk loss.

### 10. Tamper-Evident Cryptographic Audit Ledger
To guarantee non-repudiation and regulatory compliance for storage modifications:
- **Trigger-Enforced Immutability**: The SQLite ledger employs database triggers (`BEFORE UPDATE` and `BEFORE DELETE`) raising SQL abort exceptions on any modification attempt.
- **Block Chaining**: Every record computes an SHA-256 block hash incorporating the previous entry's hash:
  `hash = SHA256(prev_hash || timestamp || action || profile_id || status || details)`
- **In-Browser Chain Verification**: The Cockpit SPA executes cryptographic hash chain audits directly in WebAssembly/Web Crypto, immediately flagging any ledger divergence or missing block.
