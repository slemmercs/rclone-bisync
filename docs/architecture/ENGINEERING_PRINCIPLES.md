# Engineering Principles

## Purpose

This document defines the engineering standards and architectural principles that guide implementation decisions throughout the project.

---

# 1. Service-Oriented Architecture

Business logic belongs in domain services (`internal/services/`).

Avoid embedding business logic in:

- HTTP handlers and routing packages (`internal/server/`)
- API controllers
- UI code and frontend templates
- Background job wrappers

Services should remain testable and reusable with clear dependency injection boundaries. The application ships as a **Single Binary First**, embedding pre-compiled frontend assets (`dashboard/frontend/dist/*`) via Go's standard `embed` package without external runtime dependencies.

---

# 2. Explicit Models and Contracts

System contracts must be explicit.

Prefer:

- Typed Go structs defined in centralized schema packages (`internal/models`)
- Defined interfaces with minimal method sets
- Versioned schemas for persistent state and API communication
- Strict frontend TypeScript interfaces mirroring Go backend models (`internal/models/`) to avoid contract drift and loose runtime property assumptions

Avoid:

- Unstructured maps (`map[string]interface{}`) in core API payloads
- Anonymous payloads
- Implicit runtime contracts
- Untyped frontend state dictionaries and loose global mutations

---

# 3. Small Cohesive Components

Each package, service, and component should have a single, well-defined responsibility.

### Go Backend (`internal/`, `cmd/`)

Domain services must maintain focused architectural boundaries to avoid becoming "God Objects":

- **Sub-400 LOC Guideline**: Production Go backend files adhere to the ~400–500 LOC threshold to maintain modularity and readability.
- **Single Domain Responsibility**: Avoid multi-purpose services spanning unrelated domains. Core logic belongs in dedicated services (`TelemetryService`, `CanaryService`, `DeletionService`, `ReviewService`, `ExplorerService`, `SyncService`).
- **No Circular Coupling**: Maintain strict unidirectional dependencies between packages.

When Go services grow beyond a single responsibility or approach line-count thresholds, decompose them into cohesive domain packages or focused helper modules.

### Frontend (`dashboard/frontend/`)

Frontend components are evaluated on **component cohesion and single responsibility**, rather than an arbitrary flat line-count ceiling:

- **Cohesive Single-File Components**: A Single-File Component containing its template, scoped styles, and typed state logic is preferred over artificial file fragmentation.
- **Component Scoping**: Eliminate global window object mutation and unbundled script tags. Encapsulate local UI state within components.
- **Typed Store Governance**: Cross-cutting application state (active profiles, notifications, health telemetry) lives in dedicated, typed stores (e.g., Pinia / scoped Alpine stores).
- **Zero Runtime Compilers**: Pre-compile styles and templates ahead-of-time into static distribution assets (`dist/`), ensuring zero browser performance overhead and eliminating runtime JIT compilers.

---

# 4. Testability By Design

Code should be designed for testing before implementation.

Implications:

- External dependencies (such as Rclone RC API and subprocess execution) must be abstracted and mockable via standard `net/http/httptest`.
- Integrations must never invoke live cloud providers or destructive network calls during automated testing.
- Services should be independently testable using Arrange-Act-Assert (AAA) conventions.
- Maintain comprehensive unit test coverage (>80% statement target).

---

# 5. Atomic State Changes & Concurrency Control

Critical state updates must be atomic, and process execution must be race-free.

Partial writes and inconsistent state introduce operational and data loss risks.

Implications:

- Any telemetry, configuration, or profile persistence to disk must follow atomic write patterns (`.tmp.<pid>` + `os.Rename`).
- Critical background synchronization loops and engine operations must enforce advisory POSIX process locks (`flock`, `syscall.Flock`).
- State transitions must support clean rollback or fail-safe defaults if execution is interrupted.

---

# 6. Structured Observability

Logs and telemetry are core product capabilities, not debugging side-effects.

Logging should help users and operators answer:

- What happened?
- When did it happen?
- Why did it happen?

Logs should provide structured operational context while strictly sanitizing credentials, passwords, tokens, and webhook secrets.

---

# 7. Secure By Default

Safe behavior should be the default configuration.

Users may choose to reduce protections deliberately, but:

- Local storage browsing must be jailed within bounded directories (path traversal protection via `filepath.Rel`).
- Canary files (`RCLONE_TEST`) must be verified before mutating targets.
- The system should never require users to manually enable baseline safety controls.

---

# 8. Backward Compatibility Matters

Configuration changes should minimize disruption.

When breaking changes are unavoidable:

- Document them clearly in changelogs and release notes.
- Provide automated or guided migration paths.
- Preserve user profiles and historical audit records.

---

# 9. Simplicity Over Cleverness

Simple, idiomatic solutions are preferred over complex abstractions.

Future maintainers and contributors should be able to understand system behavior without extensive reverse engineering. Avoid metaprogramming or unnecessary dependency frameworks when standard Go library capabilities suffice.

---

# 10. Build For Maintenance

The project is intended to be maintained over time as an open-source tool.

Implementation decisions should favor:

- Readability and standard Go formatting (`gofmt`, `go vet`)
- Consistency across packages
- Operational reliability and graceful degradation

over short-term convenience or premature optimization.

---

## Related Documentation

- [Architecture Principles](ARCHITECTURE_PRINCIPLES.md)
- [Product Principles](PRODUCT_PRINCIPLES.md)
- [Security Principles](SECURITY_PRINCIPLES.md)
- [Contributor Workflow](CONTRIBUTOR_WORKFLOW.md)
- [Architecture Decision Records (ADRs)](adr/README.md)

