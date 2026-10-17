# Task Plan: Full-Repository Code Review & Safety Audit

## Goal
Conduct a systematic audit of the entire rclone-bisync repository against Section 3 standards, validating safety circuit breakers, test mocking/isolation, secret protection, and frontend reactivity, culminating in a refreshed `code-review.md`.

## Tasks
- [x] Task 1: Audit Module 1 — Core Engine & Safety (`rclone_bisync/core/`)
  - **Scope**: `core/client.py`, `core/lock.py`, `core/state.py`, `core/compiler.py`
  - **Checklist**: Race-free locking, stale PID recovery, canary integrity, rate-limit backoff, RC client timeouts/auth, type annotations & docstrings.
  - **Verification**: `./scripts/check.sh && pytest tests/test_lock.py tests/test_client.py tests/test_state.py tests/test_compiler.py`
- [ ] Task 2: Audit Module 2 — Service Layer & Path Jails (`rclone_bisync/services/`)
  - **Scope**: `services/sync_service.py`
  - **Checklist**: Path traversal jail (`commonpath`), deletion safeguards, canary protection, atomic file operations (`.tmp` + `os.replace`), secret masking, error resilience.
  - **Verification**: `./scripts/check.sh && pytest tests/test_sync_service.py`
- [ ] Task 3: Audit Module 3 — Dashboard Backend & Security (`dashboard/backend/main.py` & `config.py`)
  - **Scope**: `dashboard/backend/main.py`, `rclone_bisync/config.py`
  - **Checklist**: CSRF middleware (`X-Dashboard-Request: 1`, loopback origin), thin controller pattern, error handling, configuration parsing from `.env`, secret exposure risks.
  - **Verification**: `./scripts/check.sh && pytest tests/test_config.py`
- [ ] Task 4: Audit Module 4 — Dashboard Frontend (`dashboard/frontend/`)
  - **Scope**: `dashboard/frontend/app.js`, `dashboard/frontend/index.html`
  - **Checklist**: Alpine.js state isolation, drag-and-drop vs click interference (`draggable="false"`, `@mousedown.stop`), XSS sanitization (`escapeHtml`), network failure handling.
  - **Verification**: `./scripts/check.sh`
- [ ] Task 5: Audit Module 5 — Test Suite Quality & Synthesis (`tests/` & `code-review.md`)
  - **Scope**: All unit tests in `tests/`, test mocking (zero live subprocess/cloud calls), AAA pattern compliance, naming conventions (`test_<func>_<behavior>`).
  - **Synthesis**: Update `code-review.md` with structured verdict, grade cards, findings by severity (🔴 High, 🟡 Medium, 🟢 Low), and remediation suggestions.
  - **Verification**: `./scripts/check.sh`

## Status Updates
- 2026-10-17: Initialized audit plan on branch `audit/full-repo-safety-and-quality-review`.
- 2026-10-17: Completed Module 1 audit (Core Engine & Safety): Grade A. Verified 22 unit tests green, zero-race lock acquisition, strict atomic writes, and credential whitelisting in RC client.



