# Task Plan: Quality Engine Hardening, E2E Stability & Showcase Rendering Fixes

## Goal
Harden the Go CLI check runner (`cmd/check`), eliminate E2E smoke test race condition in CI runners, and fix showcase site navbar title and hero installation rendering regressions.

## Tasks

- [x] Task 1: Fix Showcase Site Title & Code Block (`site/`)
  - **Scope**: `site/_includes/title.html`, `site/_sass/custom/custom.scss`, `site/index.md`
  - **Verification**: Inspect generated site HTML or files, verify classes and structure

- [x] Task 2: Implement Flexible Flag Parsing & Verbose Logging (`cmd/check`)
  - **Scope**: `cmd/check/main.go`, `cmd/check/checks.go`, `cmd/check/main_test.go`
  - **Verification**: `go test -v -count=1 ./cmd/check`, `./scripts/check.sh fmt -v`, `./scripts/check.sh -v fmt`

- [x] Task 3: Stabilize Headless Chrome Smoke Test (`tests/e2e`)
  - **Scope**: `tests/e2e/dashboard_smoke_test.go`
  - **Verification**: `go test -v -count=1 ./tests/e2e`

- [x] Task 4: Run Pre-Flight Quality Gate Check
  - **Scope**: Full repository
  - **Verification**: `./scripts/check.sh all -v`

## Status Updates

- 2026-10-25: Initialized plan from implementation_plan.md on branch `feat/alpine-boolean-lint-and-e2e`.
- 2026-10-25: Completed Task 1 (Showcase site title & hero styling), Task 2 (CLI flexible flag partitioning & verbose logging), Task 3 (E2E headless Chrome dynamic polling stabilization), and Task 4 (Full quality gate verification 6/6 passed).











