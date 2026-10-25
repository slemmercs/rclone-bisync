# Implementation Plan: Documentation, Roadmap & Showcase Site Synchronization

## Proposed Changes

Reconcile and synchronize all public-facing and engineering documentation (`README.md`, `docs/roadmap/ROADMAP.md`, `docs/roadmap/backlog.md`, and the `site/` documentation portal) to reflect delivered capabilities (Immutable Audit Ledger, E2E smoke tests, `cmd/check` Go CLI quality gate, and modularized services under 400 LOC) and re-sequence the active strategic near-term priorities.

### Affected Components & Packages

- `README.md`:
  - Move delivered features (Interactive Review, Conflict Visualization, Deletion Review, Cryptographic Audit Ledger) out of "Planned Capabilities" and into "Current Capabilities".
  - Update "Roadmap Priorities" to reflect current sequence (Priority 1: Multi-Profile Workspace Manager, Priority 2: Configuration & Setup Manager, Priority 3: In-App User Guidance & Help Documents).
  - Update "Development" section to document the native Go `cmd/check` quality engine alongside `./scripts/check.sh` and its 6 execution stages.
- `docs/roadmap/ROADMAP.md`:
  - Update Status Summary table: mark *Immutable Audit Ledger* and *Go CLI Quality Engine (`cmd/check`) & E2E Testing* as **Delivered**.
  - Advance Active Strategic Priorities:
    - Priority 1: Multi-Profile Workspace Manager.
    - Priority 2: Configuration & Setup Manager.
    - Priority 3: In-App User Guidance, Info & Help Documents.
    - Priority 4: Rclone RC Framework Hardening & Mount Lifecycle Manager.
- `docs/roadmap/backlog.md`:
  - Move Epic 1 (Immutable Audit Ledger, PR #27) and recent engineering sprints (PRs #28–#32: service modularization, Alpine boolean attribute scanner, E2E tests, Go quality engine) to completed sections.
  - Re-sequence active near-term backlog items: Multi-Profile Workspace Manager (#1), Configuration & Setup Manager (#2), In-App User Guidance (#3).
- `site/roadmap.md`:
  - Update "Shipped Capabilities" to include the Go CLI Quality Engine (`cmd/check`), Chrome E2E testing, and sub-400 LOC service architecture.
  - Re-align "Near-Term Strategic Priorities" to match `docs/roadmap/ROADMAP.md`.
- `site/architecture.md`:
  - Replace outdated `gitleaks` reference with `betterleaks`.
  - Document Go quality gate architecture (`cmd/check`) with parallel check runner and true HTML attribute tokenization.
- `site/index.md`:
  - Update test gate coverage badge from 83.2% to 88.3%.
  - Clarify safety policies pillar status (core invariants active; visual policy editing planned).
- `site/quickstart.md`:
  - Fix CLI command invocations to use actual subcommands (`rclone-bisync dashboard --env-file bisync.env`, `rclone-bisync sync --dry-run`).
  - Correct systemd service `ExecStart` definition.
- `site/usps/03-safety-policies.md`:
  - Update roadmap status notice to reflect that core canary validation and deletion quotas are shipped and active in the engine.

### Technical Strategy & Data Flow

- **Single Source of Truth**: Maintain perfect fidelity across root `README.md`, developer roadmaps (`docs/roadmap/`), and showcase site (`site/`).
- **Command & Flag Fidelity**: Ensure all CLI flags, file paths, and environment variable names match the actual implementation in `main.go`, `internal/config/`, and `cmd/check/`.
- **Accurate Metric Reporting**: Reflect exact statement coverage (88.3%) verified by `go tool cover`.

### Safety & Error Handling

- **No Code Breaking**: Documentation-only changes with zero regression risk to Go or JavaScript logic.
- **Verification Integrity**: Run full `./scripts/check.sh all` to ensure format checks, template checks, LOC counts, linters, and unit tests pass cleanly.

## Verification Plan

- Run `./scripts/check.sh fmt` to verify formatting across shell scripts and Go files.
- Run `./scripts/check.sh all` to run all 6 quality stages (`fmt`, `templates`, `loc`, `lint`, `test`, `build`).
- Inspect updated files to ensure consistent cross-links, correct permalinks, and complete sections.







