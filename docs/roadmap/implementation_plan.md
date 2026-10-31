# Implementation Plan: Comprehensive HTTP Access Log Noise Demotion & Unified Ledger Filter

## Proposed Changes

Eliminate all routine HTTP access logging noise from `bisync.log` (Unified Ledger) by demoting all successful read-only requests (`GET`, `HEAD`) with `status < 400` (including static assets, favicon, remotes, profiles, and polling endpoints) to `slog.Debug` in the server access logging middleware. Furthermore, filter out routine `http_access` entries from the UI unified ledger view so that historical entries and debug traces do not clutter the business event stream.

### Affected Components & Packages

- `internal/server/server.go`:
  - Update `ServeHTTP` logging middleware: treat all successful read-only requests (`r.Method == http.MethodGet || r.Method == http.MethodHead`) with `sw.Status() < 400` as `slog.Debug`. Mutating requests (`POST`, `PUT`, `DELETE`, `PATCH`) and HTTP errors (`status >= 400`) are logged at `slog.Info` (or `slog.Warn`).
- `internal/server/server_test.go`:
  - Update tests to verify that `GET /static/...`, `GET /api/profiles`, `GET /api/remotes`, and polling requests are logged at `Debug`, while mutating requests (`POST /...`) or error status (`404`, `500`) are logged at `Info`.
- `dashboard/frontend/js/logs.js`:
  - In `filteredBisyncLogs`, filter out routine `http_access` entries where `status < 400` and method is `GET`/`HEAD`, ensuring the Unified Ledger remains dedicated to bisync jobs, sync outputs, and audit events.

### Technical Strategy & Data Flow

```
[ HTTP Request ]
       │
       ├── GET / HEAD with status < 400 (static assets, API queries, polling)
       │       └──► slog.Debug("http_access", ...) [Hidden in default INFO logs]
       │
       ├── POST / PUT / DELETE / PATCH (Sync triggers, profile saves, auth mutations)
       │       └──► slog.Info("http_access", ...)  [Visible operational trail]
       │
       └── Error responses (status >= 400, e.g. 401, 403, 404, 500)
               └──► slog.Warn("http_access", ...)  [Visible security & error alert]
```

### Safety & Error Handling

- Debug level logging preserves zero information loss: running with `--verbose` still outputs full HTTP access lines.
- Mutating and error requests are never suppressed or demoted.
- Adheres to the <400 LOC file budget for `logs.js` and `server.go`.

## Verification Plan

- `go test -count=1 ./internal/server/...`
- `./scripts/check.sh loc 400`
- `./scripts/check.sh all`



