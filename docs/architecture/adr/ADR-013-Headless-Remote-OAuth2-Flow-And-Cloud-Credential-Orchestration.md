# ADR-013: Headless Remote OAuth2 Flow & Cloud Credential Orchestration

**Status:** Accepted

**Date:** 2026-11-23

---

## Context

Cloud storage backends supported by Rclone (e.g., Google Drive, Microsoft OneDrive, Dropbox, Box) rely on OAuth 2.0 authorization grants with access and refresh tokens.

In typical CLI usage, establishing or renewing OAuth tokens requires interactive terminal commands (`rclone config reconnect remote:`), which spawn a local loopback web server and launch a desktop web browser. However, this model breaks down in server topologies:
1. **Headless & Container Deployments**: When Bisync Hub runs in Docker or on a headless server, no desktop browser is available, leading to hung processes or failed reconnect attempts.
2. **Expired or Revoked Token Failures**: Access tokens expire, and refresh tokens can be invalidated, causing unattended bisync jobs to fail abruptly with authentication errors.
3. **Shared Client ID Rate Limiting**: Cloud providers impose shared API quotas across default Rclone application client IDs. Users need a reliable mechanism to migrate existing remotes to custom credentials without manual config file surgery.
4. **Credential Exposure Risk**: Unsanitized auth errors or OAuth exchange URLs can expose client secrets or access tokens in daemon logs and API payloads.

An architectural subsystem is required to diagnose authentication health, coordinate OAuth reconnects across desktop and headless environments, facilitate credential migration, and maintain an audit trail.

---

## Decision

Bisync Hub establishes the **OAuth Connection Manager** (`internal/remoteauth/`), providing centralized credential orchestration, health monitoring, and recovery workflows:

### 1. Unified Authentication Service (`internal/remoteauth/service.go`)
- `RemoteAuthService` acts as the single authority for remote authentication state, coordinating between the Rclone Remote Control (RC) API and underlying CLI runners.
- Exposes REST API endpoints (`/api/auth/status`, `/api/auth/validate`, `/api/auth/reconnect`, `/api/auth/migrate`, `/api/auth/events`).
- Employs a 60-second TTL cache for status queries to protect cloud provider rate limits while supporting forced user refreshes.

### 2. Multi-State Health Diagnostics (`internal/remoteauth/health.go`, `validator.go`)
- Each remote's authentication state is classified into explicit statuses (`connected`, `auth_required`, `expiring_soon`, `migration_required`, `error`).
- Token expiration timestamps (`token_expiry`) are monitored continuously.
- Active non-disruptive probing (`probeRemote` via `client.About` or directory listing) verifies token validity against cloud APIs without altering profile run states.

### 3. Desktop & Headless Reconnect Orchestration (`internal/remoteauth/reconnect.go`)
- **Desktop Session Inheritance**: Collects desktop user environment variables (`DBUS_SESSION_BUS_ADDRESS`, `DISPLAY`, `XDG_CURRENT_DESKTOP`) from systemd/user configs to reliably spawn local browsers when available.
- **Concurrent Reconnect Guard**: Reconnect sessions are tracked by remote name and guarded with context cancellation (`activeReconnects map[string]context.CancelFunc`), ensuring that initiating a new flow automatically cleans up orphaned reconnect processes.
- **Headless Fallback**: Emits structured reconnect instructions and URLs, allowing users on remote hosts to authorize via an external browser and complete token exchange.

### 4. Zero-Downtime Credential Migration (`internal/remoteauth/migration.go`)
- Provides guided migration from shared Rclone client IDs to user-owned OAuth application credentials (`client_id` and `client_secret`).
- Modifies remote configurations atomically via Rclone RC (`config/update`) and immediately re-validates connectivity before confirming the migration.

### 5. Sanitized OAuth Audit Ledger (`internal/remoteauth/audit.go`)
- Maintains an in-memory and persisted audit ring buffer of OAuth events (`OAUTH_CONNECTED`, `OAUTH_RECONNECTED`, `OAUTH_MIGRATED`, `OAUTH_FAILED`).
- Strictly sanitizes all audit payloads through `security.RedactSecrets` to ensure tokens and secrets are never committed to disk or transmitted across WebSocket/SSE feeds.

---

## Rationale

1. **Elimination of Terminal Friction**: Users manage cloud credentials directly through the web UI and MCP tools, preventing dangerous manual edits to `rclone.conf`.
2. **Headless Operational Continuity**: Explicit reconnection flows and non-interactive health probes prevent background sync jobs from silently failing due to expired tokens.
3. **Decoupled Architecture**: Authentication logic is decoupled from both the sync execution engine (`internal/engine/`) and web route handlers (`internal/server/`), adhering to ADR-003.

---

## Consequences

### Positive
- Proactive warning notifications before tokens expire prevent sync aborts.
- First-class migration flow allows users to resolve cloud provider rate-limiting without reconfiguring entire profiles.
- Strict secret redaction ensures OAuth tokens and client secrets are never exposed in logs or API responses.
- Active cancellation prevents hung `rclone config reconnect` background processes.

### Negative / Trade-offs
- Headless token exchange still requires user interaction in an external web browser.
- External process execution (`exec.CommandContext`) requires careful environment sanitization and context timeout enforcement.

---

## Compliance & Verification

- `internal/remoteauth/service_test.go` verifies authentication health checks across OAuth, non-OAuth, and broken remotes.
- Reconnection concurrency and cancellation are tested using mock command runners with simulated timeouts.
- Pre-commit checks (`./scripts/check.sh`) verify zero credential leakage and full lint compliance.