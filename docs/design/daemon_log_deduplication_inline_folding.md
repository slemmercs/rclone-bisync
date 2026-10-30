# Daemon & Unified Log Inline Deduplication and Folding Specification

## 1. Problem Statement

During deep synchronization runs, Rclone emits repetitive consecutive notice logs:
```
2026/10/28 17:15:01 Duplicate directory found: march/march.go:350
2026/10/28 17:15:02 Duplicate directory found: march/march.go:350
... (150 identical occurrences) ...
```
Similarly, batch review executions log dozens of consecutive identical audit entries (`DELETION_RESTORED by USER`).

When rendered without folding:
1. Significant cognitive overload: operators miss critical warnings buried between hundreds of repetitive lines.
2. Unnecessary DOM inflation: thousands of DOM nodes cause UI lag during live polling.

---

## 2. Architecture & Normalization Engine

In `dashboard/frontend/js/logs.js`:

```javascript
function normalizeLogMessage(msg) {
    if (!msg) return "";
    return msg
        .replace(/\[cid=[a-zA-Z0-9_-]+\]/g, "")
        .replace(/\b\d+(\.\d+)?\s*(s|ms|m|h|B|KB|MB|GB|TB|PB)\b/gi, "")
        .replace(/\b\d{4}-\d{2}-\d{2}[T\s]\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:?\d{2})?\b/g, "")
        .replace(/\b\d+\s+(transferred|checked|deleted|matching|errors)\b/gi, "")
        .replace(/\s+/g, " ")
        .trim();
}
```

Consecutive entries with identical normalized keys `(level + "::" + type + "::" + normalizedMsg)` are grouped into an inline cluster object:
```javascript
{
    id: "cluster-0-12",
    key: "INFO::sync::Duplicate directory found: march/march.go:350",
    entry: firstItem,
    count: 14,
    isCluster: true,
    expanded: false,
    entries: [ ...all 14 items... ]
}
```

---

## 3. UI Presentation

- **Folded Row (Collapsed)**:
  - Renders the primary entry with an inline badge `×14 repeated`.
  - Chevron icon indicates expandable status.
- **Unfolded Row (Expanded)**:
  - Clicking the row expands an indented sub-list containing all 14 individual entries with their original timestamps and messages.
- **Master Controls**:
  - `Fold Repeated` checkbox toggle in both `pattern_timeline_ledger.html` and `modal_daemon.html`.
  - `Expand All / Collapse All` toggle button.

---

## 4. 50-Line On-Demand Pagination

To prevent multi-megabyte log downloads from locking up the browser:
- Endpoint: `GET /api/logs/bisync?page=1&limit=50`.
- Backend uses `services.TailJSONLogPaginated(logFile, page, limit)` seeking from EOF.
- Returns `PaginatedLogsResponse` with `page`, `limit`, `total`, `total_pages`, `logs`.
- Page navigation: `[← Newer]` and `[Older →]` buttons with disabled boundary states.

