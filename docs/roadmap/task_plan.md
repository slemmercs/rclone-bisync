# Task Plan: Centralized Frontend API Client & Service Decoupling

## Goal
Implement a centralized HTTP API client (`dashboard/frontend/js/api.js`) to eliminate redundant `fetch` / JSON parsing / header boilerplate across frontend modules, reduce `review.js` LOC, and standardize security headers (`X-Dashboard-Request: 1`).

## Tasks

- [x] Task 1: Create `dashboard/frontend/js/api.js` & Update `index.html`
  - **Scope**: Implement `window.API` with `request()`, `get()`, `post()`, response unwrapping, and structured error throwing. Inject `<script src="/static/js/api.js?v=...">` into `index.html`.
  - **Verification**: `npx esbuild dashboard/frontend/js/api.js --bundle --dry-run` or `./scripts/check.sh` syntax check.

- [x] Task 2: Migrate `review.js` to Centralized API Client
  - **Scope**: Refactor `fetchReviewSession`, `stageReviewDecision`, `batchStageReviewDecisions`, `batchStageAllPending`, `executeReviewPlan`, `rollbackReviewSession`, and `dismissReviewSession`.
  - **Verification**: `./scripts/check-loc.sh 400 && ./scripts/check.sh`

- [x] Task 3: Migrate `explorer.js` & `explorer_actions.js`
  - **Scope**: Refactor browse, canary provision, copy, and delete operations to use `window.API`.
  - **Verification**: `./scripts/check.sh`

- [x] Task 4: Migrate `remotes.js`, `audit.js`, `logs.js`, and `app.js`
  - **Scope**: Refactor remote management, audit ledger, daemon logs, and telemetry polling to use `window.API`.
  - **Verification**: `./scripts/check.sh`

- [x] Task 5: Final Verification & Pre-Commit Quality Gate
  - **Scope**: Run full `./scripts/check.sh` (all 8 stages) and LOC audit.
  - **Verification**: `./scripts/check.sh && ./scripts/check-loc.sh 400`

## Status Updates

- 2026-10-25: Initialized plan for Centralized Frontend API Client (`api.js`).
- 2026-10-25: Created `api.js` and loaded in `index.html`.
- 2026-10-25: Migrated `review.js`, `explorer.js`, `explorer_actions.js`, `remotes.js`, `audit.js`, `logs.js`, and `app.js` to `window.API`.
- 2026-10-25: All quality gates passed with zero files exceeding 400 LOC and 88.2% test coverage.


