# Task Plan: Documentation, Roadmap & Showcase Site Synchronization

## Goal
Synchronize `README.md`, `docs/roadmap/ROADMAP.md`, `docs/roadmap/backlog.md`, and all `site/` showcase documentation with delivered capabilities (Audit Ledger, E2E tests, Go `cmd/check` quality gate, sub-400 LOC service decoupling) and re-sequence near-term priorities.

## Tasks

- [x] Task 1: Reconcile Root `README.md`
  - **Scope**: `README.md`
  - **Verification**: Verify markdown layout, command accuracy, and roadmap alignment

- [x] Task 2: Reconcile `docs/roadmap/ROADMAP.md` and `docs/roadmap/backlog.md`
  - **Scope**: `docs/roadmap/ROADMAP.md`, `docs/roadmap/backlog.md`
  - **Verification**: Cross-reference PR history (#27–#32) and verify consistency

- [x] Task 3: Synchronize Showcase Site (`site/roadmap.md`, `site/architecture.md`, `site/index.md`, `site/quickstart.md`, `site/usps/03-safety-policies.md`)
  - **Scope**: `site/roadmap.md`, `site/architecture.md`, `site/index.md`, `site/quickstart.md`, `site/usps/03-safety-policies.md`
  - **Verification**: Inspect links, front matter, badges, and commands

- [x] Task 4: Run Quality Gate & Pre-Flight Checks
  - **Scope**: Entire repository
  - **Verification**: `./scripts/check.sh fmt` && `./scripts/check.sh all`

## Status Updates

- 2026-10-25: Initialized plan from implementation_plan.md on branch `docs/sync-roadmap-site-and-readme`.
- 2026-10-25: Completed Task 1: Reconciled root `README.md` (Current Capabilities, Planned Capabilities, Development quality gate stages, Roadmap Priorities).
- 2026-10-25: Completed Task 2: Reconciled `docs/roadmap/ROADMAP.md` and `docs/roadmap/backlog.md` with delivered PRs #27–#32 and re-sequenced active priorities (#1 Multi-Profile Workspace Manager, #2 Configuration & Setup Manager, #3 In-App Guidance).
- 2026-10-25: Completed Task 3: Synchronized showcase site (`site/roadmap.md`, `site/architecture.md`, `site/index.md`, `site/quickstart.md`, `site/usps/03-safety-policies.md`, `site/usps/index.md`) with 88.3% coverage badge, `betterleaks`, and valid CLI flags (`--env-file`).
- 2026-10-25: Completed Task 4: Ran full quality gate (`./scripts/check.sh fmt` and `./scripts/check.sh all`) - all 6 stages passed.










