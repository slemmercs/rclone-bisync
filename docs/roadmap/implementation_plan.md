# Implementation Plan: Deduplication Output UX, OneDrive Backend Safeguards & Restored Item Visibility

## Proposed Changes

Improve deduplication output formatting across the Audit Ledger drawer and the Deduplication modal, add OneDrive-specific deduplication safeguards (auto-detecting name-vs-hash support and excluding locked Personal Vault), and expose concrete file/folder paths on expanded `DELETION_RESTORED` timeline log rows.

### Affected Components & Packages

- `internal/models/api.go`:
  - Add `Object string json:"object,omitempty"` to `models.AuditLogRecord`.
- `internal/services/audit_service.go`:
  - When recording an event, extract `item_path`, `path`, or `object` from `details` map and assign it to `logEntry.Object` so mirror logs in `bisync.log` retain the item target.
- `internal/core/client/fs_ops.go`:
  - In `Dedupe()`, if the target remote or target path targets OneDrive (`onedrive:` or type `onedrive`), add `--exclude "Personal Vault/**"` so locked vault directories never cause rclone listing failures.
- `dashboard/frontend/js/utils.js`:
  - In `parseJsonLogLine()`: populate `entry.object = parsed.object || (parsed.details && (parsed.details.item_path || parsed.details.path || parsed.details.object)) || null`.
  - Add `parseDedupeOutput(rawOutput)` helper to parse rclone dedupe output into summary metrics (collisions found, deleted copies, status) and structured collision items with file/directory tags.
- `dashboard/frontend/js/remotes.js`:
  - In `openDedupeModal()`: check the remote provider type (e.g. `onedrive` vs `drive`). If the remote is `onedrive` or does not support same-folder duplicate names, default `byHash` to `true` and note that OneDrive requires hash-based deduplication.
- `dashboard/frontend/partials/modals/modal_dedupe.html`:
  - Rename title from `REMOTE HASH DEDUPLICATION` to `REMOTE DEDUPLICATION`.
  - Update subtitle: `Resolve file & directory duplicate collisions using native Rclone dedupe`.
  - Replace the plain `<pre>` execution box with the structured dedupe component (summary badges, collision file cards with copy actions, and toggle for raw console view).
  - Add backend guidance callout when remote is OneDrive.
- `dashboard/frontend/partials/components/ui_components.html`:
  - In `component_entity_drawer` when `entityDrawer.type === 'audit'`:
    - If `entityDrawer.data?.action === 'DEDUPE_EXECUTED'` or `val` key is `output`, render the structured dedupe output component instead of raw text.
- `dashboard/frontend/partials/patterns/pattern_timeline_ledger.html`:
  - In expanded cluster rows (`subEntry in cluster.entries`), display `subEntry.object` (the restored file/folder path) alongside the timestamp and action badge instead of repeating generic messages.

### Safety & Error Handling

- OneDrive `Personal Vault` exclusion prevents API failures on locked Microsoft 2FA vaults.
- Deduplication parsing is non-destructive and defensive; if the output cannot be structured, it gracefully renders the raw console output.
- All file operations in `local` storage continue strictly enforcing path confinement.

## Verification Plan

- `go test -count=1 ./internal/core/client/...`
- `go test -count=1 ./internal/server/...`
- `go test -count=1 ./internal/services/...`
- `./scripts/check.sh all` (all 6 quality stages)
- Deploy to port 8085 and verify visually in browser.



#### 3. Correlated Daemon Log Fixes (`modal_daemon.html`)
- Correct object reference in comparison settings loop to `cluster.entry.details`.
- In expanded duplicate rows: `<span class="ds-mono text-xs text-indigo-400" x-text="subEntry.object || subEntry.msg"></span>`.
- Filter source paths: `<span x-show="isDevMode && entry.source" ...>`.
- Count badges: dynamically calculate `daemonLogs.filter(e => e.level === 'error').length`, etc.

#### 4. Reverse Unified Ledger (`logs.js`)
- In `fetchBisyncLogs(page)` or rendering pipeline, sort newest first.
- In `clusterLogEntries(entries)`: set cluster timestamp to `latestEntry.timestamp` (the max timestamp in the cluster) rather than the first.

#### 5. Drawer Stacking (`design-tokens.css` & `ui_components.html`)
- Adjust `.ds-drawer-overlay` to `z-index: 40; backdrop-filter: blur(4px);`
- Adjust `.ds-drawer-panel` to `z-index: 50;`
- Ensure child elements are not rendered under backdrop-filter.

---

### Safety & Error Handling

- **Deduplication Safety**: Remote deduplication acquires an exclusive file lock (`rclone-bisync.lock`) to prevent concurrent sync operations from corrupting baseline state.
- **Circuit Breaker Protection**: Following deduplication, an automated baseline sync (`rclone bisync --resync`) is queued to ensure `TOO_MANY_DELETES` circuit breaker does not trigger.
- **Sub-400 LOC**: All touched Go files and JS files remain strictly under 400 LOC or decomposed as needed.

---

## Verification Plan

- `go test -count=1 ./internal/services/... -run TestDeletion`
- `go test -count=1 ./internal/server/...`
- `./scripts/check.sh templates`
- `./scripts/check.sh loc`
- `./scripts/check.sh all`

---

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

## 3. Phase 3: Operational Flags & Profile Schema Alignment (Active Scope)

### Proposed Changes
Implement the 3-tier operational flags architecture (`docs/design/bisync_operational_flags.md`), flag inheritance (`Execution Override > Profile Override > Global Default > Hardcoded Fallback`), POSIX shell argument sanitizer with prohibited flag security blocklist, and profile CLI argument builder.

### Affected Components & Packages
- `internal/config/dto.go`: Operational flags schema, `GlobalDefaults` DTO with canonical defaults.
- `internal/profiles/model.go`: Integration of `ProfileFlags`, `ResolvedFlags`, and `ExecutionOverrides` in `Profile`.
- `internal/profiles/validator.go`: Prohibited flag blocklist, POSIX shell quote tokenizer (`shlex` equivalent), argument safety validation.
- `internal/profiles/builder.go`: Flag inheritance engine and CLI argument generator (`BuildCLIArgs`).
- `internal/app/container.go`: Integration of `BuildCLIArgs` and global defaults resolution into profile runner instantiation.
- `internal/server/handlers_profiles.go` & `handlers_config.go`: Expose resolved flags and global operational defaults to API clients.

### Technical Strategy & Data Flow
1. **Tier Architecture & Governance**:
   - Tier 1 (Execution Controls): DryRun, Resync, Force, ConflictResolve, MaxDelete.
   - Tier 2 (Profile Settings): CanaryValidation, CompareMode, Directory Cleanup (RemoveEmptyDirs, CreateEmptyDirs), ExcludePatterns, FiltersFilePath.
   - Tier 3 (Custom Flags Escape Hatch): Prohibited flag blocklist validation, POSIX shell tokenization, strict argument slice production.
2. **Inheritance & Priority**:
   `Execution Overrides > Profile Overrides (*T pointers) > Global Defaults > Hardcoded Defaults`.
3. **Safety & Security Blocklist**:
   - Custom flags parsed as discrete tokens without shell invocation.
   - Blocklist strictly protects internal state: `--config`, `--use-mmap`, `--rc`, `--rc-addr`, `--rc-user`, `--rc-pass`, `--log-file`, `--json`.
4. **Backward Compatibility**:
   - Unset profile flags (`nil`) automatically inherit global defaults without data migration breakage.

### Verification Plan
- Unit tests in `internal/config/` for GlobalDefaults serialization and defaults.
- Unit tests in `internal/profiles/` for validator (shlex parsing, blocklist enforcement, injection prevention).
- Unit tests in `internal/profiles/` for inheritance resolution and `BuildCLIArgs`.
- Full quality gate pre-flight check via `./scripts/check.sh all`.

---

## 4. Phase 4: Hybrid Fleet Overview & Operational Flags UI

### Proposed Changes
Deliver the workload-centric dashboard hierarchy specified in `docs/design/hybrid_fleet_overview_contextual_profile_workspace.md` and the visual operational flags editor from `docs/design/bisync_operational_flags.md`:
1. **Level 1: Fleet View (`dashboard/frontend/partials/view_fleet.html`)**:
   - Central command dashboard displaying fleet health summary counters (`Total`, `Healthy`, `Needs Review`, `Failing`, `Syncing`) with interactive click-to-filter pills.
   - Interactive profile tile grid with status tags, schedule type, local/remote topology, operational telemetry (last run, duration, next run), and inline quick actions (`Sync Now`, `Review Diff`, `Settings`, `Logs`).
   - Search and sort controls (`Search profiles...`, `Sort by Status / Name`).
2. **Level 2: Scoped Header Context Switcher & Breadcrumb Navigation**:
   - Persistent global profile switcher dropdown in header replacing the static pill, featuring micro-status indicators (`● Healthy`, `⚠ Review`, `✖ Failed`).
   - Breadcrumb navigation (`Profiles / [Profile Name]`) allowing instant one-click return to Level 1 Fleet View.
   - Dynamic binding across all child tabs (`Engine & Telemetry`, `Storage Remotes`, `File Explorer`, `Audit Ledger`) to the active profile context.
3. **Visual 3-Tier Operational Flags Editor (`dashboard/frontend/partials/modals/modal_profiles.html`)**:
   - **Tier 1 (Execution Controls)**: `DryRun`, `Resync`, `Force`, `ConflictResolve` strategy dropdown, `MaxDelete` percentage input.
   - **Tier 2 (Profile Settings)**: `CanaryValidation` toggle, `CompareMode` dropdown (`size,modtime`, `checksum`, `modtime`), Directory Cleanup (`RemoveEmptyDirs`, `CreateEmptyDirs`), `FiltersFilePath`.
   - **Tier 3 (Custom Flags Escape Hatch)**: Freeform text input with client-side & server-side validation against the Phase 3 prohibited flags blocklist.
   - **Inheritance Badges**: Clear visual badges (`Inherited from Global Default` vs `Profile Override`) displaying global default values when a field is null/inherited.

### Affected Components & Packages
- `dashboard/frontend/partials/view_fleet.html`: New Level 1 Fleet View partial.
- `dashboard/frontend/index.html`: Registration of `view_fleet` partial, nav tab pill, and breadcrumb header.
- `dashboard/frontend/partials/modals/modal_profiles.html`: 3-tier operational flags editor controls and inheritance indicators in Tab 2.
- `dashboard/frontend/js/profiles.js`: Fleet summary calculations, filtering/sorting, flag overrides binding, and custom flags validation.
- `dashboard/frontend/app.js`: Tab routing update (default tab logic, breadcrumb transitions, fleet store state).
- `internal/server/server.go`: Template parsing inclusion for `view_fleet.html`.

### Technical Strategy & Data Flow
1. **Fleet Store & Aggregate State**:
   - Fleet summary metrics computed dynamically from `profilesData.list` and active worker telemetry.
   - Filter state (`filterStatus = 'all' | 'healthy' | 'review' | 'failing' | 'syncing'`) filters the tile grid in real time.
2. **Context Routing**:
   - Clicking any profile tile card in Level 1 sets `activeId` and switches tab to `telemetry` (Level 2 Scoped Workspace).
   - Breadcrumbs in header detect whether user is in Fleet View or Scoped Workspace and provide easy navigation.
3. **3-Tier Flag Binding & Inheritance**:
   - In `modal_profiles.html`, profile flags map to `profileModal.editing.flags`.
   - Each input evaluates `profileModal.editing.flags?.[key]` vs `globalDefaults?.[key]`.


---

## 5. UI/UX Refinement: Contextual Fleet vs. Workload Scoping & Header Redesign

### Proposed Changes
Refine the Phase 4 UI/UX based on industry best practices for multi-tenant and multi-workload architectures (Vercel, Portainer, Supabase):
1. **Clean 2-Tier Header**:
   - Eliminate wrapping and layout squeeze by separating global context (top bar) from tab navigation (sub-bar).
   - Top Bar: App brand, integrated breadcrumb context switcher (`Bisync Hub | ⊞ Fleet / [ Active Profile ▾ ] [Status]`), and global daemon health/latency.
   - Remove the long subtitle taking up horizontal navbar space.
2. **Context-Aware Navigation & Subnav**:
   - Level 1 (Fleet Scope): Nav tabs for `[ ⊞ Workload Fleet ]` and `[ ☁ Cloud Remotes Inventory ]`.
   - Level 2 (Workload / Profile Scope): Nav tabs for `[ ⚡ Engine & Telemetry ]`, `[ ☁ Target Storage ]`, `[ 📁 File Explorer ]`, `[ 🛡 Audit Ledger ]`.
   - Persistent view preference: `localStorage` active tab caching with smart default (Fleet when $\ge 2$ profiles, Telemetry when single profile).
3. **Fleet Card Interactive Surface**:
   - Fix premature tag closure in `view_fleet.html` so the entire card is clickable, immediately activating that profile and opening its scoped telemetry workspace.
4. **Contextual Scoping for Remotes and Audit Ledger**:
   - Remotes: Global inventory at Fleet level; filtered to active profile's local directory and remote target when inside a Profile workspace.
   - Audit Ledger: Auto-filter by `?profile_id=<active_id>` when in Profile workspace, with a 1-click toggle to clear the filter and show all system events.

### Verification Plan
- Template AST validation (`./scripts/check.sh templates`)
- Line counter audit (`./scripts/check.sh loc`)
- Headless Chrome E2E test verifying header navigation, fleet tile clicks, and context transitions (`go test -v -count=1 ./tests/e2e -run TestDashboard_HeadlessChromeSmoke`)
- Full pre-flight quality check (`./scripts/check.sh all`)

   - Reset button allows clearing a profile override back to `nil` to re-inherit global defaults.
4. **Safety & Security**:
   - Custom flags input triggers validation on change, rendering clear inline error badges if prohibited flags (`--config`, `--rc`, etc.) are detected.

### Verification Plan
- `go test -count=1 ./internal/server/...`: Verify template rendering and profile endpoints.
- `./scripts/check.sh templates`: Verify AST-tokenized Alpine.js template validity.
- `./scripts/check.sh loc`: Verify LOC budget (<= 400 lines) across all modified files.
- Manual DOM reactivity & browser verification of Fleet View and Flags Editor.
- `./scripts/check.sh all`: Run full quality gate pre-flight check.

---

## 6. Phase 5: Tiered Cloud Provider Architecture, Provider Options UI & Global Fleet Settings

### Proposed Changes
1. **Engine Decoupling**:
   - Remove Google Drive-specific flags (`--drive-export-formats`, `--drive-skip-gdocs`, `--drive-use-trash`, `--drive-acknowledge-abuse`, `--drive-skip-shortcuts`) from universal `defaultArgs` in `internal/config/config.go` and `bisync.env`.
   - Ensure global defaults are strictly provider-agnostic.
2. **Provider Options Modeling & Compiler Translation**:
   - Extend `ProfileFlags` with `ProviderOptions map[string]string` in `internal/profiles/model.go`.
   - Update `internal/profiles/builder.go` to emit provider options as CLI flags.
   - Register Tier 1 provider flags in `internal/core/compiler/compiler.go` (`--onedrive-no-versions`, `--s3-chunk-size`, `--s3-storage-class`, `--dropbox-chunk-size`, etc.) so `CompileBisyncPayload` attaches them directly to remote `path1` connection strings.
3. **Unified Provider Options UI**:
   - Consolidate provider configuration and the audited custom flags field into a single **Target Provider Configuration** panel in `modal_profiles.html`.
   - Tier 1 Providers (Google Drive, OneDrive, S3, Dropbox) get first-class typed UI controls + custom flags field for advanced arguments.
   - Tier 2 Generic Providers (WebDAV, SFTP, Box, B2, etc.) get clear guidance with the audited custom flags field as their primary interface.
4. **Profile Execution Contract in Telemetry**:
   - Replace raw host environment dump in `view_telemetry.html` with a concise **Profile Execution Contract** card displaying Source & Target paths, Remote Provider badge, schedule status, effective resolved flags pill cloud, and a 1-click **Configure Profile** shortcut.
5. **Global Fleet Settings Modal**:
   - Introduce `modal_global_settings.html` and endpoints (`GET /api/config`, `PUT /api/config`) to manage fleet-wide operational defaults, daemon polling/timeout parameters, and alert notifications.

### Affected Components & Packages
- `internal/config/config.go` & `bisync.env`: Strip provider-specific flags from universal daemon default arguments.
- `internal/core/compiler/compiler.go`: Map Tier 1 provider flags to `path1` backend options.
- `internal/profiles/model.go`: Add `ProviderOptions map[string]string` to `ProfileFlags` and `ResolvedFlags`.
- `internal/profiles/builder.go`: Translate `ProviderOptions` into CLI arguments in `BuildCLIArgs`.
- `internal/server/handlers_config.go` & `server.go`: Expose `GET /api/config` and `PUT /api/config` for global fleet settings.
- `dashboard/frontend/partials/modals/modal_profiles.html` & `profiles.js`: Unified Provider Options section with dynamic Tier 1 UI fields + audited custom flags.
- `dashboard/frontend/partials/view_telemetry.html`: Profile Execution Contract replacing raw `.env` dump.
- `dashboard/frontend/partials/modals/modal_global_settings.html`: Fleet settings manager.

### Verification Plan
- Unit tests: `go test -count=1 ./internal/config/... ./internal/profiles/... ./internal/core/compiler/... ./internal/server/...`
- Template verification: `./scripts/check.sh templates`
- LOC budget verification: `./scripts/check.sh loc 400`
- E2E smoke tests: `go test -v -count=1 ./tests/e2e -run TestDashboard_HeadlessChromeSmoke`
- Full pre-flight quality gate: `./scripts/check.sh all`



