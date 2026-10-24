---
layout: default
title: Architecture & Safety
nav_order: 7
description: "Core safety tenets, POSIX advisory locks, path confinement, and defense-in-depth architecture."
---

# Architecture & Safety Principles

Bisync Hub is designed under the assumption that networks drop, cloud APIs produce unexpected HTTP responses, and distributed clocks drift. 

Every design decision adheres to three foundational tenets:

1. **Visibility Before Action**: Never mutate remote storage silently without an immutable record and pre-flight inspection.
2. **Safety Before Automation**: If an environmental invariant is violated (e.g. missing canary, unresolvable conflict), halt immediately rather than guess.
3. **Defense in Depth**: Protect the host filesystem with path confinement jails, POSIX process flocking, and secret scrubbing.

---

## Safety Mechanisms

### 1. Filesystem Path Confinement Jails
When browsing files or preparing sync operations on local storage endpoints, Bisync Hub prevents arbitrary filesystem access by enforcing strict jail boundaries:

```go
// Enforcing local path isolation
relPath, err := filepath.Rel(jailRoot, targetPath)
if err != nil || strings.HasPrefix(relPath, "..") || strings.HasPrefix(relPath, "/") {
    return ErrPathTraversalViolation
}
```
Any request attempting directory traversal (`../`) outside the configured root returns an immediate authorization violation and is recorded in the audit log.

### 2. POSIX Flock Process Isolation
To avoid lock races between scheduled background timers, cron jobs, and interactive operator CLI requests, Bisync Hub acquires advisory file locks using native POSIX `flock(fd, LOCK_EX|LOCK_NB)`.
- Concurrent executions fail fast with a clear error rather than piling up or corrupting state.
- Stale lock detection cleans up safely if a previous process terminated abnormally.

### 3. Secret Redaction at Every Boundary
All external inputs, configuration files, and API logs pass through recursive sanitization via `security.RedactSecrets`.
- Keywords matching `PASS`, `TOKEN`, `KEY`, `SECRET`, and `CREDENTIAL` are dynamically masked (`[REDACTED]`).
- Raw tokens are never persisted in SQLite ledger records, JSON error messages, or telemetry streams.

### 4. Zero-Exposure CI/CD Security
Bisync Hub's GitHub Actions pipelines are deployed 100% on cloud-hosted ephemeral runners (`ubuntu-latest`) with zero self-hosted runner exposure. Every commit is validated with:
- Static analysis via `go vet`
- Automated leak scanning via `gitleaks`
- Vulnerability auditing via `govulncheck`
- Strict branch protection with sole review governance (`@slemmercs`)
