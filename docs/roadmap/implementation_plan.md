# Implementation Plan: GitHub Issues Realignment & Multi-Profile Review Isolation Bugfix

## Proposed Changes

Address the root cause where pending sync review sessions block other/freshly-created remotes across profiles, reconcile the GitHub epic issues (#98, #99, #100, #101), and prepare the branch for clean merge to `main`.

### 1. GitHub Issues Realignment
- Close Issue #98 (`[FEAT] Hasher Backend Integration`) as completed in `refactor/frontend-component-decomposition`.
- Update Issue #99 to explicitly target the Official Multi-Arch Container Stack (`Dockerfile`, `docker-compose.yml`, GHCR, PUID/PGID).
- Update Issue #100: Reconcile completed `<DynamicProviderForm>` (in-tree), focus on Control-Plane Safety Gates (Pre-Sync `--resync` dry-run diff gate, post-sync deep integrity audits via async `operations/check`, and profile filter simulator).
- Create/Split Issue for Remote Quota Bars (`operations/about` & `operations/cleanup`) as a quick-win, leaving Issue #101 focused on post-v1.0.0 VFS Mount Supervision & Ad-Hoc Transfers.

### 2. Multi-Profile Review Isolation Bugfix
- **Root Cause**: `ReviewService`, `TelemetryService`, and `SyncActionService` manage a single global active review session (`active_review_session.json` and `pending_deletions.json`). When profile A has pending review candidates, `HasBlockingReview()` returns `true` for all profiles, blocking sync on newly created or unaffected profiles (Profile B).
- **Domain Models (`internal/models/review.go`)**:
  - Add `ProfileID string json:"profile_id"` to `ReviewSession` and candidate records.
- **Service Layer (`internal/services/review_service.go`, `sync_actions.go`)**:
  - Update `ReviewService` to track and query sessions scoped by `profileID`.
  - `HasBlockingReview(profileID ...string) bool`: Check only whether the requested `profileID` has an active, unresolved review session.
  - `GetActiveSession(ctx context.Context, profileID ...string) (*models.ReviewSession, error)`: Filter and discover per-profile review sessions.
- **Server Handlers (`internal/server/handlers_review.go`)**:
  - Add optional `profile_id` query parameter to `GET /api/review/session`.
- **Telemetry & Frontend Alignment**:
  - In `telemetry_service.go`, include active review profile IDs.
  - In `dashboard/frontend/src/stores/telemetry.ts` and `review.ts`, ensure `fsmStore.transitionState` to `REVIEW_REQUIRED` only triggers when the pending review belongs to the currently active profile.

---

### Affected Components & Packages

- GitHub Issues: #98, #99, #100, #101 (via `gh` CLI)
- `internal/models/review.go`
- `internal/services/review_service.go`
- `internal/services/sync_actions.go`
- `internal/server/handlers_review.go`
- `internal/services/telemetry_service.go`
- `dashboard/frontend/src/stores/telemetry.ts`
- `dashboard/frontend/src/stores/review.ts`

---

### Safety & Error Handling

- **Backward Compatibility**: If a legacy review session without `profile_id` exists on disk, gracefully assign it to the default or active profile rather than crashing.
- **Fail-Safe**: If `profileID` is passed as empty, fall back to global check or active profile check.
- **Path Confinement**: Candidate review sessions must remain strictly confined to the app data directory.

---

## Verification Plan

- `gh issue list` verification of updated issues
- Unit tests: `go test -count=1 ./internal/services -run TestReview`
- Server tests: `go test -count=1 ./internal/server -run TestReview`
- Frontend type check: `npm --prefix dashboard/frontend run type-check`
- Full pre-flight verification: `./scripts/check.sh all`


