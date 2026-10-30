# ADR-006: Multi-Layered Safety Circuit Breakers & Path Confinement

**Status:** Accepted

**Date:** 2026-10-30

---

## Context

Bidirectional file synchronization carries catastrophic risks of data loss:
1. **Unmounted Disasters**: When external drives or network mounts fail to mount, their mountpoints appear empty. Standard sync engines can interpret this as user deletion of all files, immediately replicating total data destruction to the intact peer.
2. **Path Traversal**: Erroneous or malicious path inputs in local endpoints or web exploration requests could escape designated directories and mutate critical operating system files (`/etc`, `/root`).
3. **Unbounded Deletion Spikes**: Large restructuring operations, rogue scripts, or ransomware could cause thousands of unintended deletions to propagate across cloud storage.
4. **Concurrent Execution Races**: Overlapping scheduled runs or manual triggers targeting the same profile could interleave rclone operations, corrupting listing caches.

---

## Decision

Bisync Hub mandates a defense-in-depth **Multi-Layered Safety Architecture** consisting of four non-bypassable controls:

### 1. Canary Token Protection (`RCLONE_TEST`)
Before any sync run modifies files, Bisync Hub verifies designated canary token files (`RCLONE_TEST`) on both `Path1` and `Path2`.
- If either endpoint lacks the canary file, execution halts immediately with status `CANARY_MISSING`.
- An unmounted disk or empty mountpoint is never recognized as an empty folder, preventing mass-deletion propagation.
- Canary files are strictly protected by code invariants; no automated deletion or cleanup tool may touch them.

### 2. Automated Deletion Circuit Breakers
Every sync profile enforces a configurable `SafetyPolicy` defining:
- `MaxDeletePercentage`: Maximum percentage of total files permitted to be deleted in a single pass (default: 10%).
- `MaxDeleteCount`: Absolute numeric threshold of deleted files allowed before triggering a safety stop (default: 50).
- When a dry-run or pre-sync listing indicates deletions exceeding either threshold, Bisync Hub trips the safety circuit breaker, sets the profile status to `NEEDS_REVIEW`, and pushes changes to the interactive review queue. Destructive mutations halt until an operator grants explicit approval.

### 3. Local Filesystem Path Confinement (Local Jail)
All local file operations—including file browsing (`internal/services/explorer_service.go`), deduplication, deletion staging, and conflict resolution—must strictly enforce path confinement:
- Paths are canonicalized using `filepath.Clean`.
- Relative containment is validated via `filepath.Rel(root, target)`.
- Any path containing traversal prefixes (`strings.HasPrefix(rel, "..")`) or resolving outside the configured local root directory is rejected immediately with an HTTP 403 access violation error.

### 4. Advisory POSIX Process File Locking
Process execution across profiles, daemon workers, and deduplication tasks enforces race-free concurrency:
- Non-blocking advisory file locking is acquired using `syscall.Flock` (`internal/core/lock/`).
- If an operation cannot acquire the profile lock immediately, it terminates gracefully with `ERR_PROFILE_LOCKED` rather than queueing up and colliding with the running job.

---

## Rationale

1. **Safety Before Automation**: Preserving user data is strictly prioritized over unattended task completion. Halting a scheduled sync is always preferable to deleting files prematurely.
2. **Defense in Depth**: If a canary check passes, deletion thresholds provide an independent safety backstop.
3. **Exploit Prevention**: Strict path confinement ensures directory traversal vulnerabilities cannot be exploited through UI inputs or API payloads.
4. **Deterministic Atomicity**: Advisory file locks prevent race conditions and ensure cache listings remain coherent.

---

## Consequences

### Positive
- Total elimination of unmounted disk mass-deletion disasters.
- Predictable and transparent boundaries for file operations.
- Strong protection against ransomware or runaway script deletion propagation.
- Safe concurrent scheduling without state file corruption.

### Negative / Trade-offs
- Legitimate batch file renames or large reorganizations trip deletion thresholds, requiring user confirmation in the review queue.
- Initial profile setup requires placing a canary file when canary validation is enabled.

---

## Related Principles

- [Product Principle 1: Safety Before Automation](../PRODUCT_PRINCIPLES.md#1-safety-before-automation)
- [Product Principle 4: Review Before Mutation](../PRODUCT_PRINCIPLES.md#4-review-before-mutation)
- [Security Principle 1: Least Privilege](../SECURITY_PRINCIPLES.md#1-least-privilege)
- [Security Principle 3: Path Confinement](../SECURITY_PRINCIPLES.md#3-path-confinement)
- [Security Principle 4: Canary Protection](../SECURITY_PRINCIPLES.md#4-canary-protection)
- [Security Principle 5: Safe Failure Modes](../SECURITY_PRINCIPLES.md#5-safe-failure-modes)
- [Engineering Principle 5: Atomic State Changes & Concurrency Control](../ENGINEERING_PRINCIPLES.md#5-atomic-state-changes--concurrency-control)

---

## Related ADRs

- [ADR-001: Profiles As Primary User Object](ADR-001-Profiles-As-Primary-User-Object.md)
- [ADR-003: Human-In-The-Loop Review Before Mutation](ADR-003-Human-In-The-Loop-Review.md)
- [ADR-005: Local-First Single Binary Deployment Model](ADR-005-Local-First-Architecture.md)

---

## Metadata

**Owner:** @maintainer

**Tags:** safety, circuit-breakers, security, path-confinement, canary, flock
