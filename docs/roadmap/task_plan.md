# Task Plan: Refine FullRemoteTarget, Fix Static Analysis & Deep Coverage Threshold

## Goal
Resolve GitHub Actions nightly and static analysis failures by refining `FullRemoteTarget()` for local paths, updating `go.mod` to Go 1.26.11 to fix standard library vulnerability `GO-2026-6859`, and boosting repository test coverage past the 85.0% threshold under race detection.

## Tasks

- [x] Task 1: Refine `FullRemoteTarget()` for Local Paths & Add Unit Tests
  - **Scope**: `internal/profiles/model.go`, `internal/profiles/model_test.go`, `internal/profiles/store_test.go`
  - **Verification**: `go test -count=1 -v ./internal/profiles/...`

- [x] Task 2: Bump `go.mod` to Go 1.26.11 to Fix Static Analysis Vulnerability
  - **Scope**: `go.mod`, `go.sum`
  - **Verification**: `./scripts/check.sh lint`

- [x] Task 3: Elevate Test Coverage Past 85.0% Gate
  - **Scope**: `internal/server/handlers_profiles_test.go`, `internal/server/handlers_telemetry_test.go`, `internal/mcp/server_test.go`, `internal/profiles/model_test.go`
  - **Verification**: `MIN_COVERAGE=85.0 RUN_RACE=true ./scripts/check.sh test`

- [x] Task 4: Run Full Quality Gate & Final Validation
  - **Scope**: Full repository
  - **Verification**: `MIN_COVERAGE=85.0 RUN_RACE=true ./scripts/check.sh all`

## Status Updates

- 2026-11-11: Initialized plan from implementation_plan.md.
- 2026-11-11: Completed Task 1 - Refined FullRemoteTarget() to preserve raw filesystem paths when remote is 'local' or empty, and verified with internal/profiles test suite.
- 2026-11-11: Completed Task 2 - Bumped go.mod to Go 1.26.11, upgraded staticcheck to latest, and verified clean static analysis & security check.
- 2026-11-11: Completed Task 3 - Added comprehensive unit tests across internal/server, internal/mcp, internal/profiles, and internal/app, boosting test coverage to 85.1% under race detection.
- 2026-11-11: Completed Task 4 - Formatted Go files and ran full quality gate suite with MIN_COVERAGE=85.0 and RUN_RACE=true (7/7 checks passed). All tasks completed!

