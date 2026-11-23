# Implementation Plan: Update Existing ADRs and Register ADR-012, ADR-013, ADR-014

## Proposed Changes

Review and synchronize Bisync Hub's Architecture Decision Records (ADRs) with the production codebase:
1. Address architectural drift in existing ADRs:
   - **ADR-001**: Align profile manager service path with `internal/profiles/` package.
   - **ADR-005**: Eliminate outdated Alpine.js reference in favor of Vue 3 / Vite assets per ADR-011.
   - **ADR-008**: Correct path references (`review_sessions/` and `.review_backups/`), document zero-bandwidth server-side remote backup staging, `--exclude ".review_backups/**"` default exclusion, and remote TTL backup pruning.
2. Formalize missing architectural contracts into new ADRs:
   - **ADR-012**: Model Context Protocol (MCP) AI Agent Governance & Blast-Radius Circuit Breakers (`internal/mcp/`).
   - **ADR-013**: Headless Remote OAuth2 Flow & Cloud Credential Orchestration (`internal/remoteauth/`).
   - **ADR-014**: Profile Scheduler Concurrency Queue & Lifecycle Finite State Machine (`internal/profiles/`).
3. Update `docs/architecture/adr/README.md` registry table.

### Affected Components & Packages

- `docs/architecture/adr/ADR-001-Profiles-As-Primary-User-Object.md`
- `docs/architecture/adr/ADR-005-Local-First-Architecture.md`
- `docs/architecture/adr/ADR-008-WAL-Resilient-Interactive-Conflict-Review.md`
- `docs/architecture/adr/ADR-012-MCP-Agent-Tooling-And-Blast-Radius-Governance.md`
- `docs/architecture/adr/ADR-013-Headless-Remote-OAuth2-Flow-And-Cloud-Credential-Orchestration.md`
- `docs/architecture/adr/ADR-014-Profile-Scheduler-Concurrency-Queue-And-Lifecycle-FSM.md`
- `docs/architecture/adr/README.md`

### Technical Strategy & Content Structure

- **ADR-012**: Context (AI agents performing file ops and sync runs), Decision (Stdio and SSE MCP server in `internal/mcp/`, strict confirmation tokens for high-impact actions, profile jail roots, canary guards), Rationale, Consequences.
- **ADR-013**: Context (Headless server deployments without local web browser), Decision (Dedicated `RemoteAuthService`, loopback token exchange, browserless redirect callbacks, session expiry), Rationale, Consequences.
- **ADR-014**: Context (Unattended background sync scheduling, concurrency race prevention), Decision (Profile FSM states, serialized dispatch queues, advisory flock locking, interval presets), Rationale, Consequences.

## Verification Plan

- Validate all markdown links, cross-references, and formatting.
- Run `./scripts/check.sh all` to verify zero regressions across the codebase and doc validation.
