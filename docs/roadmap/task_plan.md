# Task Plan: Curate OneDrive Target Provider Options (Delta & Chunk Size)

## Goal
Add `--onedrive-delta` to Target Provider Options in `modal_provider_options.html`, ensure all OneDrive options (`onedrive_no_versions`, `onedrive_delta`, `onedrive_chunk_size`) compile in backend and synthesize in frontend CLI preview, and clean up duplicate dropdown flags.

## Tasks

- [x] Task 1: Backend Support for OneDrive Delta & Frontend Controls Integration
  - **Scope**: `internal/profiles/builder.go`, `internal/profiles/builder_test.go`, `dashboard/frontend/partials/modals/modal_provider_options.html`, `dashboard/frontend/js/flags_builder.js`, `dashboard/frontend/js/help-data-flags.js`
  - **Verification**: `go test -v -count=1 ./internal/profiles/... && ./scripts/check.sh templates && ./scripts/check.sh all`

- [ ] Task 2: E2E Headless Chrome Tests, Binary Build & Service Deployment
  - **Scope**: `tests/e2e/dashboard_smoke_test.go`, build binary, restart service
  - **Verification**: `go test -v -count=1 ./tests/e2e/... && ./scripts/check.sh all`

## Status Updates

- 2026-11-06: Initialized task plan to add `--onedrive-delta` into Target Provider Options and fix synthesis for OneDrive provider options.
- 2026-11-06: Completed Task 1: Integrated `onedrive_delta` and `onedrive_chunk_size` into Target Provider Options and backend CLI builder. All 7 check gates passed cleanly.



