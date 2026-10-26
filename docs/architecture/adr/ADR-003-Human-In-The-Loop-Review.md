# ADR-003: Human-In-The-Loop Review Before Mutation

**Status:** Accepted

**Date:** 2026-10-26

---

## Context

Bidirectional file synchronization carries inherent risks of data loss:
- File conflicts where both endpoints modified the same file independently.
- Mass deletions where accidental local directory removals are mirrored to cloud backups.
- Timestamp anomalies caused by differing filesystem time resolutions.

Automated sync utilities typically resolve conflicts with blind heuristics (e.g., "newer file wins"), which frequently leads to silent data overwrites.

---

## Decision

Bisync Hub mandates **Human-In-The-Loop Review Before Mutation** for all unresolved conflicts, deletion spikes, and safety threshold triggers.

When a sync run produces conflicts or exceeds safety parameters:
1. The execution is halted or held in a review-required state.
2. A review item is generated and queued (`internal/services/review_service.go`).
3. The Web UI and CLI present detailed file diffs, timestamps, and actionable choices (Keep Path1, Keep Path2, Keep Both, or Skip).
4. Mutations proceed **only after explicit human decision**.

---

## Rationale

1. **Safety Over Speed**: Preventing silent data destruction is far more critical than unattended automation.
2. **User Sovereignty**: Automated heuristics cannot know user intent when two conflicting versions exist.
3. **Auditability**: Every human decision is recorded and attributed in the audit ledger, establishing full operational accountability.

---

## Consequences

### Positive
- Completely eliminates silent file overwrites caused by heuristic guesswork.
- Protects users against catastrophic mass-deletion propagation.
- Increases user trust and understanding of their synchronization state.

### Negative / Trade-offs
- Synchronization jobs may pause until a human reviews pending items.
- Introduces additional state tracking and UI review workflows.

---

## Related Principles

- [Product Principle 1: Safety Before Automation](../PRODUCT_PRINCIPLES.md#1-safety-before-automation)
- [Product Principle 2: Human Decision Authority](../PRODUCT_PRINCIPLES.md#2-human-decision-authority)
- [Product Principle 4: Review Before Mutation](../PRODUCT_PRINCIPLES.md#4-review-before-mutation)
- [Security Principle 5: Safe Failure Modes](../SECURITY_PRINCIPLES.md#5-safe-failure-modes)

---

## Related ADRs

- [ADR-001: Profiles As Primary User Object](ADR-001-Profiles-As-Primary-User-Object.md)
- [ADR-004: Cryptographic Append-Only SQLite Audit Ledger](ADR-004-Immutable-Audit-Ledger.md)

---

## Metadata

**Owner:** @maintainer

**Tags:** product, safety, review, ux
