# Implementation Plan: Epic 4 — Wizard Virtual Remote Guardrails & Hasher Acceleration

## Proposed Changes

Implement contextual safeguards, canonical overlay chaining, and Hasher acceleration within the Remote Creation Wizard:
1. Prevent inverted and invalid virtual overlays by disallowing wrapping virtual providers (`hasher`, `crypt`, `union`, `combine`, `alias`) in `crypt` or `hasher`.
2. Introduce Hasher acceleration (`[x] Accelerate with Hasher`) for standard cloud, object, and protocol remotes (SFTP, WebDAV, SMB, local).
3. Enforce the canonical overlay pipeline: $\text{Base} \longrightarrow \text{Base\_crypt} \longrightarrow \text{Base\_crypt\_hasher}$.
4. Prevent subpath double-nesting by baking `RemoteSubpath` into the inner overlay target and clearing it on outer layers and the linked sync profile.
5. Require non-empty bucket subpath when wrapping bucket-based providers (`s3`, `b2`, `gcs`, `azureblob`, `swift`) in virtual overlays.
6. Guard against `union` and `combine` as targets for bidirectional synchronization (`bisync`), displaying clear warning banners and disabling linked profile creation.

### Affected Components & Packages

- `internal/models/provider.go`: Add `AccelerateHasher bool` and `HasherHashes []string` to `CreateRemoteBridgeRequest`.
- `internal/services/remote_wizard_service.go`: Inject `Hasher *HasherService`, validate bucket requirements, enforce overlay order (`Base -> Crypt -> Hasher`), prevent union/combine profile creation, and bake subpath into inner layer.
- `internal/app/container.go`: Wire `c.RemoteWizardService.SetHasherService(c.HasherService)`.
- `internal/services/remote_wizard_service_test.go`: Add unit tests for dual-wrap chaining, hasher-only wrap, bucket subpath validation, and union/combine profile guard.
- `cmd/gendocs/`: Regenerate AST TypeScript definitions (`dashboard/frontend/src/types/models.gen.ts`) and `site/technical-reference.md`.
- `dashboard/frontend/src/types/remotes.ts`: Add `isVirtualProvider` and `isBisyncIncompatible` helpers, extend `CreateRemotePayload` and `WizardConfigForm`.
- `dashboard/frontend/src/components/wizard/WizardStepConfigure.vue`: Render Hasher acceleration checkbox, contextual overlay hiding, pipeline visualization badge, and union/combine warning banner.
- `dashboard/frontend/src/components/wizard/WizardStepReview.vue`: Display Hasher acceleration in review summary table and resolved profile target remote.

### Technical Strategy & Data Flow

- The remote wizard service receives `CreateRemoteBridgeRequest`.
- If provider is `union` or `combine` and `CreateProfile` is true, return an error or skip profile provisioning.
- If provider is bucket-based (`s3`, `b2`, `gcs`, `azureblob`, `swift`) and `EncryptRemote` or `AccelerateHasher` is true, require non-empty `RemoteSubpath`.
- Inner overlay absorbs `RemoteSubpath`: `cryptTarget = cleanName + ":" + targetRemoteSubpath`, then `targetRemoteSubpath = ""`.
- If `AccelerateHasher` is true, wrap the current target (`cleanName + "_crypt"` or `cleanName`) using `HasherService.WrapRemoteInHasher`.
- Any linked sync profile targets the final outermost layer (`targetRemoteForProfile`).

### Safety & Error Handling

- Bucket protection prevents orphaned top-level buckets and Rclone overlay initialization failures.
- Non-destructive configuration creation via Rclone RC `/config/create`.
- Backend LOC strictly maintained under ~400 LOC ceiling.

## Verification Plan

- `go test -count=1 ./internal/services -run TestRemoteWizardService`
- `go test -count=1 ./internal/...`
- `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build`
- `./scripts/check.sh all`
