# Implementation Plan: Storage Remote Virtual Overlay Linkage, Badging, Action Guardrails, and Plaintext Decryption Guidance

## Proposed Changes

Address virtual remote overlay blind spots across the storage remotes view and profile management:
1. **Backend Telemetry Overlay Graph**: Enhance `TelemetryService.GetRemotesSummaryCached` (`internal/services/telemetry_remotes.go`) to parse `cfg["remote"]` on overlay remotes (`crypt`, `hasher`, `alias`), resolve underlying base remotes, and attach bidirectional overlay mappings (`underlying_remote`, `underlying_path`, `crypt_overlay`, `hasher_overlay`, `overlays`).
2. **Storage Remote Tile Badging & Action Guardrails (`RemoteCard.vue`)**:
   - On base remotes: Display `[Encrypted: remote_crypt]` cyan badge and `[Accelerated: remote_hasher]` amber badge.
   - Replace/disable `[Encrypt]` button with `[Encrypted ✓]` when a crypt overlay already exists, preventing duplicate creation collisions (`remote "xxx_crypt" already exists`).
   - If a hasher overlay already exists, route the `[Hasher]` button directly to Hasher Console/Maintenance instead of the wrap modal.
   - On virtual overlay remotes: Display `Underlying: <base_remote:subpath>` instead of generic `"Configured storage backend"`.
   - On deleting a crypt remote: Show an explicit blast-radius safeguard reminder explaining that cloud files remain ciphertext.
3. **Decryption & Reversion Education (`ModalCryptWrap.vue`, `WizardStepConfigure.vue`)**:
   - Clarify the Rclone encryption model: Local filesystem always remains plaintext; cloud storage stores opaque encrypted blobs.
   - Provide clear reversion guidance: Decryption is achieved by pausing bisync, deleting cloud ciphertext, repointing the profile to the base unencrypted remote, and running a resync (`--resync`) from the local plaintext files.
   - Prevent attempting to wrap a remote that already has a crypt overlay in `ModalCryptWrap.vue`.
4. **Profile Pipeline & Encryption Badges (`ProfileListTab.vue`, `ProfileFormTab.vue`)**:
   - In profile list: Render an `[ENCRYPTED]` badge (`pi-shield` in cyan) and `[ACCELERATED]` badge (`pi-bolt` in violet), with the pipeline topology breadcrumb (`profile -> remote_crypt -> base_remote`).
   - In profile edit form: Display pipeline details for encrypted remotes and warning when repointing between encrypted and plaintext remotes.

### Affected Components & Packages

- `internal/services/telemetry_remotes.go`: Parse `cfg["remote"]`, build overlay dependency graph, and attach `underlying_remote`, `crypt_overlay`, `hasher_overlay`, `overlays` to remote items.
- `internal/services/telemetry_service_test.go`: Unit test overlay linkage in telemetry summary.
- `dashboard/frontend/src/types/remotes.ts`: Add overlay linkage fields to remote interface.
- `dashboard/frontend/src/components/remotes/RemoteCard.vue`: Render overlay badges, disable duplicate wrap actions, display underlying storage path, and show crypt deletion warnings.
- `dashboard/frontend/src/components/ModalCryptWrap.vue`: Add existing overlay validation and plaintext/decryption reversion callout.
- `dashboard/frontend/src/components/profiles/ProfileListTab.vue`: Add `[ENCRYPTED]` and `[ACCELERATED]` badges and pipeline breadcrumbs.
- `dashboard/frontend/src/components/profiles/ProfileFormTab.vue`: Add encryption pill and resync advisories when switching remote types.

### Technical Strategy & Data Flow

- In `TelemetryService.GetRemotesSummaryCached`:
  - Pass 1: Query Rclone `/config/listremotes` and `/config/get` for each remote. Extract `rtype := cfg["type"]` and `remoteTarget := cfg["remote"]`.
  - Pass 2: For any remote with non-empty `remoteTarget` (e.g. `crypt`, `hasher`, `alias`), isolate the base remote name `baseName := strings.Split(remoteTarget, ":")[0]`.
  - Pass 3: In the returned `remoteData` list, enrich the overlay remote with `"underlying_remote": baseName` and `"underlying_path": remoteTarget`. Enrich the base remote with `"crypt_overlay": cryptName`, `"hasher_overlay": hasherName`, and `"overlays": [...]`.
- In Vue frontend:
  - `RemoteCard.vue` reads `remote.crypt_overlay` and `remote.hasher_overlay`.
  - `ProfileListTab.vue` computes remote metadata for each profile via `remotesStore.remotes`.

### Safety & Error Handling

- Read-only enrichment of `/api/remotes` response; does not alter Rclone config directly.
- Prevents confusing Rclone API error responses by validating existing overlay names client-side before dispatching `/api/remotes/crypt/wrap`.
- Preserves Go ~400-500 LOC standards.

## Verification Plan

- `go test -count=1 ./internal/services -run TestTelemetry`
- `go test -count=1 ./internal/...`
- `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build`
- Pre-flight quality suite: `./scripts/check.sh all`

