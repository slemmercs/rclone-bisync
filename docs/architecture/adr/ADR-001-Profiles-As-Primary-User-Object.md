# ADR-001: Profiles As Primary User Object

**Status:** Accepted

**Date:** 2026-10-26

---

## Context

Bidirectional synchronization requires configuring dozens of interdependent parameters: two storage endpoints (local paths or remote backends), filter rules, bandwidth limits, delete limits, canary validation files, dry-run flags, and scheduled sync intervals.

Exposing these configurations purely as transient CLI flags or loose script files creates cognitive overload, makes multi-target synchronization difficult to orchestrate, and risks error-prone manual executions.

---

## Decision

Bisync Hub establishes the **Profile** (`internal/models.Profile`) as the primary domain entity representing a complete synchronization contract and operational intent.

All user-facing workflows—including web dashboard navigation, CLI operations, background scheduling, safety checks, review queues, and audit events—are organized and scoped by Profile. Profiles are managed and persisted by a dedicated service layer (`internal/services/profile_manager.go`) using atomic writes.

---

## Rationale

1. **Clear Mental Model**: Users conceptualize synchronization around distinct jobs (e.g., *"Homelab Backup"*, *"Work Documents"*, *"Photo Archive"*), rather than abstract endpoints or command-line syntax.
2. **Safety Isolation**: Scoping operations to profiles ensures safety policies (such as canary file validation and deletion limits) are applied deterministically per workload.
3. **Explicit Contracts**: Profiles provide a strongly typed Go schema (`Profile`, `Schedule`, `SafetyPolicy`) with field validation, eliminating unstructured configuration sprawl.

---

## Consequences

### Positive
- Unified configuration model across CLI, Web UI, and background daemons.
- Easy export, import, and backup of user synchronization intent.
- Deterministic testability with mockable profile stores.

### Negative / Trade-offs
- Profile schema changes require backward-compatible migrations (`v1` to `v2`) to prevent corrupting existing user configurations.

---

## Related Principles

- [Product Principle 6: Profiles Are The Primary User Object](../PRODUCT_PRINCIPLES.md#6-profiles-are-the-primary-user-object)
- [Engineering Principle 2: Explicit Models and Contracts](../ENGINEERING_PRINCIPLES.md#2-explicit-models-and-contracts)

---

## Related ADRs

- [ADR-002: Rclone Remote Control API As Source Of Truth](ADR-002-Rclone-As-Source-Of-Truth.md)
- [ADR-005: Local-First Single Binary Deployment Model](ADR-005-Local-First-Architecture.md)

---

## Metadata

**Owner:** @maintainer

**Tags:** product, models, architecture
