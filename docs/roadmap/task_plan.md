
# Task Plan: GitHub Go-Live Preparation, Deployment Fixes & Documentation Alignment

## Goal
Fix broken deployment and documentation references resulting from the repo reorganization, update .clinerules to recognize the new doc paths, add GitHub Actions CI and community health templates, and prepare the repository for public release under Apache 2.0.

## Tasks
- [x] Task 0: Update `.clinerules` for reorganized documentation paths
  - **Scope**: `.clinerules`
  - **Verification**: Inspect `.clinerules` diff and confirm task plan path alignment
- [x] Task 1: Fix `deploy/install.sh` paths and purge deleted desktop entries
  - **Scope**: `deploy/install.sh`
  - **Verification**: `bash -n deploy/install.sh` and dry-run path resolution check
- [x] Task 2: Align markdown links and purge obsolete test references
  - **Scope**: `CONTRIBUTING.md`, `docs/roadmap/backlog.md`, `README.md`, `SECURITY.md`
  - **Verification**: Regex search verifying zero broken relative `.md` links
- [ ] Task 3: Setup GitHub Actions CI Workflow and Dependabot
  - **Scope**: `.github/workflows/ci.yml`, `.github/dependabot.yml`
  - **Verification**: Lint workflow YAML and verify test step matches `./scripts/check.sh`
- [ ] Task 4: Setup GitHub Community & Security Templates
  - **Scope**: `.github/CODEOWNERS`, `.github/ISSUE_TEMPLATE/*`, `.github/PULL_REQUEST_TEMPLATE.md`
  - **Verification**: File existence and syntax check
- [ ] Task 5: Pre-Flight Verification & Checkpoint Commit
  - **Scope**: Entire repository
  - **Verification**: `./scripts/check.sh` and `go build -o bin/rclone-bisync main.go`

## Status Updates
- 2026-10-22: Initialized plan on branch `chore/docs-deploy-and-github-prep`.
- 2026-10-22: Completed Task 0 (Updated `.clinerules` to support `docs/roadmap/task_plan.md` and `docs/architecture/code-review.md`).
- 2026-10-22: Completed Task 1 (Remediated `deploy/install.sh` dynamic repo root resolution, fixed config and systemd source paths, purged deleted manual desktop entry).
- 2026-10-22: Completed Task 2 (Aligned markdown relative links in `CONTRIBUTING.md`, `docs/roadmap/backlog.md`, `code-review.md`; updated `SECURITY.md` with GitHub advisories; purged obsolete `pytest` from `README.md`).


