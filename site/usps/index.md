---
layout: default
title: Core Pillars
nav_order: 3
has_children: true
permalink: /usps
---

# Core Architectural Pillars

Bisync Hub is engineered from first principles to make bidirectional cloud synchronization **inspectable before mutation**, **fail-safe under uncertainty**, and **verifiable after the fact**.

```text
[ Pre-Flight Canaries ] ──► [ Dry-Run Diff ] ──► [ Heuristic Scorer ]
                                                         │
        ┌────────────────────────────────────────────────┴────────┐
        ▼                                                         ▼
[ Auto-Commit / Backup ]                           [ Review Modal / Quarantine ]
        │                                                         │
        └────────────────────────┬────────────────────────────────┘
                                 ▼
                  [ Lock / Jail Guard ] ──► [ Audit Ledger ]
```

## The Six Pillars

1. **[Interactive Sync Review & Deletion Diff Visualizer]({% link usps/01-interactive-sync-review.md %})** — Inspect proposed additions, modifications, and deletions before disk mutations occur.
2. **[Multi-Analyzer Heuristic Conflict Engine]({% link usps/02-heuristic-conflict-engine.md %})** — Score dual-side edits across four signals instead of blindly trusting timestamps.
3. **[Customizable Safety Policies & Circuit Breakers]({% link usps/03-safety-policies.md %})** — Enforce canary checks, deletion limits, and rate-limit backoff.
4. **[Cryptographic Immutable Audit Ledger]({% link usps/04-audit-ledger.md %})** — Record sync actions and decisions in a hash-chained, append-only SQLite ledger.
5. **[Zero-Dependency Single Binary Distribution]({% link usps/05-single-binary.md %})** — Run a single Go binary with an embedded Vue 3 cockpit and zero external runtimes.
6. **[Native Model Context Protocol (MCP) Server]({% link usps/06-mcp-agent-server.md %})** — Connect AI agents (`stdio` / `SSE`) with read-only controls and parameter-fingerprinted confirmation tickets.

