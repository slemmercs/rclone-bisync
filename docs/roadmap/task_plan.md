
# Task Plan: OAuth Connection Manager

## Goal
Implement a centralized OAuth Connection Manager for Bisync Hub allowing users to monitor authentication health, detect and execute Google Drive shared client ID migrations, trigger desktop-integrated OAuth re-authentication, validate remote connectivity, and maintain an audit log directly from the web dashboard.

## Tasks
- [x] Task 1: Core Domain Models & Rclone RC Client Extensions
  - **Scope**: `internal/models/auth.go`, `internal/core/client/client.go`, `internal/core/client/client_test.go`
  - **Verification**: `go test -count=1 ./internal/core/client -run TestOAuthClient`
- [x] Task 2: Remote Auth Domain Service (`internal/remoteauth`)
  - **Scope**: `internal/remoteauth/` (service, health, reconnect, migration, validator, audit)
  - **Verification**: `go test -count=1 ./internal/remoteauth/...`
- [x] Task 3: Server API Endpoints & Wiring
  - **Scope**: `internal/server/server.go`, `internal/server/server_test.go`, `main.go`
  - **Verification**: `go test -count=1 ./internal/server -run TestAuthEndpoints`
- [x] Task 4: Dashboard UI & Google Drive Migration Wizard
  - **Scope**: `dashboard/frontend/js/remotes.js`, `dashboard/frontend/index.html`
  - **Verification**: `./scripts/check.sh`
- [x] Task 5: Pre-Flight Verification & Documentation Update
  - **Scope**: `docs/design/oauth-connection-manager.md`, `docs/roadmap/backlog.md`, `README.md`
  - **Verification**: `./scripts/check.sh` and `go build -o bin/rclone-bisync main.go`
- [x] Task 6: Validate Button Reactivity Fix & Option 2 Status Bar Redesign
  - **Scope**: `dashboard/frontend/js/remotes.js`, `dashboard/frontend/index.html`, `dashboard/frontend/app.js`
  - **Verification**: `./scripts/check.sh` and browser state verification
- [x] Task 7: Disentangle Passive Health Checks from Audit Log & Enhance Visual Validation Feedback
  - **Scope**: `internal/remoteauth/validator.go`, `internal/remoteauth/health.go`, `internal/remoteauth/service_test.go`, `dashboard/frontend/js/remotes.js`, `dashboard/frontend/index.html`
  - **Verification**: `./scripts/check.sh`, live unit tests, and live daemon verification
- [x] Task 8: Implement Active 'Check Auth & Quotas' Action with Dedicated Audit Logging
  - **Scope**: `internal/remoteauth/health.go`, `internal/server/server.go`, `internal/server/server_test.go`, `dashboard/frontend/js/remotes.js`, `dashboard/frontend/index.html`
  - **Verification**: `./scripts/check.sh`, live server tests, and live daemon verification

## Status Updates
- 2026-10-22: Initialized plan on branch `feat/oauth-connection-manager`.
- 2026-10-22: Completed Task 1 (Introduced `internal/models/auth.go` DTOs, added `GetRemoteAuthConfig`, `UpdateRemoteConfig`, `OAuthStatus`, and `OAuthStop` to `RcloneClient` with leak-free unit tests).
- 2026-10-22: Completed Task 2 (Created `internal/remoteauth` domain service covering auth health monitoring, shared Google Drive client ID migration, desktop-integrated OAuth reconnect runner, connectivity validator, and ring-buffer audit logger with 85.9% test coverage).
- 2026-10-22: Completed Task 3 (Wired `RemoteAuthService` into server routes with CSRF and secret leak validation, added endpoints for auth health, reconnect, credentials update, validate, audit, and oauth-stop with 85.3% coverage).
- 2026-10-22: Completed Task 4 (Implemented Dashboard UI with Authentication Health overview bar, remote card status badges, shared client ID deprecation alert banners, Google Drive Migration Wizard modal, Reconnect status modal, and OAuth Audit Trail modal).
- 2026-10-22: Completed Task 5 (Updated design documentation, backlog, and README; enhanced reconnect with `--auto-confirm`, localized audit timestamps, added persistent 'Connected ✓' feedback, and verified 100% clean check suite).
- 2026-10-22: Completed Task 6 (Fixed Validate button reactivity via immutable Alpine object updates, auto-refreshed audit modal on validation, implemented Option 2 Compact Status Bar Strip, generalized App Credentials modal for all OAuth providers, and bumped cache-busters to v11).
- 2026-10-22: Completed Task 7 (Separated passive background health probes from user-initiated audit logging, eliminated spurious duplicate audit entries on dashboard reload, isolated `showToast` in `RemotesModule`, guaranteed a minimum 600ms visual spinner duration, extended persistent "Connected ✓" pill to 4000ms, added explicit button click isolation, and deployed live binary).
- 2026-10-22: Completed Task 8 (Added `POST /api/remotes/auth/check` endpoint and `checkAuthAndQuotas()` frontend handler, ensuring explicit user clicks on "Check Auth & Quotas" actively record validation audit records, refresh open audit modals, while passive background polling remains completely clean).








