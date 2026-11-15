# Implementation Plan: Frontend Architecture & Governance Modernization

## Proposed Changes

Modernize Bisync Hub's engineering governance and architectural rules by dropping three outdated dogmas:
1. **Drop "Zero Build Tools Ever" dogma**: Replace with **"Zero Runtime Dependencies & Single Binary Invariant"**. The deployed binary remains a single standalone executable using `//go:embed`. Build-time frontend compilation (Vite / TypeScript / pre-compiled Tailwind) produces a committed `dist/` directory, allowing Go developers to compile and test with zero Node.js/npm dependencies while frontend components benefit from type safety, modularity, and pre-purged CSS.
2. **Drop "Flat 400 LOC Ceiling on Frontend"**: Replace with **"Go Package Guidelines (~400 LOC) + Frontend Cohesive Components"**. Go backend domain services maintain the ~400–500 LOC ceiling to prevent God Objects. Frontend UI components are governed by single responsibility and component cohesion (e.g. Single-File Components) rather than arbitrary line-count fragmentation that previously caused temporal global coupling across 40 disjointed files.
3. **Drop "Zero-Build Alpine.js Shared Global Object"**: Replace with **"Component Scoping & Typed API Contracts"**. Mandate component-level state isolation and typed stores (e.g., Pinia / scoped Alpine stores) backed by TypeScript interfaces aligned with Go model contracts in `internal/models/`.

### Affected Components & Packages

- `.clinerules`: Update LOC scanning rules, toolchain standards, frontend conventions, and architectural audit protocol.
- `cmd/check/checks.go`: Add `dist` to `skipDirs`, focus `checkLOC` on Go backend source files (excluding `_test.go`), and update frontend check capabilities.
- `cmd/check/main_test.go`: Update unit tests for `skipDirs` and `checkLOC`.
- `docs/architecture/ARCHITECTURE_PRINCIPLES.md`: Update Core System Invariants (Zero Runtime Dependencies & Embedded Frontend).
- `docs/architecture/ENGINEERING_PRINCIPLES.md`: Clarify Principle 3 (Small Cohesive Components) to differentiate Go service boundaries from frontend component cohesion.
- `docs/architecture/adr/ADR-010-Zero-Build-Design-System-And-Responsive-Telemetry.md`: Mark as superseded by ADR-011.
- `docs/architecture/adr/ADR-011-Modern-Frontend-Component-Architecture-And-Precompiled-Single-Binary.md`: Author new ADR formalizing the modernized frontend architecture and pre-compiled single binary embeddings.

### Technical Strategy & Data Flow

- Backend Go developers retain zero-toolchain-friction: `go build ./...` and `go test ./...` continue to work with zero external dependencies.
- Static checking via `./scripts/check.sh` skips generated `dist/` artifacts.
- The 400 LOC rule in `cmd/check/checks.go` now exclusively targets `.go` source files where package boundaries and static compilation guarantee cross-file safety.

### Safety & Error Handling

- Core single-binary invariant (`//go:embed`) and local-first architecture (ADR-005) remain 100% intact.
- Canary protection, POSIX flock locking, and path confinement rules remain strictly enforced.

## Verification Plan

- `go test -count=1 ./cmd/check/...`
- `./scripts/check.sh loc`
- `./scripts/check.sh fmt`
- `./scripts/check.sh all`

