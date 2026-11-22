# Implementation Plan: Fix HasherWrapRequest Hashes Payload Type Compatibility

## Proposed Changes

Resolve deserialization error when wrapping remotes with Hasher (`json: cannot unmarshal array into Go struct field HasherWrapRequest.hashes of type string`):
1. **Go Backend Payload Flexibility (`internal/models/hasher.go`)**:
   - Implement `UnmarshalJSON` on `models.HasherWrapRequest`.
   - Support both string (`"md5,sha1"`) and array of strings (`["md5", "sha1"]` or `["MD5", "SHA-1"]`).
   - Automatically normalize hash names to lowercase standard Rclone identifiers (`md5`, `sha1`, `dropbox`, `quickxor`, `whirlpool`, `crc32`).
   - Add unit test suite in `internal/models/hasher_test.go` covering string, array, empty, and mixed case inputs.
2. **Frontend Contract & Modal Alignment (`ModalHasherManager.vue`, `types/hasher.ts`)**:
   - Update `types/hasher.ts` so `hashes?: string[] | string`.
   - Update `ModalHasherManager.vue` to map UI display labels (`MD5`, `SHA-1`, `DropboxHash`, `QuickXorHash`, `Whirlpool`, `CRC-32`) to standard rclone identifiers (`md5`, `sha1`, `dropbox`, `quickxor`, `whirlpool`, `crc32`).
3. **End-to-End Verification**:
   - Run unit tests `go test -count=1 ./internal/models -run TestHasherWrapRequest`.
   - Run handler tests `go test -count=1 ./internal/server -run TestHasher`.
   - Run service tests `go test -count=1 ./internal/services -run TestHasher`.
   - Run frontend type check and production build `npm --prefix dashboard/frontend run build`.
   - Run full check suite `./scripts/check.sh all`.

### Affected Components & Packages

- `internal/models/hasher.go`: Flexible `UnmarshalJSON` handling string or array of strings.
- `internal/models/hasher_test.go`: Unit tests for JSON unmarshaling of `HasherWrapRequest`.
- `dashboard/frontend/src/types/hasher.ts`: Union type `string[] | string` for `hashes`.
- `dashboard/frontend/src/components/ModalHasherManager.vue`: Normalize hash options and values.

### Technical Strategy & Data Flow

- When `ModalHasherManager.vue` submits `wrapHasher`:
  - `selectedHashes` contains `['md5', 'sha1']`.
  - Backend deserializes payload via `HasherWrapRequest.UnmarshalJSON`, turning array `["md5", "sha1"]` or string `"md5,sha1"` into comma-separated `"md5,sha1"` expected by `HasherService.WrapRemoteInHasher` and Rclone RC.

### Safety & Error Handling

- Zero breaking changes for existing callers (e.g. `remote_wizard_service.go` passes comma-separated string; UI passes array; both work).
- Defense-in-depth: Malformed elements in array are skipped; valid names are trimmed and normalized.

## Verification Plan

- `go test -count=1 ./internal/models -run TestHasher`
- `go test -count=1 ./internal/server -run TestVirtualRemotes`
- `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build`
- `./scripts/check.sh all`

