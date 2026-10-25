# Implementation Plan: Native Go CLI Quality Gate (`cmd/check`) with Hybrid Wrapper

## Proposed Changes

Migrate repository quality gate auditing from Bash (`scripts/check.sh`) to a high-performance, concurrent Go CLI (`cmd/check`). Replace fragile line-based regexes with a robust HTML attribute scanner, run independent fast checks concurrently, and provide a lightweight shell wrapper (`scripts/check.sh`) for seamless backward compatibility.

### Affected Components & Packages

- `cmd/check/main.go`:
  - New Go entrypoint implementing quality gate orchestration:
    - Stage routing: `all`, `fmt`, `lint`, `templates`, `loc`, `test`, `build`.
    - CLI flags: `-v` / `--verbose`, `-loc <int>`, `-min-coverage <float>`, `-race`, `-target <path>`, `-coverage-file <path>`.
    - Parallel execution: fast checks (`fmt`, `templates`, `loc`) execute concurrently using goroutines; heavy checks (`lint`, `test`, `build`) execute sequentially.
    - Robust HTML attribute tokenizer: handles multi-line expressions, single/double quotes, and ignores HTML comments.
    - Precise LOC counter: scans `.go` (excluding `_test.go`), `.js`, and `.html` files.
    - Formatter & shell script validator (`gofmt` + `bash -n`).
    - Linter & security suite (`go vet`, `staticcheck`, `govulncheck`, `betterleaks`, `esbuild`, `tidy`).
    - Coverage validator: executes `go test` with coverage and verifies minimum statement coverage threshold (80.0%).
    - Cross-compiler: builds native and Linux target binaries (`linux/amd64`, `linux/arm64`).
- `cmd/check/main_test.go`:
  - Unit tests covering the HTML tokenizer, LOC counter, and configuration argument parser.
- `scripts/check.sh`:
  - Refactored into a thin delegator executing `exec go run "${BASH_SOURCE[0]%/*}/../cmd/check" "$@"`.

### Technical Strategy & Data Flow

- **True HTML Tokenization**: Scan tags, attributes, and values across newlines. Ignore text inside `<!-- ... -->`. Detect dotted boolean attributes (`:(disabled|checked|readonly|hidden|required|selected)="*. *"`) safely.
- **Concurrent Non-Blocking Execution**: `filepath.WalkDir` audits run in parallel goroutines, dropping check overhead from hundreds of milliseconds to <30ms.
- **Deterministic Output & Exit Codes**: Stage results maintain deterministic ordering and standard POSIX exit codes (0 for pass, 1 for fail).

### Safety & Error Handling

- **Zero Global Pollution**: Pure Go single binary / `go run` workflow with zero third-party dependencies beyond existing modules.
- **Circuit Breakers**: Graceful fallback when optional external linters (`staticcheck`, `govulncheck`, `betterleaks`, `esbuild`, `tidy`) are not found in `$PATH`.
- **Atomic Temporary Builds**: Builds and temporary test coverage files use `os.MkdirTemp` with deferred cleanup (`defer os.RemoveAll(tmpDir)`).

## Verification Plan

- Unit tests for `cmd/check`:
  - `go test -count=1 ./cmd/check/...`
- Test individual check stages via Go CLI:
  - `go run ./cmd/check fmt`
  - `go run ./cmd/check templates`
  - `go run ./cmd/check loc`
  - `go run ./cmd/check loc 400`
  - `go run ./cmd/check loc 300` (verify failure on low threshold)
  - `go run ./cmd/check lint`
  - `go run ./cmd/check test`
  - `go run ./cmd/check build`
  - `go run ./cmd/check all`
- Test thin wrapper:
  - `./scripts/check.sh all`
  - `./scripts/check.sh loc 400`
  - `./scripts/check.sh templates`






