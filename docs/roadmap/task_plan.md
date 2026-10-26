# Task Plan: Harden Headless Chrome E2E Smoke Tests for CI

## Goal
Harden `TestDashboard_HeadlessChromeSmoke` against transient CI timeouts and container resource constraints by adding container resilience flags, dynamic CI timeout margins, and diagnostic error output.

## Tasks

- [x] Task 1: Update Allocator Options, CI Timeout & Error Diagnostics in E2E Smoke Test
  - **Scope**: `tests/e2e/dashboard_smoke_test.go`
  - **Verification**: `go test -v -count=1 ./tests/e2e -run TestDashboard_HeadlessChromeSmoke` and `CI=true go test -v -count=1 ./tests/e2e -run TestDashboard_HeadlessChromeSmoke`

- [x] Task 2: Full Quality Gate Verification & Pre-Commit Audit
  - **Scope**: Codebase validation via native check engine
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-10-26: Initialized plan from implementation_plan.md on branch `fix/e2e-headless-chrome-ci-flakiness`.
- 2026-10-26: Completed Task 1 (Added container resilience flags, dynamic CI timeout margin, and console error diagnostics on navigation failure).
- 2026-10-26: Completed Task 2 (Formatted cmd/check/checks.go, full 6/6 quality gate passed in 9.8s).











