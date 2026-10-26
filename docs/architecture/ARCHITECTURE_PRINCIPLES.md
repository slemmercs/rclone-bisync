# Architecture Principles

These principles define the foundational architecture, system invariants, and engineering standards for Bisync Hub.

Any significant change to the platform must align with these principles.

---

## The Four Architectural Pillars

Bisync Hub's technical governance is structured across four specialized specifications:

1. 📋 **[Product Principles](PRODUCT_PRINCIPLES.md)**  
   Defines the user-centric mission: *Review. Decide. Sync.* Prioritizes human decision authority, explainability over black boxes, safety before automation, and profiles as primary user abstractions.

2. ⚙️ **[Engineering Principles](ENGINEERING_PRINCIPLES.md)**  
   Governs code organization, domain service separation (`internal/services/`), sub-400 LOC component budgets, atomic file write patterns (`.tmp.<pid>` + `os.Rename`), POSIX advisory locking (`flock`), and testability by design.

3. 🛡️ **[Security Principles](SECURITY_PRINCIPLES.md)**  
   Mandates local filesystem path confinement jails, canary file protection (`RCLONE_TEST`), recursive credential sanitization (`security.RedactSecrets`), safe failure modes, and tamper-evident audit trails.

4. 🔄 **[Contributor Workflow](CONTRIBUTOR_WORKFLOW.md)**  
   Prescribes the standard development lifecycle: design-first planning, small verifiable tasks, continuous pre-flight verification via `./scripts/check.sh all`, 3-try circuit breakers, and checkpoint commits.

---

## Core System Invariants

### 1. Single Binary First
Bisync Hub ships as a single, self-contained executable binary. It requires zero external language runtimes (no Node.js, Python, or Ruby) and zero external background daemons to operate.

### 2. Embedded Frontend
Web dashboard assets (HTML, Alpine.js, Tailwind CSS, Chart.js) are embedded directly into the Go executable via standard `//go:embed`. Users never need separate frontend server deployments.

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

