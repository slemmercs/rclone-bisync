# Product Principles

## Purpose

This document defines the fundamental product principles that guide all design, architecture, and implementation decisions in Bisync Hub.

These principles take precedence over convenience, feature requests, and implementation preferences.

---

# 1. Safety Before Automation

Bisync Hub prioritizes safe outcomes over maximum automation.

When safety and automation conflict, safety wins.

The platform exists to reduce the risk of unintended synchronization outcomes, particularly during conflicts, deletions, authentication issues, and recovery situations.

Implications:

- Destructive actions should require explicit user intent.
- Uncertainty should favor caution.
- Risky operations should be visible before execution.
- Related Decision: [ADR-003: Human-In-The-Loop Review](adr/ADR-003-Human-In-The-Loop-Review.md)

---

# 2. Human Decision Authority

Users remain the ultimate decision makers.

Recommendations may assist users but never replace user judgement.

Bisync Hub is designed to support informed decisions, not silently make them.

Implications:

- Recommendations are advisory.
- Users retain final approval authority.
- Destructive actions should not be automatically approved based solely on confidence scores.

---

# 3. Explainability Over Black Boxes

Every significant recommendation or safety decision should be understandable.

Users should be able to answer:

- What happened?
- Why did this happen?
- Why is this recommendation being made?

Implications:

- Recommendations should include reasoning.
- Confidence indicators must be understandable.
- System behavior should be observable and traceable.

---

# 4. Review Before Mutation

Potentially destructive operations should be reviewed before changes occur.

Prevention is preferred over recovery.

Implications:

- Conflict resolution should occur before synchronization proceeds.
- Dangerous deletions should be reviewed.
- Safety policies should be evaluated before execution.
- Related Decision: [ADR-003: Human-In-The-Loop Review](adr/ADR-003-Human-In-The-Loop-Review.md)

---

# 5. Audit Significant Decisions

Important actions must be traceable.

Users should be able to reconstruct what happened and why.

Implications:

- Approvals should be recorded.
- Policy violations should be recorded.
- Synchronization outcomes should be recorded.
- Audit records should be append-only and cryptographically verified.
- Related Decision: [ADR-004: Immutable Audit Ledger](adr/ADR-004-Immutable-Audit-Ledger.md)

---

# 6. Profiles Are The Primary User Object

Users work with Profiles rather than low-level synchronization primitives.

Profiles represent operational intent.

Implications:

- Features should be organized around Profiles.
- Profile lifecycle management should remain simple and understandable.
- Internal implementation details should not leak into user workflows.
- Related Decision: [ADR-001: Profiles As Primary User Object](adr/ADR-001-Profiles-As-Primary-User-Object.md)

---

# 7. Rclone Remains The Source Of Truth

Bisync Hub augments Rclone rather than replacing it.

Rclone remains responsible for synchronization execution and storage integration.

Implications:

- Avoid duplicating Rclone capabilities.
- Prefer leveraging Rclone interfaces and APIs.
- Bisync Hub should focus on orchestration, visibility, review, and governance.
- Related Decision: [ADR-002: Rclone As Source Of Truth](adr/ADR-002-Rclone-As-Source-Of-Truth.md)

---

# 8. Local First

Bisync Hub should operate effectively without cloud-hosted control planes or external platform dependencies.

Implications:

- Core functionality should remain self-hosted.
- Critical workflows should not require internet connectivity beyond configured storage providers.
- User data remains under user control.
- Related Decision: [ADR-005: Local-First Architecture](adr/ADR-005-Local-First-Architecture.md)

---

# Product Mission

**Review. Decide. Sync.**

---

## Related Documentation

- [Architecture Principles](ARCHITECTURE_PRINCIPLES.md)
- [Engineering Principles](ENGINEERING_PRINCIPLES.md)
- [Security Principles](SECURITY_PRINCIPLES.md)
- [Contributor Workflow](CONTRIBUTOR_WORKFLOW.md)
- [Architecture Decision Records (ADRs)](adr/README.md)

