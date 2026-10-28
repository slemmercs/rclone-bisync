# Task Plan: Phase 5 - Tiered Cloud Provider Architecture, Provider Options UI & Global Fleet Settings

## Goal
Implement a tiered cloud provider settings architecture with first-class UI controls for Tier-1 providers (Google Drive, OneDrive, S3, Dropbox) and an audited custom flags field for Tier-2 generic providers, decouple daemon engine defaults from workload sync payloads, surface a Profile Execution Contract in Telemetry, and introduce a Global Fleet Settings manager.

## Tasks

- [x] Task 1: Engine Flag Decoupling & Compiler Tier-1 Provider Flag Registration
  - **Scope**: `internal/config/config.go`, `bisync.env`, `internal/config/config_test.go`, `internal/core/compiler/compiler.go`, `internal/core/compiler/compiler_test.go`
  - **Verification**: `go test -count=1 ./internal/config/... ./internal/core/compiler/...`

- [x] Task 2: Provider Options Model & Flag Builder Implementation
  - **Scope**: `internal/profiles/model.go`, `internal/profiles/builder.go`, `internal/profiles/builder_test.go`
  - **Verification**: `go test -count=1 ./internal/profiles/...`

- [x] Task 3: Global Fleet Settings Backend Endpoints (`GET/PUT /api/config`)
  - **Scope**: `internal/server/handlers_config.go`, `internal/server/server.go`, `internal/server/server_test.go`
  - **Verification**: `go test -count=1 ./internal/server/...`

- [x] Task 4: Unified Provider Options UI in Profile Modal (`modal_profiles.html`)
  - **Scope**: `dashboard/frontend/partials/modals/modal_profiles.html`, `dashboard/frontend/js/profiles.js`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh loc`

- [x] Task 5: Profile Execution Contract in Telemetry View (`view_telemetry.html`)
  - **Scope**: `dashboard/frontend/partials/view_telemetry.html`, `dashboard/frontend/app.js`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh loc`

- [x] Task 6: Global Fleet Settings Modal Component (`modal_global_settings.html`)
  - **Scope**: `dashboard/frontend/partials/modals/modal_global_settings.html`, `dashboard/frontend/index.html`, `dashboard/frontend/app.js`
  - **Verification**: `./scripts/check.sh templates && ./scripts/check.sh loc`

- [x] Task 7: Full Quality Verification & Headless Chrome Testing
  - **Scope**: Full repository
  - **Verification**: `./scripts/check.sh all && go test -v -count=1 ./tests/e2e -run TestDashboard_HeadlessChromeSmoke`

## Status Updates

- 2026-10-28: Completed Phase 5 (Tiered Cloud Provider Architecture, Provider Options UI & Global Fleet Settings):
  1. Decoupled engine defaults in internal/config/config.go and bisync.env by removing Google Drive flags, ensuring global defaults are strictly provider-agnostic.
  2. Registered Tier 1 provider flags in compiler.CLIToRCMap (OneDrive, S3, Dropbox, Google Drive) with full connection-string path parameter translation.
  3. Extended ProfileFlags and ResolvedFlags with ProviderOptions map[string]string and implemented typed CLI argument builder with test coverage.
  4. Introduced GET/PUT /api/config handlers with runtime validation and secrets redaction.
  5. Implemented unified Target Provider Configuration panel (modal_provider_options.html) in profile modal with Tier 1 first-class controls (GDrive, OneDrive, S3, Dropbox) and audited custom flags for both Tier 1 and Tier 2 generic providers.
  6. Replaced raw .env dump in view_telemetry.html with a concise Profile Execution Contract card displaying topology, schedule, provider badge, and effective flags pill cloud.
  7. Created Global Fleet Settings modal (modal_global_settings.html, settings.js) accessible via Tier 1 header and Fleet banner.
  8. Verified all 6 stages of ./scripts/check.sh all and headless Chrome E2E smoke tests passing with zero errors.
- 2026-10-28: Initialized Phase 5 Task Plan from approved architectural design.





