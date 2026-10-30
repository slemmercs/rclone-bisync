# ADR-004: Cryptographic Append-Only SQLite Audit Ledger

**Status:** Accepted

**Date:** 2026-10-26 (Updated 2026-10-30: Decoupled Log Streaming & Bounded Schema)

---

## Context

Synchronization operations perform high-impact mutations: creating, updating, and deleting files across local disks and remote cloud storage. In corporate, homelab, and personal environments, users must be able to verify what actions were performed, who approved them, and whether historical logs have been tampered with or accidentally truncated.

Plain text log files and standard mutable database tables lack cryptographic proof of immutability and can be modified or deleted without detection.

Furthermore, synchronization engines and maintenance utilities (such as `rclone dedupe` or full dry-runs) generate voluminous, unbounded terminal outputs containing thousands of lines of verbose transport output. Embedding unbounded raw stdout/stderr directly into immutable SQLite rows creates severe operational problems:
1. Rapid, unbounded SQLite database bloat and memory thrashing during JSON payload deserialization.
2. Inefficient querying, slow table scans, and sluggish UI audit drawer rendering.
3. Blurred architectural boundaries between immutable forensic governance records and transient, rotating debug execution streams.

---

## Decision

Bisync Hub implements a **Cryptographic Append-Only Audit Ledger** (`internal/audit/ledger.go`) backed by local embedded SQLite, paired with a **Decoupled Execution Log Streaming Architecture**.

### 1. Cryptographic Ledger Core
Every significant operational event (profile creation, sync runs, conflict resolutions, deletions, deduplications, and safety overrides) is recorded as an immutable ledger entry. Each entry contains:
- Sequential auto-incrementing ID.
- ISO-8601 UTC timestamp.
- Action type, actor, profile name, and structured payload details.
- Target object paths (`subEntry.object`) affected by the action.
- Previous entry's hash (`prev_hash`).
- SHA-256 cryptographic hash of the current entry combined with `prev_hash` (`curr_hash`).

The ledger provides automated cryptographic verification (`ledger.VerifyIntegrity()`) to detect any inserted, deleted, or altered records.

### 2. Decoupled Execution Streaming & Correlation Keys
To guarantee a strictly bounded database footprint:
- **Unbounded Terminal Logs** (raw CLI stdout/stderr) are **never** stored inside SQLite audit ledger payloads.
- Terminal outputs are streamed in real time to rotating filesystem log files (`bisync.log`).
- Every high-impact operation is tagged with a unique correlation identifier (`run_id`, e.g., `dedupe-1730307600` or `sync-1730307600`).
- The SQLite audit record stores only structured decision summaries (e.g., `remote`, `path`, `mode`, `by_hash`, `collisions`, `deleted`, `run_id`).
- Deep-linking in the Web UI (`openDaemonModal({ run_id })`) and CLI queries use `run_id` to correlate immutable ledger entries with streaming execution log offsets on demand.

---

## Rationale

1. **Tamper Evidence**: Hash chaining guarantees that modifying or deleting past audit records breaks the hash chain, making tampering immediately visible.
2. **Zero External Dependencies**: SQLite is embedded directly into the Go binary (`modernc.org/sqlite`), requiring no external database server installations.
3. **Bounded Storage Footprint**: Separating structured decision metrics from unbounded terminal output keeps `audit_ledger.db` compact, fast to index, and responsive even over millions of synchronization events.
4. **Structured Queryability**: SQLite allows fast filtering, pagination, and reporting by profile, action type, or date range while preserving immutability guarantees.
5. **Separation of Concerns**: Audit ledgers govern *who authorized what and what decisions occurred*; rotating file logs record *low-level transport debugging and byte transfers*.

---

## Consequences

### Positive
- Verifiable, audit-ready operational history.
- Immediate detection of corrupted or tampered historical records.
- Zero external runtime infrastructure required.
- High-performance UI rendering and search speed across audit records without JSON deserialization bottlenecks.
- Bounded database file sizes with predictable growth profiles.

### Negative / Trade-offs
- Local disk storage for SQLite database files (`*.db`, `*.sqlite`).
- SQLite database migrations must preserve the hash chain sequence.
- Inspecting raw command outputs requires reading correlated log files (`bisync.log`) via `run_id` rather than querying SQLite directly.

---

## Related Principles

- [Product Principle 3: Explainability Over Black Boxes](../PRODUCT_PRINCIPLES.md#3-explainability-over-black-boxes)
- [Product Principle 5: Audit Significant Decisions](../PRODUCT_PRINCIPLES.md#5-audit-significant-decisions)
- [Security Principle 8: Audit Integrity](../SECURITY_PRINCIPLES.md#8-audit-integrity)
- [Engineering Principle 2: Explicit Models and Contracts](../ENGINEERING_PRINCIPLES.md#2-explicit-models-and-contracts)
- [Engineering Principle 5: Atomic State Changes & Concurrency Control](../ENGINEERING_PRINCIPLES.md#5-atomic-state-changes--concurrency-control)

---

## Related ADRs

- [ADR-001: Profiles As Primary User Object](ADR-001-Profiles-As-Primary-User-Object.md)
- [ADR-003: Human-In-The-Loop Review Before Mutation](ADR-003-Human-In-The-Loop-Review.md)
- [ADR-005: Local-First Single Binary Deployment Model](ADR-005-Local-First-Architecture.md)

---

## Metadata

**Owner:** @maintainer

**Tags:** security, audit, sqlite, cryptography, logging, telemetry

