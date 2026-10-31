# Design Specifications Registry

This directory contains technical design specifications detailing interface models, component interactions, user workflows, and state transitions across Bisync Hub.

While **[Architecture Decision Records (ADRs)](../architecture/adr/README.md)** govern the overarching architectural decisions, trade-offs, and invariants ("Why"), these **Design Specifications** define concrete interface schemas, user journeys, layout systems, and domain service mechanics ("How").

---

## Master Specifications Registry

| Specification | Domain | Governing ADR(s) | Status | Key Focus |
|:---|:---|:---|:---|:---|
| **[UI Design System Specification](bisync-hub-design-system-specification.md)** | Frontend & UI/UX | [ADR-010](../architecture/adr/ADR-010-Zero-Build-Design-System-And-Responsive-Telemetry.md) | `Shipped` | Zero-build CSS tokens, 7 canonical status labels, 2-tier header, atomic primitives, and slide-over drawer patterns. |
| **[Bisync Operational Flags](bisync_operational_flags.md)** | Engine & Config | [ADR-007](../architecture/adr/ADR-007-Cascading-Operational-Flags-Hierarchy.md) | `Shipped` | 3-tier precedence hierarchy (global, profile, runtime), flag allowlisting, and provider presets. |
| **[Interactive Sync Review Engine](interactive-sync-review.md)** | Safety & UX | [ADR-003](../architecture/adr/ADR-003-Human-In-The-Loop-Review.md), [ADR-008](../architecture/adr/ADR-008-WAL-Resilient-Interactive-Conflict-Review.md) | `Shipped` | Write-Ahead Log (WAL) session engine, staged file actions, batch execution, and pre-mutation backups. |
| **[Heuristic Conflict Recommendation Engine](heuristic_conflict_recommendation_engine_design_specification.md)** | Safety & State | [ADR-003](../architecture/adr/ADR-003-Human-In-The-Loop-Review.md), [ADR-008](../architecture/adr/ADR-008-WAL-Resilient-Interactive-Conflict-Review.md) | `Shipped` | Intelligent recommendation heuristics (modtime, size, content hash), confidence scores, and conflict resolution staging. |
| **[Immutable Audit Ledger Specification](immutable_audit_ledger_design_specification.md)** | Security & Audit | [ADR-004](../architecture/adr/ADR-004-Immutable-Audit-Ledger.md) | `Shipped` | Cryptographic SHA-256 hash chains, append-only SQLite schema, decoupled execution log streaming, and tampering verification. |
| **[Audit Ledger Dual-Context Workspace](audit-ledger-architecture-dual-context-workspace.md)** | Security & UX | [ADR-004](../architecture/adr/ADR-004-Immutable-Audit-Ledger.md), [ADR-005](../architecture/adr/ADR-005-Local-First-Architecture.md) | `Shipped` | Global fleet ledger view vs. profile-scoped workspace ledger, cryptographic integrity continuity status. |
| **[Customizable Safety Policies](customizable_safety_policies_design_specification.md)** | Safety & Security | [ADR-006](../architecture/adr/ADR-006-Multi-Layered-Safety-Circuit-Breakers.md) | `Shipped` | Canary verification (`RCLONE_TEST`), deletion thresholds (`MaxDeletePercentage`, `MaxDeleteCount`), and path confinement jails. |
| **[Profile Manager Architecture](profile-manager.md)** | Product & State | [ADR-001](../architecture/adr/ADR-001-Profiles-As-Primary-User-Object.md), [ADR-005](../architecture/adr/ADR-005-Local-First-Architecture.md) | `Shipped` | Multi-profile lifecycle, isolated SQLite profile stores, independent scheduling, and per-profile runtime state. |
| **[Hybrid Fleet Overview & Contextual Workspace](hybrid_fleet_overview_contextual_profile_workspace.md)** | Frontend & UX | [ADR-005](../architecture/adr/ADR-005-Local-First-Architecture.md), [ADR-010](../architecture/adr/ADR-010-Zero-Build-Design-System-And-Responsive-Telemetry.md) | `Shipped` | Multi-tenant fleet grid view, single-workload scoped cockpit, and dynamic tab navigation routing. |
| **[Daemon & Log Inline Deduplication Folding](daemon_log_deduplication_inline_folding.md)** | Frontend & Telemetry | [ADR-004](../architecture/adr/ADR-004-Immutable-Audit-Ledger.md), [ADR-010](../architecture/adr/ADR-010-Zero-Build-Design-System-And-Responsive-Telemetry.md) | `Shipped` | Repetitive log folding, consecutive notice clustering (`×N repeated`), and run-aggregated expandable timelines. |
| **[Telemetry Tab & Run-Aggregated Ledger](telemetry_tab_redesign_run_aggregated_log_ledger.md)** | Frontend & Telemetry | [ADR-004](../architecture/adr/ADR-004-Immutable-Audit-Ledger.md), [ADR-010](../architecture/adr/ADR-010-Zero-Build-Design-System-And-Responsive-Telemetry.md) | `Shipped` | Correlated `run_id` session aggregation, audit decision pills, live transfer progress gauges, and latency monitoring. |
| **[OAuth Connection Manager](oauth-connection-manager.md)** | Storage & Auth | [ADR-002](../architecture/adr/ADR-002-Rclone-As-Source-Of-Truth.md) | `Shipped` | In-app loopback OAuth2 authorize/exchange workflow, token refresh monitoring, and remote configuration via Rclone RC API. |
| **[Rclone RC Integration Framework](rclone-rc-framework.md)** | Architecture & Remote | [ADR-002](../architecture/adr/ADR-002-Rclone-As-Source-Of-Truth.md) | `Shipped` | HTTP connection pooling, retry policies, async job tracking (`job/status`), remote operations, and filesystem primitives. |
| **[System Integration Architecture Specification](system_integration_architecture_specification.md)** | System Architecture | Cross-Cutting | `Shipped` | End-to-end component coordination across Runner, Compiler, Audit, Heuristics, and Remote Auth subsystems. |
| **[Operational Visibility & Queued Uploads](operational_visibility_and_queued_upload_specification.md)** | Storage & Operations | [ADR-002](../architecture/adr/ADR-002-Rclone-As-Source-Of-Truth.md), [ADR-010](../architecture/adr/ADR-010-Zero-Build-Design-System-And-Responsive-Telemetry.md) | `Draft` | Remote cloud links, asynchronous upload queuing, and background transfer monitoring. |
| **[Remote Mount Manager & Pooled Storage Engine](remote_mount_manager.md)** | Storage & Mounts | [ADR-002](../architecture/adr/ADR-002-Rclone-As-Source-Of-Truth.md) | `Draft` | On-demand VFS streaming mounts via Rclone RC (`mount/mount`), multi-cloud storage pooling, and mount lifecycle management. |

---

## Domain Groupings

### 1. Frontend & Design System
- **[bisync-hub-design-system-specification.md](bisync-hub-design-system-specification.md)**: Master design token scale, semantic palette, canonical 7-state status vocabulary, composite patterns, and component catalog.
- **[hybrid_fleet_overview_contextual_profile_workspace.md](hybrid_fleet_overview_contextual_profile_workspace.md)**: Two-tier navigation model (Fleet scope vs. Profile workspace).
- **[daemon_log_deduplication_inline_folding.md](daemon_log_deduplication_inline_folding.md)**: High-density log stream deduplication and repeated notice clustering.

### 2. Safety, Circuit Breakers & Review
- **[customizable_safety_policies_design_specification.md](customizable_safety_policies_design_specification.md)**: Multi-layer safety architecture, canary validation (`RCLONE_TEST`), automated deletion barriers, and path confinement.
- **[interactive-sync-review.md](interactive-sync-review.md)**: Pre-mutation conflict review workflow, Write-Ahead Log (WAL) crash resilience, and staging actions.
- **[heuristic_conflict_recommendation_engine_design_specification.md](heuristic_conflict_recommendation_engine_design_specification.md)**: Metadata-driven conflict resolution scoring and recommendation pipeline.

### 3. Execution Engine & Configuration
- **[bisync_operational_flags.md](bisync_operational_flags.md)**: 3-tier operational flags hierarchy, security allowlisting, and cloud provider presets.
- **[profile-manager.md](profile-manager.md)**: Profile lifecycle, multi-tenant isolation, cron scheduling, and runtime context.
- **[system_integration_architecture_specification.md](system_integration_architecture_specification.md)**: Subsystem orchestration and cross-domain data contracts.

### 4. Telemetry, Security & Audit
- **[immutable_audit_ledger_design_specification.md](immutable_audit_ledger_design_specification.md)**: Cryptographic SHA-256 hash chaining, SQLite append-only audit events, and tamper detection.
- **[audit-ledger-architecture-dual-context-workspace.md](audit-ledger-architecture-dual-context-workspace.md)**: Fleet-wide vs. profile-scoped ledger views and cryptographic continuity verification.
- **[telemetry_tab_redesign_run_aggregated_log_ledger.md](telemetry_tab_redesign_run_aggregated_log_ledger.md)**: Run-aggregated log ledger, live transfer statistics, and daemon health indicators.

### 5. Storage, Remotes & Rclone RC
- **[rclone-rc-framework.md](rclone-rc-framework.md)**: Rclone RC client primitives, job execution, and remote filesystem abstractions.
- **[oauth-connection-manager.md](oauth-connection-manager.md)**: Embedded OAuth authorization flow, loopback callbacks, and credential security.
- **[operational_visibility_and_queued_upload_specification.md](operational_visibility_and_queued_upload_specification.md)**: Remote explorer file transfer queues.
- **[remote_mount_manager.md](remote_mount_manager.md)**: VFS mounts and multi-cloud virtual file system aggregation.

