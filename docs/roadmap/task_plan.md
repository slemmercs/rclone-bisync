# Task Plan: Native Go CLI Quality Gate (`cmd/check`) with Hybrid Wrapper

## Goal
Implement a high-performance Go CLI tool in `cmd/check/main.go` that replaces bash quality scripts, provides robust HTML token scanning and parallel auditing, and wire it to `scripts/check.sh` for full backwards compatibility.

## Tasks

- [x] Task 1: Implement robust HTML template scanner and LOC auditor in `cmd/check/main.go`
  - **Scope**: `cmd/check/main.go`, HTML tokenizer, LOC scanner
  - **Verification**: `go run ./cmd/check templates` && `go run ./cmd/check loc 400`

- [x] Task 2: Implement full suite of checks in `cmd/check/main.go` (`fmt`, `lint`, `test`, `build`, `all`)
  - **Scope**: `cmd/check/main.go`
  - **Verification**: `go run ./cmd/check all`

- [x] Task 3: Add unit tests for `cmd/check` in `cmd/check/main_test.go`
  - **Scope**: `cmd/check/main_test.go`
  - **Verification**: `go test -count=1 ./cmd/check/...`

- [x] Task 4: Refactor `scripts/check.sh` to delegate to `cmd/check` and verify CI compatibility
  - **Scope**: `scripts/check.sh`
  - **Verification**: `./scripts/check.sh all` and `./scripts/check.sh loc 400`

## Status Updates

- 2026-10-25: Initialized plan for Go CLI migration (`cmd/check`) on branch `feat/alpine-boolean-lint-and-e2e`.
- 2026-10-25: Completed Task 1: Implemented custom HTML attribute tokenizer (`html_scanner.go`) and LOC auditor (`checks.go`).
- 2026-10-25: Completed Task 2: Implemented full suite of checks in `cmd/check` (`fmt`, `lint`, `test`, `build`, `all`) with parallel execution.
- 2026-10-25: Completed Task 3: Added unit tests for HTML scanner, LOC counter, and formatter in `cmd/check/main_test.go` (100% passing in 3ms).
- 2026-10-25: Completed Task 4: Refactored `scripts/check.sh` into a thin delegator forwarding to `cmd/check`, verified full 6-stage suite passes in ~10s.










