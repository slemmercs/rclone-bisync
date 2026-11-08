# Task Plan: Tier 4 Rclone Sandbox UI Journey E2E Suite

## Goal
Implement and verify the full browser-driven user journey across the File Explorer Workspace using a real ephemeral `rclone rcd` sandbox and live local filesystem folders.

## Tasks

- [x] Task 1: Sandbox Fixture & Harness Construction
  - **Scope**: `tests/e2e/sandbox_ui_journey_test.go` (scaffolding: ephemeral `rclone rcd` with local remote `sandrem:`, directory tree seeding, server initialization)
  - **Verification**: `go test -v -count=1 ./tests/e2e -run "TestDashboard_SandboxUIJourney/HarnessSetup"`

- [x] Task 2: Real Dual-Pane Exploration & Navigation Journey
  - **Scope**: `tests/e2e/sandbox_ui_journey_test.go` (navigate to Explorer, verify left pane lists `localDir` items, right pane lists `sandrem:` items via live RC)
  - **Verification**: `go test -v -count=1 ./tests/e2e -run "TestDashboard_SandboxUIJourney/RealExploration"`

- [x] Task 3: Real File Move/Rename via UI Modal
  - **Scope**: `tests/e2e/sandbox_ui_journey_test.go` (trigger move modal in browser, submit rename, assert file renamed physically on disk)
  - **Verification**: `go test -v -count=1 ./tests/e2e -run "TestDashboard_SandboxUIJourney/RealMoveRename"`

- [x] Task 4: Real Directory Purge & Canary Safeguard UI Journey
  - **Scope**: `tests/e2e/sandbox_ui_journey_test.go` (test directory purge deleting real disk folder, test canary directory move/purge blocked by UI safeguards)
  - **Verification**: `go test -v -count=1 ./tests/e2e -run "TestDashboard_SandboxUIJourney/RealPurgeAndCanary"`

- [ ] Task 5: Upload Dock Staging & Flush Journey
  - **Scope**: `tests/e2e/sandbox_ui_journey_test.go` (stage upload in dock, assert pending count in UI badge, flush dock, verify file appears on disk)
  - **Verification**: `go test -v -count=1 ./tests/e2e -run "TestDashboard_SandboxUIJourney/RealUploadDock"`

- [ ] Task 6: Quality Gates & Comprehensive E2E Verification
  - **Scope**: Code formatting, staticcheck, LOC audit (<400 LOC), full test suite
  - **Verification**: `./scripts/check.sh all && go test -v -count=1 ./tests/e2e/...`

## Status Updates

- 2026-11-08: Initialized Sandbox UI Journey plan from approved implementation plan.
