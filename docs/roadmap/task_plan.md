# Task Plan: Automated Code Documentation Generator & Sandbox Screenshot Capture

## Goal
Implement a native Go AST documentation generator (`cmd/gendocs`) producing search-indexed GitHub Pages documentation, and build an automated headless Chrome sandbox screenshot capture suite for the Vue Cockpit SPA.

## Tasks

- [x] Task 1: Native Go Documentation Generator (`cmd/gendocs`)
  - **Scope**: `cmd/gendocs/main.go`, `cmd/gendocs/parser.go`
  - **Verification**: `go run ./cmd/gendocs --out site/technical-reference.md && head -n 30 site/technical-reference.md`

- [ ] Task 2: Automated Sandbox Screenshot Suite for Vue Cockpit
  - **Scope**: `tests/e2e/screenshots_test.go`
  - **Verification**: `go test -count=1 -v ./tests/e2e -run TestCaptureSandboxScreenshots`

- [ ] Task 3: CI Workflow & Build Integration
  - **Scope**: `.github/workflows/pages.yml`, `Makefile`, `site/gallery.md`, `site/_config.yml`
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-11-19: Initialized plan for native Go AST documentation generator and sandbox screenshot capture.
- 2026-11-19: Completed Task 1 - Implemented native Go AST documentation generator in `cmd/gendocs/` (`main.go` and `parser.go`) to automatically extract global settings, profile policies, and MCP tool definitions into `site/technical-reference.md`. Verified via `./scripts/check.sh all` and live generation.

