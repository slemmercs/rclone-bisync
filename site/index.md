---
layout: default
title: Home
nav_order: 1
description: "Bisync Hub: Production-grade bidirectional synchronization, audit ledger, and safety orchestrator for Rclone."
permalink: /
---

<div class="hero-box">
  <img src="{{ '/assets/images/logo.svg' | relative_url }}" alt="Bisync Hub Logo" class="hero-logo" />
  <div class="hero-badges">
    <span class="badge-online">● Test Gate Passed (88.3% Coverage)</span>
    <span class="badge-indigo">Go 1.24+</span>
    <span class="badge-indigo">Single Binary</span>
  </div>
  <h1>Bisync Hub</h1>
  <p class="hero-tagline">Review. Decide. Sync. The safety layer missing from automated cloud synchronization.</p>
  
  ```bash
  curl -sSL https://raw.githubusercontent.com/slemmercs/rclone-bisync/main/deploy/install.sh | bash
  ```
</div>
<div class="screenshot-frame">
  <img src="{{ '/assets/images/screenshots/bisync-hub_engine-telemetry.png' | relative_url }}" alt="Bisync Hub Real-Time Telemetry Cockpit" />
  <div class="caption">
    <span class="title">Production Cockpit</span>
    <span>Real-time transfer telemetry, sync metrics & status indicators</span>
  </div>
</div>


## The Bidirectional Dilemma

Standard bidirectional synchronization tools face a fundamental challenge: **How do you reconcile silent file deletions and conflicting concurrent edits without risking catastrophic data loss?**

Raw `rclone bisync` provides raw power across 40+ storage providers, but in production, silent automated runs can propagate accidental directory purges, overwrite newer files due to cloud timestamp skews, or deadlock when token expirations occur.

**Bisync Hub** surrounds `rclone bisync` with an unyielding safety envelope:

<div class="feature-grid">
  <div class="feature-card">
    <span class="badge-online">● Available Now</span>
    <h3>Interactive Sync Review</h3>
    <p>Diff previews and dry-run deletion guards block bulk modifications until explicitly approved by an operator or policy.</p>
  </div>
  <div class="feature-card">
    <span class="badge-online">● Available Now</span>
    <h3>Confidence-Scored Conflicts</h3>
    <p>Multi-analyzer resolution evaluating size divergence, binary signatures, cloud clock skew, and edit velocity.</p>
  </div>
  <div class="feature-card">
    <span class="badge-online">● Available Now</span>
    <h3>Configurable Safety Policies</h3>
    <p>Pre-execution canary guards (`RCLONE_TEST`), path confinement jails, and deletion quota circuit breakers.</p>
  </div>
  <div class="feature-card">
    <span class="badge-online">● Available Now</span>
    <h3>Cryptographic Audit Ledger</h3>
    <p>Tamper-evident audit trail with SHA-256 hash chains, trigger-enforced append-only tables, and POSIX advisory locking.</p>
  </div>
</div>

---

## Architectural Pillars & Delivery Status

Explore the deep technical architecture powering Bisync Hub:

1. **[Interactive Sync Review & Deletion Diff Visualizer]({% link usps/01-interactive-sync-review.md %})** <span class="badge-online">Available</span>: Full visibility into pending cloud mutations before any files are altered or removed.
2. **[Multi-Analyzer Heuristic Conflict Engine]({% link usps/02-heuristic-conflict-engine.md %})** <span class="badge-online">Available</span>: Multi-signal confidence calculation eliminating blind last-modified-wins data loss.
3. **[Customizable Safety Policies & Circuit Breakers]({% link usps/03-safety-policies.md %})** <span class="badge-online">Available</span>: Pre-execution canaries, max-delete ratios, and rate-limit backoff handling.
4. **[Cryptographic Immutable Audit Ledger]({% link usps/04-audit-ledger.md %})** <span class="badge-online">Available</span>: SQLite triggers and SHA-256 hash chains providing tamper-proof sync accountability.
5. **[Zero-Dependency Single Binary Distribution]({% link usps/05-single-binary.md %})** <span class="badge-online">Available</span>: Native Go executable with embedded Web UI, zero Python/pip/npm runtime requirements.

---

## Embedded Dashboard Tour

Bisync Hub includes a rich, compiled-in Web Cockpit requiring zero external dependencies:

<div class="screenshot-grid">
  <div class="screenshot-frame">
    <img src="{{ '/assets/images/screenshots/bisync-hub_storage-remotes.png' | relative_url }}" alt="Storage Remotes Overview" />
    <div class="caption">
      <span class="title">Storage Remotes</span>
      <span>Remote connectivity & health overview</span>
    </div>
  </div>
  <div class="screenshot-frame">
    <img src="{{ '/assets/images/screenshots/bisync-hub_file-explorer.png' | relative_url }}" alt="Dual-Pane Remote File Explorer" />
    <div class="caption">
      <span class="title">Dual-Pane Explorer</span>
      <span>Side-by-side replica inspection</span>
    </div>
  </div>
</div>

<p style="text-align: center; margin-top: 1rem;">
  <a href="{{ '/gallery.html' | relative_url }}" class="btn btn-primary">Browse Full UI Screenshot Gallery →</a>
</p>

---

## Quick Navigation

- **[UI Gallery]({% link gallery.md %})**: Complete screenshot walkthrough of the telemetry cockpit, file explorer, and OAuth managers.

- **[Project Roadmap]({% link roadmap.md %})**: Detailed breakdown of shipped features, near-term priorities, and planned capabilities.
- **[Feature Comparison Matrix]({% link comparison.md %})**: How Bisync Hub stacks up against cron scripts, GUI clients, and raw CLI.
- **[Quickstart Guide]({% link quickstart.md %})**: 5-minute setup, systemd automation, and environment configuration.
- **[System Architecture & Safety Tenets]({% link architecture.md %})**: Path jail confinement, advisory locking, and state atomicity.

