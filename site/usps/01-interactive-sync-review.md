---
layout: default
title: "1. Interactive Sync Review"
parent: Core Pillars
nav_order: 1
---

# Interactive Sync Review & Deletion Diff Visualizer

Standard automated sync daemons run unattended in background crons. When an unexpected deletion wave occurs—such as an operator accidentally moving a root directory or an external service wiping a folder—unattended sync immediately replicates the wipe across all replicas.

**Bisync Hub introduces an interactive human-in-the-loop inspection barrier before destructive mutations are committed.**

---

## How It Works

```text
[ Trigger Sync ] ──► [ Dry-Run Operation ] ──► [ Parse Mutation List ]
                                                       │
                           ┌───────────────────────────┴───────────────────────────┐
                           ▼                                                       ▼
                Deletions <= Max Threshold                              Deletions > Max Threshold
                           │                                                       │
                           ▼                                                       ▼
                [ Auto-Apply Policies ]                                [ HALT & Emit Review Event ]
                                                                                   │
                                                                                   ▼
                                                                       [ Operator Decision Modal ]
                                                                        • Confirm Specific Deletes
                                                                        • Reject / Abort Run
                                                                        • Override Heuristics
```

### 1. Dry-Run Mutation Analysis
Every scheduled or manual sync cycle executes in `--dry-run` inspection mode first. The engine streams the stdout/stderr mutation log and categorizes every proposed operation:
- **`+ CREATE`**: New file appearing on either Path1 or Path2.
- **`~ MODIFY`**: Content or metadata discrepancy requiring replication.
- **`- DELETE`**: Proposed removal of a file or directory.

### 2. High-Severity Quota Halts
If proposed deletions exceed either:
- The absolute file limit (e.g. `MAX_DELETE_COUNT=50`), or
- The directory percentage ratio (e.g. `MAX_DELETE_PERCENT=10%`),

the sync engine immediately trips the circuit breaker, writes an atomic `.review_pending` state, and emits real-time WebSocket notifications to the dashboard and external webhooks (Discord, Telegram, Healthchecks.io).

### 3. Granular Operator Triage
Operators can open the Web Dashboard review modal to:
- Inspect file diffs, file sizes, and last modified timestamps side-by-side.
- Selectively whitelist legitimate bulk deletions while rejecting questionable removals.
- Commit the approved batch or abort the run cleanly without leaving locks in place.

{: .safety }
> **Safety Guarantee**: No file is ever removed when a review threshold is tripped until an authorized operator explicitly signs off or an automated timeout policy executes an abort.
