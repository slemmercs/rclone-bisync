# Task Plan: CodeQL Path Injection Remediation, PR #80 Merge & Release v1.0.0-beta.30

## Goal
Remediate the two high-severity CodeQL path-traversal alerts (`go/path-injection`) in `internal/services/explorer_move.go`, verify the security and functional test gates, update and merge Pull Request #80, and release version `v1.0.0-beta.30`.

## Tasks

- [x] Task 1: Sanitize Local Path Resolution in `internal/services/explorer_move.go` & Add Unit Tests
  - **Scope**: `internal/services/explorer_move.go`, `internal/services/explorer_ops_test.go`
  - **Verification**: `go test -count=1 ./internal/services/... -run TestExplorerOps_MoveItem`

- [x] Task 2: Quality Gate Verification & Pre-Commit Validation
  - **Scope**: Full repository check suite and end-to-end tests
  - **Verification**: `./scripts/check.sh all && go test -count=1 ./tests/e2e/...`

- [x] Task 3: Commit, Push & Verify GitHub CodeQL Run on PR #80
  - **Scope**: Remote tracking branch `feat/milestone-2-sync-staging-workspace`
  - **Verification**: `gh pr checks 80` (CodeQL workflow clean)

- [x] Task 4: Merge PR #80 & Release v1.0.0-beta.30
  - **Scope**: Merge to `main`, create tag and release `v1.0.0-beta.30`
  - **Verification**: `gh pr merge 80 --squash && gh release view v1.0.0-beta.30`

## Status Updates

- 2026-11-09: Initialized task plan from implementation_plan.md.
- 2026-11-09: Task 1 verified: Applied DotDotCheck and PrefixCheck barrier guards directly to req paths and local cleanTargetDst/cleanDstDir sinks in explorer_move.go. Added traversal rejection unit test cases; all tests passed.
- 2026-11-09: Task 2 verified: Formatted explorer_ops_test.go, passed all 7/7 check.sh gates, verified all headless Chrome and journey E2E tests, and confirmed all internal packages satisfy the 400-LOC limit.
- 2026-11-09: Task 3 verified: Committed, pushed to origin/feat/milestone-2-sync-staging-workspace. All 12/12 CI and CodeQL checks passed with zero open security alerts.
- 2026-11-09: Task 4 verified: Merged PR #80 into main, tagged and published pre-release v1.0.0-beta.30 on GitHub.

