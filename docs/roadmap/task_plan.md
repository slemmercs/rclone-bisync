# Task Plan: Profile-Driven Dual-Pane Explorer Integration, Safeguarded Local Path Resolution & Custom Deletion Modal

## Goal
Integrate workspace profiles with the dual-pane file explorer (auto-binding, orientation swap, exclude rule visual dimming), safeguard local path auto-resolution, decouple configuration audit history, implement dark UI delete confirmation modal with trash SVG, and ensure all files adhere to the 400 LOC ceiling and Alpine safety checks.

## Tasks

- [x] Task 1: Audit Domain Decoupling & Exclude Patterns Backend Support
  - **Scope**: `internal/profiles/model.go`, `internal/profiles/store.go`, `internal/profiles/service.go`, `internal/profiles/service_test.go`
  - **Verification**: `go test -count=1 ./internal/profiles/...`

- [x] Task 2: Home Directory Detection & Path Safeguarding
  - **Scope**: `internal/profiles/service.go`, `internal/server/handlers_profiles.go`, `internal/server/handlers_profiles_test.go`, `internal/server/server.go`
  - **Verification**: `go test -count=1 ./internal/server/...`

- [x] Task 3: Profile Management UI, Trash SVG & Delete Confirmation Modal
  - **Scope**: `dashboard/frontend/js/profiles.js`, `dashboard/frontend/partials/modals/modal_profiles.html`
  - **Verification**: `./scripts/check.sh templates`

- [x] Task 4: Explorer Integration, Orientation Swap & Exclude Pattern Dimming
  - **Scope**: `dashboard/frontend/js/explorer.js`, `dashboard/frontend/js/review.js`, `dashboard/frontend/partials/view_explorer.html`
  - **Verification**: `./scripts/check.sh loc` and `./scripts/check.sh templates`

- [x] Task 5: Pre-Flight Verification & LOC Audit Compliance
  - **Scope**: Entire codebase
  - **Verification**: `./scripts/check.sh all`

- [x] Task 6: Active Profile Local Path Routing in Dual-Pane Explorer & Human-Friendly Profile Names in Audit Ledger
  - **Scope**: `internal/models/api.go`, `internal/services/explorer_service.go`, `internal/services/explorer_ops.go`, `internal/services/canary_service.go`, `internal/server/handlers_explorer.go`, `internal/server/handlers_audit.go`, `internal/audit/ledger.go`, `dashboard/frontend/js/explorer.js`, `dashboard/frontend/js/explorer_actions.js`, `dashboard/frontend/js/audit.js`, `dashboard/frontend/partials/view_audit.html`
  - **Verification**: `go test -count=1 ./internal/...` and `./scripts/check.sh all`


## Status Updates

- 2026-10-26: Created branch `feat/profile-explorer-integration-and-modal` and initialized plan from implementation_plan.md.
- 2026-10-26: Completed Task 1 (Decoupled profile lifecycle events into profiles_history.json with Action discriminator, added ExcludePatterns to Profile model).
- 2026-10-26: Completed Task 2 (Added home_dir endpoint and path safeguarding for outside-home directories).
- 2026-10-26: Completed Task 3 (Implemented Lucide trash SVG, dark confirmation dialog, exclude input, and isProfileDeleting helper).
- 2026-10-26: Completed Task 4 (Integrated profile target auto-binding in explorer, orientation swap button ⇄ with array-destructuring state swap, and exclude-pattern visual dimming/badge).
- 2026-10-26: Completed Task 5 (Optimized LOC in explorer.js, explorer_actions.js, and view_explorer.html ensuring strict adherence to the 400 LOC ceiling; verified all 6 quality stages pass via `./scripts/check.sh all`).

- 2026-10-26: Completed Task 6 (Plumbed profile local path resolution to explorer browse/canary/copy/delete handlers and services; bound Alpine active profile ID to all explorer fetch requests; enriched immutable audit ledger with friendly profile names and profile filter autocomplete).

