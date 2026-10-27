# Implementation Plan: Four-Phase Multi-Profile Stabilization & Architecture Evolution

This plan establishes the architectural roadmap across all four strategic phases:
- **Phase 1: Multi-Profile UI & Engine Cohesion** (Active Execution Scope)
- **Phase 2: Core Refactoring & Subsystem Consolidation**
- **Phase 3: Operational Flags & Profile Schema Alignment**
- **Phase 4: Dashboard UI Evolution & Config Flow**

---

## 1. Phase 1: Multi-Profile UI & Engine Cohesion (Active Scope)

### Proposed Changes
Resolve Alpine.js reactive null pointer exceptions on profile modal close, unify telemetry and sync status across multi-profile schedulers and single-tenant runners, bind cockpit engine badges to the active profile's execution state, and forward profile-specific exclude patterns to the underlying Rclone sync runner.

### Affected Components & Packages
- `dashboard/frontend/js/profiles.js`:
  - Introduce `createDefaultProfile()` helper.
  - Reset `profileModal.editing` to safe default object in `closeProfilesModal()`.
- `dashboard/frontend/partials/modals/modal_profiles.html`:
  - Refactor Tab 2 to use `<template x-if="profileModal.tab === 'edit'">` for DOM detachment.
  - Add null-safe bindings (`profileModal.editing?.name`, `profileModal.editing?.id`).
- `internal/profiles/scheduler.go`:
  - Expose `IsAnyRunning() bool` and `IsProfileRunning(profileID string) bool`.
- `internal/profiles/service.go`:
  - Add delegate methods `IsAnySyncing() bool` and `IsProfileSyncing(id string) bool`.
- `internal/server/handlers_telemetry.go`:
  - Return `"execution_status": "SYNCING" | "IDLE"` in `/api/telemetry` payload.
  - Return `is_syncing: true` if global lock acquired or any profile scheduler worker running.
- `dashboard/frontend/partials/view_telemetry.html`:
  - Update status badge to evaluate `telemetry.execution_status` and active profile state.
- `internal/server/server.go`:
  - In `executor`, map `p.ExcludePatterns` into `profCfg.RcloneCommonArgs` as `--exclude` flags.

### Technical Strategy & Data Flow
1. **Alpine State Safety**:
   - `createDefaultProfile()` initializes non-null defaults for all bound properties.
   - On modal close, `editing` resets to defaults, eliminating `TypeError: Cannot read properties of null`.
2. **Telemetry & Concurrency Harmonization**:
   - Scheduler tracks active worker sync runs.
   - `/api/telemetry` checks both single-tenant lock and active profile workers.
3. **Exclude Rule Forwarding**:
   - Profile `ExcludePatterns` are appended to `profCfg.RcloneCommonArgs` before running `SyncRunner`.

---

## 2. Phase 2: Core Refactoring & Subsystem Consolidation (Active Scope)

### Proposed Changes
1. **Central DI Container (`internal/app/container.go`)**:
   - Centralize lifecycle management, dependency construction, and wiring for core services: Rclone RC client, SyncService, TelemetryService, CanaryService, DeletionService, ExplorerService, ReviewService, AuditService, LogService, RemoteAuthService, ProfileStore, ProfileScheduler, and ProfileService.
   - De-duplicate and decouple service creation from `main.go` and `internal/server/server.go`.
   - Provide clean lifecycle methods (`Start(ctx)`, `Close()`) to manage background profile schedulers, loggers, and database handles gracefully.
2. **Subsystem Consolidation (`internal/review/` -> `internal/engine/review/`)**:
   - Relocate the review package (`backup.go`, `discovery.go`, `executor.go`, `journal.go`, `rollback.go`, `session.go` and their unit tests) into `internal/engine/review/` to respect domain-driven bounded contexts under `engine`.
   - Update imports in `internal/services/review_service.go`, `internal/services/review_recommendations.go`, and test suites.
   - Remove the orphaned `internal/review/` directory.
3. **Stream/Line Secret Redactor (`internal/core/logger/redact.go`)**:
   - Implement regex-based stream and line masking engine for sensitive parameters (`RC_PASS`, OAuth access/refresh tokens, webhook secrets, authorization headers, passwords).
   - Wire the masking engine into log writing (`rotator.go` / `logger.go`) and file tailing (`internal/services/filetail.go`) as a deep defense-in-depth sanitization layer.

### Affected Components & Packages
- `internal/app/container.go` (new), `internal/app/container_test.go` (new):
  - Container struct, constructor `NewContainer(cfg *config.Config) (*Container, error)`, and lifecycle hooks.
- `main.go`, `internal/server/server.go`:
  - Wire application via `internal/app/container.go`.
- `internal/review/` -> `internal/engine/review/`:
  - Package relocation and import harmonization across services and tests.
- `internal/core/logger/redact.go` (new), `internal/core/logger/redact_test.go` (new):
  - Sanitization patterns, redaction streams, and line transformers.
- `internal/core/logger/logger.go`, `internal/services/filetail.go`:
  - Hook secret redaction into file sink and log tail readers.

### Technical Strategy & Data Flow
1. **DI Container**:
   - `Container` manages service instantiation order: Config -> Client -> Core Services (Canary, Deletion, Telemetry, Explorer, Review, Audit, Log, Auth) -> SyncService -> Profile Store/Scheduler/Service.
   - Server takes `Container` or builds one from config fallback, retaining backwards-compatible accessors for existing test fixtures.
   - Background processes (profile scheduler) are managed explicitly via `Container.Start(ctx)` and `Container.Close()`.
2. **Review Relocation**:
   - Bounded context: The review engine is an execution subsystem of the bisync workflow, situated alongside `runner` and `compiler` in `internal/engine/`.
   - Package statement updated to `package review` under `internal/engine/review`.
3. **Secret Redaction**:
   - Multi-pattern regex matching: `token=[^&\s]+`, `secret=[^&\s]+`, `pass(word)?=[^&\s]+`, `Bearer\s+[A-Za-z0-9\-._~+/]+=*`, etc.
   - Replacement with safe masked tokens: `[REDACTED]`.

### Safety & Error Handling
- Nil-safe container construction: All service accessors check for nil configuration and fail with descriptive errors or fallback to safe defaults.
- Graceful shutdown: `Container.Close()` cleanly terminates profile worker routines and flushes log/audit handles.
- No live RC or cloud network calls during container initialization in test environments.

---

## 3. Phase 3: Operational Flags & Profile Schema Alignment

### Proposed Changes
Implement 3-tier operational flags architecture (`docs/design/bisync_operational_flags.md`), flag inheritance (`Profile Override > Global Default > Hardcoded Fallback`), and security blocklist validation.

### Affected Components & Packages
- `internal/config/dto.go`: Operational flags schema and DTO definitions.
- `internal/config/inheritance.go`: Flag inheritance resolver engine.
- `internal/config/validator.go`: Prohibited flag blocklist and POSIX shlex sanitizer.

---

## 4. Phase 4: Dashboard UI Evolution & Config Flow

### Proposed Changes
Deliver profile-first Cockpit UI redesign: quick-switcher in main navigation, visual flag and safety policy editor with inheritance badges, and step-by-step profile onboarding wizard.

### Affected Components & Packages
- `dashboard/frontend/partials/header.html` & `view_telemetry.html`: Profile quick-switcher.
- `dashboard/frontend/partials/modals/modal_profiles.html`: Visual flag editor and inheritance indicators.
- `dashboard/frontend/js/profiles.js` & `app.js`: Onboarding state machine.

---

## Verification Plan

### Phase 1 Polish: Confirmation Modals & Dynamic Target Binding
- `dashboard/frontend/partials/modals/modal_confirm.html`:
  - Introduce themed, dark-mode confirmation modal to replace native browser `confirm(...)` dialogs.
- `dashboard/frontend/app.js`:
  - Add `confirmModal` reactive state and `askConfirmation({ title, message, confirmText, confirmClass, icon, onConfirm })` helper.
- `dashboard/frontend/js/logs.js` & `explorer_actions.js`:
  - Refactor `confirmResync()` and `syncCurrentDirectory()` to trigger `askConfirmation(...)`.
- `dashboard/frontend/js/remotes.js`:
  - Introduce `isRemoteActiveTarget(remote)` helper dynamically checking against `activeProfile.remote`.
- `dashboard/frontend/partials/view_remotes.html`:
  - Bind `ACTIVE TARGET` badge to `isRemoteActiveTarget(remote)`.
- `dashboard/frontend/partials/view_telemetry.html`:
  - Dynamically display `activeProfile.local_path` and `activeProfile.remote` in `Active Profile Topology` banner.

### Phase 1 Verification
- `go test -count=1 ./internal/profiles/...`: Verify scheduler running status checks.
- `go test -count=1 ./internal/server/...`: Verify `/api/telemetry` reports `execution_status` and exclude patterns forward to runner.
- `./scripts/check.sh templates`: Verify Alpine.js template safety.
- `./scripts/check.sh loc`: Verify LOC constraints (<= 400 lines).
- `./scripts/check.sh all`: Full 8-stage pre-flight quality check.


