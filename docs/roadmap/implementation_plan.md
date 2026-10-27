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

## 2. Phase 2: Core Refactoring & Subsystem Consolidation

### Proposed Changes
Introduce central Dependency Injection (DI) Container (`internal/app/container.go`), consolidate `internal/review/` domain into `internal/engine/review/`, and implement a recursive logger secret-redaction masking engine.

### Affected Components & Packages
- `internal/app/container.go`: Central application container wiring services.
- `internal/engine/review/`: Relocate review session manager and diff calculator.
- `internal/core/logger/redact.go`: Stream/line secret redactor for logs.

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


