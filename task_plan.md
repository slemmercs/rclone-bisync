# Task Plan: Branch 3 - Security Pydantic DTOs & Recursive Redaction

## Goal
Establish strict Pydantic response models (DTOs) for all FastAPI dashboard endpoints and implement a deep recursive secret sanitization pipeline (`redact_secrets()`) to prevent token, password, or webhook leakage over HTTP.

## Tasks
- [x] Task 1: Recursive Secret Sanitization Engine (`rclone_bisync/core/security.py`)
  - **Scope**: Create `rclone_bisync/core/security.py` with `redact_secrets()` supporting recursive dicts, lists, sets, tuples, and Pydantic models. Add comprehensive unit tests in `tests/test_security.py`.
  - **Verification**: `/home/csaba/rclone-bisync/.venv/bin/pytest tests/test_security.py`
- [ ] Task 2: Centralized API Response Models & DTOs (`rclone_bisync/models/api.py`)
  - **Scope**: Create `rclone_bisync/models/__init__.py` and `rclone_bisync/models/api.py`. Define `ActionResponse`, `BrowseItem`, `BrowseResponse`, `CanaryResponse`, `CopyResponse`, `DeleteResponse`, `PendingDeletionEntry`, `RemoteCardInfo`, `RemotesSummary`, `RemotesResponse`, `TelemetryPayload`.
  - **Verification**: `python3 -m py_compile rclone_bisync/models/api.py`
- [ ] Task 3: Integrate Models and Recursive Redaction in Dashboard Backend (`dashboard/backend/main.py`)
  - **Scope**: Update `dashboard/backend/main.py` to import models from `rclone_bisync.models.api`, apply `response_model` across all endpoints (`/api/action`, `/api/browse`, `/api/browse/canary`, `/api/browse/copy`, `/api/browse/delete`, `/api/remotes`, `/api/deletions/pending`, `/api/telemetry`), and apply `redact_secrets()` middleware/interceptor to sanitize all outgoing payloads defense-in-depth.
  - **Verification**: `/home/csaba/rclone-bisync/.venv/bin/pytest tests/test_dashboard_backend.py`
- [ ] Task 4: Backend Contract Validation & Leak-Proofing Tests (`tests/test_dashboard_backend.py`)
  - **Scope**: Extend `tests/test_dashboard_backend.py` with contract compliance tests for all newly typed endpoints and verify recursive secret redaction under simulated leaks. Run full pre-flight check `./scripts/check.sh`.
  - **Verification**: `./scripts/check.sh`

## Status Updates
- 2026-10-17: Initialized Branch 3 task plan on `feat/security-pydantic-dtos-and-redaction`.
- 2026-10-17: Completed Task 1 (Recursive Secret Sanitization Engine). Added `rclone_bisync/core/security.py` and 5 unit tests in `tests/test_security.py`. All 73 tests green.





