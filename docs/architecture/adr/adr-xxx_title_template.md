# ADR-XXX: <Decision Title>

**Status:** Proposed | Accepted | Superseded | Deprecated

**Date:** YYYY-MM-DD

---

## Context

What problem are we solving?

Briefly describe:

- the issue
- constraints
- why a decision is needed

---

## Decision

State the decision clearly.

Describe:

- what was chosen
- what contributors should assume going forward

---

## Rationale

Why was this decision made?

Include the most important benefits and trade-offs.

---

## Consequences

### Positive

- ...
- ...

### Negative

- ...
- ...

---

## Related Principles

- Product Principle A
- Product Principle B

---

## Related ADRs

- ADR-XXX
- ADR-YYY

---

## Metadata

**Owner:** @maintainer

**Review Date:** YYYY-MM-DD (optional)

**Tags:** architecture, security, product, workflow

---

# ADR Naming Convention

ADRs are numbered sequentially and must never be renumbered after publication.

Format:

ADR-XXX-Short-Decision-Name.md

Examples:

- ADR-001-Profiles-As-Primary-User-Object.md
- ADR-002-Rclone-As-Source-Of-Truth.md
- ADR-003-Human-In-The-Loop-Review.md
- ADR-004-Immutable-Audit-Ledger.md
- ADR-005-Local-First-Architecture.md

Guidelines:

- Use a three-digit numeric identifier.
- Use Title Case words separated by hyphens.
- Name the architectural decision, not the implementation.
- Prefer product and architecture language over package, class, or file names.
- ADR numbers are permanent and never reused.
- Superseded ADRs remain in the repository for historical reference.

Examples:

✅ ADR-006-Safety-Before-Automation.md

✅ ADR-007-Recommendations-Are-Advisory.md

✅ ADR-008-Review-Before-Destructive-Operations.md

❌ ADR-008-New-Sync-Service.md

❌ ADR-009-refactor-sync-engine.md

❌ ADR-010-added-audit-feature.md

---

# ADR Status Values

### Proposed

The decision is under discussion and has not yet been adopted.

### Accepted

The decision has been approved and represents the current architecture.

### Superseded

The decision has been replaced by a newer ADR.

Example:

ADR-004 superseded by ADR-017.

### Deprecated

The decision is no longer recommended but has not been formally replaced.
