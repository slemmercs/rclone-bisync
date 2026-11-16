# Implementation Plan: Unified FSM Storage Control Plane & Vue 3 + PrimeVue Cockpit Modernization

## Overview
Transform Bisync Hub into an autonomous, deterministic Storage Control Plane by replacing implicit boolean state tracking with a thread-safe Go Finite State Machine (`SyncFSM`) backed by data-plane failure contract tests, while co-designing a desktop-grade cockpit Single-Page Application (SPA) powered by Vue 3, PrimeVue v4, Tailwind CSS, and TypeScript that natively consumes and visualizes FSM state transitions.

The architecture strictly preserves all core system invariants established in ADR-005 and ADR-011: the Go application compiles into a 100% self-contained static executable embedding `dashboard/frontend/dist/*` with zero runtime dependencies. Pre-compiled production bundles are committed to version control so backend Go developers can build and test with zero Node.js/npm dependencies installed. Frontend developers utilize Node 22 LTS via NVM for lightning-fast Vite Hot Module Replacement (HMR) and end-to-end TypeScript validation directly against Go FSM models.

### Affected Components & Packages
- `internal/engine/state_machine.go`: Pure Go, thread-safe `SyncFSM` implementation (~160 LOC), transition matrix validation, listener hooks.
- `internal/engine/state_machine_test.go`: Unit tests for all valid/invalid transitions, concurrency races, and transition event ordering.
- `internal/engine/runner/runner_contract_test.go`: Data-plane failure contract tests simulating daemon crash, socket drop, lock contention, and WAL recovery.
- `internal/models/fsm.go`: DTOs exposing FSM state over HTTP API.
- `internal/engine/runner/runner.go` & `runner_handlers.go`: Drive explicit FSM transitions during sync lifecycle (`QUEUED` -> `RUNNING` -> `COMPLETED`/`ABORTED`/`REVIEW_REQUIRED`).
- `internal/services/sync_service.go`: Replace ad-hoc boolean lookups with `fsm.IsActive()`.
- `dashboard/frontend/`: Vue 3 + PrimeVue v4 + Tailwind CSS + Pinia SPA source code.
- `internal/server/routes.go`: Serve SPA static bundle from `dashboard/frontend/dist/` with client-side fallback routing.

## Verification Plan
- `go test -v -race -count=1 ./internal/engine/...`
- `go test -v -count=1 ./internal/engine/runner/... -run "TestContract_"`
- `npm run type-check && npm run build`
- `./scripts/check.sh all`


