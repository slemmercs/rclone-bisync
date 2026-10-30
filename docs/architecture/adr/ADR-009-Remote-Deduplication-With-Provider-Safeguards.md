# ADR-009: Remote Hash Deduplication with Provider Safeguards and Baseline Resynchronization

**Status:** Accepted

**Date:** 2026-10-30

---

## Context

Cloud storage backends (Google Drive, Microsoft OneDrive, Dropbox, Nextcloud) frequently accumulate duplicate files sharing identical filenames or cryptographic hashes due to overlapping multi-device uploads or aborted sync tasks.

These duplicate objects cause Rclone bisync to abort with fatal errors (`duplicate object found`). However, running unmanaged CLI deduplication introduces severe operational hazards:
1. **Provider Constraints**: Microsoft OneDrive enforces strict unique-names-per-folder constraints, causing single-directory dedupe passes to fail. Additionally, OneDrive includes locked two-factor authentication paths (`Personal Vault/**`) that cause recursive rclone operations to fail with access errors.
2. **Concurrency Hazards**: Running deduplication while background daemons or scheduled cron workers are actively transferring files leads to race conditions and corrupted remote state.
3. **Bisync Baseline Invalidation (`TOO_MANY_DELETES`)**: Removing duplicate files directly on a cloud remote modifies remote object counts and IDs without updating Bisync's local baseline listing files. On the very next scheduled sync run, Bisync detects unexpected remote deletions, assumes data destruction has occurred, and trips the `TOO_MANY_DELETES` safety breaker.

---

## Decision

Bisync Hub implements an integrated **Remote Deduplication Engine with Provider Safeguards and Baseline Resynchronization** (`internal/core/client/fs_ops.go`):

### 1. Automated Provider Safeguards
When initiating deduplication against a cloud backend:
- **OneDrive Personal Vault Exclusion**: If the target remote is OneDrive (`onedrive:` or provider type `onedrive`), the engine automatically injects `--exclude "Personal Vault/**"`, preventing traversal failures on locked 2FA vaults.
- **Provider-Aware Mode Selection**: The dashboard UI and backend automatically default to `--by-hash` mode for single-name backends like OneDrive, guiding users to cross-folder hash deduplication.

### 2. Process Concurrency Locks
- Before executing deduplication, Bisync Hub acquires advisory process locks (`flock`) on all profiles targeting the specified remote.
- Background sync jobs attempting to run against the remote during deduplication defer gracefully, preventing race conditions.

### 3. Automated Baseline Listing Rebuild
- Immediately following deduplication, Bisync Hub refreshes internal baseline listing files for all affected profiles.
- Synchronizing baseline state with the cleaned remote guarantees that subsequent scheduled sync runs will not misinterpret deduplication cleanups as unauthorized data deletions, completely preventing false-positive `TOO_MANY_DELETES` halts.

### 4. Decoupled Execution & Audit Attestation
- Raw terminal output streams directly to `bisync.log` tagged with correlation identifier `run_id` (`dedupe-<timestamp>`).
- Only structured summary metrics (`remote`, `path`, `mode`, `by_hash`, `collisions`, `deleted`, `run_id`) are written to the cryptographic audit ledger (`audit_ledger.db`, per ADR-004).
- The web UI entity drawer renders clean metric summary cards with deep links to correlated daemon logs.

---

## Rationale

1. **Automated Safety Over Manual Scripting**: Enforcing provider-specific flags in the backend prevents users from encountering obscure cloud API failures.
2. **Cache Coherency**: Automatic baseline resynchronization preserves bisync state integrity without requiring operators to manually execute error-prone CLI resync sequences.
3. **Audit Transparency**: Recording structured results in the immutable ledger preserves accountability for remote deletions while keeping the database bounded.

---

## Consequences

### Positive
- Transparent, reliable duplicate consolidation across major cloud storage providers.
- Total elimination of OneDrive `Personal Vault` API errors.
- Prevention of post-deduplication `TOO_MANY_DELETES` bisync lockouts.
- Race-free execution guaranteed by process locks.

### Negative / Trade-offs
- Remote deduplication on large multi-terabyte cloud remotes can take substantial time, holding profile locks until completion.
- Cross-folder hash deduplication requires calculating and comparing remote checksums, which can be I/O intensive for providers that do not provide native hashes.

---

## Related Principles

- [Product Principle 1: Safety Before Automation](../PRODUCT_PRINCIPLES.md#1-safety-before-automation)
- [Product Principle 7: Rclone Remains The Source Of Truth](../PRODUCT_PRINCIPLES.md#7-rclone-remains-the-source-of-truth)
- [Security Principle 4: Canary Protection](../SECURITY_PRINCIPLES.md#4-canary-protection)
- [Security Principle 5: Safe Failure Modes](../SECURITY_PRINCIPLES.md#5-safe-failure-modes)
- [Engineering Principle 5: Atomic State Changes & Concurrency Control](../ENGINEERING_PRINCIPLES.md#5-atomic-state-changes--concurrency-control)

---

## Related ADRs

- [ADR-002: Rclone Remote Control API As Source Of Truth](ADR-002-Rclone-As-Source-Of-Truth.md)
- [ADR-004: Cryptographic Append-Only SQLite Audit Ledger](ADR-004-Immutable-Audit-Ledger.md)
- [ADR-006: Multi-Layered Safety Circuit Breakers & Path Confinement](ADR-006-Multi-Layered-Safety-Circuit-Breakers.md)

---

## Metadata

**Owner:** @maintainer

**Tags:** storage, deduplication, onedrive, safeguards, baseline, locks
