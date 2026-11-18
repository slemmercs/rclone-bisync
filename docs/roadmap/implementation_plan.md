# Implementation Plan: 100% Feature Parity Bridge (Alpine.js to Vue 3 Cockpit)

## Overview
Bridge the remaining four functional and interaction gaps between the legacy Alpine.js dashboard and the modern Vue 3 Cockpit SPA to achieve 100% complete feature and UX parity.

The consolidated Vue 3 frontend currently implements ~95% of legacy functionality while introducing typed Pinia stores, PrimeVue UI dialogs, global keyboard shortcuts, and operator polling controls. However, a meticulous comparative audit identified four concrete capability gaps: (1) Fleet view profile search, status filtering, and tile-click workspace navigation into the File Explorer; (2) Heuristic Conflict Recommendation Engine scoring display and auto-apply batch resolution (`POST /api/review/action-batch`) in the Conflict Review modal; (3) Deduplication collision search and candidate filtering in the Dedupe Workspace modal; and (4) HTML5 drag-and-drop file and directory transfers between explorer panes.

This plan details the exact architectural, TypeScript, Pinia, and component changes required to eliminate these gaps without regressing existing workflows or violating codebase constraints.

---

## Types

### 1. `dashboard/frontend/src/types/models.ts`
Add heuristic recommendation interfaces and update candidate definitions:
```typescript
export interface RuleResult {
  rule_id: string
  confidence: number
  risk_tier: 'LOW' | 'MEDIUM' | 'HIGH' | 'CRITICAL'
  suggested_action: ActionDecision
  reason: string
}

export interface ConflictRecommendation {
  suggested_action: ActionDecision
  confidence: number
  risk_tier: 'LOW' | 'MEDIUM' | 'HIGH' | 'CRITICAL'
  primary_reason: string
  rule_evaluations?: RuleResult[]
}

// Update DeletionCandidate to include optional heuristic recommendation:
export interface DeletionCandidate {
  id: string
  path: string
  name: string
  is_dir: boolean
  source: 'local' | 'remote'
  target_action: string
  size?: number
  mod_time?: string
  condition: string
  decision?: ActionDecision
  resolved: boolean
  recommendation?: ConflictRecommendation
}

// Update ConflictCandidate to include optional heuristic recommendation:
export interface ConflictCandidate {
  id: string
  path: string
  name: string
  local_size: number
  local_mod_time: string
  remote_size: number
  remote_mod_time: string
  condition: string
  decision?: ActionDecision
  resolved: boolean
  recommendation?: ConflictRecommendation
}

export interface BatchReviewActionRequest {
  session_id?: string
  min_confidence: number
  max_allowed_risk: 'LOW' | 'MEDIUM' | 'HIGH'
}

export interface BatchReviewActionResponse {
  status: string
  session?: ReviewSession
  staged_count: number
  error?: string
}
```

### 2. `dashboard/frontend/src/types/explorer.ts`
Add drag-and-drop payload interface:
```typescript
export interface ExplorerDragPayload {
  item: BrowseItem
  sourcePane: 'left' | 'right'
  sourceRemote: string
  sourcePath: string
}
```

---

## Files

### New Files
- None required (all enhancements extend existing cohesive Single-File Components and Pinia stores).

### Existing Files to Modify
- `dashboard/frontend/src/types/models.ts`:
  - Add `ConflictRecommendation`, `RuleResult`, `BatchReviewActionRequest`, `BatchReviewActionResponse`.
  - Extend `ConflictCandidate` and `DeletionCandidate` with `recommendation?: ConflictRecommendation`.
- `dashboard/frontend/src/types/explorer.ts`:
  - Add `ExplorerDragPayload` interface.
- `dashboard/frontend/src/stores/review.ts`:
  - Add `applyBatchRecommendations(minConfidence?: number, maxRisk?: 'LOW' | 'MEDIUM' | 'HIGH'): Promise<number>` calling `POST /api/review/action-batch`.
- `dashboard/frontend/src/views/ViewFleet.vue`:
  - Add `searchQuery` and `statusFilter` reactive state (`'all' | 'running' | 'idle' | 'aborted' | 'disabled'`).
  - Add search and filter ribbon above profiles grid.
  - Implement `openProfileWorkspace(profile: Profile)` to select profile in store and switch active tab to `'explorer'`.
  - Add open workspace trigger on profile card tiles.
- `dashboard/frontend/src/components/ConflictReviewModal.vue`:
  - Add "Auto-Apply Recommendations" button in the Quick Batch toolbar.
  - Add heuristic recommendation badge display (suggested action chip, confidence score percentage meter, and risk tier pill) for each conflict and deletion candidate.
  - Add confirmation toast upon auto-applying batch recommendations.
- `dashboard/frontend/src/components/ModalDedupe.vue`:
  - Add `filterQuery` reactive search ref.
  - Add search input field in the Candidates tab header (`placeholder="Filter collisions by name or path..."`).
  - Compute `filteredItems` to filter duplicate collisions by file name, relative path, or parent directory.
- `dashboard/frontend/src/components/explorer/ExplorerTable.vue`:
  - Add HTML5 `draggable="item.Name !== 'RCLONE_TEST'"` and `@dragstart` event handler to draggable row items.
  - Add `draggable="false"` and `@mousedown.stop` to interactive row children (checkboxes, buttons) per engineering rules.
  - Add `@dragover.prevent` and `@drop` event listeners with visual drop highlight overlay.
  - Emit `'itemDrop'` event with `ExplorerDragPayload`.
- `dashboard/frontend/src/views/ViewExplorer.vue`:
  - Handle `'itemDrop'` emitted by `ExplorerPane` / `ExplorerTable`.
  - Open move/copy confirmation dialog (`ModalExplorerMove.vue` or prompt) to safely execute cross-pane operations.
- `tests/e2e/cockpit_spa_smoke_test.go`:
  - Add headless Chrome E2E tests for Fleet search/filter & workspace jump, Conflict Review auto-apply button, Dedupe collision search, and Explorer drag-and-drop events.

### Files to Delete or Move
- None.

---

## Functions

### New Functions
- `applyBatchRecommendations(minConfidence?: number, maxRisk?: string)`:
  - **File**: `dashboard/frontend/src/stores/review.ts`
  - **Signature**: `async function applyBatchRecommendations(minConfidence = 0.70, maxRisk: 'LOW' | 'MEDIUM' | 'HIGH' = 'MEDIUM'): Promise<number>`
  - **Purpose**: Issues `POST /api/review/action-batch` with `X-Dashboard-Request` header to auto-stage high-confidence heuristic recommendations, updates active review session state, and returns count of staged resolutions.
- `openProfileWorkspace(profile: Profile)`:
  - **File**: `dashboard/frontend/src/views/ViewFleet.vue`
  - **Signature**: `function openProfileWorkspace(profile: Profile): void`
  - **Purpose**: Sets active profile in `useProfilesStore().selectProfile(profile.id)` and switches global tab via `useAppStore().switchTab('explorer')`.
- `handleRowDragStart(item: BrowseItem, event: DragEvent)`:
  - **File**: `dashboard/frontend/src/components/explorer/ExplorerTable.vue`
  - **Signature**: `function handleRowDragStart(item: BrowseItem, event: DragEvent): void`
  - **Purpose**: Populates `dataTransfer` payload with JSON-serialized item metadata, source remote, and parent path.
- `handleTableDrop(event: DragEvent)`:
  - **File**: `dashboard/frontend/src/components/explorer/ExplorerTable.vue`
  - **Signature**: `function handleTableDrop(event: DragEvent): void`
  - **Purpose**: Parses drop payload, validates drop target is not the identical item or canary file, and emits `itemDrop`.
- `handleCrossPaneDrop(payload: ExplorerDragPayload, targetPaneKey: 'left' | 'right')`:
  - **File**: `dashboard/frontend/src/views/ViewExplorer.vue`
  - **Signature**: `async function handleCrossPaneDrop(payload: ExplorerDragPayload, targetPaneKey: 'left' | 'right'): Promise<void>`
  - **Purpose**: Coordinates cross-pane copy/move confirmation dialog to prevent accidental mutations.

---

## Classes
No ES6 / TypeScript classes are introduced; all frontend state follows Pinia compositional setup stores (`defineStore`) and Vue 3 `<script setup>` idioms.

---

## Dependencies
- Zero new external runtime or build dependencies required.
- Uses existing PrimeVue components, Lucide / PrimeIcons, and Vue 3 standard library.

---

## Testing

### Automated Headless Chrome E2E Suite (`tests/e2e/cockpit_spa_smoke_test.go`)
1. **Fleet Search & Navigation Subtest**:
   - Filter profiles by text query (e.g., query matching only 1 profile).
   - Assert filtered card count equals 1.
   - Click the "Open Workspace" icon on the card.
   - Assert current active tab is `'File Explorer'` and `profilesStore.selectedProfileId` matches the clicked profile.
2. **Conflict Review Heuristic Auto-Apply Subtest**:
   - Seed an active review session with simulated heuristic confidence metadata.
   - Open Conflict Review dialog.
   - Assert recommendation badges and confidence percentages render in candidate rows.
   - Click "Auto-Apply Recommendations" button.
   - Verify `POST /api/review/action-batch` is received and candidate decisions transition to staged.
3. **Dedupe Collision Search Subtest**:
   - Seed mock duplicate collision items in `ModalDedupe.vue`.
   - Enter search term in collision search input.
   - Assert rendered collision candidate rows filter down to matching file names.
4. **Explorer Drag & Drop Integrity Subtest**:
   - Verify `draggable="true"` attribute on file rows (excluding `RCLONE_TEST`).
   - Verify `draggable="false"` on child checkboxes and action buttons.
   - Dispatch `dragstart` and `drop` custom events between panes and verify confirmation dialog displays.

### Validation Script
- Full validation via `./scripts/check.sh all` (all 7 quality stages: Go formatting, vet, tests, vue-tsc typecheck, vite bundle compilation, LOC audit, and secret scanning).

---

## Implementation Order

1. **Step 1: TypeScript Models & Pinia Review Store Extension**
   - Update `dashboard/frontend/src/types/models.ts` with `ConflictRecommendation`, `RuleResult`, `BatchReviewActionRequest`, `BatchReviewActionResponse`.
   - Add `ExplorerDragPayload` to `dashboard/frontend/src/types/explorer.ts`.
   - Add `applyBatchRecommendations` in `dashboard/frontend/src/stores/review.ts`.
   - Verification: `npm --prefix dashboard/frontend run type-check`.

2. **Step 2: Fleet View Search, Status Filters & Workspace Jump**
   - Add `searchQuery` and `statusFilter` refs in `ViewFleet.vue`.
   - Render search input and status pill filters above profiles grid.
   - Implement `openProfileWorkspace` and bind to card header / "Open in Explorer" quick action button.
   - Verification: `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build`.

3. **Step 3: Conflict Review Modal Heuristic Badges & Auto-Apply**
   - Add "Auto-Apply Recommendations" button and risk threshold popover in `ConflictReviewModal.vue`.
   - Render heuristic badge (suggested action, confidence %, risk tier) on conflict and deletion cards.
   - Connect auto-apply button to `reviewStore.applyBatchRecommendations()`.
   - Verification: `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build`.

4. **Step 4: Dedupe Workspace Collision Search Filter**
   - Add `filterQuery` ref and `filteredItems` computed in `ModalDedupe.vue`.
   - Render search input in the Candidates tab toolbar.
   - Verification: `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build`.

5. **Step 5: Explorer HTML5 Drag-and-Drop Cross-Pane Transfers**
   - Add drag event handlers (`@dragstart`, `@dragover`, `@dragleave`, `@drop`) in `ExplorerTable.vue`.
   - Enforce `.stop` and `draggable="false"` on interactive child elements to prevent accidental drag initiation.
   - Wire `itemDrop` event in `ViewExplorer.vue` to launch `ModalExplorerMove.vue`.
   - Verification: `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build`.

6. **Step 6: Headless Chrome E2E Suite Expansion & Pre-Flight Quality Gates**
   - Extend `tests/e2e/cockpit_spa_smoke_test.go` with subtests covering the 4 bridged features.
   - Run `go test -v -count=1 ./tests/e2e -run "TestCockpitSPA_HeadlessChromeSmoke"`.
   - Run full verification suite: `./scripts/check.sh all`.
