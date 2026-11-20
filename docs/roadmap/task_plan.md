# Task Plan: Option A — Sundown Alpine.js & Default Route to Vue 3 SPA

## Goal
Sundown the legacy Go-templated Alpine.js frontend, make the embedded Vue 3 / PrimeVue 4 SPA the default UI served at `GET /`, update static asset routing, adjust `cmd/check` scanners (HTML templates, licenses), and align E2E/integration tests.

## Tasks

- [x] Task 1: Compile Production Vue 3 SPA Bundle
  - **Scope**: `dashboard/frontend/package.json`, `dashboard/frontend/dist/*`, `dashboard/frontend/vite.config.ts`, `dashboard/frontend/index.html`
  - **Verification**: `npm --prefix dashboard/frontend run build`

- [ ] Task 2: Serve Vue 3 SPA at `GET /` and Route Assets from `dist`
  - **Scope**: `internal/server/routes.go`, `internal/server/server.go`, `internal/server/server_test.go`
  - **Verification**: `go test -count=1 ./internal/server/...`

- [ ] Task 3: Sundown Legacy Alpine Files
  - **Scope**: `dashboard/frontend/cockpit.html`, `dashboard/frontend/partials/`, `dashboard/frontend/js/`, `dashboard/frontend/vendor/`, `dashboard/frontend/app.js`, `dashboard/frontend/css/design-tokens.css`
  - **Verification**: `git status --porcelain` confirms removal of legacy assets

- [ ] Task 4: Update `cmd/check` HTML and License Scanners
  - **Scope**: `cmd/check/html_scanner.go`, `cmd/check/licenses.go`, `cmd/check/main.go`, `cmd/check/main_test.go`
  - **Verification**: `go test -count=1 ./cmd/check/...`

- [ ] Task 5: Align E2E and Integration Tests to Vue 3 DOM
  - **Scope**: `tests/e2e/dashboard_smoke_test.go`, `tests/e2e/cockpit_spa_smoke_test.go`, `main_test.go`
  - **Verification**: `go test -count=1 ./tests/e2e/...`

- [ ] Task 6: Pre-commit Validation & Quality Gate Verification
  - **Scope**: Repository-wide audit
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-11-20: Completed Task 1 (Compile Production Vue 3 SPA Bundle). Upgraded frontend build with Vite 8 + vue-tsc 3, updated `vite.config.ts` manualChunks function for Rolldown, standardized entry point to `dashboard/frontend/index.html`, and generated production distribution in `dashboard/frontend/dist/`.
- 2026-11-20: Initialized Option A plan. Merged Dependabot PRs #92 and #91 (Vite 8 & vue-tsc 3 upgrade), closed #93 as superseded. Created feature branch `feat/sundown-alpine-spa-default`.




