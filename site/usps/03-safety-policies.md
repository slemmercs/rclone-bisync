---
layout: default
title: "3. Customizable Safety Policies"
parent: Core Pillars
nav_order: 3
---

# Customizable Safety Policies & Circuit Breakers

<span class="badge-online">● Shipped & Active</span>

{: .note }
> **Core Safety Tenets**: Pre-execution canary protection (`RCLONE_TEST`), path confinement filesystem jails, and pre-mutation deletion quotas (`SAFETY_MAX_DELETE_COUNT`, `SAFETY_MAX_DELETE_RATIO`) are fully active in the core engine. Dynamic in-dashboard parameter configuration is scheduled for the upcoming [Configuration & Setup Manager](https://github.com/slemmercs/rclone-bisync/blob/main/docs/roadmap/ROADMAP.md) milestone.

Automated tools must know when to stop. Running bidirectional synchronization when a network share is unmounted, when a cloud provider returns HTTP 429 rate limits, or when a disk is failing will corrupt sync state and trigger cascading deletions.

**Bisync Hub implements a two-stage defensive policy barrier: Pre-Execution and Pre-Mutation.**

---

## Defensive Gates

### 1. Pre-Execution Validation Gates
Before invoking any sync or check command, Bisync Hub checks environmental invariants:

- **Canary File Verification (`RCLONE_TEST`)**:
  - Requires a known marker file to exist on both local and remote mount roots.
  - If a network volume unmounts or cloud sync token expires, the canary file disappears. The engine detects this and aborts instantly, preventing the empty directory from being treated as a mass deletion.
- **Path Confinement (Filesystem Jail)**:
  - All local file manipulations are strictly validated with `filepath.Rel` and prefix checks to ensure zero path traversal escapes outside designated sync roots.
- **Advisory POSIX Locking (`flock`)**:
  - Prevents overlapping sync jobs from corrupting local metadata or triggering race conditions across systemd timers or manual CLI runs.

### 2. Pre-Mutation Circuit Breakers
During sync execution, incoming mutation requests pass through configurable circuit breakers:

```yaml
# bisync.env safety settings
SAFETY_MAX_DELETE_COUNT=25       # Hard limit on total file removals per run
SAFETY_MAX_DELETE_RATIO=0.10     # Maximum 10% of total files may be deleted
SAFETY_ENABLE_CANARY=true        # Canary file mandatory on both remotes
SAFETY_RATE_LIMIT_BACKOFF=true   # Exponential backoff on provider HTTP 429
```

### 3. Automatic Backoff & Quarantine
- **Provider Throttling**: When Google Drive or Microsoft OneDrive signals rate limits (`429 Too Many Requests`), Bisync Hub immediately halts further calls and enforces exponential backoff with jitter.
- **Fail-Safe Abort**: Any violation of a safety policy halts execution cleanly and logs the event to the audit ledger.

{: .warning }
> **Zero Canary Bypass Policy**: Canary check routines are hardcoded to fail-closed. If canary status is ambiguous, sync execution is blocked.
