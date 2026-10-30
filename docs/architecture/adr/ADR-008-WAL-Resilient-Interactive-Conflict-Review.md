# ADR-008: Write-Ahead Log (WAL) Resilient Interactive Conflict Review

**Status:** Accepted

**Date:** 2026-10-30

---

## Context

When bidirectional synchronization detects file conflicts (concurrent modifications on both `Path1` and `Path2`) or deletion spikes, Bisync Hub halts execution and queues items in an interactive review session (`internal/services/review_service.go`).

Resolving conflicting items requires complex, multi-step operations across local disks and remote cloud endpoints: creating backup copies, transferring files, deleting obsolete candidates, and rebuilding rclone listing caches.

If the daemon process terminates, the host reboots, or network connectivity drops halfway through applying a 50-file resolution batch:
1. The filesystem can be left in an inconsistent, partially modified state.
2. The user interface cannot reliably determine which decisions took effect and which failed.
3. Rclone listing caches fall out of sync with actual file states, forcing an error-prone manual `--resync` recovery.

A stateless or un-journaled execution model is unacceptable for high-consequence file mutations.

---

## Decision

Bisync Hub implements a **Write-Ahead Log (WAL) Resilient Interactive Review Engine** (`internal/services/review_execution.go`):

### 1. Pre-Mutation WAL Transaction Staging
Before initiating any physical file transfer or deletion:
- The execution plan is serialized and committed to an on-disk WAL transaction file (`.bisync_wal/<session_id>.json`).
- Each resolution step records the target path, source/destination endpoints, action type (`keep_path1`, `keep_path2`, `keep_both`, `delete`, `restore`), and status transitions (`STAGED` -> `IN_PROGRESS` -> `COMPLETED` -> `FAILED`).
- Disk writes to the WAL are flushed prior to issuing rclone file operations.

### 2. Pre-Mutation Backup Snapshots
- For any destructive operation where a candidate file will be overwritten or removed, Bisync Hub first creates a local snapshot copy within a quarantined session backup directory (`.bisync_backups/<session_id>/`).
- If an operation fails midway or cannot be verified, the rollback subsystem uses these snapshot files to restore the endpoints to their exact pre-resolution state.

### 3. Crash Recovery & Idempotent Resume
- Upon daemon startup or review session reload, the WAL engine scans for uncommitted or interrupted sessions.
- Operations are designed to be idempotent: re-executing a partially completed transaction checks destination hashes and timestamps, resuming from the last unfulfilled step without corrupting already transferred files.

### 4. Atomic Commit & Cryptographic Ledger Attestation
- When all planned actions in a review batch complete successfully, the WAL state is atomically updated to `COMMITTED` via `.tmp.<pid>` + `os.Rename`.
- A structured event (`CONFLICT_RESOLVED` or `DELETIONS_APPROVED`) is written to the append-only SQLite audit ledger (per ADR-004), cryptographically linking the human review decision with the resulting mutation.
- Ephemeral backup snapshots are safely retired once commit integrity is verified.

---

## Rationale

1. **ACID-Like Reliability**: Treating batch conflict resolutions as transactional units prevents partial execution states and corrupt cache baselines.
2. **Reversible Mutations**: Pre-mutation snapshots eliminate the risk of accidental permanent loss during human decision execution.
3. **Operational Confidence**: Users can safely execute large multi-file resolutions over unreliable network connections without fear of irrecoverable disconnects.

---

## Consequences

### Positive
- Zero corrupted or half-resolved states during power outages, process terminations, or network drops.
- Automatic recovery and seamless session resumption on startup.
- Complete audit ledger attestation linking human decisions directly to verified outcomes.
- Failsafe rollback mechanisms via quarantined pre-mutation snapshots.

### Negative / Trade-offs
- Slight operational I/O overhead to stage WAL transactions and create pre-mutation snapshots before executing large file moves.
- Requires temporary disk capacity to store quarantined backups until resolution completion.

---

## Related Principles

- [Product Principle 1: Safety Before Automation](../PRODUCT_PRINCIPLES.md#1-safety-before-automation)
- [Product Principle 2: Human Decision Authority](../PRODUCT_PRINCIPLES.md#2-human-decision-authority)
- [Product Principle 4: Review Before Mutation](../PRODUCT_PRINCIPLES.md#4-review-before-mutation)
- [Engineering Principle 5: Atomic State Changes & Concurrency Control](../ENGINEERING_PRINCIPLES.md#5-atomic-state-changes--concurrency-control)
- [Security Principle 5: Safe Failure Modes](../SECURITY_PRINCIPLES.md#5-safe-failure-modes)

---

## Related ADRs

- [ADR-001: Profiles As Primary User Object](ADR-001-Profiles-As-Primary-User-Object.md)
- [ADR-003: Human-In-The-Loop Review Before Mutation](ADR-003-Human-In-The-Loop-Review.md)
- [ADR-004: Cryptographic Append-Only SQLite Audit Ledger](ADR-004-Immutable-Audit-Ledger.md)
- [ADR-006: Multi-Layered Safety Circuit Breakers & Path Confinement](ADR-006-Multi-Layered-Safety-Circuit-Breakers.md)

---

## Metadata

**Owner:** @maintainer

**Tags:** review, wal, atomicity, crash-resilience, conflicts, transactions
