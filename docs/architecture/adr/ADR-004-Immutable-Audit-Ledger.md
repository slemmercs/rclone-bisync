# ADR-004: Cryptographic Append-Only SQLite Audit Ledger

**Status:** Accepted

**Date:** 2026-10-26

---

## Context

Synchronization operations perform high-impact mutations: creating, updating, and deleting files across local disks and remote cloud storage. In corporate, homelab, and personal environments, users must be able to verify what actions were performed, who approved them, and whether historical logs have been tampered with or accidentally truncated.

Plain text log files and standard mutable database tables lack cryptographic proof of immutability and can be modified or deleted without detection.

---

## Decision

Bisync Hub implements a **Cryptographic Append-Only Audit Ledger** (`internal/audit/ledger.go`) backed by local embedded SQLite.

Every significant operational event (profile creation, sync runs, conflict resolutions, deletions, and safety overrides) is recorded as an immutable ledger entry. Each entry contains:
- Sequential auto-incrementing ID.
- ISO-8601 UTC timestamp.
- Action type, actor, profile name, and payload details.
- Previous entry's hash (`prev_hash`).
- SHA-256 cryptographic hash of the current entry combined with `prev_hash` (`curr_hash`).

The ledger provides automated cryptographic verification (`ledger.VerifyIntegrity()`) to detect any inserted, deleted, or altered records.

---

## Rationale

1. **Tamper Evidence**: Hash chaining guarantees that modifying or deleting past audit records breaks the hash chain, making tampering immediately visible.
2. **Zero External Dependencies**: SQLite is embedded directly into the Go binary (`modernc.org/sqlite`), requiring no external database server installations.
3. **Structured Queryability**: SQLite allows fast filtering, pagination, and reporting by profile, action type, or date range while preserving immutability guarantees.

---

## Consequences

### Positive
- Verifiable, audit-ready operational history.
- Immediate detection of corrupted or tampered historical records.
- Zero external runtime infrastructure required.

### Negative / Trade-offs
- Local disk storage for SQLite database files (`*.db`, `*.sqlite`).
- SQLite database migrations must preserve the hash chain sequence.

---

## Related Principles

- [Product Principle 5: Audit Significant Decisions](../PRODUCT_PRINCIPLES.md#5-audit-significant-decisions)
- [Security Principle 8: Audit Integrity](../SECURITY_PRINCIPLES.md#8-audit-integrity)
- [Engineering Principle 5: Atomic State Changes & Concurrency Control](../ENGINEERING_PRINCIPLES.md#5-atomic-state-changes--concurrency-control)

---

## Related ADRs

- [ADR-001: Profiles As Primary User Object](ADR-001-Profiles-As-Primary-User-Object.md)
- [ADR-003: Human-In-The-Loop Review Before Mutation](ADR-003-Human-In-The-Loop-Review.md)
- [ADR-005: Local-First Single Binary Deployment Model](ADR-005-Local-First-Architecture.md)

---

## Metadata

**Owner:** @maintainer

**Tags:** security, audit, sqlite, cryptography
