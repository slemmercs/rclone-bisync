# Architecture Decision Records (ADRs)

## Purpose

Architecture Decision Records (ADRs) document significant technical, architectural, and governance decisions made in Bisync Hub. They capture the context, rationale, trade-offs, and consequences of key choices to maintain architectural consistency over time.

---

## Registry of Decisions

| ADR | Title | Status | Date | Primary Domain |
|:---|:---|:---|:---|:---|
| [ADR-000](ADR-000-Template.md) | Architecture Decision Record Template | Accepted | 2026-10-26 | Governance |
| [ADR-001](ADR-001-Profiles-As-Primary-User-Object.md) | Profiles As Primary User Object | Accepted | 2026-10-26 | Product & State |
| [ADR-002](ADR-002-Rclone-As-Source-Of-Truth.md) | Rclone Remote Control API As Source Of Truth | Accepted | 2026-10-26 | Architecture |
| [ADR-003](ADR-003-Human-In-The-Loop-Review.md) | Human-In-The-Loop Review Before Mutation | Accepted | 2026-10-26 | Safety & UX |
| [ADR-004](ADR-004-Immutable-Audit-Ledger.md) | Cryptographic Append-Only SQLite Audit Ledger | Accepted | 2026-10-26 | Security & Audit |
| [ADR-005](ADR-005-Local-First-Architecture.md) | Local-First Single Binary Deployment Model | Accepted | 2026-10-26 | Engineering |

---

## ADR Lifecycle & Status Values

- **Proposed**: The decision is under discussion and has not yet been accepted.
- **Accepted**: The decision has been approved and represents the binding architecture.
- **Superseded**: The decision has been superseded by a newer ADR (e.g., *ADR-004 superseded by ADR-017*). Superseded records remain in the repository for historical rationale.
- **Deprecated**: The decision is no longer recommended or active, but has not been formally replaced.

---

## Authoring Guidelines & Naming Conventions

### 1. Naming Format
ADRs use sequential three-digit identifiers and hyphen-separated Title Case:

```text
ADR-XXX-Short-Decision-Name.md
```

### 2. Examples
- ✅ `ADR-001-Profiles-As-Primary-User-Object.md`
- ✅ `ADR-006-Safety-Before-Automation.md`
- ✅ `ADR-007-Advisory-Locking-For-Daemon-Isolation.md`
- ❌ `ADR-008-New-Sync-Service.md` *(Avoid code artifact/class names)*
- ❌ `adr-009-refactor.md` *(Must use Title Case and three digits)*

### 3. How to Propose a New ADR
1. Copy [`ADR-000-Template.md`](ADR-000-Template.md) to `ADR-XXX-Title.md` with the next sequential number.
2. Fill out all sections: Context, Decision, Rationale, Consequences, and Related Principles.
3. Add a row to the table in this `README.md` with status `Proposed`.
4. Open a pull request for team and community review.
