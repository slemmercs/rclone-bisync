# Task Plan: Fix HasherWrapRequest Hashes Payload Type Compatibility

## Goal
Resolve JSON unmarshaling type collision for `HasherWrapRequest.hashes` between frontend array and backend string representation, ensuring seamless remote wrapping from Hasher Console Modal.

## Tasks

- [x] Task 1: Go Backend Flexible Unmarshaling & Normalization
  - **Scope**: `internal/models/hasher.go`, `internal/models/hasher_test.go`
  - **Verification**: `go test -count=1 -v ./internal/models -run TestHasherWrapRequest`

- [ ] Task 2: Frontend Hasher Manager Modal Contract & Options Alignment
  - **Scope**: `dashboard/frontend/src/types/hasher.ts`, `dashboard/frontend/src/components/ModalHasherManager.vue`
  - **Verification**: `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build`

- [ ] Task 3: Full Quality Verification Suite
  - **Scope**: Pre-flight verification across all stages
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-11-22: Completed Task 1. Added `UnmarshalJSON` to `HasherWrapRequest` in `internal/models/hasher.go` accepting both string and array for `hashes`, with hash identifier normalization. Added unit tests in `internal/models/hasher_test.go`. Verified with `go test -count=1 -v ./internal/models -run TestHasherWrapRequest`.

- 2026-11-22: Initialized implementation plan and task plan to fix HasherWrapRequest hashes deserialization error.







