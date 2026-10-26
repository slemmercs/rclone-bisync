# ADR-005: Local-First Single Binary Deployment Model

**Status:** Accepted

**Date:** 2026-10-26

---

## Context

Many modern developer and synchronization tools require multi-container Docker compositions, external relational databases, Node.js runtimes, cloud-hosted SaaS control planes, or outbound telemetry reporting.

Users managing sensitive backups and personal data on laptops, workstations, and homelabs strongly prioritize privacy, fast setup, and offline resilience. Introducing distributed architectures or mandatory third-party cloud infrastructure alienates these core users.

---

## Decision

Bisync Hub is architected as a **Local-First, Single-Binary Application**:

1. **Native Single Binary**: Compiled via standard Go toolchain into a standalone static binary with zero external language runtimes (no Node.js, Python, or Ruby dependencies).
2. **Embedded Frontend**: Dashboard static assets (HTML, Alpine.js, Tailwind CSS, Chart.js) are embedded into the binary using Go's `//go:embed`.
3. **Local State & Storage**: State, profile configurations, and audit ledgers reside entirely on the local host filesystem.
4. **Advisory POSIX Process Locking**: Concurrent process safety is guaranteed via local advisory file locking (`flock`, `syscall.Flock`), preventing multiple daemons or sync runners from colliding.
5. **Zero External Telemetry**: No remote analytics, tracking pixels, or outbound reporting are included.

---

## Rationale

1. **User Privacy & Sovereignty**: User credentials, sync profiles, and file lists never leave the host machine.
2. **Operational Simplicity**: Deployment requires nothing more than downloading and running a single executable.
3. **High Reliability**: Operates completely offline, with network access required only when communicating with configured cloud storage remotes.

---

## Consequences

### Positive
- Instant startup and minimal memory footprint.
- Straightforward operational troubleshooting and atomic backup of the application directory.
- Frictionless installation on bare metal, VMs, or single-container Docker setups.

### Negative / Trade-offs
- Multi-host clustering or centralized multi-node fleet management is intentionally out of scope for the core daemon.

---

## Related Principles

- [Product Principle 8: Local First](../PRODUCT_PRINCIPLES.md#8-local-first)
- [Architecture Principles: Core Invariants](../ARCHITECTURE_PRINCIPLES.md#core-system-invariants)
- [Engineering Principle 1: Service-Oriented Architecture](../ENGINEERING_PRINCIPLES.md#1-service-oriented-architecture)
- [Security Principle 1: Least Privilege](../SECURITY_PRINCIPLES.md#1-least-privilege)

---

## Related ADRs

- [ADR-001: Profiles As Primary User Object](ADR-001-Profiles-As-Primary-User-Object.md)
- [ADR-002: Rclone Remote Control API As Source Of Truth](ADR-002-Rclone-As-Source-Of-Truth.md)
- [ADR-004: Cryptographic Append-Only SQLite Audit Ledger](ADR-004-Immutable-Audit-Ledger.md)

---

## Metadata

**Owner:** @maintainer

**Tags:** architecture, deployment, privacy, local-first
