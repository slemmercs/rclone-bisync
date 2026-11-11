# Task Plan: Refine FullRemoteTarget, Fix Static Analysis & Deep Coverage Threshold

## Goal
Resolve GitHub Actions nightly and static analysis failures by refining `FullRemoteTarget()` for local paths, updating `go.mod` to Go 1.26.11 to fix standard library vulnerability `GO-2026-6859`, and boosting repository test coverage past the 85.0% threshold under race detection.

## Tasks

- [x] Task 1: Refine `FullRemoteTarget()` for Local Paths & Add Unit Tests
  - **Scope**: `internal/profiles/model.go`, `internal/profiles/model_test.go`, `internal/profiles/store_test.go`
  - **Verification**: `go test -count=1 -v ./internal/profiles/...`

- [ ] Task 2: Bump `go.mod` to Go 1.26.11 to Fix Static Analysis Vulnerability
  - **Scope**: `go.mod`, `go.sum`
  - **Verification**: `./scripts/check.sh lint`

- [ ] Task 3: Elevate Test Coverage Past 85.0% Gate
  - **Scope**: `internal/server/handlers_profiles_test.go`, `internal/server/handlers_telemetry_test.go`, `internal/mcp/server_test.go`, `internal/profiles/model_test.go`
  - **Verification**: `MIN_COVERAGE=85.0 RUN_RACE=true ./scripts/check.sh test`

- [ ] Task 4: Run Full Quality Gate & Final Validation
  - **Scope**: Full repository
  - **Verification**: `MIN_COVERAGE=85.0 RUN_RACE=true ./scripts/check.sh all`

## Status Updates

- 2026-11-11: Initialized plan from implementation_plan.md.
- 2026-11-11: Completed Task 1 - Refined FullRemoteTarget() to preserve raw filesystem paths when remote is 'local' or empty, and verified with internal/profiles test suite.

