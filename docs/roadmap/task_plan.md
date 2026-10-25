# Task Plan: Consolidate Quality Scripts into Unified check.sh with Standalone CI Targets

## Goal
Integrate `lint-templates.sh`, `check-loc.sh`, and `check.sh` into a single, unified `check.sh` script where each check can be executed independently for standalone CI jobs.

## Tasks

- [x] Task 1: Consolidate template linting and LOC auditing into `scripts/check.sh`
  - **Scope**: `scripts/check.sh`
  - **Verification**: `./scripts/check.sh templates` && `./scripts/check.sh loc 400`

- [x] Task 2: Remove redundant standalone scripts and update rule references
  - **Scope**: `scripts/lint-templates.sh`, `scripts/check-loc.sh`, `.clinerules`
  - **Verification**: `git status -s` && `./scripts/check.sh fmt`

- [x] Task 3: Update CI workflows for standalone quality check jobs
  - **Scope**: `.github/workflows/ci.yml`, `.github/workflows/nightly-quality.yml`
  - **Verification**: `git diff .github/workflows/`

- [x] Task 4: Full verification of all stages and pre-flight commit
  - **Scope**: `./scripts/check.sh [fmt|lint|templates|loc|test|build|all]`
  - **Verification**: All individual check invocations and full `./scripts/check.sh` pass cleanly

## Status Updates

- 2026-10-25: Initialized plan from implementation_plan.md on branch `feat/alpine-boolean-lint-and-e2e`.
- 2026-10-25: Completed Task 1: Consolidated `check_alpine_templates` and `check_loc` into `scripts/check.sh`.
- 2026-10-25: Completed Task 2: Removed `scripts/lint-templates.sh` and `scripts/check-loc.sh` and updated `.clinerules`.
- 2026-10-25: Completed Task 3: Added dedicated standalone CI jobs for `templates` and `loc` in `.github/workflows/ci.yml` and `.github/workflows/nightly-quality.yml`.
- 2026-10-25: Completed Task 4: Verified each individual stage (`fmt`, `lint`, `templates`, `loc`, `test`, `build`) and the unified 6-stage `./scripts/check.sh all` run cleanly.








