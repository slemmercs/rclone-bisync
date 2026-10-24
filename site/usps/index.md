---
layout: default
title: Core Pillars
nav_order: 2
has_children: true
permalink: /usps
---

# Core Architectural Pillars

Bisync Hub was engineered from first principles to turn unpredictable bidirectional cloud sync into a deterministic, observable, and fully auditable operation.

```text
┌─────────────────────────────────────────────────────────────────────────┐
│                           Bisync Hub Engine                             │
│                                                                         │
│  [ Pre-Flight Canaries ]  ──►  [ Dry-Run Detection ]                    │
│                                        │                                │
│                                        ▼                                │
│                          [ Heuristic Conflict Engine ]                  │
│                                        │                                │
│                                        ▼                                │
│                          [ Interactive Review Modal ]                   │
│                                        │                                │
│                                        ▼                                │
│  [ Lock / Jail Guard ]    ──►  [ Controlled Mutation ]  ──► [ Ledger ]  │
└─────────────────────────────────────────────────────────────────────────┘
```

## The Pillars

1. **[Interactive Sync Review & Deletion Diff Visualizer]({% link usps/01-interactive-sync-review.md %})**
   Inspect proposed additions, modifications, and deletions before disk mutations occur.
2. **[Multi-Analyzer Heuristic Conflict Engine]({% link usps/02-heuristic-conflict-engine.md %})**
   Resolve non-trivial file conflicts with multi-signal confidence scoring and skew tolerance.
3. **[Customizable Safety Policies & Circuit Breakers]({% link usps/03-safety-policies.md %})**
   Enforce strict deletion quotas, canary file invariants, and automatic backoff throttles.
4. **[Cryptographic Immutable Audit Ledger]({% link usps/04-audit-ledger.md %})**
   Maintain verifiable SQLite-backed records of every sync action, decision, and error.
5. **[Zero-Dependency Single Binary Distribution]({% link usps/05-single-binary.md %})**
   Deploy anywhere with a single Go binary and an embedded Alpine.js/Tailwind web dashboard.
