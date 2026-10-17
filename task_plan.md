# Task Plan: Phase 1 - Go Core Engine & Configuration Loader

## Goal
Establish the foundational Go module, port the environment/secret configuration parser, port the POSIX flock lock manager with PID liveness tracking, and port the desktop notification dispatcher (`notify-send` / webhooks).

## Tasks
- [x] Task 1: Bootstrap Go Toolchain & Module
  - **Scope**: Ensure Go 1.22+ is available in the environment, initialize `go.mod` with module `github.com/slemmercs/rclone-bisync`. Setup basic directory layout (`internal/config`, `internal/core/lock`, `internal/core/notifier`).
  - **Verification**: `go version` and `go build ./...`
- [x] Task 2: Port Configuration Loader (`internal/config/config.go`)
  - **Scope**: Implement `.env` and `rc-secrets.env` parsing, dynamic search directory traversal, fail-fast validation for mandatory `RC_PASS`, defaults for all bisync options (`CHECK_ACCESS`, `MAX_DELETE_RATIO`, etc.).
  - **Verification**: Unit tests in `internal/config/config_test.go` (`go test ./internal/config/...`)
- [x] Task 3: Port POSIX Advisory Lock Manager (`internal/core/lock/lock.go`)
  - **Scope**: Port `fcntl.flock` semantics using `golang.org/x/sys/unix` (or `syscall`), PID recording, stale PID detection via `kill -0` (`unix.Kill(pid, 0)`), probe execution, and safe cleanup.
  - **Verification**: Unit tests in `internal/core/lock/lock_test.go` (`go test ./internal/core/lock/...`)
- [x] Task 4: Port Notification Dispatcher (`internal/core/notifier/notifier.go`)
  - **Scope**: Implement `NotificationDispatcher` supporting `notify-send` subprocess invocation with icon path resolution (`cloud-sync-light.svg`), urgency handling, and HTTP webhook dispatch.
  - **Verification**: Unit tests in `internal/core/notifier/notifier_test.go` (`go test ./internal/core/notifier/...`)

## Status Updates
- 2026-10-17: Transitioned active roadmap to Go Single-Binary Architecture. Created Phase 1 task plan.
- 2026-10-17: Bootstrapped Go toolchain (go 1.26/1.22), initialized `go.mod`, implemented `internal/config`, `internal/core/lock`, and `internal/core/notifier` with comprehensive unit tests. Integrated `go test ./...` into `./scripts/check.sh`. 100% tests passing (all Go unit tests + 77 Python tests).









