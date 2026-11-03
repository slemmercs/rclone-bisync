# Task Plan: RC Argument Compilation, Rename Tracking Flags & Review Session Cache Fix

## Goal
Fix Rclone RC argument compilation for profile flags (`--track-renames`, `--track-renames-strategy`, and `--max-delete`), invalidate stale in-memory review session cache, fix browser DOM select parsing in the flags builder UI, and add regression tests.

## Tasks

- [x] Task 1: RC Argument Compiler Fixes & Unit Tests
  - **Scope**: `internal/core/compiler/compiler.go`, `internal/core/compiler/compiler_test.go`
  - **Verification**: `go test -v -count=1 ./internal/core/compiler/...`

- [x] Task 2: Review Service In-Memory Cache Invalidation & Unit Tests
  - **Scope**: `internal/services/review_execution.go`, `internal/services/review_service_test.go`
  - **Verification**: `go test -v -count=1 ./internal/services/... -run "TestReview"`

- [x] Task 3: Flags Builder Select DOM Fix, Curated Catalog Expansion & E2E Tests
  - **Scope**: `dashboard/frontend/partials/modals/modal_provider_options.html`, `dashboard/frontend/js/flags_builder.js`, `tests/e2e/dashboard_smoke_test.go`
  - **Verification**: `go test -v -count=1 ./tests/e2e/...` and `./scripts/check.sh all`

- [ ] Task 4: Daemon Deployment & Live Sync Verification
  - **Scope**: Build binary, update user profile flags if needed, restart dashboard service, live dry-run sync
  - **Verification**: Verify clean sync execution with `rclone rc` and dashboard API

## Status Updates

- 2026-11-03: Initialized plan from implementation_plan.md on branch fix/rc-flags-compilation-and-review-cache.












