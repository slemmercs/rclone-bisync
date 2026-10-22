# Architecture Principles

These principles guide technical decisions within Bisync Hub.

Any significant change should align with these principles.

---

# Single Binary First

Bisync Hub should remain deployable as a single executable.

Avoid runtime dependency chains that increase deployment complexity.

---

# Embedded Frontend

Frontend assets should be bundled directly into the application where practical.

Users should not need separate frontend deployments.

---

# Local-First Design

The primary deployment target is:

- Individual users
- Power users
- Homelabs
- Small environments

Distributed deployments are secondary concerns.

---

# Safety Before Automation

Data safety takes priority over convenience.

Synchronization workflows should prioritize:

- Visibility
- Validation
- Recovery
- User approval

especially for destructive actions.

---

# Review Before Destruction

Users should understand:

- What is changing
- Why it is changing
- What the consequences are

before synchronization proceeds.

---

# Security By Default

Credentials and secrets must never be exposed through:

- Logs
- Telemetry
- Exports
- Debug output

---

# Explainability

Whenever practical:

- Show cause
- Show impact
- Show proposed action

before execution.

---

# Maintainability Over Cleverness

Readable code is preferred.

Simple solutions are preferred.

Avoid unnecessary complexity when simpler approaches exist.

---

# Minimal Runtime Dependencies

Avoid introducing:

- Databases
- Mandatory cloud services
- Heavy runtime stacks

unless they provide substantial user value.
