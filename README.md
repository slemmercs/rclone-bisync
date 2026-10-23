# Bisync Hub

> Review. Decide. Sync.

**Bisync Hub is a safety-first operational platform for Rclone Bisync.**

Most synchronization tools focus on executing sync jobs.

Bisync Hub focuses on helping users understand synchronization outcomes before data changes occur.

Instead of relying on logs, terminal output, and recovery procedures after something goes wrong, Bisync Hub aims to provide visibility, review workflows, and guided decision-making before synchronization is executed.

Bisync Hub provides automated, bidirectional synchronization between your local machine and cloud storage using the **Rclone Remote Control (RC) API**, paired with a single-binary native Go operational dashboard and synchronization orchestrator.

The project focuses on making Rclone Bisync easier to understand, operate, and recover when synchronization becomes complex.

---

# Why Bisync Hub?

Rclone is one of the most powerful file synchronization tools available.

However, when synchronization becomes complex, users are often left answering difficult questions:

- Why is this file in conflict?
- Why is Bisync trying to delete these files?
- What changed since the previous sync?
- Is it safe to continue?
- How do I recover from a failed synchronization?

Bisync Hub exists to answer those questions.

Instead of exposing users to raw logs and force flags, Bisync Hub aims to provide operational visibility, safety controls, and synchronization intelligence.

---

# How Bisync Hub Differs

Bisync Hub is inspired by the Rclone ecosystem and benefits from lessons learned across existing synchronization tools and automation projects.

While many projects help users run, schedule, or automate synchronizations, Bisync Hub focuses on helping users understand synchronization outcomes before changes occur.

## Comparison at a Glance

| Capability | Sync Scripts | Bisync Managers | File Managers | Bisync Hub |
|------------|-------------|----------------|---------------|------------|
| Run Rclone Bisync | ✅ | ✅ | ✅ | ✅ |
| Scheduled Synchronization | ✅ | ✅ | ⚠️ | ✅ |
| Systemd Integration | ✅ | ✅ | ❌ | ✅ |
| Operational Dashboard | ❌ | ⚠️ | ✅ | ✅ |
| Real-Time Telemetry | ❌ | ⚠️ | ⚠️ | ✅ |
| Authentication Health Monitoring | ❌ | ❌ | ❌ | ✅ |
| Synchronization Visibility | ⚠️ | ⚠️ | ❌ | ✅ |
| Conflict Review Experience | ❌ | ❌ | ⚠️ | ✅ |
| Deletion Review Workflow | ❌ | ❌ | ❌ | ✅ |
| Guided Recovery Workflows | ❌ | ❌ | ❌ | ✅ |
| Per-File Sync Decisions | ❌ | ❌ | ❌ | ✅ |
| Rollback-Oriented Design | ❌ | ❌ | ❌ | ✅ |

## Existing Approaches

### Synchronization Automation

Many synchronization projects focus on:

- Running scheduled synchronizations
- Managing system timers
- Applying recommended safety flags
- Capturing logs and errors

These tools make synchronization easier to automate.

### Synchronization Management

Some projects expand automation with:

- Multiple synchronization profiles
- Scheduling
- Background daemons
- Configuration management
- Tray integration

These tools make synchronization easier to operate.

### File Management

Many graphical Rclone frontends focus on:

- Browsing files
- Managing cloud storage
- Copying and moving data
- Exploring remotes

These tools make storage easier to navigate.

## The Bisync Hub Approach

Bisync Hub focuses on a different question:

> How can users understand the impact of a synchronization before data changes occur?

The project is being built around the concept of **synchronization intelligence**, not simply synchronization execution.

Future workflows are intended to help answer questions such as:

- Why is this file in conflict?
- Why is a deletion being proposed?
- What changed since the previous sync?
- What will happen if synchronization proceeds?
- Which files require user attention?
- How can a failed synchronization be recovered safely?

Most synchronization tools help execute a sync.

**Bisync Hub aims to help users understand a sync.**

---

# Not Another File Manager

Many Rclone-based tools focus on:

- File browsing
- File transfers
- Cloud storage management
- Remote administration

Bisync Hub focuses on a different problem:

**Synchronization Intelligence**

The goal is not simply to execute a synchronization.

The goal is to help users understand, review, and control synchronization outcomes.

The integrated file explorer exists to provide synchronization context, not to become a full-featured cloud file manager.

---

# Core Principles

## Visibility Before Action

Users should understand what will happen before synchronization runs.

## Safety Before Automation

Automation should not come at the cost of accidental data loss.

## Recovery Before Rebuild

When synchronization fails, recovery should be guided and understandable.

## Operational Clarity

Synchronization state, remote health, conflicts, and risks should be visible without reading log files.

---

# Current Capabilities

- Real-time synchronization telemetry
- Browser-based operational dashboard
- Embedded single-page application UI
- Native Go single-binary deployment
- Integrated dual-pane file explorer
- OAuth connection health monitoring
- Authentication recovery workflows
- Canary validation and access checks
- Synchronization orchestration and scheduling
- Systemd integration
- Safety-focused execution controls
- Remote health monitoring
- Structured logging and diagnostics

---

# Future Vision

## Review. Decide. Sync.

The long-term vision is to transform synchronization from a command into a reviewable workflow.

```text
Scan
 ↓
Analyze
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
```

Rather than investigating problems after synchronization has completed, users should be able to understand and influence synchronization outcomes beforehand.

---

# Planned Capabilities

## Interactive Sync Review

Review synchronization actions before execution.

Planned capabilities:

- Dry-run analysis
- Pending action review
- Synchronization previews
- Risk visibility

## Conflict Visualization

Provide clear explanations of synchronization conflicts.

Planned capabilities:

- Side-by-side comparison
- Conflict timelines
- Recommended actions
- Per-file decisions

## Deletion Review

Help users understand deletions before they occur.

Planned capabilities:

- Deletion analysis
- Directory impact summaries
- Approval workflows
- Delete-risk visibility

## Guided Recovery

Transform synchronization recovery into a guided experience.

Planned capabilities:

- Recovery recommendations
- Recovery automation
- Failure diagnostics
- Assisted resynchronization

## Rollback Support

Provide rollback-oriented synchronization workflows.

Planned capabilities:

- Configuration snapshots
- Recovery checkpoints
- Synchronization rollback foundations

## Multi-Profile Synchronization

Manage multiple synchronization environments from a single operational hub.

Examples:

- Personal Google Drive
- Business OneDrive
- NAS synchronization
- Archive storage

---

# Core Components

## 1. Rclone Daemon

`rclone-daemon.service`

Background controller exposing internal statistics and control APIs.

## 2. Telemetry UI Backend

`rclone-dashboard.service`

Native Go web server running the embedded Alpine.js and Tailwind UI.

Responsibilities:

- Real-time telemetry
- State aggregation
- Log streaming
- Dashboard APIs

## 3. Synchronization Engine

`bisync-gdrive.service`

Native Go synchronization runner.

Responsibilities:

- Safe execution
- Canary validation
- Lock management
- Synchronization orchestration

## 4. OAuth Connection Manager

Provides:

- Authentication health monitoring
- OAuth reconnect workflows
- Credential validation
- Audit history

## 5. Systemd Timer

`bisync-gdrive.timer`

Automatically executes periodic synchronization cycles.

---

# Installation

The application compiles into a self-contained single-binary executable with no Python runtime dependencies.

## Install

```bash
sudo bash deploy/install.sh
```

---

# CLI Commands

```bash
# Launch dashboard
./bin/rclone-bisync dashboard

# Execute synchronization
./bin/rclone-bisync sync

# Force resync
./bin/rclone-bisync sync --resync

# Dry-run synchronization
./bin/rclone-bisync sync --dry-run

# Provision canary
./bin/rclone-bisync canary
```

---

# Accessing Bisync Hub

Preferred user experience:

```text
bisync-hub dashboard
        ↓
Browser Opens
        ↓
Bisync Hub Dashboard
```

Future releases may provide:

```text
http://bisync.localhost
```

as a stable local endpoint while maintaining automatic port management internally.

---

# Development

Run the pre-flight quality gate before opening a pull request.

```bash
./scripts/check.sh
```

This verifies:

1. Shell script syntax
2. Secret leak protection
3. Go static analysis
4. Automated Go tests

---

# Roadmap Priorities

## Near-Term

- Test coverage >80%
- OAuth reconnect improvements
- Interactive Sync Review
- Multi-profile synchronization
- Configuration management
- Security hardening

## Medium-Term

- Conflict visualization
- Deletion review workflows
- Guided recovery
- Rollback capabilities

## Long-Term

Bisync Hub aims to become the safest and most understandable way to operate Rclone Bisync.

The project is intentionally focused on synchronization safety, operational visibility, and decision support rather than generic file management.

---

# What Bisync Hub Is Not

Bisync Hub is not:

- A replacement for Rclone
- A cloud storage provider
- A hosted SaaS platform
- A generic cloud file manager
- A competitor to storage providers

Bisync Hub exists to improve the operational experience surrounding Rclone Bisync.

---

# Support Development

Bisync Hub is developed and maintained in personal time.

Support helps fund:

- Ongoing development
- AI-assisted engineering tooling
- Testing infrastructure
- Documentation
- Release management
- Long-term maintenance

Support is completely optional.

The project will remain free and open source regardless of sponsorship levels.

## Sponsorship Options

- ❤️ GitHub Sponsors
- ☕ Buy Me a Coffee

---

# Transparency on AI-Assisted Development

AI tools are used to accelerate:

- Implementation
- Testing
- Documentation
- Architecture reviews
- Maintenance tasks

All project decisions, reviews, and releases remain under maintainer control.

For details, see: [AI DISCLOSURE](./AI_DISCLOSURE.md)

---

# Acknowledgements

Bisync Hub integrates with and depends on **Rclone**, the open-source cloud storage synchronization and file transfer tool created by Nick Craig-Wood and maintained by the Rclone community.

Rclone is distributed under the MIT License.

Bisync Hub is an independent project and is not affiliated with or endorsed by the Rclone project.

---

**Review. Decide. Sync.**
