# Design Specification: System Integration & Architectural Alignment

**Feature Name**: System Integration & Architectural Alignment

**Status**: Approved / Design Baseline

**Target Milestones**: Phase 4 & Phase 5

**Package References**: `internal/remoteauth`, `internal/review`, `internal/engine/heuristics`, `internal/audit`

---

## 1. Executive Summary & Purpose

Bisync Hub provides a unified, safety-first operational platform built on top of Rclone Bisync. To prevent silent data corruption, accidental mass deletions, and unrecoverable sync failures, the platform connects four core operational modules into an integrated lifecycle pipeline:

1. **OAuth Connection Manager** (`internal/remoteauth`): Ensures authentication health and guides credential reconnects.
2. **Interactive Sync Review** (`internal/review`): Intercepts bisync safety aborts and provides a transactional, file-by-file resolution state machine.
3. **Heuristic Conflict Recommendation Engine** (`internal/engine/heuristics`): Analyzes candidate conflicts using pure-Go heuristics to provide confidence-scored recommendations.
4. **Immutable Audit Ledger** (`internal/audit`): Logs executed operational decisions into a cryptographically chained, append-only SQLite database.

This document formalizes the end-to-end operational pipeline, component hand-offs, and mandatory integration directives across all four modules.

---

## 2. End-to-End Pipeline & Data Flow

```text
               ┌─────────────────────────────────────────┐
               │   OAuth Connection Manager (Implemented)│
               │  - Pre-flight auth checks & health bar  │
               └────────────────────┬────────────────────┘
                                    │ (Pre-Flight Auth Gate)
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│               rclone bisync Execution (Background / Timer)              │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │ (Aborts on TOO_MANY_DELETES / Conflicts)
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│             Interactive Sync Review (Currently In Code / Phase 4)      │
│  - Traps abort, creates ReviewSession, locks sync runner               │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │ (Candidate List)
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│          Heuristic Conflict Recommendation Engine (Designed)           │
│  - Parallel worker pool evaluates hashes, clock skew, truncation       │
│  - Enriches candidates with Confidence, Risk, and Suggested Actions    │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │ (User Approval / Batch Auto-Apply)
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│             Targeted RC Resolution & Baseline Reset                    │
│  - Pre-mutation backups -> Rclone RC operations/copyfile               │
│  - Triggers rclone bisync --resync baseline reset                      │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │ (Log Executed Action)
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│                  Immutable Audit Ledger (Designed)                     │
│  - Pure-Go SQLite + Goose embedded migrations                         │
│  - Cryptographic SHA-256 hash chain & DB immutability triggers         │
└────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Component Interaction Matrix

### 3.1 OAuth Connection Manager (`internal/remoteauth`)
* **Role**: Operational authentication gate and recovery orchestrator.
* **Pipeline Triggers**:
  * Blocks scheduled background synchronization if remote state is `AUTHORIZATION_REQUIRED`.
  * Intercepts `403 Access Denied` or token expiration errors occurring during Stage 3 resolution execution, launching an inline OAuth reconnect workflow without destroying the active `ReviewSession`.
  * Emits `ActionOAuthReconnected` and `ActionConfigUpdated` entries directly to the `internal/audit` ledger.

### 3.2 Interactive Sync Review (`internal/review`)
* **Role**: State machine managing transient review sessions (`CREATED`, `REVIEWING`, `APPLYING`, `READY`, `RUNNING_BISYNC`).
* **Pipeline Triggers**:
  * Traps bisync abort states (`TOO_MANY_DELETES`, unhandled conflicts) and constructs the candidate item payload.
  * Supplies candidate lists to `heuristics.Engine.Evaluate()` for annotation.
  * Creates pre-mutation localized backups before invoking atomic Rclone RC primitives (`operations/copyfile`, `operations/deletefile`).
  * Triggers `rclone bisync --resync` after all session decisions are successfully applied.

### 3.3 Heuristic Conflict Recommendation Engine (`internal/engine/heuristics`)
* **Role**: Non-invasive decision intelligence layer.
* **Pipeline Triggers**:
  * Evaluates candidate files concurrently using a bounded worker pool while Rclone is idle between interception and resolution execution.
  * Populates UI badges with confidence scores ($0.00$ to $1.00$), risk tiers (`LOW`, `MEDIUM`, `HIGH`, `CRITICAL`), and primary explanation strings.
  * Drives the `POST /api/review/action-batch` endpoint, allowing users to auto-apply low-risk, high-confidence recommendations while holding critical items for manual review.

### 3.4 Immutable Audit Ledger (`internal/audit`)
* **Role**: Permanent, tamper-evident record keeper.
* **Pipeline Triggers**:
  * Receives event payloads after mutations are completed (`ActionConflictResolved`, `ActionDeletionAccepted`, `ActionDeletionRestored`, `ActionRollbackExecuted`).
  * Uses pure-Go SQLite (`modernc.org/sqlite`) with Goose embedded migrations, SHA-256 hash chaining (`prev_hash` $\rightarrow$ `hash`), and engine-level triggers to enforce append-only immutability.
  * Scrubs sensitive tokens and credentials via `security.RedactSecrets()` before computing cryptographic hashes.

---

## 4. Key Integration Directives

### Directive 1: Separation of Transient Review State and Permanent Ledger
* **Transient Session State** (`ReviewSession`): Managed in atomic temporary JSON files (`state/review-sessions/<session_id>.json`) or an in-memory session manager. Holds active candidate lists, pending user choices, and backup snapshot paths. Transient sessions can be edited, reset, rolled back, or dismissed.
* **Permanent Audit Ledger** (`internal/audit` SQLite DB): Holds only finalized, successfully executed actions. Once written, entries are cryptographically locked and cannot be removed or updated.

### Directive 2: Profile ID Fallback Architecture (Pre-Phase 5 Mapping)
* The `audit_logs` database schema includes a mandatory `profile_id` column.
* Prior to the completion of Phase 5 (Multi-Profile Synchronization Hub), all Phase 4 audit records must populate `profile_id` with `"default"` or the underlying Rclone remote name (e.g., `"gdrive"`).
* When Phase 5 is deployed, the column will seamlessly map to dynamic profile identifiers without requiring database schema alterations or migrations.

### Directive 3: Mid-Resolution Authentication Recovery Protocol
If an individual RC operation (`operations/copyfile`) fails during Stage 3 resolution due to token expiration:
1. Stage 3 execution pauses immediately without rolling back or invalidating previously completed operations.
2. The active `ReviewSession` enters `APPLY_FAILED` state.
3. The dashboard triggers the **OAuth Connection Manager** reconnect modal.
4. Upon successful re-authentication, the user resumes resolution execution to process remaining candidates.

### Directive 4: Pre-Sync Baseline Reset Barrier
* After all file-by-file RC resolutions complete, Rclone's internal listing files (`.lst` files) still contain pre-conflict state markers.
* Bisync Hub must issue an `rclone bisync --resync` command before clearing the active `ReviewSession` and unlocking scheduled background timer cycles.

---

## 5. Security & Verification Guarantees

1. **Zero Secret Persistence**: No access token, refresh token, or client secret is recorded in the audit database, review session journals, or telemetry payloads.
2. **Idempotent Resolution**: All targeted RC resolution routines are idempotent. If interrupted by a system restart, resuming the review session re-verifies target file states before executing mutations.
3. **Auditability**: Running `bisync-hub audit verify` re-evaluates the SHA-256 hash chain from genesis to tail, ensuring no historical records have been modified or deleted.