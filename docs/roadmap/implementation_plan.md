# Implementation Plan: Streamline ROADMAP.md and backlog.md

## Proposed Changes

Streamline `docs/roadmap/ROADMAP.md` and `docs/roadmap/backlog.md` to eliminate massive duplication of historical completed milestones (which are already documented in GitHub Releases `v1.0.0-beta.1` through `beta.27` and git history). Establish crisp role boundaries between strategic roadmap vs. tactical backlog.

### Affected Components & Packages

- `docs/roadmap/ROADMAP.md`: Strategic vision, guiding principles, forward-looking horizons towards v1.0.0-GA and beyond, and non-goals. Direct link to GitHub Releases for historical changelog.
- `docs/roadmap/backlog.md`: Tactical, prioritized engineering backlog with scoped work items, estimates (story points & hours), and references to technical design specifications.
- `site/`: Ensure any site documentation or links referencing roadmap/backlog remain valid.

### Technical Strategy & Content Boundaries

1. **`ROADMAP.md` (Strategic "Why & Where")**:
   - Target audience: Users, contributors, evaluators.
   - Forward-looking horizons:
     - **Horizon 1 (Target v1.0.0-GA)**: Configuration Snapshot & Rollback Engine, Homelab Reverse Proxy Hardening & Standards Observability (OTel/OTLP).
     - **Horizon 2 (Post-GA Operational Visibility)**: High-Density Unified Log Ledger & Deep Path Navigation, Remote Cloud Web Links, Queued File Explorer Direct Upload.
     - **Horizon 3 (Storage & Infrastructure Expansion)**: Remote Mount Manager & Pooled Virtual Storage Engine, Local IPC via Unix Sockets, Native Desktop Packaging (Wails).
   - Clear link to GitHub Releases for past delivered versions.
   - Non-goals section (multi-tenant SaaS, diff merge editor, upstream rclone replacement).

2. **`backlog.md` (Tactical "What & How")**:
   - Target audience: Developers, sprint planners.
   - Groomed, sequenced work queue with estimates (SP and hours), architecture impact, and design doc references.
   - Strict removal of delivered sprints (Phases 1-6, PR #9-76).

### Verification Plan

- Inspect both files for formatting, links, and markdown integrity.
- Run `./scripts/check.sh all` to guarantee that all quality gates (including license and link checks) pass cleanly.

