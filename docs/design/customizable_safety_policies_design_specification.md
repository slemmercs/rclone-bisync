# Design Specification: Customizable Safety Policies

**Feature Name**: Customizable Safety Policies (Custom Circuit Breakers, Deletion Guards, & Policy Guardrails)

**Status**: Draft / Proposed

**Target Milestone**: Phase 5 (Configuration Management & 3-Layer Safety Pipeline)

**Package Path**: `internal/engine/policies`

**Dependencies**: `internal/config`, `internal/audit` (Immutable Audit Ledger), `internal/services/deletion_service.go`, `internal/engine/runner`, `internal/services/review_service.go`

## 1. Executive Summary & Purpose

While Rclone provides basic command-line safety flags like `--max-delete`, automated synchronization in real-world homelab and enterprise setups requires more context-aware guardrails. Common synchronization disasters occur when:

1. A local or remote drive unmounts or mounts empty, causing Rclone to interpret missing files as mass deletions.

2. A script or user mistakenly deletes or moves a critical top-level folder (e.g., `/archive/` or `/tax-records/`).

3. Ransomware or application failure alters or truncates an entire directory tree, dropping total storage volume significantly.

**Customizable Safety Policies** provide declarative, fine-grained safety rules beyond standard Rclone flags. Operating directly within Bisync Hub's Go synchronization engine, policies evaluate conditions **before** any destructive data changes occur. If a rule threshold is breached, synchronization halts safely, locking background automated runs and launching an **Interactive Sync Review Session** with an explicit policy violation warning.

## 2. Policy Architecture & Dual-Stage Evaluation Lifecycle

Customizable Safety Policies run at two distinct execution checkpoints to maximize safety while minimizing cloud API quota consumption:

```
+-----------------------------------------------------------------------------------+
| STAGE 1: Pre-Execution Gate (Lightweight Single-Item Probes)                     |
|                                                                                   |
|  [Sync Trigger] --> [Evaluate Pre-Execution Policy Rules]                         |
|                             ├── Check Mandatory Subdirectory / Canary Presence    |
|                             ├── Verify Local Path Syscalls (os.Stat)              |
|                             └── Verify Remote Single-Path Stat (RC /operations/stat)
|                                         |                                         |
|                       +-----------------+-----------------+                       |
|                       |                                   |                       |
|                 (Pass │)                            (Fail │)                      |
|                       v                                   v                       |
|         [Proceed to Rclone Scan]                 [ABORT SYNC RUN]                 |
|                                                  [Log Audit Event]                |
+-----------------------------------------------------------------------------------+
                                                      |
                                                      v
+-----------------------------------------------------------------------------------+
| STAGE 2: Pre-Mutation Interceptor (Post-Scan Listing Analysis)                     |
|                                                                                   |
|  [Rclone Scan / Local .lst Parsing] --> [Evaluate Pre-Mutation Policy Rules]      |
|                                                ├── Max Delete Count / Percentage   |
|                                                ├── Protected Path Violation Check |
|                                                ├── Total Size Volume Drift        |
|                                                └── Extension Shift Thresholds     |
|                                                            |                      |
|                       +------------------------------------+--------------------+ |
|                       |                                                         | |
|                 (Pass │)                                                  (Fail │) |
|                       v                                                         v |
|          [Execute Synchronization]                          [CREATE REVIEW SESSION]
|          [Write Audit Record]                               [Tag POLICY_VIOLATION]
|                                                             [Lock Background Sync]
+-----------------------------------------------------------------------------------+
```

### Stage 1: Pre-Execution Gate (Lightweight Probes)

Evaluates baseline environment health before launching `rclone bisync`. To avoid burning cloud provider API quotas (such as Google Drive TPS limits or OneDrive throttling thresholds), Stage 1 does **not** run recursive directory scans or dry-run transfers. Instead, it uses targeted, single-item lookups:

* **Local Endpoints**: Evaluated using local OS syscalls (`os.Stat`). Overhead: $<1\text{ ms}$, 0 API calls.

* **Remote Endpoints**: Evaluated using a single targeted Rclone Remote Control (RC) HTTP request (`POST /operations/stat`) for specific required path markers (e.g., `.mount_check`). Overhead: ~100–200ms, exactly 1 API call.

If a required path is missing or an unmounted drive is detected, execution aborts immediately in under 200ms without initiating Rclone listing scans.

### Stage 2: Pre-Mutation Interceptor (Post-Scan Analysis)

Intercepts proposed changes after Rclone calculates synchronization deltas but **before** file operations are executed. Stage 2 evaluates the `.lst` listing files already generated locally by Rclone during its initial scan. Zero additional API calls are made. If proposed deletions, path modifications, or volume drops exceed configured policy thresholds, execution is blocked, and candidate changes are passed to the Interactive Sync Review workflow.

## 3. Quota & Performance Impact Analysis

| Rule Category | Stage | Execution Mechanism | Cloud API Cost per Run | Daily API Cost (15-min Syncs) |
| ----- | ----- | ----- | ----- | ----- |
| **`MANDATORY_SUBDIR`** | Stage 1 | Single-item `operations/stat` RC API probe | 1 API call per required path | 96 calls / day (<0.01% of quota) |
| **`MAX_DELETE_COUNT`** | Stage 2 | In-memory parsing of local Rclone `.lst` work files | 0 API calls | 0 calls / day |
| **`PROTECTED_PATH`** | Stage 2 | Regex matching against local `.lst` candidate list | 0 API calls | 0 calls / day |
| **`SIZE_DRIFT_THRESHOLD`**| Stage 2 | Size delta calculation over local Rclone listings | 0 API calls | 0 calls / day |

By catching unmounted drives or renamed folders at Stage 1 using a single 1-call probe, Bisync Hub prevents Rclone from attempting full recursive bucket listings, saving thousands of unnecessary API calls per failed run.

## 4. Data Contracts & Rule Models (`internal/engine/policies`)

```go
package policies

import (
	"context"
	"time"
)

type PolicyRuleType string
type PolicyAction string

const (
	// Rule Types
	RuleMaxDeleteCount     PolicyRuleType = "MAX_DELETE_COUNT"
	RuleMaxDeletePercent   PolicyRuleType = "MAX_DELETE_PERCENTAGE"
	RuleProtectedPath      PolicyRuleType = "PROTECTED_PATH"
	RuleMandatorySubdir    PolicyRuleType = "MANDATORY_SUBDIR"
	RuleSizeDriftThreshold PolicyRuleType = "SIZE_DRIFT_THRESHOLD"
	RuleConsecutiveFail    PolicyRuleType = "CONSECUTIVE_FAILURES"

	// Enforcement Actions
	ActionBlockAndReview PolicyAction = "BLOCK_AND_REVIEW" // Halts sync & creates Review Session
	ActionAbortSync      PolicyAction = "ABORT_SYNC"        // Cancels sync immediately without review
	ActionWarnOnly       PolicyAction = "WARN_ONLY"         // Logs warning & proceeds
)

// PolicyRule defines a single declarative safety rule.
type PolicyRule struct {
	ID          string                 `json:"id" yaml:"id"`
	Name        string                 `json:"name" yaml:"name"`
	Type        PolicyRuleType         `json:"type" yaml:"type"`
	Enabled     bool                   `json:"enabled" yaml:"enabled"`
	Action      PolicyAction           `json:"action" yaml:"action"`
	Parameters  map[string]interface{} `json:"parameters" yaml:"parameters"`
	Description string                 `json:"description" yaml:"description"`
}

// PolicyEvaluationContext contains runtime sync details for evaluation.
type PolicyEvaluationContext struct {
	ProfileID         string   `json:"profile_id"`
	LocalRoot         string   `json:"local_root"`
	RemoteRoot        string   `json:"remote_root"`
	TotalLocalFiles   int64    `json:"total_local_files"`
	TotalRemoteFiles  int64    `json:"total_remote_files"`
	TotalLocalBytes   int64    `json:"total_local_bytes"`
	TotalRemoteBytes  int64    `json:"total_remote_bytes"`
	ProposedDeletes   []string `json:"proposed_deletes"`
	ProposedModifies  []string `json:"proposed_modifies"`
	ConsecutiveErrors int      `json:"consecutive_errors"`
}

// PolicyViolation details a specific rule breach.
type PolicyViolation struct {
	RuleID         string         `json:"rule_id"`
	RuleName       string         `json:"rule_name"`
	RuleType       PolicyRuleType `json:"rule_type"`
	Action         PolicyAction   `json:"action"`
	Message        string         `json:"message"`
	MeasuredValue  interface{}    `json:"measured_value"`
	ThresholdValue interface{}    `json:"threshold_value"`
	OffendingPaths []string       `json:"offending_paths,omitempty"`
}

// PolicyEvaluationResult holds the overall evaluation outcome.
type PolicyEvaluationResult struct {
	Passed      bool              `json:"passed"`
	Violations  []PolicyViolation `json:"violations"`
	EvaluatedAt time.Time         `json:"evaluated_at"`
}

// Evaluator defines the interface for policy rule logic.
type Evaluator interface {
	Type() PolicyRuleType
	Evaluate(ctx context.Context, rule PolicyRule, evalCtx PolicyEvaluationContext) (*PolicyViolation, error)
}
```

## 5. Rule Taxonomy & Guardrail Behavior

| Rule Type | Stage | Trigger Condition | Parameter Example | Default Action | Primary Operational Explanation |
| :--- | :--- | :--- | :--- | :---: | :--- |
| **`MANDATORY_SUBDIR`** | Stage 1 | Targeted `operations/stat` lookup fails on required path | `required_paths: [".mount_check", "Documents"]` | `ABORT_SYNC` | "Mandatory path `.mount_check` missing on remote endpoint. Drive may be unmounted." |
| **`CONSECUTIVE_FAILURES`** | Stage 1 | Profile experiences $N$ consecutive failed sync attempts | `max_failures: 3` | `ABORT_SYNC` | "Profile paused automatically after 3 consecutive failed execution attempts." |
| **`MAX_DELETE_COUNT`** | Stage 2 | Absolute count of proposed deletions exceeds threshold | `max_count: 50` | `BLOCK_AND_REVIEW` | "Proposed deletion count (84 files) exceeds safety threshold (50 files)." |
| **`MAX_DELETE_PERCENTAGE`** | Stage 2 | Deletions exceed $X\%$ of total remote/local inventory | `max_percentage: 10.0`, `min_file_threshold: 20` | `BLOCK_AND_REVIEW` | "Proposed deletions represent 14.2% of total inventory (limit: 10.0%)." |
| **`PROTECTED_PATH`** | Stage 2 | Any proposed delete or overwrite matches glob/regex patterns | `patterns: ["/archive/**", "/tax-records/*"]` | `BLOCK_AND_REVIEW` | "Deletion proposed for protected path `/archive/2024/ledger.pdf`." |
| **`SIZE_DRIFT_THRESHOLD`** | Stage 2 | Total volume storage drops by more than $Y\%$ in a single cycle | `max_drop_percentage: 25.0` | `BLOCK_AND_REVIEW` | "Total storage volume dropped by 31.5% (limit: 25.0%). Possible mass truncation." |

## 6. Declarative Policy Configuration Example (`policies.yaml`)

```yaml
policies:
  - id: "guard-unmounted-drives"
    name: "Mount Verification Guard"
    type: "MANDATORY_SUBDIR"
    enabled: true
    action: "ABORT_SYNC"
    parameters:
      required_paths:
        - ".mount_check"
        - "Documents"
    description: "Aborts synchronization if top-level mount markers are missing."

  - id: "protect-critical-archives"
    name: "Protected Directory Guard"
    type: "PROTECTED_PATH"
    enabled: true
    action: "BLOCK_AND_REVIEW"
    parameters:
      patterns:
        - "/archive/**"
        - "/legal/**"
        - "**/photos/originals/**"
    description: "Prevents automatic deletion or overwrite of archival assets."

  - id: "limit-deletion-volume"
    name: "Mass Deletion Percentage Limit"
    type: "MAX_DELETE_PERCENTAGE"
    enabled: true
    action: "BLOCK_AND_REVIEW"
    parameters:
      max_percentage: 10.0
      min_file_threshold: 25
    description: "Triggers review if deletions exceed 10% of total profile inventory."
```

## 7. Review Integration & Immutable Audit Ledger

### 7.1 Review Session Payload

When a policy triggers a `BLOCK_AND_REVIEW` violation:

1. `SyncRunner` stops execution and persists a `ReviewSession` with `status: "POLICY_VIOLATION"`.

2. The dashboard displays a dedicated violation card highlighting the policy name, breached threshold vs. measured value, and the list of offending paths.

3. **User Action Options**:
   * **Override Policy & Proceed**: Applies a one-time administrative override for the active run.
   * **Selectively Approve/Restore**: Uses the Interactive Review Modal to approve valid items or restore blocked items.
   * **Abort & Modify Policy**: Cancels the run and opens the Policy Configuration editor.

```json
{
  "session_id": "rev_01JAX990",
  "status": "POLICY_VIOLATION",
  "policy_violations": [
    {
      "rule_id": "limit-deletion-volume",
      "rule_name": "Mass Deletion Percentage Limit",
      "rule_type": "MAX_DELETE_PERCENTAGE",
      "action": "BLOCK_AND_REVIEW",
      "message": "Proposed deletions represent 18.5% of total inventory (limit: 10.0%).",
      "measured_value": 18.5,
      "threshold_value": 10.0,
      "offending_paths": [
        "documents/projects/2024_old_archive.zip",
        "photos/raw/DSC_0012.NEF"
      ]
    }
  ]
}
```

### 7.2 Audit Ledger Event Logging

All safety policy events emit cryptographic records to the **Immutable Audit Ledger**:

* **`ActionPolicyViolated`**: Logged whenever a safety rule is breached.

* **`ActionPolicyOverridden`**: Logged when a user explicitly overrides a policy violation in the UI.

* **`ActionPolicyUpdated`**: Logged when a policy definition or rule threshold is modified.

## 8. Next Steps & Implementation Roadmap

1. **Sprint 1 (`internal/engine/policies`)**: Implement core policy structures, `Evaluator` interfaces, `ProtectedPathEvaluator`, `MaxDeletePercentEvaluator`, and `MandatorySubdirEvaluator` with >85% unit test coverage.

2. **Sprint 2 (`internal/engine/runner`)**: Wire Stage 1 Pre-Execution and Stage 2 Pre-Mutation hooks into `SyncRunner`.

3. **Sprint 3 (`dashboard/frontend`)**: Build the **Policy Management Panel** and violation alert card UI in the Alpine.js SPA.