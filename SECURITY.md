# Security Policy

## Reporting a Vulnerability

Please do not report security vulnerabilities through public GitHub issues.

Instead, contact the project maintainer privately with:

- A description of the issue
- Steps to reproduce
- Potential impact
- Any proposed mitigations

Security reports will be acknowledged as quickly as possible.

---

## Scope

Security-sensitive areas include:

- Authentication and authorization
- Credential handling
- Rclone RC integration
- Dashboard APIs
- File operations
- Synchronization workflows
- Configuration management
- Session management

---

## Examples of Security Issues

Please report issues involving:

- Credential disclosure
- Authentication bypass
- Authorization bypass
- Directory traversal
- Arbitrary file access
- Remote code execution
- Unsafe synchronization actions
- Exposure of secrets in logs or telemetry

---

## Responsible Disclosure

Please provide reasonable time for investigation and remediation before publicly disclosing any security vulnerability.

The goal is to protect users while addressing issues responsibly.

---

## Security Design Principles

Bisync Hub follows several core security principles:

- Local-first operation
- Principle of least privilege
- Credential redaction by default
- Explicit user actions for destructive operations
- Review before synchronization changes
- No intentional collection of user data

Users remain responsible for securing their own systems, accounts, and Rclone configurations.
