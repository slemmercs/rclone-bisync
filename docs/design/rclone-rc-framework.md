# Rclone RC Integration Framework

| Metadata | Details |
|---|---|
| **Status** | `Shipped` (v1.0.0-beta.12) |
| **Domain** | Remote Control Client & Filesystem Primitives |
| **Governing ADR** | [ADR-002: Rclone Remote Control API As Source Of Truth](../architecture/adr/ADR-002-Rclone-As-Source-Of-Truth.md) |
| **Core Principles** | [Architecture Principles](../architecture/ARCHITECTURE_PRINCIPLES.md), [Engineering Principles](../architecture/ENGINEERING_PRINCIPLES.md) |

---

## Delivered Components

- Package `internal/core/client/`: HTTP connection pooling, basic auth, retry logic, timeout handling (`client.go`).
- RC Job Management: Async job triggers, status polling, stop jobs (`jobs.go`).
- Remote Operations: `ListRemotes`, `GetRemoteConfig`, `GetRemoteAbout`, `UpdateRemoteConfig`, `OAuthStatus`, `OAuthStop` (`remotes.go`).
- Filesystem Primitives: `ListDirectory`, `CopyFile`, `CopyDir`, `DeleteFile`, `PurgeDir`, `TouchFile`, `NormalizeFs` (`fs_ops.go`).
- RC Stream Monitoring: Daemon real-time log stream subscription (`monitor.go`).

## Remaining Planned Scope

- Provider Metadata Discovery (`/config/providers`) & Dynamic Form Generation (`internal/providers/`).
- Full Remote CRUD (`/config/create`, `/config/delete`) abstraction.


## Priority

Critical Foundation

## Goal

Provide a unified abstraction layer over the Rclone Remote Control (RC) API that enables Bisync Hub to manage remotes, authentication, diagnostics, profiles, and synchronization workflows without directly modifying rclone.conf.

Bisync Hub does not replace Rclone.

Bisync Hub builds on top of:

- Rclone
- Rclone RC API
- Rclone backend definitions
- Rclone configuration management

The framework exists to translate Rclone capabilities into a modern operational user experience.

---

# Design Philosophy

## Rclone Owns Configuration

Rclone remains the authoritative source for:

- Remote definitions
- OAuth implementations
- Backend implementations
- Provider support
- Authentication tokens
- Synchronization operations
- Configuration persistence

Examples:

drive
onedrive
dropbox
s3
webdav
mega

Bisync Hub does not duplicate this functionality.

---

## Bisync Hub Owns Operations

Bisync Hub provides:

- Dashboard
- Profile management
- Health monitoring
- Interactive Sync Review
- OAuth recovery
- Scheduling
- Notifications
- Audit logging
- Diagnostics
- Configuration guidance

---

# Architecture

```text
Bisync Hub
      │
      ▼
Rclone RC Integration Framework
      │
      ▼
Rclone RC API
      │
      ▼
rclone.conf
      │
      ▼
Rclone
      │
      ▼
Storage Providers
```

---

# Core Principle

Every remote visible in Bisync Hub originates from Rclone.

Every configuration change made in Bisync Hub is executed through RC APIs.

Bisync Hub never edits:

rclone.conf

directly.

---

# RC API Dependencies

Primary configuration endpoints:

/config/create

/config/update

/config/delete

/config/get

/config/listremotes

/config/dump

/config/providers

The framework acts as a structured consumer of these APIs.

---

# Framework Responsibilities

## Remote Discovery

Query:

/config/listremotes

Purpose:

- Discover configured remotes
- Build profile catalog
- Populate dashboard views

Result:

```text
gdrive
onedrive
dropbox
```

---

## Remote Inspection

Query:

/config/get

Purpose:

- View remote configuration
- Build edit forms
- Drive diagnostics

---

## Remote Creation

Query:

/config/create

Purpose:

- Create new remotes
- Support onboarding workflows
- Enable profile creation

---

## Remote Updates

Query:

/config/update

Purpose:

- Modify remote settings
- Update credentials
- Change provider options

---

## Remote Removal

Query:

/config/delete

Purpose:

- Remove configuration
- Remove profile references

---

## Provider Discovery

Query:

/config/providers

Purpose:

- Enumerate supported backends
- Discover provider metadata
- Generate configuration experiences

---

# Profiles

Bisync Hub introduces Profiles.

Profiles are an operational abstraction.

Example:

```yaml
name: Personal Google Drive

remote: gdrive

provider: drive
```

---

# Relationship Model

```text
Rclone Remote
      │
      ▼
Bisync Hub Profile
      │
      ▼
Operational Features
```

Example:

```text
Remote:
gdrive

Profile:
Personal Google Drive
```

---

# Multiple Profiles

Supported.

```text
Personal Google Drive

Work Google Drive

Archive Google Drive
```

All may reference independent Rclone remotes.

---

# Provider Catalog

Provider information is obtained from Rclone.

Bisync Hub does not maintain a hardcoded provider registry.

Source:

/config/providers

---

# Provider Metadata

Examples:

```text
Google Drive

OneDrive

Dropbox

S3

WebDAV
```

Displayed as:

```text
Supported Providers
```

inside the UI.

---

# Dynamic Form Generation

Provider configuration screens are generated using Rclone metadata.

Workflow:

Provider
    ↓
/config/providers
    ↓
Metadata
    ↓
Generated Form
```

This eliminates duplicate configuration definitions.

---

# Configuration Translation

Example:

User enters:

```text
Client ID

Client Secret

Scope
```

Bisync Hub submits:

```text
/config/update
```

Rclone persists configuration.

No custom configuration writer exists.

---

# OAuth Integration

Authentication remains implemented by Rclone.

Bisync Hub acts as an orchestration layer.

---

## OAuth Connection Manager

Responsible for:

- Detecting failures
- Guiding recovery
- Launching reconnect workflows
- Displaying status

Not responsible for:

- OAuth implementation
- Token management
- Backend authentication logic

---

# Credential Management

Source of truth:

Rclone

Bisync Hub displays:

- Auth status
- Validation state
- Migration warnings

Bisync Hub must never expose:

- Access Tokens
- Refresh Tokens
- Client Secrets

---

# Diagnostics Framework

Diagnostics combine:

Rclone metadata

RC responses

Bisync operational checks

---

## Example

Google Drive

```text
✓ Remote Exists

✓ Authentication Valid

✓ API Reachable

⚠ Shared Client ID Detected
```

---

# Provider Extensions

Some providers require additional guidance.

Examples:

Google Drive

- Client ID Migration

OneDrive

- Tenant Validation

Dropbox

- Team Space Validation

These are operational overlays.

They do not modify backend behavior.

---

# Community Extensions

Community contributions focus on:

- Provider guidance
- Diagnostics
- Validation rules
- UI enhancements
- Documentation

They do not introduce new storage backends.

Storage backend support remains owned by Rclone.

---

# Internal Packages

```text
internal/rc/
```

Responsibilities:

- RC Client
- Endpoint Management
- Authentication
- Retry Logic
- Error Handling

---

```text
internal/remotes/
```

Responsibilities:

- Remote Catalog
- Remote Translation
- Profile Mapping

---

```text
internal/providers/
```

Responsibilities:

- Provider Metadata Cache
- Dynamic Form Generation
- Validation Rules
- Guidance Workflows

---

# Security Principles

All RC access is high privilege.

The RC API can:

- Read remotes
- Modify remotes
- Create remotes
- Delete remotes

Access must be treated as administrative access.

Authentication and transport security must be enforced. 【1-3e7601】

---

# MVP Acceptance Criteria

- RC client implemented
- Remote discovery implemented
- Remote inspection implemented
- Remote creation implemented
- Remote update implemented
- Remote deletion implemented
- Provider discovery implemented
- Dynamic forms implemented
- Profile abstraction implemented
- OAuth Connection Manager integrated
- Diagnostics integrated

---

# Success Criteria

Bisync Hub supports any Rclone backend supported by the connected Rclone instance without requiring backend-specific implementation work.

When Rclone adds support for a new backend, Bisync Hub can surface and manage that backend through the RC API with minimal additional development.

Rclone remains the system of record.

Bisync Hub remains the operational, diagnostic, and user experience layer built on top of Rclone.