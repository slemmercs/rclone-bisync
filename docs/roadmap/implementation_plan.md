# Implementation Plan: Quality Engine Hardening, E2E Stability & Showcase Rendering Fixes

## Proposed Changes

Address three operational regressions on branch `feat/alpine-boolean-lint-and-e2e`:
1. **Showcase Site (`site/`)**:
   - Fix missing site title text next to logo in navbar caused by `just-the-docs` logo replacement behavior. Override `_includes/title.html` and style `.site-title-container`, `.site-title-logo`, and `.site-title-text`.
   - Fix unparsed backtick markdown in `site/index.md` inside `<div class="hero-box">` by replacing it with a styled HTML code block.
2. **Quality Engine CLI (`cmd/check`)**:
   - Pre-process CLI arguments in `cmd/check/main.go` so flags (`-v`, `--verbose`, `-loc`, `-min-coverage`, `-race`, `-target`, etc.) are recognized regardless of whether they appear before or after the stage name.
   - Wire `cfg.Verbose` into `cmd/check/checks.go` and `cmd/check/main.go` to provide meaningful live progress and test output when verbose mode is enabled.
   - Add unit test coverage in `cmd/check/main_test.go` for flexible flag and argument parsing.
3. **Headless Chrome E2E Smoke Test (`tests/e2e/dashboard_smoke_test.go`)**:
   - Replace brittle fixed `time.Sleep(300 * time.Millisecond)` after tab switching with dynamic polling that waits for Alpine.js async loading (`remotesData.loading === false`, `auditData.loading === false`) before asserting button disabled states.

### Affected Components & Packages

- `site/_includes/title.html`: Custom header title component rendering both logo and text.
- `site/_sass/custom/custom.scss`: Navbar title and hero installation code container styling.
- `site/index.md`: Standard HTML markup for hero installation command.
- `cmd/check/main.go`: Flexible argument partitioning, flag parsing, and stage progress logging.
- `cmd/check/checks.go`: Streaming/verbose execution for tests, linters, and builds.
- `cmd/check/main_test.go`: Tests for CLI argument parsing.
- `tests/e2e/dashboard_smoke_test.go`: Polling wait for async Alpine.js tab load settlement.

### Technical Strategy & Data Flow

- **Jekyll Include Override**: `just-the-docs` looks for `_includes/title.html` to render the content of `<a class="site-title">`. Rendering `<img>` and `<span>` together with flex alignment provides clean brand presentation.
- **Argument Partitioning**: Partition `os.Args[1:]` into flags and positional tokens. Flag parsing evaluates all flag arguments via `flag.CommandLine.Parse()`, while positional tokens determine stage and stage-specific arguments.
- **E2E Async Settlement**: In headless Chrome, wait for `window.Alpine.$data(el).remotesData.loading === false` to eliminate timing flakes on slow CI environments.

### Safety & Error Handling

- Zero API or daemon contract breakage.
- All checks adhere to POSIX exit codes (0 for pass, 1 for failure).

## Verification Plan

- Run `./scripts/check.sh fmt` and `./scripts/check.sh all -v` to verify flag parsing and live output.
- Run `go test -v -count=1 ./cmd/check` to verify CLI parser tests.
- Run `go test -v -count=1 ./tests/e2e` repeatedly to verify zero race flakes.








