# Task Plan: Phase 2 - Core Refactoring & Subsystem Consolidation

## Goal
Centralize application lifecycle and dependency wiring via a DI container, consolidate the review domain under `internal/engine/review/`, and implement a deep defense-in-depth secret-redaction masking engine for logging and file tailing.

## Tasks

- [x] Task 1: Central DI Container (`internal/app/container.go`)
  - **Scope**: `internal/app/container.go`, `internal/app/container_test.go`, `main.go`, `internal/server/server.go`
  - **Verification**: `go test -count=1 ./internal/app/...` and `go test -count=1 ./internal/server/...` and `./scripts/check.sh all`

- [x] Task 2: Subsystem Consolidation (`internal/review/` -> `internal/engine/review/`)
  - **Scope**: `internal/review/` -> `internal/engine/review/`, `internal/services/review_service.go`, `internal/services/review_recommendations.go`, `internal/app/container.go`
  - **Verification**: `go test -count=1 ./internal/engine/review/...` and `go test -count=1 ./internal/services/...` and `./scripts/check.sh all`

- [x] Task 3: Stream/Line Secret Redactor (`internal/core/logger/redact.go`)
  - **Scope**: `internal/core/logger/redact.go`, `internal/core/logger/redact_test.go`, `internal/core/logger/logger.go`, `internal/services/filetail.go`
  - **Verification**: `go test -count=1 ./internal/core/logger/...` and `go test -count=1 ./internal/services/...` and `./scripts/check.sh all`

- [ ] Task 4: Full Quality Gate & Pre-Flight Verification
  - **Scope**: Full repository
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-10-27: Completed Task 3 (Stream/Line Secret Redactor). Implemented regex-based stream/line masking in internal/core/logger/redact.go, wrapped logger sinks with MaskingWriter, and integrated secret masking into TailLines and TailJSONLog with unit tests in logger and services packages. All 6 quality gate stages passed clean.
- 2026-10-27: Completed Task 2 (Subsystem Consolidation). Relocated internal/review/ into internal/engine/review/ under engine bounded domain, updating service imports and documentation references. All 6 quality gate stages passed clean.
- 2026-10-27: Completed Task 1 (Central DI Container). Extracted manual service wiring and scheduler orchestration out of main.go and internal/server/server.go into internal/app/container.go. All 6 quality gate stages passed clean.
- 2026-10-27: Initialized Phase 2 Task Plan from implementation_plan.md.



