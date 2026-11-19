# Task Plan: TypeScript Contract Generator, Documentation Showcase Redesign & Generator Modularization

## Goal
Implement native Go AST $\rightarrow$ TypeScript interface generator (`--ts-out`) as Priority 1, modularize `cmd/gendocs/` (< 200 LOC per file), gate `TestCaptureSandboxScreenshots` behind `CAPTURE_SCREENSHOTS=1`, add retina + zero-animation rendering, and overhaul GitHub Pages showcase (`site/`) to match Agnoster Carbon & Electric Blue theme.

## Tasks

- [x] Task 1: Native Go AST $\rightarrow$ TypeScript Contract Generator & `cmd/gendocs` Modularization
  - **Scope**: `cmd/gendocs/main.go`, `cmd/gendocs/ast_parser.go`, `cmd/gendocs/ts_generator.go`, `cmd/gendocs/mcp_docs.go`, `cmd/gendocs/render.go`, `cmd/gendocs/gendocs_test.go`, `dashboard/frontend/src/types/models.gen.ts`, `Makefile`
  - **Verification**: `make generate-ts && npm --prefix dashboard/frontend run type-check && make docs-generate`

- [x] Task 2: Gate E2E Screenshots & High-DPI Polish
  - **Scope**: `tests/e2e/screenshots_test.go`, `Makefile`
  - **Verification**: `go test -count=1 ./tests/e2e -run TestCaptureSandboxScreenshots` (skips cleanly) and `CAPTURE_SCREENSHOTS=1 make docs-screenshots`

- [x] Task 3: Agnoster Carbon & Electric Blue Theme Redesign
  - **Scope**: `site/_sass/custom/setup.scss`, `site/_sass/custom/custom.scss`, `site/assets/images/logo.svg`, `site/assets/images/favicon.svg`
  - **Verification**: Inspect generated styles, verify SCSS syntax validity, and ensure Electric Blue brand assets compile

- [x] Task 4: Content Overhaul for Vue 3 Architecture & Production Milestones
  - **Scope**: `site/usps/05-single-binary.md`, `site/usps/index.md`, `site/roadmap.md`, `.clinerules`
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-11-19: Completed Task 4 (Content Overhaul for Vue 3 Architecture & Production Milestones). Replaced outdated Alpine.js references across site USPs, roadmap, architecture docs, and .clinerules with Vue 3 Cockpit architecture and Agnoster Carbon design system. All 7 quality gates passed via ./scripts/check.sh all.
- 2026-11-19: Completed Task 3 (Agnoster Carbon & Electric Blue Theme Redesign). Overhauled setup.scss and custom.scss with Agnoster Carbon tokens (#090A0D canvas, #111318 surfaces, #161922 card/code containers, #0ea5e9 electric blue accents), resolved interleaved SCSS blocks, and updated brand SVGs with electric blue gradient (#0ea5e9 -> #0284c7).
- 2026-11-19: Completed Task 2 (Gate E2E Screenshots & High-DPI Polish). Gated screenshot capture behind CAPTURE_SCREENSHOTS=1, added 2x Retina scale factor, zero-animation style injection, and frontend-build dependency in Makefile.
- 2026-11-19: Completed Task 1 (Native Go AST -> TypeScript Contract Generator & cmd/gendocs Modularization). All 5 cmd/gendocs modules are <200 LOC, generate-ts target added to Makefile, and models.gen.ts verified with vue-tsc.
- 2026-11-19: Added Priority 1 Go AST -> TypeScript contract generator (`--ts-out`) to implementation plan and task plan.



