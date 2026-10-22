# OAuth Connection Manager

## Status

Planned

## Priority

High

## Goal

Allow users to diagnose, migrate, reconnect, validate, and recover OAuth-based remotes directly from the Bisync Hub dashboard.

The OAuth Connection Manager provides a centralized interface for remote authentication lifecycle management without requiring direct use of Rclone configuration commands.

The feature is responsible for:

- Authentication health monitoring
- OAuth reconnect workflows
- Expired token recovery
- Revoked authorization recovery
- Credential validation
- Provider migration workflows
- Authentication diagnostics
- Secure credential management
- Audit logging

The objective is to make authentication and authorization issues understandable and recoverable without requiring manual edits to the Rclone configuration file.

---

# Background

Many cloud providers supported by Rclone rely on OAuth authentication.

Examples include:

- Google Drive
- OneDrive
- Dropbox
- Box

When authentication problems occur, synchronization can fail due to:

- Expired access tokens
- Revoked authorization grants
- Invalid refresh tokens
- Invalid OAuth credentials
- Provider-side changes
- Configuration errors

Historically, recovery requires command-line interaction:

```bash
rclone config reconnect remote:
```

or direct modification of:

```text
rclone.conf
```

This creates operational challenges:

- Requires terminal access
- Requires knowledge of Rclone internals
- Breaks dashboard-first workflows
- Increases support burden
- Makes troubleshooting difficult

Bisync Hub should provide a guided, discoverable recovery experience.

---

# Google Drive Migration Support

Google Drive represents a special case.

Rclone's shared Google Drive client ID is being retired and users must migrate to personal OAuth credentials to avoid future service disruption.

Users frequently do not know:

- Whether they are using the shared client ID
- Whether migration is required
- How to create OAuth credentials
- How to reconnect after migration

The OAuth Connection Manager should guide users through migration and reauthorization workflows.

---

# User Story

As a Bisync Hub user,

I want to manage the authentication lifecycle of my remotes through the dashboard,

so that I can maintain synchronization without needing command-line recovery procedures.

---

# Scope

## Included

- Authentication health monitoring
- OAuth reconnect workflows
- Credential validation
- Provider migration detection
- Google Drive migration assistance
- Browser authorization flows
- Authentication diagnostics
- Credential storage
- Status monitoring
- Audit logging

---

## Excluded

### Initial Release

- Full remote creation wizard
- Generic configuration editor
- OAuth provider administration
- Multi-user identity management
- Team credential sharing

---

# Authentication States

Every OAuth-capable remote reports one of the following states.

---

## Connected

Remote authentication is healthy.

```text
Connected
```

---

## Connected (Migration Required)

Remote is operational but future action is required.

Example:

```text
Google Drive remote uses shared Rclone Client ID.

Migration recommended.
```

---

## Authorization Required

Remote requires OAuth reauthorization.

```text
Authorization Required
```

---

## Configuration Required

Credential configuration is incomplete.

```text
Client ID Missing
Client Secret Missing
```

---

## Error

Authentication validation failed.

```text
Authentication Error
```

---

# User Experience

## Authentication Center

New dashboard section:

```text
Settings
└── Authentication
```

Authentication Overview:

```text
Authentication Health

Connected:                3
Migration Required:       1
Authorization Required:   1
Errors:                   0

[Manage Authentication]
```

---

## Remote List

Example:

```text
Remote         Provider       Status

gdrive         Google Drive   Migration Required
onedrive       OneDrive       Connected
dropbox        Dropbox        Authorization Required
```

---

## Available Actions

Per remote:

```text
[ Reconnect ]

[ Validate ]

[ View Details ]
```

Google Drive only:

```text
[ Migration Wizard ]
```

---

# Reconnect Workflow

## Step 1

User selects:

```text
Reconnect
```

---

## Step 2

Dashboard shows:

```text
Reconnect Remote

Remote:
gdrive

Provider:
Google Drive

This process will obtain a new authorization token.
```

Actions:

```text
[ Continue ]
[ Cancel ]
```

---

## Step 3

Bisync Hub requests reconnect flow.

Backend initiates:

```text
rclone config reconnect
```

or equivalent supported workflow.

---

## Step 4

User completes authorization through browser.

Example:

```text
accounts.google.com
login.microsoftonline.com
```

---

## Step 5

OAuth callback received.

Display:

```text
Authorization Successful
```

---

## Step 6

Validation performed.

Checks:

- Token exists
- Approval successful
- Remote responds
- Authentication status healthy

Status becomes:

```text
Connected
```

---

# Google Drive Migration Wizard

## Purpose

Assist users in migrating away from the shared Rclone Google Drive Client ID.

---

## Step 1 - Detection

Bisync Hub inspects remote configuration.

Example:

```ini
type = drive
client_id =
client_secret =
```

Result:

```text
Shared Client ID Detected
```

---

## Step 2 - Explain Requirement

Display:

```text
Migration Recommended

This remote is currently using Rclone's
shared Google OAuth application.

To avoid future service disruption,
configure your own Google OAuth credentials.
```

---

## Step 3 - Guided Instructions

Provide:

```text
1. Create Google Cloud Project
2. Enable Google Drive API
3. Create OAuth Credentials
4. Obtain Client ID
5. Obtain Client Secret
```

Include links to official documentation.

---

## Step 4 - Credential Entry

User enters:

```text
Client ID
Client Secret
```

through the dashboard.

---

## Step 5 - Secure Storage

Credentials are securely stored.

Validation occurs immediately.

---

## Step 6 - Reauthorization

Browser authorization starts.

User grants access.

---

## Step 7 - Validation

System validates:

```text
OAuth Token Exists
Remote Accessible
Credentials Valid
```

Status becomes:

```text
Connected
```

---

# Credential Storage Security Definition

## Objective

Protect OAuth credentials and prevent accidental disclosure.

---

## Protected Secrets

The following values are considered secrets:

```text
Access Tokens
Refresh Tokens
Client Secrets
Authorization Codes
Session Tokens
```

The following are considered sensitive:

```text
Client ID
OAuth Scopes
Provider Metadata
```

---

## Storage Principles

### Storage at Rest

Credentials must never be stored in plaintext.

Preferred approaches:

#### Linux Desktop

```text
libsecret
GNOME Keyring
KWallet
```

#### Cross Platform (Future)

```text
OS Keychain Integration
```

Examples:

```text
Windows Credential Manager
macOS Keychain
Linux Secret Service
```

---

## Configuration Files

Rclone configuration may continue storing:

```text
client_id
```

within configuration files.

However:

```text
client_secret
access_token
refresh_token
```

should, where practical, be moved into secure credential storage.

---

## Dashboard Behavior

The dashboard must never display:

```text
access_token
refresh_token
client_secret
```

Display example:

```text
Google Drive

Client ID:
1234567890-abc...xyz.apps.googleusercontent.com

Client Secret:
**************
```

---

## Logging Requirements

The following must never appear in logs:

```text
OAuth Tokens
Refresh Tokens
Client Secrets
Authorization Codes
```

Example:

```text
OAuth reconnect successful
```

Allowed.

Example:

```text
refresh_token=abc123...
```

Forbidden.

---

## Diagnostic Exports

Diagnostic bundles must automatically redact:

```text
Tokens
Secrets
Passwords
Cookies
API Keys
```

before export.

---

## Memory Handling

Secret values should:

- Exist only as long as necessary
- Not be cached permanently in memory
- Be excluded from telemetry exports

---

## Backup Policy

Credential stores must never be exported automatically.

Backups should:

```text
Exclude OAuth secrets by default.
```

User must explicitly opt-in before exporting secrets.

---

## Security Levels

### Level 1 (MVP)

- Secret redaction
- Hidden UI values
- Protected exports
- Limited memory exposure

### Level 2

- Secret Service integration
- Keychain integration
- Credential rotation support

### Level 3

- Hardware-backed secure storage
- TPM integration
- FIDO-backed secret protection

---

# Dashboard Integration

## Authentication Widget

Dashboard card:

```text
Authentication Health

Connected:                3
Migration Required:       1
Authorization Required:   1

[ Manage Authentication ]
```

---

## Sync Failure Integration

Authentication-related failures should surface directly.

Example:

```text
Synchronization Failed

Reason:
Authorization Required

Remote:
gdrive

[ Reconnect ]
```

---

# Architecture

## New Package

```text
internal/remoteauth/
```

Suggested structure:

```text
internal/remoteauth/
├── service.go
├── health.go
├── reconnect.go
├── migration.go
├── validator.go
├── callback.go
├── credential_store.go
├── audit.go
└── dto.go
```

---

## Responsibilities

### Health Checks

Determine:

```text
Connected
Migration Required
Authorization Required
Error
```

---

### Credential Validation

Verify:

```text
Client ID Present
Client Secret Present
Token Valid
Remote Reachable
```

---

### Migration Workflows

Detect:

```text
Shared Client ID Usage
```

Launch:

```text
Migration Wizard
```

---

### Secure Storage

Manage:

```text
Client Secrets
Tokens
Refresh Tokens
```

---

### Audit Logging

Record:

```json
{
  "event":"oauth_reconnect",
  "remote":"gdrive",
  "status":"success"
}
```

---

# Audit Events

## Successful Reconnect

```json
{
  "event":"oauth_reconnect",
  "remote":"gdrive",
  "status":"success",
  "timestamp":"2026-10-22T10:15:00Z"
}
```

---

## Failed Reconnect

```json
{
  "event":"oauth_reconnect",
  "remote":"gdrive",
  "status":"failed",
  "reason":"access_denied"
}
```

---

## Migration Completed

```json
{
  "event":"oauth_migration",
  "remote":"gdrive",
  "status":"completed"
}
```

---

# Error Handling

## Authorization Cancelled

```text
Authorization Cancelled

No changes were made.
```

---

## Provider Error

```text
Authorization Failed

The provider returned an error.

Please retry or review your credentials.
```

---

## Validation Failure

```text
Credentials Accepted

Remote Validation Failed
```

Provide troubleshooting guidance.

---

## Timeout

```text
Authorization Timed Out

Please retry the operation.
```

---

# MVP Acceptance Criteria

- Authentication health is visible
- OAuth failures are detected
- Users can reconnect remotes from the UI
- Google Drive migration detection exists
- Users can enter custom Client ID and Client Secret
- Validation occurs automatically
- Authentication status refreshes without restart
- Audit events are recorded
- Secrets never appear in logs
- Secrets never appear in diagnostic bundles
- Secrets are masked in the UI

---

# Future Enhancements

## V2

- Secret Service integration
- OAuth expiration warnings
- Reconnect history
- Multi-profile support
- Backup and restore of metadata

---

## V3

- Full remote creation wizard
- Provider-specific diagnostics
- Credential rotation workflows
- Secure keychain integration
- Hardware-backed secret storage

---

# Success Criteria

A user can:

1. Detect authentication problems.
2. Determine if migration is required.
3. Configure personal OAuth credentials.
4. Reauthorize remotes.
5. Validate authentication status.
6. Resume synchronization.

without editing Rclone configuration files or executing Rclone authentication commands manually.

The OAuth Connection Manager becomes the central authentication and credential-management component of Bisync Hub.
