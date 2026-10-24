---
layout: default
title: Home
nav_order: 1
description: "Bisync Hub: Production-grade bidirectional synchronization, audit ledger, and safety orchestrator for Rclone."
permalink: /
---

<div class="hero-box">
  <div class="hero-badges">
    <span class="badge-online">● Google Drive & OneDrive Verified</span>
    <span class="badge-indigo">Go 1.24+</span>
    <span class="badge-indigo">Single Binary</span>
  </div>
  <h1>Bisync Hub</h1>
  <p class="hero-tagline">Review. Decide. Sync. The safety layer missing from automated cloud synchronization.</p>
  
  ```bash
  curl -sSL https://raw.githubusercontent.com/slemmercs/rclone-bisync/main/deploy/install.sh | bash
  ```
</div>

## The Bidirectional Dilemma

Standard bidirectional synchronization tools face a fundamental challenge: **How do you reconcile silent file deletions and conflicting concurrent edits without risking catastrophic data loss?**

Raw `rclone bisync` provides raw power across 40+ storage providers, but in production, silent automated runs can propagate accidental directory purges, overwrite newer files due to cloud timestamp skews, or deadlock when token expirations occur.

**Bisync Hub** surrounds `rclone bisync` with an unyielding safety envelope:

<div class="feature-grid">
  <div class="feature-card">
    <span class="badge-delete">Safety Layer</span>
    <h3>Interactive Sync Review</h3>
    <p>Diff previews and dry-run deletion guards block bulk modifications until explicitly approved by an operator or policy.</p>
  </div>
  <div class="feature-card">
    <span class="badge-conflict">Heuristic Engine</span>
    <h3>Confidence-Scored Conflicts</h3>
    <p>Multi-analyzer resolution evaluating size divergence, binary signatures, cloud clock skew, and edit velocity.</p>
  </div>
  <div class="feature-card">
    <span class="badge-indigo">Policy Enforcement</span>
    <h3>Configurable Safety Thresholds</h3>
    <p>Declarative pre-execution and pre-mutation rules with strict delete percentage bounds and canary canary protection.</p>
  </div>
  <div class="feature-card">
    <span class="badge-online">Immutable Proof</span>
    <h3>SQLite Cryptographic Ledger</h3>
    <p>Tamper-evident audit trail with SHA-256 hash chains, trigger-enforced append-only tables, and POSIX advisory locking.</p>
  </div>
</div>

---

## The 5 Core Architectural Pillars

Explore the deep technical architecture powering Bisync Hub:

1. **[Interactive Sync Review & Deletion Diff Visualizer]({% link usps/01-interactive-sync-review.md %})**: Full visibility into pending cloud mutations before any files are altered or removed.
2. **[Multi-Analyzer Heuristic Conflict Engine]({% link usps/02-heuristic-conflict-engine.md %})**: Multi-signal confidence calculation eliminating blind last-modified-wins data loss.
3. **[Customizable Safety Policies & Circuit Breakers]({% link usps/03-safety-policies.md %})**: Pre-execution canaries, max-delete ratios, and rate-limit backoff handling.
4. **[Cryptographic Immutable Audit Ledger]({% link usps/04-audit-ledger.md %})**: SQLite triggers and SHA-256 hash chains providing tamper-proof sync accountability.
5. **[Zero-Dependency Single Binary Distribution]({% link usps/05-single-binary.md %})**: Native Go executable with embedded Web UI, zero Python/pip/npm runtime requirements.

---

## Quick Navigation

- **[Feature Comparison Matrix]({% link comparison.md %})**: How Bisync Hub stacks up against cron scripts, GUI clients, and raw CLI.
- **[Quickstart Guide]({% link quickstart.md %})**: 5-minute setup, systemd automation, and environment configuration.
- **[System Architecture & Safety Tenets]({% link architecture.md %})**: Path jail confinement, advisory locking, and state atomicity.
