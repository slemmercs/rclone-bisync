# Implementation Plan: Consolidate Quality Scripts into Unified check.sh with Standalone CI Targets

## Proposed Changes

Consolidate `scripts/lint-templates.sh`, `scripts/check-loc.sh`, and `scripts/check.sh` into a single, self-contained `scripts/check.sh` script. Every check stage will be decoupled so it can be executed standalone in GitHub Actions CI jobs as well as locally via unified flags.

### Affected Components & Packages

- `scripts/check.sh`:
  - Integrate `check_alpine_templates()`: scans HTML templates for dotted boolean attribute bindings (`:disabled="x.y"`), preventing Alpine v3 boolean coercion bugs.
  - Integrate `check_loc()`: audits Go, JavaScript, and HTML files against LOC limits (default 400 LOC, configurable via CLI arg or `LOC_THRESHOLD`). Fails if files exceed the threshold.
  - Expand stage routing:
    - `all`: runs all 6 stages sequentially (`fmt`, `lint`, `templates`, `loc`, `test`, `build`).
    - `fmt`: runs `check_fmt_and_scripts`.
    - `lint`: runs `check_lint_and_security`.
    - `templates`: runs `check_alpine_templates`.
    - `loc [limit]`: runs `check_loc` with optional threshold.
    - `test`: runs `check_tests_and_coverage`.
    - `build`: runs `check_builds`.
  - Add help documentation (`--help`), argument validation, and support for `-v` / `--verbose` streaming.
- `scripts/lint-templates.sh`:
  - Removed (logic absorbed into `scripts/check.sh`).
- `scripts/check-loc.sh`:
  - Removed (logic absorbed into `scripts/check.sh`).
- `.clinerules`:
  - Update LOC inspection command from `./scripts/check-loc.sh 400` to `./scripts/check.sh loc 400`.
- `.github/workflows/ci.yml` & `.github/workflows/nightly-quality.yml`:
  - Add standalone CI jobs for `templates` (`./scripts/check.sh templates`) and `loc` (`./scripts/check.sh loc`).

### Technical Strategy & Data Flow

- **Decoupled Stages**: Each quality stage functions independently without relying on intermediate state or order of execution.
- **Fail-Fast Error Propagation**: `check_alpine_templates` and `check_loc` return non-zero exit codes on violations, properly failing standalone CI jobs or pre-flight checks.
- **Zero External Dependencies**: Both template linting and LOC auditing use standard POSIX utilities (`grep`, `find`, `awk`, `wc`) already present in all environments.

### Safety & Error Handling

- Unknown stage names exit immediately with code 1 and display available options.
- Clean cleanup trap ensures temporary logs and builds are purged on exit.
- Preserves all existing stage configurations (`MIN_COVERAGE`, `RUN_RACE`, `BUILD_TARGET`).

## Verification Plan

- Test each stage independently:
  - `./scripts/check.sh fmt`
  - `./scripts/check.sh lint`
  - `./scripts/check.sh templates`
  - `./scripts/check.sh loc` & `./scripts/check.sh loc 400` & `./scripts/check.sh loc 300` (verifying failure on intentional low threshold)
  - `./scripts/check.sh test`
  - `./scripts/check.sh build`
  - `./scripts/check.sh all` (all 6 stages pass)
  - `./scripts/check.sh --help`
- Run `./scripts/check.sh` pre-flight commit validation.





