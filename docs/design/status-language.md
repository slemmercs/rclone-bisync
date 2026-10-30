# Bisync Hub Design System: Status Language Specification

## 1. Principles

In a safety-critical bidirectional cloud synchronization tool where operators manage destructive file mutations, status terminology must be unambiguous, concise, and predictable.

Status indicators communicate two orthogonal pieces of information:
1. **Health State**: Is the system or workload functioning correctly?
2. **Execution State**: Is an operation currently active?

---

## 2. The 7 Approved Status Vocabulary States

Every badge, tile, indicator dot, and alert across the Bisync Hub interface must use **only** one of these seven canonical labels:

| Canonical Label | Semantic Meaning | Color Token | CSS Badge Class | Disallowed Legacy Terms |
|---|---|---|---|---|
| **`Healthy`** | Normal operation, listings aligned, baseline clean | Emerald (`--color-success`) | `.ds-badge-healthy` | `OPERATIONAL`, `ONLINE`, `OK`, `READY`, `PASSING` |
| **`Warning`** | Non-blocking anomalies, quota threshold (>85%), dry-run divergence | Amber (`--color-warning`) | `.ds-badge-warning` | `ATTENTION REQUIRED`, `NEEDS ATTENTION`, `DEGRADED`, `WARN` |
| **`Critical`** | Unrecoverable error, process fault, connection failure, canary missing | Rose (`--color-danger`) | `.ds-badge-critical` | `FAULT`, `OFFLINE`, `ERROR`, `DANGER`, `FAILED` |
| **`Running`** | Active bidirectional sync pass, dry-run simulation, or file transfer | Indigo / Sky (`--color-info`) | `.ds-badge-running` | `SYNCING`, `SYNC IN PROGRESS`, `IN PROGRESS`, `ACTIVE` |
| **`Idle`** | System initialized, standby between scheduler intervals | Slate (`--color-text-secondary`) | `.ds-badge-idle` | `STANDBY`, `WAITING`, `SLEEPING` |
| **`Blocked`** | Stale lock detected, rate limit exponential backoff, or lock contention | Orange (`--color-blocked`) | `.ds-badge-blocked` | `LOCKED`, `STALE_LOCK`, `RATE_LIMIT_WAIT` |
| **`Review Required`** | Pending mass deletions exceeding safety ceiling or path conflict requiring operator decision | Purple (`--color-review`) | `.ds-badge-review` | `NEEDS REVIEW`, `MANUAL APPROVAL`, `PENDING APPROVAL`, `CONFLICT` |

---

## 3. Visual Representation Matrix

```
[Healthy]         • Green dot   | bg-emerald-500/10 text-emerald-400 border-emerald-500/20
[Warning]         • Amber dot   | bg-amber-500/10   text-amber-400   border-amber-500/20
[Critical]        • Red dot     | bg-rose-500/10    text-rose-400    border-rose-500/20
[Running]         • Blue pulse  | bg-indigo-500/10  text-indigo-400  border-indigo-500/20 (animate-pulse)
[Idle]            • Gray dot    | bg-slate-800      text-slate-400   border-slate-700/60
[Blocked]         • Orange dot  | bg-orange-500/10  text-orange-400  border-orange-500/20
[Review Required] • Purple dot  | bg-purple-500/10  text-purple-300  border-purple-500/20
```

---

## 4. Normalization Engine

In `dashboard/frontend/js/utils.js`, `normalizeStatus(rawStatus)` normalizes any raw backend code (`OPERATIONAL`, `FAULT`, `SYNCING`, `DELETION_THRESHOLD_EXCEEDED`) into its canonical approved status label before rendering.
