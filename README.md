# Bisync Hub

> The safety-first dashboard for Rclone Bisync.

**Review conflicts, inspect deletions, and make informed synchronization decisions before data changes occur.**

Bisync Hub provides automated, bidirectional synchronization between your local machine and cloud storage using the **Rclone Remote Control (RC) API**, paired with a single-binary native Go operational dashboard and sync orchestrator.

The project focuses on making Rclone Bisync easier to understand, operate, and recover when synchronization becomes complex. Instead of relying solely on logs, listing files, and force flags, Bisync Hub aims to provide visibility, safety controls, and guided recovery workflows for synchronization operations.

## Why Bisync Hub?

There are already excellent tools built on top of Rclone that provide file browsers, cloud explorers, transfer managers, and synchronization capabilities.

Bisync Hub focuses on a different problem.

Traditional Rclone frontends typically help users browse files and execute synchronization operations. Bisync Hub focuses on helping users understand synchronization outcomes before changes are applied.

### What Makes Bisync Hub Different?

| Capability | Traditional Rclone GUIs | Bisync Hub |
|------------|-------------------------|------------|
| Browse files | ✅ | ✅ |
| Transfer files | ✅ | ✅ |
| Execute synchronization | ✅ | ✅ |
| View logs | ✅ | ✅ |
| Review pending deletions before sync | ⚠️ Limited | ✅ |
| Review synchronization conflicts | ⚠️ Limited | ✅ |
| Explain synchronization outcomes | ❌ | ✅ |
| Per-file synchronization decisions | ❌ | ✅ Planned |
| Guided recovery from sync failures | ❌ | ✅ Planned |
| Rollback-oriented workflows | ❌ | ✅ Planned |
| Synchronization-focused operational dashboard | ⚠️ Partial | ✅ |

### Review. Decide. Sync.

Bisync Hub is designed around a simple principle:

**Synchronization should be understandable before it is executed.**

Instead of treating conflicts, mass deletions, and synchronization failures as events that are investigated after the fact, Bisync Hub aims to make them visible before data changes occur.

### What Bisync Hub Is Not

Bisync Hub is not intended to become:

- A general-purpose cloud file manager
- A replacement for Rclone
- A hosted SaaS platform
- A file storage provider

The goal is to improve the operational experience of Rclone Bisync by providing visibility, safety controls, review workflows, and guided recovery capabilities.

### Long-Term Vision

The long-term direction of the project is to evolve synchronization into a review-driven process:

```text
Scan
 ↓
Review Changes
 ↓
Review Conflicts
 ↓
Review Deletions
 ↓
Make Decisions
 ↓
Execute Synchronization
 ↓
Recover if Required


## Core Components

1. **Rclone Daemon (`rclone-daemon.service`)**: Background controller exposing internal statistics and control APIs.
2. **Telemetry UI Backend (`rclone-dashboard.service`)**: High-performance native Go web server (`rclone-bisync dashboard`) running continuously with embedded Alpine.js + Tailwind frontend, aggregating log streams, operational state vectors, and real-time charts.
3. **Synchronization Engine (`bisync-gdrive.service`)**: Native Go execution engine (`rclone-bisync sync`) triggered by systemd timer or manual command to execute bidirectional sync cycles safely with kernel-level flock locking and canary validation.
4. **OAuth Connection Manager**: Centralized remote authentication health monitor, Google Drive shared client ID deprecation detector and migration wizard, desktop-integrated OAuth re-authentication runner, and in-memory audit trail.
5. **Systemd Timer (`bisync-gdrive.timer`)**: Automatically runs the sync iteration every 15 minutes.

## Prerequisites & Installation

The application compiles into a self-contained, single-binary executable with no external Python runtime dependencies required for production operation.

### Installation

1. Move to the source workspace directory.
2. Run the automated deployment script with administrative elevations:

    ```bash
    sudo bash deploy/install.sh
    ```

## CLI Commands

The unified single binary supports modular subcommands:

```bash
# Launch telemetry dashboard (embedded UI)
./bin/rclone-bisync dashboard --host 127.0.0.1 --port 8080

# Execute bidirectional sync cycle
./bin/rclone-bisync sync [--resync] [--force] [--dry-run] [--no-notify] [--origin=cli|timer|dashboard]

# Provision RCLONE_TEST canary file
./bin/rclone-bisync canary [--remote local|gdrive] [--path subfolder]
```

## Access & Monitoring Commands

- **Telemetry UI Instance**: Open your Application Launcher and select **Rclone Custom Dashboard**, or directly navigate to `http://localhost:8080` in your browser.
- **Manual Sync Execution**: Run synchronization manually via the CLI or trigger the systemd service:
    ```bash
    ./bin/rclone-bisync sync
    # Or via systemd:
    systemctl --user start bisync-gdrive.service
    ```
- **Real-Time Service Journal Streams**:

    ```bash
    # View dashboard web service engine logs
    journalctl --user -u rclone-dashboard.service -f
    
    # View synchronization runner history
    journalctl --user -u bisync-gdrive.service -f
    ```

## Development & Testing

Run the automated pre-flight quality and safety gate before committing or opening a PR:

```bash
./scripts/check.sh
```

This verifies:
1. Shell script syntax linting (`bash -n`).
2. Secret isolation (preventing `rc-secrets.env` or plaintext credentials from entering git).
3. Go static analysis (`go vet ./...`).
4. Complete Go automated test suite (`go test -count=1 ./...`).

## Support Development

Bisync Hub is developed and maintained in personal time.

This project makes extensive use of modern AI-assisted engineering workflows for implementation, testing, documentation, architecture reviews, and ongoing maintenance. While the software is free and open source, the tools used to build and evolve it incur recurring costs.

If this project helps you manage your synchronization workflows, improve reliability, or save time, please consider supporting continued development.

Your support helps fund:

- AI-assisted development and code review tooling
- Automated testing and quality improvements
- Documentation and user guides
- New features and long-term maintenance
- Infrastructure required for project releases and distribution

Support is completely optional. The project will remain free and open source regardless of sponsorship levels.

### Sponsorship Options

- ❤️ GitHub Sponsors (recommended for recurring support)
- ☕ Buy Me a Coffee (one-time contributions)

### Transparency on AI-Assisted Development

AI tools are used to accelerate implementation, testing, documentation, and architectural exploration, while all project decisions, reviews, and releases remain under maintainer control.

For details, see [AI_DISCLOSURE.md](AI_DISCLOSURE.md).

## Acknowledgements

This project integrates with and depends on [Rclone](https://rclone.org), the open-source cloud storage synchronization and file transfer tool created by Nick Craig-Wood and maintained by the Rclone community.
Rclone is distributed under the MIT License. 【1-26a01c】【2-46d30b】

This project is independent and is not affiliated with or endorsed by the Rclone project.

---

Special thanks to everyone who reports bugs, submits pull requests, improves documentation, shares feedback, or supports the project financially. All forms of contribution are equally appreciated.
