# Architecture Principles

These principles define the foundational architecture, system invariants, and engineering standards for Bisync Hub.

Any significant change to the platform must align with these principles.

---

## The Four Architectural Pillars

Bisync Hub's technical governance is structured across four specialized specifications:

1. 📋 **[Product Principles](PRODUCT_PRINCIPLES.md)**  
   Defines the user-centric mission: *Review. Decide. Sync.* Prioritizes human decision authority, explainability over black boxes, safety before automation, and profiles as primary user abstractions.

2. ⚙️ **[Engineering Principles](ENGINEERING_PRINCIPLES.md)**  
   Governs code organization, domain service separation (`internal/services/`), Go service sub-400 LOC budgets and cohesive frontend components, atomic file write patterns (`.tmp.<pid>` + `os.Rename`), POSIX advisory locking (`flock`), and testability by design.

3. 🛡️ **[Security Principles](SECURITY_PRINCIPLES.md)**  
   Mandates local filesystem path confinement jails, canary file protection (`RCLONE_TEST`), recursive credential sanitization (`security.RedactSecrets`), safe failure modes, and tamper-evident audit trails.

4. 🔄 **[Contributor Workflow](CONTRIBUTOR_WORKFLOW.md)**  
   Prescribes the standard development lifecycle: design-first planning, small verifiable tasks, continuous pre-flight verification via `./scripts/check.sh all`, 3-try circuit breakers, and checkpoint commits.

---

## Core System Invariants

### 1. Single Binary First & Zero Runtime Dependencies
Bisync Hub ships as a single, self-contained executable binary. It requires zero external language runtimes (no Node.js, Python, or Ruby) and zero external background daemons to operate. Contributors working exclusively on Go backend services, MCP tools, or sync algorithms can clone and run `go build ./...` or `go test ./...` with zero Node.js/npm tooling installed.

### 2. Embedded Pre-Compiled Frontend
Web dashboard production assets (`dashboard/frontend/dist/*`) are compiled ahead-of-time and embedded directly into the Go executable via standard `//go:embed`. Pre-compiled distribution bundles are committed to version control, preserving instant Go builds without runtime compilers (no browser-side Tailwind JIT) and eliminating external frontend server deployments.

### 3. Local-First Design
The platform is designed primarily for individual power users, homelabs, and self-hosted environments. It operates completely independently of third-party cloud control planes, centralized telemetry servers, or remote licensing authorities.

### 4. Rclone as the Storage Authority
Bisync Hub orchestrates and governs synchronization; it does not reimplement cloud storage drivers or low-level transport protocols. Rclone Remote Control (RC) remains the single source of truth for file operations.

### 5. Safety & Review Before Mutation
Destructive synchronization actions (deletions, conflict overrides) require human awareness and explicit intent. Automated execution is bounded by safety policies and circuit breakers.

### 6. Minimal Runtime Footprint
Avoid introducing external database servers (PostgreSQL, MySQL, Redis) or heavy middleware stacks. Internal state is maintained locally using atomic JSON stores and an embedded SQLite cryptographic audit ledger.

---

## Architecture Decision Records (ADRs)

Architectural and design decisions are formally documented and tracked in the ADR registry:

👉 **[Architecture Decision Records (ADRs)](adr/README.md)**

