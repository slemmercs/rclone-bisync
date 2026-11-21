# Implementation Plan: Universal Dynamic Remote Configuration Engine

## Proposed Changes

Implement a universal dynamic schema engine for all non-OAuth backends in Bisync Hub's Cockpit UI (`ModalRemoteWizard.vue`), powered directly by Rclone's `/config/providers` RC API.

### Affected Components & Packages

- `internal/models/provider.go`:
  - Add `Provider string` and `Hide int` to `ProviderOption`.
  - Add `Provider string` to `ProviderExample`.
- `internal/models/provider_test.go`:
  - Unit tests verifying JSON serialization and deserialization of provider options with flavor filters and hide bitmasks.
- `cmd/gendocs/`:
  - Regenerate `dashboard/frontend/src/types/models.gen.ts`.
- `dashboard/frontend/src/types/dedupe.ts`:
  - Update `ProviderOption` and `ProviderExample` to expose `Provider?: string` and `Hide?: number`.
  - Add `opt?: Record<string, any>` to `CreateRemotePayload`.
- `dashboard/frontend/src/components/DynamicProviderForm.vue`:
  - New component rendering dynamic input controls (`InputText`, `Password`, `Select`, `ToggleSwitch`).
  - Implements `matchesProviderFlavor` (with `!` negation and comma-separated tokens).
  - Implements `(opt.Hide & 3) === 0` filtering.
  - Implements exclusive dropdowns vs. open editable preset suggestions.
  - Collapsible Advanced Options accordion with badge counter.
- `dashboard/frontend/src/components/ModalRemoteWizard.vue`:
  - Replace static S3, B2, SFTP, and WebDAV blocks with `<DynamicProviderForm>`.
  - Pass `opt: { obscure: true }` when passwords are submitted.

### Technical Strategy & Data Flow

1. **Provider Discovery**:
   - `dedupeStore.fetchProviders()` retrieves cached providers from `GET /api/remotes/providers`.
2. **Dynamic Rendering**:
   - For non-OAuth providers, Step 2 mounts `<DynamicProviderForm>`.
   - If provider is `s3`, the `provider` option renders first as the S3 Flavor picker (58 presets).
   - Changing the flavor reactively triggers `matchesProviderFlavor()` across all other options.
3. **Payload Sanitization & Obscuring**:
   - Only user-modified, non-empty parameters are serialized.
   - If any `IsPassword: true` field has a value, set `opt: { obscure: true }` in `CreateRemotePayload`.

### Safety & Error Handling

- **Bitmask Filtering**: Enforces `(opt.Hide & 3) === 0` to prevent CLI configurator internal state tokens from surfacing.
- **Credential Protection**: Masks all `IsPassword: true` and `Sensitive: true` fields.
- **Input Sanitization**: Empty strings and unchanged defaults are stripped to prevent bloat in `rclone.conf`.

## Verification Plan

- Go backend tests:
  - `go test -count=1 ./internal/models/...`
  - `go test -count=1 ./internal/services/... -run TestRemoteWizard`
- Frontend type safety & build:
  - `make generate-ts`
  - `npm --prefix dashboard/frontend run type-check`
  - `npm --prefix dashboard/frontend run build`
- Full CI pre-flight check:
  - `./scripts/check.sh all`






