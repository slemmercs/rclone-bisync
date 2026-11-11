# Implementation Plan: Deep Path Navigation & Pinned Log Audit Dock

## Overview
Deliver Deep Path Navigation and an IDE-style Pinned Log Audit Dock in the Bisync Hub cockpit, combining a hardened regex tokenizer for log mutations, synchronized dual-pane Explorer navigation with targeted row highlighting, and an in-context bottom tray for rapid multi-file stepping.

When Rclone bisync or the daemon processes file mutations (`Duplicate ignored`, `Deleted`, `File was deleted`, `Transferring`, `Conflict`, `Copied`, `Updated`), operators currently see flat text log lines with no direct link to the affected files. Navigating via repeated full-page tab switches between the Audit Ledger and File Explorer breaks operational flow during batch reviews. This implementation introduces:
1. A **hardened contextual tokenizer** (`extractPathFromLogMsg`) that extracts, sanitizes, and normalizes file paths from both structured JSON logs and raw log strings while stripping remote anchors (e.g. `gdrive{tCwNP}:`) and rejecting non-file error summaries.
2. A **synchronized dual-pane navigation engine** (`jumpToExplorerPath`) that safely switches profiles without async overwrite races, navigates both Local and Remote panes to the enclosing directory, and waits for directory listings before executing smooth DOM scroll-into-view with an animated pulse highlight.
3. An **IDE-style Pinned Log Audit Dock** (`component_log_dock.html` & `log_dock.js`) anchored at the bottom of the File Explorer, enabling operators to step through file mutations using `N` (Next) / `P` (Previous) keyboard shortcuts or direct row clicks without leaving the workspace.

---

## Types

### Frontend Runtime Model Specifications (`dashboard/frontend/js/`)

No new backend Go structs are needed because existing endpoints (`GET /api/logs/bisync`, `GET /api/logs/daemon`, and `POST /api/browse`) already transport JSON line streams and directory listings. In the frontend Alpine.js runtime, log entries and dock state are enriched with the following explicit contracts:

```typescript
// Enriched structured representation of a log line
interface LogEntryEnrichment {
    time: string;                     // e.g. "21:04:02"
    level: string;                    // e.g. "WARN   "
    levelClass: string;               // Tailwind class e.g. "text-amber-400 font-medium"
    msg: string;                      // Sanitized message string
    object: string | null;            // Normalized relative path (e.g. "docs/contracts/agreement.pdf")
    detected_dir?: string;           // Enclosing directory path (e.g. "docs/contracts" or "")
    detected_filename?: string;      // Target file name (e.g. "agreement.pdf")
    run_id: string | null;            // Correlated run ID (e.g. "run-20261111-01")
    job_id: number | null;            // Correlated background job ID
    profile_id: string | null;        // Target profile ID
    type: "sync" | "audit" | "http_access";
}

// Pinned Log Dock state machine contract
interface LogDockState {
    isOpen: boolean;                  // Dock tray visibility toggle
    isPinned: boolean;                // Auto-open on jump toggle (persisted to localStorage)
    heightMode: "docked" | "collapsed" | "expanded"; // 220px | 36px | 380px
    filter: "mutations" | "errors" | "all"; // Filter presets
    activeIndex: number;              // Index of currently focused mutation entry
    entries: LogEntryEnrichment[];    // Filtered actionable items
}
```

---

## Files

### New Files
1. `dashboard/frontend/js/log_dock.js`:
   - Dedicated Alpine.js module (`window.LogDockModule`) managing dock state, mutation filtering, keyboard shortcuts (`N`, `P`, `Esc`, `ArrowUp`, `ArrowDown`), multi-file stepping, and height presets.
   - Enforces repository modularity and guarantees all frontend files remain below the **400 LOC ceiling** (~115 LOC).
2. `dashboard/frontend/partials/components/component_log_dock.html`:
   - High-density IDE output bottom tray template.
   - Header with mutation count, filter buttons (`[ ⚠️ Mutations ]`, `[ 🔴 Errors ]`, `[ All ]`), keyboard shortcut indicators (`[N] Next`, `[P] Prev`), height cycle buttons, and close button (`×`).
   - Body with high-density table (`TIME | LEVEL | RUN ID | ACTION | PATH | ACTIONS`), active row indicator, clickable path chips, and direct `[ 🎯 Jump ]` buttons (~85 LOC).

### Existing Files to Modify
1. `dashboard/frontend/index.html`: Load `/static/js/log_dock.js?v=26`.
2. `dashboard/frontend/app.js`: Register `window.LogDockModule` into Alpine root initialization via `Object.assign`.
3. `dashboard/frontend/js/utils.js`: Implement `extractPathFromLogMsg(msg)` contextual tokenizer; update `parseJsonLogLine`.
4. `dashboard/frontend/js/profiles.js`: Upgrade `selectProfile(id, overrideDir = null)` to return a `Promise`.
5. `dashboard/frontend/js/logs.js`: Add `getEntryPath(entry)`, `scrollToHighlightedRow(fileName)`, and `jumpToExplorerPath(rawPath, entry, options)`.
6. `dashboard/frontend/js/review_actions.js`: Update `getItemReviewRowClass` to append pulse animation when `explorer.highlightName === item.Name`.
7. `dashboard/frontend/partials/view_explorer.html`: Add `:data-item-name="item.Name"`; embed `{{ template "component_log_dock" . }}`.
8. `dashboard/frontend/partials/patterns/pattern_timeline_ledger.html`: Clickable path chips; add `[ 📌 Dock ]` action button.
9. `dashboard/frontend/partials/modals/modal_daemon.html`: Interactive `jumpToExplorerPath` chips.
10. `docs/roadmap/backlog.md` & `docs/roadmap/ROADMAP.md`: Synchronize deliverables through `v1.0.0-beta.31`.
11. `tests/e2e/dashboard_smoke_test.go`: Add automated chromedp E2E tests.

---

## Functions

### New Functions

- `extractPathFromLogMsg(msg: string): string | null` (`dashboard/frontend/js/utils.js`):
  - Pre-filter: Rejects HTTP access entries (`GET /...`), web URLs (`https?://`), and bisync safety summaries (`too many deletes`, `exceed safety threshold`).
  - Prefix stripping: Strips leading timestamps (`\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}`) and log level labels (`INFO :`, `NOTICE:`, `ERROR :`).
  - Regex mutation matching:
    1. `/Duplicate ignored:\s*(.+)/i`
    2. `/(?:-\s+)?(?:Path[12]\s+)?File was deleted(?:\s*-\s*|\s*:\s*)(.+)/i`
    3. `/(?:-\s+)?(?:Path[12]\s+)?Deleted(?:\s*-\s*|\s*:\s*)(.+)/i`
    4. `/Transferring(?:\s+file)?:\s*(.+)/i`
    5. `/Conflict:\s*(.+?)\s+differs between/i`
    6. `/Copied(?:\s*\([^)]+\))?:\s*(.+)/i`
    7. `/Updated:\s*(.+)/i`
  - Sanitization: Removes surrounding quotes (`"`, `'`, `` ` ``), strips remote anchors (`^[a-zA-Z0-9_.-]+(?:\{[^}]+\})?:[/\\]?`), normalizes Windows backslashes (`\`) to forward slashes (`/`), and trims trailing punctuation (`,`, `;`).

- `getEntryPath(entry: object): string | null` (`dashboard/frontend/js/logs.js`):
  - Returns `entry.object || (entry.details && (entry.details.path || entry.details.item_path)) || extractPathFromLogMsg(entry.msg)`.

- `scrollToHighlightedRow(fileName: string): void` (`dashboard/frontend/js/logs.js`):
  - Escapes `fileName` using `CSS.escape`.
  - Queries `document.querySelector(`[data-item-name="${escaped}"]`)`.
  - Executes `targetRow.scrollIntoView({ behavior: 'smooth', block: 'center' })`.

- `async jumpToExplorerPath(rawPath: string, entry: object = {}, options: object = {}): Promise<void>` (`dashboard/frontend/js/logs.js`):
  - Parses `rawPath` into `{ dir, fileName, cleanPath }`.
  - If `entry.profile_id` differs from active profile: `await this.selectProfile(entry.profile_id, dir)`.
  - Otherwise: sets `this.explorer.left.path = dir`, `this.explorer.right.path = dir`, and `await Promise.all([this.fetchExplorerDirectory("left", this.explorer.left.target, dir), this.fetchExplorerDirectory("right", this.explorer.right.target, dir)])`.
  - Sets `this.explorer.highlightPath = cleanPath` and `this.explorer.highlightName = fileName`.
  - Auto-clears `highlightName` after 6000ms.
  - Closes daemon modal if open (`this.closeDaemonModal()`).
  - Switches view tab: `this.switchTab('explorer')`.
  - Executes `this.$nextTick(() => this.scrollToHighlightedRow(fileName))`.
  - If `options.openDock !== false` and `this.logDock`: calls `this.openLogDockWithEntry(entry)`.

- `openLogDockWithEntry(entry: object): void` (`dashboard/frontend/js/log_dock.js`):
  - Gathers actionable log lines from `this.filteredBisyncLogs` matching mutations/errors.
  - Sets `this.logDock.entries = actionableLogs`.
  - Locates `entry` within `this.logDock.entries` and assigns `this.logDock.activeIndex`.
  - Sets `this.logDock.isOpen = true`.

- `selectNext(): void` & `selectPrev(): void` (`dashboard/frontend/js/log_dock.js`):
  - Increments or decrements `logDock.activeIndex` with boundary clipping and calls `jumpCurrent()`.

- `jumpCurrent(): void` (`dashboard/frontend/js/log_dock.js`):
  - Invokes `jumpToExplorerPath(curr.object, curr, { keepDockOpen: true })`.

- `handleLogDockKeydown(e: KeyboardEvent): void` (`dashboard/frontend/js/log_dock.js`):
  - Binds global `window` keydown listener when `currentTab === 'explorer' && logDock.isOpen`.
  - Ignores keypresses when typing inside `<input>`, `<textarea>`, or `<select>`.
  - Maps `n` / `ArrowDown` to `selectNext()`, `p` / `ArrowUp` to `selectPrev()`, and `Escape` to closing the dock.

### Modified Functions

- `parseJsonLogLine(rawLine: string): object | null` (`dashboard/frontend/js/utils.js`):
  - Populates `object` via `parsed.object || extractPathFromLogMsg(cleanMsg)` for JSON lines, and applies `extractPathFromLogMsg` to legacy and raw fallbacks.

- `selectProfile(id: string, overrideDir: string | null = null): Promise<void>` (`dashboard/frontend/js/profiles.js`):
  - Converted to `async`. When `overrideDir` is provided, skips default profile root resetting and executes `await Promise.all([this.fetchExplorerDirectory('left', ...), this.fetchExplorerDirectory('right', ...)])` with `overrideDir`.

- `getItemReviewRowClass(paneKey: string, item: object): string` (`dashboard/frontend/js/review_actions.js`):
  - Appends `'ring-2 ring-indigo-500/80 bg-indigo-500/20 animate-pulse'` when `this.explorer?.highlightName === item.Name`.

---

## Testing

### Automated Headless Chrome E2E Test Suite (`tests/e2e/dashboard_smoke_test.go`)

1. **Contextual Tokenizer Vectors**:
   Validate `extractPathFromLogMsg` against 12 test vectors (spaces, quotes, remote prefixes, backslashes, negative cases).
2. **Synchronized Navigation Flow**:
   Assert tab switch, dual-pane path synchronization, and target row highlight.
3. **Pinned Log Dock Traversal**:
   Assert open state, active index, `N`/`P` keyboard stepping, and `Escape` close.

### Pre-Commit Quality Gate
Execute repository quality gate:
```bash
./scripts/check.sh templates
./scripts/check.sh loc 400
MIN_COVERAGE=85.0 RUN_RACE=true ./scripts/check.sh test
```

---

## Implementation Order

1. **Phase 1: Contextual Tokenizer Core (`utils.js`)**: Implement and export `extractPathFromLogMsg`.
2. **Phase 2: Concurrency & Async Navigation Engine (`profiles.js` & `logs.js`)**: Async profile switching with directory override.
3. **Phase 3: Visual Row Highlighting & Explorer Attributes (`review_actions.js` & `view_explorer.html`)**: Data attributes and pulse animation.
4. **Phase 4: Pinned Log Audit Dock Engine (`log_dock.js` & `component_log_dock.html`)**: Bottom dock partial, registration, and module.
5. **Phase 5: Ledger & Daemon Modal Integration (`pattern_timeline_ledger.html` & `modal_daemon.html`)**: Interactive chips and dock buttons.
6. **Phase 6: Roadmap Documentation & E2E Validation (`backlog.md`, `ROADMAP.md`, `dashboard_smoke_test.go`)**: Sync roadmap and pass quality checks.


