---
layout: default
title: "2. Heuristic Conflict Engine"
parent: Core Pillars
nav_order: 2
---

# Multi-Analyzer Heuristic Conflict Engine

Most bidirectional sync tools rely on a naive, binary rule: **"newer timestamp wins"**. In distributed cloud environments, this rule causes silent data loss because:
1. Cloud storage APIs (e.g. Google Drive, Microsoft OneDrive, SharePoint) round timestamps to 1-second or 2-second windows or update them asynchronously.
2. Device clocks can drift or suffer skew during daylight savings transitions.
3. Rapid concurrent edits on mobile and desktop produce collisions where the "newer" file is actually an incomplete draft.

**Bisync Hub replaces timestamp gambling with a multi-analyzer heuristic confidence engine.**

---

## The Evaluation Pipeline

When both Path1 and Path2 report modifications to the same file path, the engine executes four independent heuristic analyzers:

| Heuristic Analyzer | Signal Analyzed | Weight | Description |
| :--- | :--- | :--- | :--- |
| **SizeDivergenceAnalyzer** | Content volume delta | **30%** | Detects dramatic file shrinks or expansions indicating truncation or major additions. |
| **BinarySignatureAnalyzer** | MIME & header magic bytes | **25%** | Verifies format validity (e.g. SQLite database header, JPEG signatures) to penalize corrupt partial writes. |
| **ClockSkewAnalyzer** | Cross-remote timestamp drift | **25%** | Evaluates cloud API timestamp granularity and compensates for known provider skews (e.g., OneDrive 2s window). |
| **VelocityAnalyzer** | Modification frequency | **20%** | Scores edit frequency to distinguish steady active editing from stale background touches. |

---

## Confidence Calculation & Safety Bands

The engine aggregates individual analyzer scores into a unified **Confidence Index (0% - 100%)**:

```text
Score >= 85%  ──► [ HIGH CONFIDENCE ]  ──► Auto-resolve in favor of dominant replica
60% - 84%    ──► [ MEDIUM CONFIDENCE ]──► Backup losing candidate to .conflict-bak, then sync
Score < 60%   ──► [ LOW CONFIDENCE ]   ──► QUARANTINE: Halt file mutation & prompt Operator
```

### Safety Features
- **Deterministic Quarantine**: Unresolvable collisions are safely preserved in a `.bisync_quarantine/` isolation jail.
- **Conflict Backup Preserves**: Even on high-confidence resolutions, the displaced version is moved to the backup store before overwriting.
- **Provider Tolerance**: Verified compatibility with Google Drive v3 APIs and Microsoft OneDrive Graph API timestamp quirks.

{: .note }
> All heuristic scoring weights and confidence thresholds are fully configurable in `bisync.env` to match your organization's risk tolerance.
