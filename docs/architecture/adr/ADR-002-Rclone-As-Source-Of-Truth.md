# ADR-002: Rclone Remote Control API As Source Of Truth

**Status:** Accepted

**Date:** 2026-10-26

---

## Context

Bisync Hub requires deep interoperability with cloud storage providers and local filesystems to inspect directory trees, fetch transfer statistics, list configured remotes, and execute bidirectional synchronization.

Re-implementing storage provider SDKs (Google Drive, S3, OneDrive, Dropbox, WebDAV) or writing a native Bisync algorithm inside Bisync Hub would create an unsustainable maintenance burden and introduce divergence from upstream bug fixes.

---

## Decision

Bisync Hub establishes the **Rclone Remote Control (RC) HTTP API** (`internal/core/client.go`) as the sole storage connector and synchronization execution engine.

Bisync Hub operates strictly as a governance, orchestration, and review platform:
- All remote listing and authentication is delegated to Rclone (`/config/listremotes`).
- All file listing and directory traversal is delegated to Rclone (`/operations/list`).
- All bisync runs are initiated and monitored via Rclone RC endpoints (`/sync/bisync`, `/core/stats`).
- Bisync Hub does **not** implement storage drivers or bypass Rclone's synchronization logic.

---

## Rationale

1. **Ecosystem Leverage**: Rclone supports 70+ storage providers and actively maintains API compatibility, OAuth credential refreshing, rate limiting, and chunked uploads.
2. **Separation of Concerns**: Rclone excels at file transport; Bisync Hub excels at safety guardrails, human review workflows, and configuration management.
3. **Safety & Stability**: Upstream bisync bug fixes and improvements immediately benefit Bisync Hub without requiring engine rewrites.

---

## Consequences

### Positive
- Zero storage-specific vendor code in Bisync Hub.
- Automatic support for new storage remotes configured in user `rclone.conf`.
- Standardized HTTP/JSON interaction allowing 100% test isolation via `net/http/httptest`.

### Negative / Trade-offs
- Bisync Hub requires an active Rclone instance with RC enabled (`rclone rcd`).
- Bisync Hub is constrained by the features and parameters exposed through Rclone's RC API.

---

## Related Principles

- [Product Principle 7: Rclone Remains The Source Of Truth](../PRODUCT_PRINCIPLES.md#7-rclone-remains-the-source-of-truth)
- [Engineering Principle 1: Service-Oriented Architecture](../ENGINEERING_PRINCIPLES.md#1-service-oriented-architecture)
- [Engineering Principle 4: Testability By Design](../ENGINEERING_PRINCIPLES.md#4-testability-by-design)

---

## Related ADRs

- [ADR-001: Profiles As Primary User Object](ADR-001-Profiles-As-Primary-User-Object.md)
- [ADR-003: Human-In-The-Loop Review Before Mutation](ADR-003-Human-In-The-Loop-Review.md)

---

## Metadata

**Owner:** @maintainer

**Tags:** architecture, rclone, integration
