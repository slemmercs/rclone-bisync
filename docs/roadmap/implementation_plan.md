# Implementation Plan: Architecture Principles Harmonization, ADR Foundation & Local Tooling Hygiene

## Proposed Changes

Establish a cohesive architectural governance and ADR framework for Bisync Hub, align all core principles documents, standardize the ADR directory structure with foundational records (ADR-001 through ADR-005), and untrack local developer tool configurations (`.clinerules`, `.clineignore`) from Git tracking.

### Affected Components & Packages

- `.gitignore`: Add `.clinerules` and `.clineignore` to ignore patterns.
- `.clinerules` & `.clineignore`: Untrack from Git index using `git rm --cached` while preserving files locally.
- `docs/architecture/CONTRIBUTOR_WORKFLOW.md`:
  - Complete Section 8 by adding missing product principles (Principle 6: Profiles As Primary Object, Principle 7: Rclone Remains Source Of Truth, Principle 8: Local First) with links to `PRODUCT_PRINCIPLES.md`.
  - Update Section 4 ("Verify Continuously") to explicitly specify the native quality gate `./scripts/check.sh all` and `cmd/check`.
- `docs/architecture/ENGINEERING_PRINCIPLES.md`:
  - Incorporate advisory POSIX locking (`flock`) under Principle 5 ("Atomic State Changes") for concurrency isolation.
  - Formalize single-binary packaging with embedded frontend under Principle 1 / 3.
- `docs/architecture/PRODUCT_PRINCIPLES.md`:
  - Cross-link product principles directly to foundational ADRs and contributor workflow.
- `docs/architecture/ARCHITECTURE_PRINCIPLES.md`:
  - Restructure as the central umbrella architecture index/hub connecting Product Principles, Engineering Principles, Security Principles, and Contributor Workflow.
- `docs/architecture/adr/`:
  - `README.md`: Create ADR registry index table and lifecycle guidance.
  - `ADR-000-Template.md`: Create standardized template matching naming convention.
  - `adr-xxx_title_template.md`: Remove in favor of `ADR-000-Template.md`.
  - `ADR-001-Profiles-As-Primary-User-Object.md`: Foundational ADR establishing profiles as primary configuration and intent abstraction.
  - `ADR-002-Rclone-As-Source-Of-Truth.md`: Foundational ADR establishing Rclone RC API as sole storage connector and engine.
  - `ADR-003-Human-In-The-Loop-Review.md`: Foundational ADR formalizing interactive conflict and deletion review over silent mutation.
  - `ADR-004-Immutable-Audit-Ledger.md`: Foundational ADR establishing append-only cryptographic SQLite audit ledger.
  - `ADR-005-Local-First-Architecture.md`: Foundational ADR establishing local-first single binary, embedded UI, and zero external telemetry/cloud dependencies.
- `CONTRIBUTING.md` & `.github/PULL_REQUEST_TEMPLATE.md`:
  - Update links and checklists to reference the unified architecture suite.

### Technical Strategy & Data Flow

- **Git Tracking Isolation**: Use `git rm --cached` to delete `.clinerules` and `.clineignore` from index and future commits without affecting local working tree copies.
- **Architectural Link Topology**:
  - `ARCHITECTURE_PRINCIPLES.md` serves as Root Hub.
  - Pillars branch into `PRODUCT_PRINCIPLES.md`, `ENGINEERING_PRINCIPLES.md`, `SECURITY_PRINCIPLES.md`, and `CONTRIBUTOR_WORKFLOW.md`.
  - Concrete decisions branch into `docs/architecture/adr/ADR-XXX-*.md` referenced by `docs/architecture/adr/README.md`.
- **Foundational ADR Alignment**: Capture existing, already-built architectural decisions (Milestone 1–4, PR #27, PR #33) as accepted ADRs to provide historical and future rationale for contributors and maintainers.

### Safety & Error Handling

- **No Code/Runtime Regressions**: Changes are strictly documentation, governance, gitignore, and ADR records. No Go runtime code is modified.
- **Canary & Concurrency Guarantees**: Documented principles explicitly enforce canary file verification (`RCLONE_TEST`) and POSIX flock locking across all sync operations.
- **Clean Git Status**: Ensure local `.clinerules` and `.clineignore` remain available on disk for active development while git status remains clean.

## Verification Plan

- `git status`: Confirm `.clinerules` and `.clineignore` are untracked and ignored, but preserved locally.
- Link Integrity: Verify all relative markdown links between `docs/architecture/`, `docs/architecture/adr/`, `CONTRIBUTING.md`, and `.github/`.
- `./scripts/check.sh fmt`: Validate formatting across all markdown and shell scripts.
- `./scripts/check.sh all`: Run full 8-stage pre-flight quality gate to verify zero regressions.









