# Implementation Plan: Centralized Frontend API Client & Service Decoupling

## Proposed Changes

Extract a centralized HTTP client (`dashboard/frontend/js/api.js`) to unify all REST API communication across the dashboard, standardize required security headers (`X-Dashboard-Request: 1`, `Content-Type: application/json`), normalize error handling, and eliminate duplicated fetch/JSON boilerplate across `review.js`, `explorer.js`, `explorer_actions.js`, `remotes.js`, `audit.js`, `logs.js`, and `app.js`.

### Affected Components & Packages

- `dashboard/frontend/js/api.js` (New, ~75 LOC): Unified `window.API` client with `.get()`, `.post()`, and custom request wrapping with JSON response parsing and HTTP error raising.
- `dashboard/frontend/index.html`: Load `api.js` prior to any modules that make network requests.
- `dashboard/frontend/js/review.js`: Refactor 7 REST API calls to use `window.API`, reducing LOC from 384 down to ~320 LOC.
- `dashboard/frontend/js/explorer.js` & `explorer_actions.js`: Use `window.API.post()` for browse, copy, delete, and canary provisioning.
- `dashboard/frontend/js/remotes.js`: Use `window.API` for remote auth, validation, credentials, and OAuth control.
- `dashboard/frontend/js/audit.js`: Use `window.API.get()` for ledger records and verification checks.
- `dashboard/frontend/js/logs.js`: Use `window.API.get()` for daemon log retrieval.
- `dashboard/frontend/app.js`: Use `window.API` for telemetry polling and engine actions.

### Technical Strategy & Data Flow

- The `window.API` object exposes:
  - `request(endpoint, options)`: Core wrapper attaching `X-Dashboard-Request: 1`, parsing JSON safely, and throwing descriptive Error objects on non-2xx responses.
  - `get(endpoint, options)`: Shortcut for GET requests.
  - `post(endpoint, body, options)`: Shortcut for POST requests with auto-JSON body serialization.
- Retain existing UI behavior, toast feedback, and error handling contracts.

### Safety & Error Handling

- Ensures every API request includes `X-Dashboard-Request: 1` header, preventing CSRF / unauthorized browser cross-origin submissions.
- Ensures non-OK HTTP status codes throw actionable Errors with response details attached.

## Verification Plan

- `esbuild` JS syntax validation via `./scripts/check.sh`.
- HTML Tidy structural validation via `./scripts/check.sh`.
- `./scripts/check-loc.sh 400` ensuring all JS files remain strictly < 400 LOC.
- Full 8-stage quality gate `./scripts/check.sh`.

