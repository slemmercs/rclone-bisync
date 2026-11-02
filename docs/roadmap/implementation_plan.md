# Implementation Plan: Profile Configuration Redesign & Guided Flag Builder

## Proposed Changes

Redesign the Profile Configuration and Operational Flags experience in Bisync Hub (Active Strategic Priority #1). This replaces the unguided single-string escape hatch with an interactive, type-safe `+ Add Flag` row builder, fixes checkbox persistence and boolean serialization bugs for cloud providers, reclassifies provider configurations into distinct Tier 1 (curated) and Tier 2 (lean) interfaces, and provides a real-time synthesized CLI invocation preview.

---

### Affected Components & Packages

1. **Backend Flag Generation (`internal/profiles/`)**:
   - `builder.go`:
     - Fix boolean provider options serialization to emit explicit flags when disabled or enabled (e.g., `--drive-use-trash=false` when unchecked vs `--drive-use-trash` when checked).
     - Add curated support for `--drive-shared-with-me` (`drive_shared_with_me`).
     - Support explicit boolean serialization for OneDrive, Dropbox, and S3 flags.
   - `builder_test.go`:
     - Unit tests for explicit boolean flag generation (`"true"`, `"false"`, unset).
     - Tests for `--drive-shared-with-me`, `--drive-use-trash=false`, and combined arguments.
   - `validator.go`:
     - Validate both space-separated and equal-sign arguments (`--flag=value`) consistently.

2. **Frontend Architecture & Modules (`dashboard/frontend/js/`)**:
   - `flags_builder.js` (NEW):
     - Catalog of standard Rclone bisync flags (`--fast-list`, `--checkers`, `--transfers`, `--bwlimit`, `--tpslimit`, `--drive-shared-with-me`, `--checksum`, `--ignore-checksum`, `--size-only`, `--resilient`, `--recover`, etc.).
     - Parser: parses `custom_flags` string into structured row models `[{ id, flag, value, isBool, isCustom }]`.
     - Serializer: converts row models into sanitized POSIX-compliant CLI flag strings.
     - Live Preview generator: computes complete synthesized `rclone bisync <path1> <path2> [args...]` preview.
   - `profiles_modal.js`:
     - Integrate `flags_builder.js` for row management, parsing, and serialization.
     - Fix checkbox state persistence in `getProviderOption` / `setProviderOption`.
   - `index.html`:
     - Register `<script src="/js/flags_builder.js"></script>`.

3. **Dashboard Modals & Templates (`dashboard/frontend/partials/modals/`)**:
   - `modal_provider_options.html`:
     - Redesign Tier 1 UI: Curated toggles for Google Drive, OneDrive, S3, Dropbox with robust boolean checkboxes.
     - Redesign Tier 2 UI: Lean context notice for generic remotes (WebDAV, SFTP, Nextcloud, etc.).
     - Replace raw custom flags textarea with interactive `+ Add Flag` row builder:
       - Row controls: Flag dropdown, adaptive toggle/value input, delete button.
       - Quick add action: `+ Add Flag` button.
       - Real-time CLI preview card.
       - Inline prohibited flags validation warnings.
   - `modal_profiles.html`:
     - Streamlined layout, visual hierarchy, and section badges adhering to the design system.

---

### Technical Strategy & Data Flow

#### 1. Explicit Boolean Flag Serialization (Backend)
Rclone commands interpret `--flag` as enabling the flag, but when an option defaults to `true` (like `--drive-use-trash`), disabling it requires `--drive-use-trash=false`.
- If `providerOpts["drive_use_trash"] == "false"`, serialize `--drive-use-trash=false`.
- If `providerOpts["drive_use_trash"] == "true"` or `"1"`, serialize `--drive-use-trash`.
- If unset, omit (rely on rclone default or explicit configuration).

#### 2. Interactive Flag Builder Catalog & Data Flow (Frontend)
- Pre-populated popular rclone flags with types (`boolean` vs `value`) and units.
- Opening `modal_profiles.html` parses `custom_flags` string into structured rows via `FlagsBuilder.parseCustomFlags(...)`.
- Mutating rows updates `custom_flags` and recomputes the CLI preview in real time.
- Validates rows against prohibited flags blocklist in real time.

#### 3. Strict LOC & Modular Governance
- Keep `flags_builder.js` (<200 LOC) and `profiles_modal.js` (<300 LOC) strictly below the 400 LOC ceiling.
- Verify with `./scripts/check-loc.sh 400`.

---

### Safety & Error Handling

- **Prohibited Flags Enforcement**: Disallows restricted flags (`--config`, `-c`, `--rc`, `--use-mmap`, `--log-file`, `--json`) and highlights invalid rows.
- **Shell Injection Protection**: The builder handles POSIX escaping without arbitrary shell interpolation.
- **Backward Compatibility**: Existing profiles with string `custom_flags` seamlessly parse into rows; unknown flags map to `Custom...` without loss of user configuration.

---

## Verification Plan

### Automated Unit & Integration Tests
- `go test -v -count=1 ./internal/profiles/...` to verify `BuildCLIArgs`, explicit boolean flags, `--drive-use-trash=false`, and `--drive-shared-with-me`.
- `go test -count=1 ./cmd/check/...` and `./scripts/check.sh all` to verify all quality gates pass including LOC limits, HTML tokenizer, and license checks.

### Frontend & End-to-End Tests
- Verify modal rendering, row additions/removals, flag selection, and CLI preview in browser.
- Run headless Chrome E2E suite: `go test -v -count=1 ./tests/e2e/...`.








