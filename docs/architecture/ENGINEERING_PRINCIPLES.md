# Engineering Principles

## Purpose

This document defines the engineering standards and architectural principles that guide implementation decisions throughout the project.

---

# 1. Service-Oriented Architecture

Business logic belongs in domain services.

Avoid embedding business logic in:

- HTTP handlers
- API controllers
- UI code
- Background job wrappers

Services should remain testable and reusable.

---

# 2. Explicit Models and Contracts

System contracts must be explicit.

Prefer:

- Typed structs
- Defined interfaces
- Versioned schemas

Avoid:

- Unstructured maps
- Anonymous payloads
- Implicit runtime contracts

---

# 3. Small Cohesive Components

Each package should have a clear responsibility.

Avoid:

- God objects
- Multi-purpose services
- Cross-domain coupling

When components grow beyond a single responsibility, they should be decomposed.

---

# 4. Testability By Design

Code should be designed for testing before implementation.

Implications:

- External dependencies should be abstracted.
- Integrations should be mockable.
- Services should be independently testable.

---

# 5. Atomic State Changes

Critical state updates should be atomic.

Partial writes and inconsistent state introduce operational risk.

Implications:

- Use temporary files and atomic rename strategies.
- Avoid partially committed state.

---

# 6. Structured Observability

Logs are a product capability.

Logging should help users answer:

- What happened?
- When did it happen?
- Why did it happen?

Logs should provide operational context.

---

# 7. Secure By Default

Safe behavior should be the default configuration.

Users may choose to reduce protections.

The system should never require users to manually enable basic safety controls.

---

# 8. Backward Compatibility Matters

Configuration changes should minimize disruption.

When breaking changes are unavoidable:

- Document them clearly.
- Provide migration guidance.
- Preserve user data.

---

# 9. Simplicity Over Cleverness

Simple solutions are preferred over clever solutions.

Future maintainers should be able to understand system behavior without extensive reverse engineering.

---

# 10. Build For Maintenance

The project is intended to be maintained over time.

Implementation decisions should favor:

- Readability
- Consistency
- Operational reliability

over short-term convenience.
