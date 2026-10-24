---
layout: default
title: "5. Zero-Dependency Single Binary"
parent: Core Pillars
nav_order: 5
---

# Zero-Dependency Single Binary Architecture

Many modern orchestrators require Node.js runtimes, Python virtual environments, Redis message brokers, or heavy Docker containers just to expose a dashboard or coordinate periodic sync tasks.

**Bisync Hub is distributed as a single statically compiled Go binary with zero external runtime dependencies.**

---

## Architectural Principles

```text
┌──────────────────────────────────────────────────────────────┐
│                    bisync-hub executable                     │
│                                                              │
│  [ Go 1.24 Runtime ]                                         │
│  ├── Sync Orchestrator & Rclone RC Client                    │
│  ├── Multi-Analyzer Heuristic Engine                         │
│  ├── SQLite Engine & POSIX Flock Locks                       │
│  └── Embedded Web Assets (go:embed)                          │
│      ├── Tailwind CSS & Alpine.js Dashboard                  │
│      └── Static HTML & WebSocket Handlers                    │
└──────────────────────────────────────────────────────────────┘
```

### 1. `go:embed` Frontend Architecture
The complete interactive management dashboard—including Alpine.js components, Tailwind CSS styling, Chart.js telemetry graphs, and WebSocket listeners—is embedded directly into the binary at compile time.
- No `node_modules` directory in production.
- No external npm CDN requests; runs 100% offline in air-gapped data centers.
- Instant startup (< 15ms) with sub-30MB memory footprint.
<div class="screenshot-frame">
  <img src="{{ '/assets/images/screenshots/bisync-hub_file-explorer.png' | relative_url }}" alt="Bisync Hub Dual-Pane File Explorer" />
  <div class="caption">
    <span class="title">Dual-Pane Explorer</span>
    <span>Zero-dependency browser for local files and remote cloud storage</span>
  </div>
</div>

<div class="screenshot-grid">
  <div class="screenshot-frame">
    <img src="{{ '/assets/images/screenshots/bisync-hub_bisync-log-panel.png' | relative_url }}" alt="Bisync Execution Log Panel" />
    <div class="caption">
      <span class="title">Execution Log Panel</span>
      <span>Color-coded live sync output</span>
    </div>
  </div>
  <div class="screenshot-frame">
    <img src="{{ '/assets/images/screenshots/bisync-hub_correlated-daemon-log.png' | relative_url }}" alt="Correlated Daemon Log Modal" />
    <div class="caption">
      <span class="title">Correlated Daemon Log</span>
      <span>Systemd & service diagnostics</span>
    </div>
  </div>
</div>
{: .note }
> For a full visual tour of all dashboard modules and modals, visit the [UI Gallery]({% link gallery.md %}).



### 2. POSIX Flocking & Atomic File Writes
Process concurrency and state isolation rely entirely on OS-native POSIX advisory locks (`syscall.Flock`).
- Zero reliance on external lock daemons (Redis / Zookeeper).
- State persistence and telemetry use atomic rename semantics (`.tmp.<pid>` + `os.Rename`) to prevent half-written files during sudden power loss or server restarts.

### 3. Multi-Platform Cross-Compilation
Compiles cleanly across modern platforms with standard Go tooling:
- `linux/amd64` (Standard servers & workstations)
- `linux/arm64` (Raspberry Pi, AWS Graviton, Apple Silicon Linux VMs)
- `darwin/amd64` & `darwin/arm64` (macOS workstations)

{: .note }
> You can deploy Bisync Hub to an edge Raspberry Pi, a corporate Linux server, or a NAS box with a single `scp` command.
