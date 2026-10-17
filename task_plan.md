# Task Plan: Phase 2 - Go RC Client & Recursive Security Engine

## Goal
Port the Rclone Remote Control (RC) HTTP client (`internal/core/client`) and recursive secret redaction engine (`internal/core/security`) into Go with 100% test coverage.

## Tasks
- [x] Task 1: Recursive Secret Redaction Engine (`internal/core/security/security.go`)
  - **Scope**: Implement recursive data structure traversal (maps, slices, arrays, structs) with key substring matching (`PASS`, `SECRET`, `TOKEN`, `KEY`, `CREDENTIAL`, `AUTH`, `WEBHOOK`, `PRIVATE`) and value regex signatures (`ghp_*`, `xox*`, `ya29.*`).
  - **Verification**: `go test -v ./internal/core/security`
- [x] Task 2: Rclone RC HTTP Client (`internal/core/client/client.go`)
  - **Scope**: Port connection-pooled HTTP client with basic auth support covering `/core/stats` (health), `/sync/bisync` (trigger), `/job/status`, `/job/stop`, `/job/list`, `/config/listremotes`, `/config/get` (with safe key filtering), `/operations/about`, `/operations/list` (with canary detection), `/operations/touch`, `/operations/copyfile`, `/sync/copy`, `/operations/deletefile`, `/operations/purge`.
  - **Verification**: `go test -v ./internal/core/client`
- [x] Task 3: Quality Gate Verification
  - **Scope**: Run full pre-flight check `./scripts/check.sh` ensuring all 5 Go packages compile and pass tests, alongside Python tests and secret scanners.
  - **Verification**: `./scripts/check.sh`

## Status Updates
- 2026-10-17: Initialized Phase 2 implementation.
- 2026-10-17: Implemented `internal/core/security` with 4 test scenarios (basic, nested, regex, struct). Passed.
- 2026-10-17: Implemented `internal/core/client` with 4 test scenarios (health, bisync trigger, canary list, config sanitization). Passed.
- 2026-10-17: Executed `./scripts/check.sh`. 100% clean across Go and Python suites.


