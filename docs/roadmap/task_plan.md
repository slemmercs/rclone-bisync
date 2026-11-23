# Task Plan: Clustered Remote Storage Families with Vertical Overlay Pipeline Trees

## Goal
Harden Cockpit storage remotes UI by clustering virtual overlay remotes (`crypt`, `hasher`, `alias`, etc.) under their root storage families using a vertical indented pipeline tree with precise active profile targeting, full per-overlay action suites, and cascade delete safeguards.

## Tasks

- [x] Task 1: Create `RemoteOverlayTree.vue` Component
  - **Scope**: `dashboard/frontend/src/components/remotes/RemoteOverlayTree.vue`
  - **Verification**: `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build`

- [x] Task 2: Refactor `RemoteCard.vue` to Embed Overlay Tree & Cascade Delete Safeguards
  - **Scope**: `dashboard/frontend/src/components/remotes/RemoteCard.vue`
  - **Verification**: `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build`

- [x] Task 3: Update `ViewRemotes.vue` with Topology Builder & Composite Section
  - **Scope**: `dashboard/frontend/src/views/ViewRemotes.vue`
  - **Verification**: `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build`

- [x] Task 4: Full Quality Verification Suite
  - **Scope**: Pre-flight verification across all 7 stages
  - **Verification**: `./scripts/check.sh all`

## Status Updates

- 2026-11-22: Completed Task 4. Executed full test and quality suite (`./scripts/check.sh all`): passed all 7/7 verification gates (vet, unit tests, coverage, security scanning, frontend type checking, and production asset bundle generation). All tasks in Epic complete!

- 2026-11-22: Completed Task 3. Updated `ViewRemotes.vue` with adjacency graph builder (`childrenByParent`), root base remotes vs composite/orphan separation (`compositeOrOrphanRemotes`), deep family search and category filtering traversing overlay pipelines, node-exact active profile derivation, and cascade warning prompts on remote deletion. Verified with `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build`.

- 2026-11-22: Completed Task 2. Refactored `RemoteCard.vue` to embed `RemoteOverlayTree`, modernize base card overlay wrapping actions (`+ Crypt`, `+ Hasher` with wrapped guards), isolate node-exact active indicators, and incorporate cascade deletion warnings. Verified with `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build`.

- 2026-11-22: Completed Task 1. Created `RemoteOverlayTree.vue` component implementing a vertical indented tree structure for child overlays with depth markers, type badges, upstream references, active profile targeting, and full action suites (`Test`, `Dedupe`, `Delete`, contextual `+ Wrap`). Verified with `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build`.

- 2026-11-22: Initialized implementation plan and task plan incorporating all 5 UX and Rclone domain recommendations.








