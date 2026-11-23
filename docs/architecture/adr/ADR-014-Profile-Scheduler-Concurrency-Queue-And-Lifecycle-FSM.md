# ADR-014: Profile Scheduler Concurrency Queue & Lifecycle Finite State Machine

**Status:** Accepted

**Date:** 2026-11-23

---

## Context

Bisync Hub enables multi-profile bidirectional synchronization across local directories and cloud storage remotes. Each profile can define its own synchronization schedule (e.g., 15m, 30m, 1h, daily) or run manually on demand.

Uncontrolled concurrent profile execution presents critical engineering hazards:
1. **Engine Option Contention**: In Rclone Remote Control (RC) architecture, certain tuning options (e.g., `--transfers`, `--checkers`, `--bwlimit`, `--drive-chunk-size`) are process-global rather than per-request. Running concurrent profile syncs with divergent flags causes cross-tenant configuration clobbering.
2. **Resource Exhaustion & Cloud Rate Limiting**: Spawning unthrottled concurrent bisync processes risks exhausting host memory, CPU, network bandwidth, and cloud provider API rate limits.
3. **Race Conditions & Shared Path Overlap**: Overlapping executions on the same profile or intersecting paths risk data corruption, lock file conflicts, and bisync cache invalidation.
4. **Indeterminate Profile States**: Without a formal lifecycle Finite State Machine (FSM), background jobs and UI clients lack clear visibility into whether a profile is idle, queued, synchronizing, awaiting review, or halted due to authentication failure.

A resilient concurrency scheduler and lifecycle state machine are necessary to govern profile execution safely.

---

## Decision

Bisync Hub implements a **Cohort-Aware Scheduler and Lifecycle Finite State Machine** (`internal/profiles/scheduler.go`, `dispatcher.go`, `worker.go`):

### 1. Centralized FIFO Queue & Bounded Concurrency
- Replaces uncoordinated background timers with a unified execution queue (`queue []*QueueItem`).
- Imposes an explicit concurrency limit (`maxConcurrency`, default: 4 parallel workers).
- Guarantees per-profile single-flight execution: at most one active sync job per profile ID can run or queue at any time (`activeJobs map[string]*QueueItem`). Duplicate trigger requests while a profile is running or queued return `ErrSyncInProgress`.

### 2. Engine Signature Cohort Batching & SLA Starvation Avoidance
- **Engine Signature (`EngineSignature`)**: Computed dynamically from profile custom flags and global fleet settings (`ResolveEngineSignature`). Profiles sharing the same process-global options share an engine signature.
- **Opportunistic Batching**: The dispatcher (`runDispatcher`) prioritizes queued jobs that match the currently active signature (`activeSignature`), minimizing disruptive `rc options/set` reconfigurations on the running Rclone engine.
- **Starvation-Free Cohort Switching**: If the oldest item in the queue requires a different engine signature and exceeds the maximum wait duration (`maxQueueWait`, default: 60s), the dispatcher halts new admissions, permits active jobs to drain, and performs a synchronized engine option switch (`switchSignatureLocked`).
- **Baseline Restoration**: When the queue empties and all active jobs finish, the scheduler automatically restores baseline global engine options (`restoreBaselineLocked`).

### 3. POSIX Advisory File Isolation (`flock`)
- Every profile execution acquires an advisory POSIX file lock (`flock`) on disk prior to executing bisync routines.
- Prevents collisions between Bisync Hub background schedules, manual CLI triggers, and external automation scripts targeting the same profile root.

### 4. Profile Lifecycle Finite State Machine (FSM)
Each profile advances through well-defined lifecycle states (`internal/profiles/model.go`):
- `healthy`: Idle profile whose last sync succeeded without errors.
- `running`: Active execution running within a bounded worker thread.
- `sync_failed`: Execution terminated with non-zero exit code or bisync abort.
- `authentication_required`: Intercepted OAuth 401 or token revocation requiring user reconnect.
- `paused`: Background schedule suspended by operator; queued jobs cancelled.
- `warning`: Completed with non-fatal warnings or soft-abort triggers awaiting review.

### 5. Operator Control & Telemetry Introspection
- **Dispatcher Controls**: Admins can pause admissions (`PauseDispatcher`), resume processing (`ResumeDispatcher`), or cancel specific queued jobs (`CancelQueued`).
- **Real-Time Snapshot DTOs**: Exposes `GetQueueSnapshot()` (`QueueSnapshotDTO`, `QueueJobDTO`) supplying running jobs, queue depth, wait latencies, and active engine signatures to the dashboard and MCP inspection endpoints.

---

## Rationale

1. **Deterministic Execution**: Bounded worker pools and POSIX flock locks eliminate out-of-memory crashes and file descriptor exhaustion.
2. **Process-Global Option Isolation**: Cohort scheduling ensures that specialized profile flags do not bleed into standard profiles.
3. **Predictable Recovery**: Formal lifecycle states ensure UI components and MCP agents can reliably diagnose why a profile stopped and offer targeted recovery actions.

---

## Consequences

### Positive
- Prevents resource contention and cloud API throttling across multiple profiles.
- Eliminates configuration race conditions through engine signature cohorting.
- Provides sub-second queue observability and operator intervention capabilities.

### Negative / Trade-offs
- Jobs with rare custom flags may experience queue wait latency during high-load periods until an SLA cohort switch triggers.
- Requires maintenance of engine option mapping when Rclone introduces new global flags.

---

## Compliance & Verification

- `internal/profiles/scheduler_test.go` verifies bounded concurrency, fair queue ordering, cohort switching, starvation avoidance, and baseline option restoration.
- End-to-end scheduler lifecycle is verified through `go test -count=1 ./internal/profiles/...`.
- Static analysis and code quality are validated via `./scripts/check.sh`.