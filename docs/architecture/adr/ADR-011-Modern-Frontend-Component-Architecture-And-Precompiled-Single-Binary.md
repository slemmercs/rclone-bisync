# ADR-011: Modern Frontend Component Architecture & Pre-Compiled Single Binary Embeddings

**Status:** Accepted

**Date:** 2026-11-15

---

## Context

Bisync Hub's frontend provides a rich management console featuring multi-profile fleet metrics, dual-pane directory tree browsing, real-time streaming logs, conflict review workflows, and cryptographic audit ledger inspection.

Under [ADR-005](ADR-005-Local-First-Architecture.md) and [ADR-010](ADR-010-Zero-Build-Design-System-And-Responsive-Telemetry.md), the system adopted a "Zero-Build Dogma" attempting to eliminate all Node.js and build toolchains. At the time, this was motivated by preserving single-binary simplicity and avoiding heavy frontend toolchains.

However, as the application matured, the Zero-Build Dogma coupled with an arbitrary sub-400 LOC ceiling on frontend files caused severe engineering dysfunctions:
1. **Unnatural Code Fragmentation**: Complex interactive features (such as the dual-pane file explorer, conflict review modal, and fleet manager) were artificially sliced across 40 disjointed JavaScript files solely to evade the 400 LOC check.
2. **Global Namespace Pollution & Temporal Coupling**: Without a modern module bundler, all scripts executed in a single global browser window scope (`window.*`). Component state and helper functions relied on fragile, implicit HTML script tag ordering, making refactoring treacherous.
3. **Absence of Type Safety & Contract Drift**: Frontend models were loose dictionaries, creating risk of silent runtime crashes when backend Go API models (`internal/models/`) changed.
4. **Client-Side Compilation Overhead**: Attempting modern styling without ahead-of-time compilation required either massive unpurged CSS files or runtime JIT compilers running in the browser, wasting client CPU and memory.

A critical architectural distinction was recognized:
- **Runtime Invariant**: The production binary must have **Zero Runtime Dependencies** and run as a 100% self-contained static executable with embedded assets (`//go:embed dashboard/frontend/dist/*`).
- **Developer Tooling Invariant**: Backend Go contributors must remain **Build-Artifact Independent**—cloning the repo and running `go build ./...` or `go test ./...` requires zero Node.js/npm tooling because pre-compiled `dist/` assets are tracked in version control.
- **Frontend Development Invariant**: Frontend UI development requires modern component scoping, typed contracts, and ahead-of-time bundling rather than unbundled global script spaghetti.

---

## Decision

Bisync Hub transitions to a **Modern Frontend Component Architecture with Pre-Compiled Single-Binary Embeddings**:

### 1. Zero Runtime Dependencies & Single Binary Invariant
- The production distribution remains a single, standalone Go executable.
- Production assets are embedded into the Go executable via `//go:embed dashboard/frontend/dist/*`.
- Zero external language runtimes (no Node.js, Python, or Ruby) and zero external background daemons are required to execute Bisync Hub.

### 2. Build-Artifact Independence for Go Contributors
- The pre-compiled frontend distribution (`dashboard/frontend/dist/`) is committed directly to version control.
- Go backend developers, MCP tool implementers, and system operators can build, test, and run the full application natively using standard Go toolchains (`go build`, `go test`) with zero Node.js or npm dependencies installed on their systems.
- Node.js/npm tooling is an optional prerequisite required only when modifying and compiling frontend source components.


### 3. Separation of LOC Scope: Go Services vs. Frontend Components
- **Go Backend (`internal/`, `cmd/`)**: Retains the strict ~400–500 LOC maintainability ceiling enforced by `cmd/check loc` (`./scripts/check.sh loc`) to prevent monolithic God Objects and maintain clean domain service boundaries (`TelemetryService`, `CanaryService`, `DeletionService`, `ReviewService`, `ExplorerService`, `SyncService`).
- **Frontend Components (`dashboard/frontend/src/`)**: Evaluated on **component cohesion and single responsibility**, rather than arbitrary line-count fragmentation. A cohesive Single-File Component containing its template, scoped styles, and typed state logic is preferred over artificially splitting cohesive state across multiple global files.
- `cmd/check` excludes frontend source and pre-compiled distribution directories (`dist/`) from Go LOC budget audits.

### 4. Component Scoping & Typed API Contracts
- Frontend modules utilize standard ES modules and scoped Single-File Components, eliminating `window.*` global mutation and temporal script loading hazards.
- Application-wide cross-cutting state (active profile, notifications, system health) lives in dedicated, typed state stores (e.g., Pinia / scoped Alpine stores). Local view state remains encapsulated within its respective component.
- Frontend TypeScript interfaces strictly mirror the Go model schemas defined in `internal/models/`, providing compile-time type validation against contract drift.

### 5. Ahead-of-Time Pre-Compilation & Purging
- Tailwind CSS and component assets are compiled and purged ahead of time into static distribution assets (`dist/app.css`, `dist/app.js`).
- Zero runtime compiler scripts or dynamic style injections execute in production browsers.

---

## Rationale

1. **Best of Both Worlds**: Preserves 100% of the operational simplicity and single-binary portability of Go while providing a robust, modern component model for UI development.
2. **Eliminates Invisible Coupling**: Component encapsulation eliminates order-dependent global script execution and prevents accidental state leakage across tabs or panes.
3. **Compile-Time Contract Guarantees**: Frontend type checking catches schema regressions before runtime, eliminating unexpected errors when Go backend payloads evolve.
4. **Zero Impact on Go-Only Workflows**: Go developers experience zero toolchain friction, zero extra build steps, and zero CI overhead when running Go tests and builds.

---

## Consequences

### Positive
- Production deployment model remains an untainted, self-contained single binary.
- Go developers require zero Node.js/npm environment setup.
- UI components are cohesive, scoped, and maintainable without arbitrary line-count fragmentation.
- Full type safety between Go backend models (`internal/models/`) and frontend state contracts.
- Ahead-of-time CSS purging drastically reduces asset sizes and eliminates client-side compilation lag.

### Negative / Trade-offs
- Frontend contributors who modify UI code must have Node.js and a package manager installed to run the build step and regenerate `dist/`.
- Git commits touching frontend source code must include updated pre-compiled assets in `dashboard/frontend/dist/`.

---

## Related Principles

- [Architecture Principles: Core Invariants 1 & 2](../ARCHITECTURE_PRINCIPLES.md#core-system-invariants)
- [Engineering Principle 1: Service-Oriented Architecture](../ENGINEERING_PRINCIPLES.md#1-service-oriented-architecture)
- [Engineering Principle 2: Explicit Models and Contracts](../ENGINEERING_PRINCIPLES.md#2-explicit-models-and-contracts)
- [Engineering Principle 3: Small Cohesive Components](../ENGINEERING_PRINCIPLES.md#3-small-cohesive-components)

---

## Related ADRs

- [ADR-005: Local-First Single Binary Deployment Model](ADR-005-Local-First-Architecture.md)
- [ADR-010: Zero-Build Component Design System & Responsive Telemetry Architecture](ADR-010-Zero-Build-Design-System-And-Responsive-Telemetry.md) *(Superseded)*

---

## Metadata

**Owner:** @maintainer

**Tags:** frontend, architecture, single-binary, components, types, governance
