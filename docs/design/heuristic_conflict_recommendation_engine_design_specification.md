# Design Specification: Heuristic Conflict Recommendation Engine

**Feature Name**: Heuristic Conflict Recommendation Engine

**Status**: Draft / Proposed

**Target Milestone**: Phase 4 (Interactive Sync Review)

**Package Path**: `internal/engine/heuristics`

**Dependencies**: Rclone Remote Control (RC) API, `internal/audit` (Immutable Audit Ledger)

---

## 1. Executive Summary & Purpose

When `rclone bisync` encounters bidirectional file modifications or safety circuit breakers (such as `TOO_MANY_DELETES`), it aborts the synchronization run. Rclone itself operates in batch mode and does not provide an interactive, file-by-file decision pipeline during execution.

The **Heuristic Conflict Recommendation Engine** acts as an operational bridge. It intercepts bisync abort events, extracts the candidate conflict list, and evaluates each file pair using a chain of deterministic pure-Go heuristics. It then presents non-binding, confidence-scored recommendations (`keep-local`, `keep-remote`, `keep-both`) within Bisync Hub's Interactive Sync Review dashboard. Resolutions are executed using targeted Rclone Remote Control (RC) primitive operations (`operations/copyfile`), followed by a state reset (`--resync`) to restore clean automated synchronization.

---

## 2. Orchestration Architecture & Execution Model

Bisync Hub separates batch synchronization execution from fine-grained decision intelligence through a three-stage lifecycle:

```text
+-----------------------------------------------------------------------------------+
| STAGE 1: Interception & Extraction                                                |
|                                                                                   |
|  [rclone bisync] --(Aborts on Conflict)--> [Bisync Hub Sync Runner]               |
|                                                     |                             |
|                                                     v                             |
|                                     [Parse Work Files / Log Output]               |
|                                                     |                             |
|                                                     v                             |
|                                       [Create Pending Review Session]             |
+-----------------------------------------------------------------------------------+
                                                      |
                                                      v
+-----------------------------------------------------------------------------------+
| STAGE 2: Parallel Offline Analysis                                                |
|                                                                                   |
|  Candidate List --> [Bounded Worker Pool (G=4)]                                    |
|                              |                                                    |
|                              v                                                    |
|                  [Analyzer Pipeline Chain]                                        |
|                  ├── Pattern / Temp Lock Analyzer                                 |
|                  ├── Clock Skew & DST Analyzer                                    |
|                  ├── Truncation / 0-Byte Guard                                    |
|                  └── Size Divergence Analyzer                                     |
|                              |                                                    |
|                              v                                                    |
|                  [Weighted Scoring Aggregator]                                    |
|                              |                                                    |
|                              v                                                    |
|           [Dashboard UI: Conflict Review Modal]                                   |
+-----------------------------------------------------------------------------------+
                                                      |
                                                      v
+-----------------------------------------------------------------------------------+
| STAGE 3: Targeted RC Resolution & Baseline Reset                                  |
|                                                                                   |
|  User Decision / Auto-Apply --> [Rclone RC API (operations/copyfile, etc.)]       |
|                                              |                                    |
|                                              v                                    |
|                                 [Immutable Audit Ledger]                          |
|                                              |                                    |
|                                              v                                    |
|                                 [rclone bisync --resync Execution]                |
+-----------------------------------------------------------------------------------+
```

### Stage 1: Interception & Candidate Extraction
1. `rclone bisync` exits with a conflict code or safety limit threshold.
2. Bisync Hub's `SyncRunner` traps the process exit state and parses Rclone's listing/work directory files (`.lst` files) and log output to isolate conflicting paths.
3. Automated background timer syncs are temporarily locked to prevent retry races.

### Stage 2: Offline Parallel Heuristic Analysis
1. Rclone is idle. Bisync Hub initiates an in-memory `ReviewSession`.
2. Missing file metadata (hashes, mime-types) is fetched on demand via low-overhead `operations/stat` RC API calls.
3. The candidate list is passed to `heuristics.Engine`, which processes files concurrently across pure-Go analyzer rules.
4. Each candidate receives a normalized `Recommendation` (action, confidence score, risk level, primary explanation).

### Stage 3: Targeted RC Resolution & Baseline Reset
1. The user accepts recommendations individually or in bulk via the dashboard review modal.
2. Bisync Hub executes specific, atomic Rclone RC primitives (`POST /operations/copyfile`, `POST /operations/deletefile`) for each file decision.
3. Decisions are recorded in the **Immutable Audit Ledger**.
4. Once all candidates are resolved and local/remote endpoints match, Bisync Hub executes `rclone bisync --resync` to establish a clean baseline for subsequent automated runs.

---

## 3. Data Contracts & Domain Models

```go
package heuristics

import (
	"context"
	"time"
)

type Action string
type RiskLevel string

const (
	ActionKeepLocal  Action = "keep-local"
	ActionKeepRemote Action = "keep-remote"
	ActionKeepBoth   Action = "keep-both"
	ActionManual     Action = "manual-review"

	RiskLow      RiskLevel = "LOW"
	RiskMedium   RiskLevel = "MEDIUM"
	RiskHigh     RiskLevel = "HIGH"
	RiskCritical RiskLevel = "CRITICAL"
)

// FileMeta captures endpoint file properties for analysis.
type FileMeta struct {
	Path     string    `json:"path"`
	Size     int64     `json:"size"`
	ModTime  time.Time `json:"mod_time"`
	Hash     string    `json:"hash"`
	MimeType string    `json:"mime_type"`
}

// Candidate represents a conflicting file pair.
type Candidate struct {
	Path   string   `json:"path"`
	Local  FileMeta `json:"local"`
	Remote FileMeta `json:"remote"`
}

// RuleResult represents the evaluation output from a single analyzer.
type RuleResult struct {
	RuleName string    `json:"rule_name"`
	Action   Action    `json:"action"`
	Score    float64   `json:"score"`  // Range: -1.0 (Strong Local) to +1.0 (Strong Remote)
	Weight   float64   `json:"weight"` // Priority multiplier
	Risk     RiskLevel `json:"risk"`
	Reason   string    `json:"reason"`
	Matched  bool      `json:"matched"`
}

// Recommendation represents the aggregated output for a single conflict.
type Recommendation struct {
	CandidatePath   string       `json:"candidate_path"`
	SuggestedAction Action       `json:"suggested_action"`
	Confidence      float64      `json:"confidence"` // Normalized 0.00 to 1.00
	Risk            RiskLevel    `json:"risk"`
	PrimaryReason   string       `json:"primary_reason"`
	EvaluatedRules  []RuleResult `json:"evaluated_rules"`
}

// Analyzer defines the interface implemented by all rule modules.
type Analyzer interface {
	Name() string
	Analyze(ctx context.Context, c Candidate) (RuleResult, error)
}
```

---

## 4. Analyzer Ruleset Matrix

The engine executes an array of focused analyzers:

| Rule Name | Trigger Condition | Recommended Action | Weight | Risk Level | Primary Explanation |
| :--- | :--- | :--- | :---: | :---: | :--- |
| **`TempLockAnalyzer`** | Path matches temporary/lock patterns (`~$*`, `*.tmp`, `*.swp`, `.DS_Store`, `desktop.ini`, `*.lock`) | `keep-local` or `keep-remote` (Keep primary document side) | 1.5 | **LOW** | "Target is an application lock or temporary file. Preserving primary document." |
| **`HashMatchAnalyzer`** | Content hashes match (`Local.Hash == Remote.Hash`), but modification timestamps differ | `keep-local` | 1.0 | **LOW** | "Content hashes are identical across endpoints despite timestamp differences." |
| **`ClockSkewAnalyzer`** | Timestamp delta is exactly 1 or 2 hours ($\pm 3600\text{s}$ or $\pm 7200\text{s}$) or $\le 2\text{s}$ filesystem clock drift | `keep-newer` | 0.8 | **MEDIUM** | "Timestamp delta matches an exact 1- or 2-hour Daylight Saving Time boundary." |
| **`TruncationAnalyzer`** | One side is 0 bytes or dropped $>90\%$ size while the other contains valid data | `keep-populated` | 2.0 | **CRITICAL** | "Potential 0-byte truncation detected. Preserving non-empty file." |
| **`SizeDivergenceAnalyzer`** | Binary asset size divergence $>20\%$ (`.sqlite`, `.zip`, `.psd`, `.iso`) | `manual-review` | 1.2 | **HIGH** | "Significant binary size divergence detected ($+45\%$). Manual review required." |
| **`ExtensionMutationAnalyzer`** | File path root is identical but extension changed on remote (e.g. `.docx` to `.docx.old`) | `keep-both` | 1.0 | **MEDIUM** | "File extension altered on remote endpoint. Preserving both copies." |

---

## 5. Scoring & Risk Aggregator

### 5.1 Scoring Formula

For each candidate, total score ($S$) and total weight ($W$) are computed across all matched rules:

$$S = \sum_{i=1}^{n} (\text{Score}_i \times \text{Weight}_i)$$

$$W = \sum_{i=1}^{n} \text{Weight}_i$$

The normalized score $N = \frac{S}{W}$ falls in the range $[-1.0, +1.0]$:
* $N \le -0.4 \longrightarrow \text{ActionKeepLocal}$
* $N \ge +0.4 \longrightarrow \text{ActionKeepRemote}$
* $-0.4 < N < +0.4 \longrightarrow \text{ActionKeepBoth}$

Confidence is derived from absolute magnitude: $\text{Confidence} = |N|$.

### 5.2 Safety Circuit Breakers
1. **Critical Risk Override**: If any rule triggers `RiskCritical` (e.g., `TruncationAnalyzer`), the aggregated action defaults to `ActionManual` regardless of timestamp recency.
2. **Low Confidence Fallback**: If $\text{Confidence} < 0.50$, the system forces `ActionManual`.
3. **Automated Batch-Apply Threshold**: Dashboard batch auto-resolution requires $\text{Confidence} \ge 0.85$ and $\text{Risk} \le \text{RiskMedium}$.

---

## 6. Dashboard Integration & Batch Endpoints

### 6.1 Review Session API (`GET /api/review/session`)

Returns candidate conflicts populated with heuristic recommendations:

```json
{
  "session_id": "rev_01JAX7B8",
  "status": "PENDING_DECISION",
  "candidates": [
    {
      "path": "projects/report.docx",
      "local": { "size": 1048576, "mod_time": "2026-10-23T18:00:00Z" },
      "remote": { "size": 1052000, "mod_time": "2026-10-23T20:30:00Z" },
      "recommendation": {
        "suggested_action": "keep-remote",
        "confidence": 0.88,
        "risk": "LOW",
        "primary_reason": "Remote file was continuously updated 2.5 hours later with valid size growth.",
        "evaluated_rules": [
          { "rule_name": "TimestampRecencyAnalyzer", "action": "keep-remote", "score": 0.8, "weight": 1.0, "risk": "LOW" }
        ]
      }
    }
  ]
}
```

### 6.2 Batch Action Endpoint (`POST /api/review/action-batch`)

Allows users to batch-apply safe, high-confidence recommendations:

```http
POST /api/review/action-batch
Content-Type: application/json

{
  "session_id": "rev_01JAX7B8",
  "min_confidence": 0.85,
  "max_allowed_risk": "MEDIUM"
}
```

Every decision executed via this batch call emits an entry to the Immutable Audit Ledger (`ACTION_CONFLICT_RESOLVED`).

---

## 7. Next Steps & Implementation Roadmap

1. **Sprint 1 (`internal/engine/heuristics`)**: Implement core types, scoring aggregator, `TempLockAnalyzer`, `ClockSkewAnalyzer`, and `TruncationAnalyzer` with >85% unit test coverage.
2. **Sprint 2 (`internal/services/review_service.go`)**: Wire candidate extraction from bisync abort logs and integrate RC `operations/copyfile` resolution execution.
3. **Sprint 3 (`dashboard/frontend`)**: Update the interactive Review Modal UI with recommendation badges, confidence meters, and batch action triggers.