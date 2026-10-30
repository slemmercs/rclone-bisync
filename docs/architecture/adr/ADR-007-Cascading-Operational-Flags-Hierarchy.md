# ADR-007: 3-Tier Cascading Operational Flags Hierarchy

**Status:** Accepted

**Date:** 2026-10-30

---

## Context

Rclone bisync provides dozens of operational flags controlling synchronization behavior: `--resync`, `--check-access`, `--slow-hash`, `--drive-skip-gdocs`, `--resilient`, `--recover`, bandwidth caps, and filter rules.

Across multi-profile deployments, managing these parameters introduces critical operational problems:
1. **Configuration Drift**: Duplicating identical baseline flags across dozens of profiles is error-prone and leads to inconsistent fleet safety settings.
2. **Provider Specifics**: Different cloud backends require distinct flag sets (e.g., Google Drive requires `--drive-skip-gdocs` to prevent binary conversion errors; OneDrive requires specific hash checks).
3. **Recovery vs. Persistence**: Corrective recovery operations often require one-off flags (such as `--resync` or `--dry-run`) that must execute once without permanently altering stored profile configurations.
4. **Injection Vulnerabilities**: Unsanitized user inputs or arbitrary custom flag fields could allow parameter injection or trigger unintended rclone behavior.

---

## Decision

Bisync Hub establishes a **Deterministic 3-Tier Cascading Operational Flags Hierarchy** with security allowlisting and cloud provider presets:

### Tier 1: Global Fleet Defaults (`config.GlobalDefaults`)
- Managed centrally in `app_config.json` and editable via the Global Fleet Settings modal.
- Defines baseline operational flags inherited by all profiles across the instance (e.g., `check_access: true`, `resilient: true`, `lock_explorer_remote: true`).
- Serves as the single source of truth for global organization-wide sync policy.

### Tier 2: Profile-Level Configuration (`Profile.BisyncFlags`)
- Persisted per profile within `profiles.json`.
- Explicitly overrides Tier 1 defaults where specific profile requirements diverge (e.g., enabling `--slow-hash` for specific un-hashed storage or configuring custom filters).
- Integrates **Provider Presets** (`gdrive`, `onedrive`, `dropbox`, `s3`) that automatically apply optimized flag combinations based on detected remote types.

### Tier 3: Interactive Runtime Flags (Transient Execution Options)
- Provided dynamically during manual trigger requests (`DryRun: true`, `Resync: true`).
- Preempts Tier 1 and Tier 2 parameters **solely for the active execution run**.
- Runtime flags are strictly ephemeral; they execute the requested job and immediately discard transient overrides without altering the persistent profile configuration.

### Flag Allowlisting & Security Validation
- All custom flags entered via the advanced escape hatch (`CustomFlags`) are strictly validated against an approved allowlist of valid rclone bisync options (`internal/config/validator.go`).
- Shell metacharacters (`;`, `&`, `|`, `` ` ``, `$`) and dangerous commands are rejected before dispatch.

---

## Rationale

1. **DRY Configuration**: Centralized fleet defaults eliminate redundant boilerplate across profile definitions.
2. **Explainability**: The three tiers establish unambiguous precedence: `Global Defaults` < `Profile Overrides` < `Runtime Flags`. The UI renders visual inheritance badges to ensure users understand effective active flags.
3. **Safe Remediation**: Operators can perform emergency resynchronizations or dry-runs without accidentally persisting dangerous flags to scheduled cron jobs.
4. **Defense in Depth**: Strict flag allowlisting prevents command injection vulnerabilities.

---

## Consequences

### Positive
- Consistent fleet-wide safety baselines with zero configuration drift.
- Effortless onboarding via provider presets.
- Risk-free interactive troubleshooting via transient runtime flags.
- Comprehensive security validation against unauthorized parameter injection.

### Negative / Trade-offs
- Resolving effective configuration requires computing the 3-tier cascade before invoking rclone.
- Visual inheritance badges and indicator states are required in the dashboard UI so users can distinguish inherited defaults from explicit overrides.

---

## Related Principles

- [Product Principle 1: Safety Before Automation](../PRODUCT_PRINCIPLES.md#1-safety-before-automation)
- [Product Principle 3: Explainability Over Black Boxes](../PRODUCT_PRINCIPLES.md#3-explainability-over-black-boxes)
- [Product Principle 6: Profiles Are The Primary User Object](../PRODUCT_PRINCIPLES.md#6-profiles-are-the-primary-user-object)
- [Engineering Principle 2: Explicit Models and Contracts](../ENGINEERING_PRINCIPLES.md#2-explicit-models-and-contracts)
- [Security Principle 1: Least Privilege](../SECURITY_PRINCIPLES.md#1-least-privilege)

---

## Related ADRs

- [ADR-001: Profiles As Primary User Object](ADR-001-Profiles-As-Primary-User-Object.md)
- [ADR-002: Rclone Remote Control API As Source Of Truth](ADR-002-Rclone-As-Source-Of-Truth.md)
- [ADR-006: Multi-Layered Safety Circuit Breakers & Path Confinement](ADR-006-Multi-Layered-Safety-Circuit-Breakers.md)

---

## Metadata

**Owner:** @maintainer

**Tags:** configuration, flags, profiles, hierarchy, security
