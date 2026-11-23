# Task Plan: Update Existing ADRs and Author ADR-012, ADR-013, ADR-014

## Goal
Synchronize the Architecture Decision Record (ADR) registry with the production codebase by fixing minor drift in ADR-001, ADR-005, and ADR-008, authoring new records for MCP AI agent tooling (ADR-012), headless OAuth2 orchestration (ADR-013), and scheduler concurrency FSM (ADR-014), and registering them in `docs/architecture/adr/README.md`.

## Tasks

- [x] Task 1: Update existing ADRs (ADR-001, ADR-005, ADR-008) to eliminate architectural drift
  - **Scope**: `docs/architecture/adr/ADR-001-Profiles-As-Primary-User-Object.md`, `docs/architecture/adr/ADR-005-Local-First-Architecture.md`, `docs/architecture/adr/ADR-008-WAL-Resilient-Interactive-Conflict-Review.md`
  - **Verification**: `git diff docs/architecture/adr/`

- [x] Task 2: Author ADR-012 (Model Context Protocol AI Agent Governance)
  - **Scope**: `docs/architecture/adr/ADR-012-MCP-Agent-Tooling-And-Blast-Radius-Governance.md`
  - **Verification**: Verify markdown structure, headings, links, and template consistency.

- [x] Task 3: Author ADR-013 (Headless Remote OAuth2 Flow & Cloud Credential Orchestration)
  - **Scope**: `docs/architecture/adr/ADR-013-Headless-Remote-OAuth2-Flow-And-Cloud-Credential-Orchestration.md`
  - **Verification**: Verify markdown structure, headings, links, and template consistency.

- [ ] Task 4: Author ADR-014 (Profile Scheduler Concurrency Queue & Lifecycle FSM) and update `docs/architecture/adr/README.md`
  - **Scope**: `docs/architecture/adr/ADR-014-Profile-Scheduler-Concurrency-Queue-And-Lifecycle-FSM.md`, `docs/architecture/adr/README.md`
  - **Verification**: Run `./scripts/check.sh all` to ensure all quality gates pass.

## Status Updates

- 2026-11-23: Completed Task 3. Authored ADR-013 covering Headless Remote OAuth2 Flow & Cloud Credential Orchestration (`internal/remoteauth/`), status diagnostics, reconnect orchestration, zero-downtime credential migration, and audit logging.
- 2026-11-23: Completed Task 2. Authored ADR-012 covering Model Context Protocol (MCP) AI Agent Governance & Blast-Radius Circuit Breakers with dual transport (stdio/SSE), tiered tooling, confirmation tokens, jail containment, and secret redaction.
- 2026-11-23: Completed Task 1. Updated ADR-001 (`internal/profiles/`), ADR-005 (Vue 3/Vite embedded assets per ADR-011), and ADR-008 (review_sessions, .review_backups, server-side remote copy, exclude rule, remote TTL pruning).
- 2026-11-23: Initialized task plan for ADR documentation synchronization.
