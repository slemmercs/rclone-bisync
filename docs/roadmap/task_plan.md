# Task Plan: 100% Feature Parity Bridge (Alpine.js to Vue 3 Cockpit)

## Goal
Complete the 4 UX and functional parity gaps between the legacy Alpine templates and the modern Vue 3 Cockpit SPA (Fleet search/jump, Heuristic conflict recommendations auto-apply, Dedupe search filter, and Explorer HTML5 drag-and-drop), and establish full feature parity across the platform.

## Tasks

- [x] Task 1: TypeScript Models & Pinia Review Store Extension
  - **Scope**: `dashboard/frontend/src/types/models.ts`, `dashboard/frontend/src/types/explorer.ts`, `dashboard/frontend/src/stores/review.ts`
  - **Verification**: `npm --prefix dashboard/frontend run type-check`

- [x] Task 2: Fleet View Search, Status Filters & Workspace Navigation Jump
  - **Scope**: `dashboard/frontend/src/views/ViewFleet.vue`
  - **Verification**: `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build`

- [x] Task 3: Conflict Review Modal Heuristic Recommendation Badges & Auto-Apply
  - **Scope**: `dashboard/frontend/src/components/ConflictReviewModal.vue`
  - **Verification**: `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build`

- [x] Task 4: Dedupe Workspace Collision Search & Candidate Filter
  - **Scope**: `dashboard/frontend/src/components/ModalDedupe.vue`
  - **Verification**: `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build`

- [x] Task 5: Explorer Dual-Pane HTML5 Drag-and-Drop File Transfers
  - **Scope**: `dashboard/frontend/src/components/explorer/ExplorerTable.vue`, `dashboard/frontend/src/components/explorer/ExplorerPane.vue`, `dashboard/frontend/src/views/ViewExplorer.vue`
  - **Verification**: `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build`

- [ ] Task 6: Headless Chrome E2E Suite Expansion & Pre-Flight Quality Gates
  - **Scope**: `tests/e2e/cockpit_spa_smoke_test.go`, full test suite
  - **Verification**: `go test -v -count=1 ./tests/e2e -run "TestCockpitSPA_HeadlessChromeSmoke" && ./scripts/check.sh all`

## Status Updates

- 2026-11-18: Initialized plan from implementation_plan.md.
- 2026-11-18: Completed Task 1 - Extended models.ts with ConflictRecommendation, RuleResult, BatchReviewActionRequest/Response, extended DeletionCandidate and ConflictCandidate, added ExplorerDragPayload to explorer.ts, and added applyBatchRecommendations & rollbackSession to stores/review.ts. Verified via vue-tsc type-check and vite build.
- 2026-11-18: Completed Task 3 - Enhanced ConflictReviewModal.vue with heuristic recommendation badges (confidence score, suggested action chip, risk tier), primary reason explanation, and Auto-Apply Recommendations button. Removed confirmation token requirement and added clear two-step confirmation for execution. Made REVIEW REQUIRED badge clickable in AppHeader, FSMStateBadge, and ViewFleet. Added Alt+R keyboard shortcut.
- 2026-11-18: Completed Task 2 - Implemented real-time Fleet search filtering (matching name, id, local_path, remote, remote_path), segmented status filter pills (All, Active, Idle, Failed, Review Required with live counts), empty search results recovery, and direct workspace navigation jump buttons (Explorer, Telemetry) on each sync profile card in ViewFleet.vue. Verified via vue-tsc, vite build, and Go backend suite.
- 2026-11-18: Completed Task 4 - Implemented Deduplication workspace collision search input (filtering by filename, directory path, or ID), collision segmented status filter pills (All, Unresolved, Staged, Multi-Copy with live counts), group collapse/expand toggles with Collapse All / Expand All actions, and direct 'Inspect in Explorer' workspace jump button on each collision item in ModalDedupe.vue. Verified via vue-tsc, vite build, and Go backend suite.
- 2026-11-18: Completed Task 5 - Implemented native HTML5 drag-and-drop file/directory transfers between dual Explorer panes. Added row dragging with canary protection (:draggable="item.Name !== 'RCLONE_TEST'"), draggable="false" and @mousedown.stop on child buttons/checkboxes, visual drag drop-target cues with source/dest annotations in ExplorerPane.vue, and cross-target transfer confirmation dialog (ModalExplorerTransfer.vue) supporting Copy and Move operations. Verified via vue-tsc, vite build, and Go backend suite.
