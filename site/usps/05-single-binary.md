---
layout: default
title: "5. Zero-Dependency Single Binary"
parent: Core Pillars
nav_order: 5
---

# Zero-Dependency Single Binary Architecture

Many sync tools and dashboards require Node.js, Python environments, message brokers, or heavy container stacks just to coordinate periodic jobs. **Bisync Hub ships as a single statically compiled Go binary with zero external runtime dependencies.**

---

## Why Single-Binary Matters

* **Embedded Management Cockpit (`go:embed`):** The Vue 3, TypeScript, PrimeVue, and Tailwind dashboard is compiled straight into `bisync-hub`. There is no `node_modules` tree, no external CDN dependency for UI assets, fast startup, and a sub-30MB memory footprint.
* **OS-Native Locking & Atomic Writes:** Concurrency uses POSIX `flock` rather than external coordination services, while state and telemetry updates rely on temporary-file atomic renames (`os.Rename`) to prevent half-written files during power loss or restarts.
* **Easy Cross-Platform Deployment:** Pre-built binaries target `linux/amd64`, `linux/arm64` (Raspberry Pi, ARM servers), and macOS (`darwin/amd64`, `darwin/arm64`), making upgrades or rollbacks as simple as replacing a single executable.

<div class="screenshot-frame">
  <img src="{{ '/assets/images/screenshots/bisync-hub_file-explorer.png' | relative_url }}" alt="Bisync Hub Dual-Pane File Explorer" />
  <div class="caption">
    <span class="title">Dual-Pane Explorer</span>
    <span>Browse local and remote directories directly from the embedded cockpit</span>
  </div>
</div>

<div class="screenshot-grid">
  <div class="screenshot-frame">
    <img src="{{ '/assets/images/screenshots/bisync-hub_bisync-log-panel.png' | relative_url }}" alt="Bisync Execution Log Panel" />
    <div class="caption">
      <span class="title">Execution Log Panel</span>
      <span>Real-time sync output stream</span>
    </div>
  </div>
  <div class="screenshot-frame">
    <img src="{{ '/assets/images/screenshots/bisync-hub_correlated-daemon-log.png' | relative_url }}" alt="Correlated Daemon Log Modal" />
    <div class="caption">
      <span class="title">Correlated Daemon Log</span>
      <span>Service and journal diagnostics</span>
    </div>
  </div>
</div>

> Explore more views in the [UI Gallery]({% link gallery.md %}) or follow the [Quickstart Guide]({% link quickstart.md %}) to deploy.

