# Implementation Plan: Option A — Sundown Alpine.js & Default Route to Vue 3 SPA

## Proposed Changes

Sundown the legacy Go-templated Alpine.js frontend (`cockpit.html`, `partials/`, `js/`, `vendor/`), make the embedded Vue 3 / PrimeVue 4 SPA the default UI served at `GET /`, update static asset routing, adjust `cmd/check` scanners (HTML templates, licenses), and align E2E/integration tests.

---

### Affected Components & Packages

- `dashboard/frontend/`:
  - Run fresh production build (`npm run build`) with updated Vite 8 + vue-tsc 3 toolchain.
  - Remove legacy Alpine assets: `cockpit.html`, `app.js`, `css/design-tokens.css`, `partials/` (all 40 sub-templates), `js/` (legacy scripts), `vendor/`.
- `internal/server/`:
  - `routes.go`: Remove `//go:embed cockpit.html` and Alpine template rendering. Serve `dist/index.html` at `GET /` and `GET /cockpit`. Serve assets directly from embedded filesystem.
  - `server_test.go` & `handlers_*.go`: Update any tests expecting `cockpit.html` or Alpine-specific headers.
- `cmd/check/`:
  - `html_scanner.go` & `html_scanner_test.go`: Remove or adapt HTML scanner rules that targeted Alpine template attributes (`:disabled`, `x-cloak`).
  - `licenses.go`: Update third-party license inspection to reflect Vue 3 package tree.
- `tests/e2e/`:
  - `dashboard_smoke_test.go`: Update assertions from Alpine DOM IDs to Vue 3 Cockpit DOM (`#app`, PrimeVue elements).
- `Makefile`:
  - Keep build targets streamlined.

---

### Technical Strategy & Data Flow

1. **Embedded Assets**:
   - `internal/server/routes.go` embeds `all:dashboard/frontend/dist` (or utilizes existing embedded `frontendFS`).
   - `GET /` serves `dashboard/frontend/dist/index.html` with `Content-Type: text/html; charset=utf-8`.
   - `GET /cockpit` redirects (or serves directly) to `GET /`.
   - Asset routes (`/assets/*`, `/favicon.ico`, `/favicon.svg`) served directly from embedded `dist`.
2. **Clean Sundown**:
   - Deleting the legacy Alpine codebase removes ~40 HTML files and ~20 JS files, dramatically shrinking source complexity and preventing template maintenance confusion.
3. **Quality Gates**:
   - `cmd/check` (HTML scanner, license scanner, LOC checker) passes cleanly with 0 warnings.
   - All Go unit tests, MCP tests, and E2E tests pass.

---

### Safety & Error Handling

- **SPA Routing**: Any non-API HTML navigation defaults safely to `index.html` without 404s.
- **Cache Control**: Embedded static assets served with appropriate caching headers while `index.html` remains `no-cache` to ensure instant updates on binary deployment.
- **Single-Binary Invariant**: Distribution remains 100% self-contained static executable with zero external runtime dependencies.

## Verification Plan

1. Frontend Build: `npm --prefix dashboard/frontend run build` outputs clean `dist/`.
2. Go Tests: `go test -count=1 ./...` passes across all packages.
3. Static & Quality Audits: `./scripts/check.sh all` passes all stages (Go build, vet, test, check tool, licenses, LOC).
4. Smoke Tests: `go test -count=1 ./tests/e2e -run TestDashboardSmoke` passes against Vue 3 DOM.




