# Task Plan: Phase 3 - Operational Flags & Profile Schema Alignment

## Goal
Implement the 3-tier operational flags architecture (`docs/design/bisync_operational_flags.md`), flag inheritance (`Execution Override > Profile Override > Global Default > Hardcoded Fallback`), POSIX shell argument sanitizer with prohibited flag security blocklist, and profile CLI argument builder.

## Tasks

- [x] Task 1: Operational Flags DTOs & Global Defaults
  - **Scope**: `internal/config/dto.go`, `internal/config/dto_test.go`, `internal/profiles/model.go`
  - **Verification**: `go test -count=1 ./internal/config/... ./internal/profiles/...`

- [x] Task 2: Custom Flags Validator & Shell Tokenizer
  - **Scope**: `internal/profiles/validator.go`, `internal/profiles/validator_test.go`
  - **Verification**: `go test -count=1 ./internal/profiles/... -run "Test(SplitCommandLine|ValidateCustomFlags)"`

- [x] Task 3: Flag Inheritance & CLI Argument Builder
  - **Scope**: `internal/profiles/builder.go`, `internal/profiles/builder_test.go`
  - **Verification**: `go test -count=1 ./internal/profiles/... -run "Test(ResolveFlags|BuildCLIArgs)"`

- [x] Task 4: App Wiring & Server Integration
  - **Scope**: `internal/app/container.go`, `internal/server/handlers_profiles.go`, `internal/server/handlers_config.go`
  - **Verification**: `go test -count=1 ./internal/app/... ./internal/server/...`

- [x] Task 5: Full Quality Gate & Pre-Flight Verification
  - **Scope**: Full repository
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-10-27: Completed Task 5 (Full Quality Gate & Pre-Flight Verification). Executed all 6 automated quality stages via `./scripts/check.sh all` (formatting, shell check, go vet, lint, tests with coverage, and loc limits). All checks passed clean.
- 2026-10-27: Completed Task 4 (App Wiring & Server Integration). Integrated BuildCLIArgs and 3-tier flag hierarchy into DefaultProfileExecutor in internal/app/container.go, added custom flag validation on profile create/update, and returned resolved_flags and global_defaults in internal/server/handlers_profiles.go with full test coverage.
- 2026-10-27: Completed Task 3 (Flag Inheritance & CLI Argument Builder). Implemented ResolveFlags and BuildCLIArgs in internal/profiles/builder.go executing the full 3-tier hierarchy (Execution Override > Profile Override > Global Default > Hardcoded Fallback) with complete unit test coverage.
- 2026-10-27: Completed Task 2 (Custom Flags Validator & Shell Tokenizer). Implemented POSIX shell argument tokenizer and prohibited flag blocklist validator in internal/profiles/validator.go with complete test coverage for quotes, spaces, equals syntax, and security rejections.
- 2026-10-27: Completed Task 1 (Operational Flags DTOs & Global Defaults). Implemented GlobalDefaults in internal/config/dto.go, integrated GlobalDefaults into runtime Config, and defined ProfileFlags, ResolvedFlags, and ExecutionOverrides in internal/profiles/model.go with complete JSON serialization tests.
- 2026-10-27: Initialized Phase 3 Task Plan from implementation_plan.md.




