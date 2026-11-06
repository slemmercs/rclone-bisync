# Task Plan: Alternative A - Single-Source Declarative Schema Engine for Provider Options

## Goal
Implement Alternative A: Single-Source Declarative Schema Engine for Target Provider Options, replacing duplicated Go, JavaScript, and HTML logic with a single declarative schema.

## Tasks

- [x] Task 1: Go Provider Options Declarative Schema Registry & Generic Backend Compiler
  - **Scope**: `internal/profiles/provider_options.json`, `internal/profiles/registry.go`, `internal/profiles/builder.go`, `internal/profiles/builder_test.go`
  - **Verification**: `go test -v -count=1 ./internal/profiles/...`

- [x] Task 2: Frontend Declarative Provider Options Engine & Dynamic UI Rendering
  - **Scope**: `dashboard/frontend/js/provider_options.js`, `dashboard/frontend/index.html`, `dashboard/frontend/js/profiles_modal.js`, `dashboard/frontend/partials/modals/modal_provider_options.html`, `dashboard/frontend/js/flags_builder.js`
  - **Verification**: `./scripts/check.sh templates && go test -v -count=1 ./tests/e2e/...`

- [ ] Task 3: Quality Gates, Smoke Verification, Binary Build & Deployment
  - **Scope**: `./scripts/check.sh all`, binary build, deployment to `/home/csaba/rclone-bisync/bin/rclone-bisync`, service restart, live verification
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-11-06: Initialized task plan for Alternative A (Declarative Provider Options Schema Engine).
- 2026-11-06: Completed Task 1: Embedded `provider_options.json` registry in Go, replaced hardcoded `buildProviderCLIArgs` in `builder.go` with generic registry iteration, added registry unit tests in `builder_test.go`. All tests passing.
- 2026-11-06: Completed Task 2: Created `provider_options.js` module, registered in `index.html` and `app.js`, dynamically rendered options via Alpine `template x-for` loops in `modal_provider_options.html`, updated `generateCLIPreview` in `flags_builder.js`, added and passed E2E headless Chrome parity tests.






