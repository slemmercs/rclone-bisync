# User Story: Interactive Sync Review & Resolution Workflow

## Status

Implemented (Phase 4 - `feat/interactive-sync-review`)

## Delivered Components

- Package `internal/engine/review/`: state machine, abort discovery, atomic write-ahead log (WAL), dual backup staging (zero-byte local hardlink + cloud server-side copy), reversible resolution executor, saga rollback runner.
- Package `internal/services/review_service.go`: thread-safe facade, pre-sync safety barrier, lazy-load session recovery.
- Package `internal/server/handlers_review.go`: REST endpoints (`/api/review/session`, `/api/review/action`, `/api/review/execute`, `/api/review/rollback`, `/api/review/dismiss`).
- Frontend `dashboard/frontend/js/review.js`: interactive review modal, batch staging, visual candidate indicators in dual-pane explorer.


## Epic

**As a user of Rclone Bisync Enterprise Automation Hub, I want to review and selectively resolve sync conflicts and mass deletion events before synchronization proceeds, so that I can safely control the outcome of synchronization without manually inspecting logs or using command-line recovery procedures.**

---

# Background

Today, when Rclone Bisync detects a dangerous synchronization condition such as:

```text
ERROR: TOO_MANY_DELETES
```

the synchronization aborts to prevent accidental data loss.

The user is then forced to:

1. Inspect logs manually.
2. Determine which files and folders are affected.
3. Decide whether the deletions are legitimate.
4. Potentially execute:

```bash
rclone bisync --force
```

However, this applies the entire synchronization plan as calculated by Bisync, including:

- Deletions
- Conflict resolutions
- Overwrites

The user cannot selectively choose outcomes on a per-file basis.

---

# Goal

Provide an **Interactive Sync Review Session** that allows users to:

- Review pending deletions.
- Review file conflicts.
- Visualize affected items in the Explorer.
- Make per-item decisions.
- Apply resolutions safely.
- Roll back resolution actions before final synchronization.
- Resume sessions after crashes or restarts.

---

# User Story 1: Launch Review Session

## Scenario

A synchronization run encounters:

```text
TOO_MANY_DELETES
```

or:

```text
Unresolved file conflicts
```

## Acceptance Criteria

### Given

A Bisync run is aborted because user review is required.

### When

The dashboard receives the synchronization result.

### Then

A new Review Session is created.

### And

The dashboard displays:

```text
Sync Review Required

Pending Deletions: 12
Pending Conflicts: 3

[Review Changes]
```

### And

Synchronization will not continue until all required decisions are resolved.

---

# User Story 2: Review Pending Deletions

## Scenario

The remote side contains several deleted folders.

### Example

```text
Remote:
  FolderA (deleted)
  FolderB (deleted)
  FolderC (deleted)
```

## Acceptance Criteria

The dashboard presents each deletion candidate individually.

Example:

```text
FolderA

Detected:
Deleted on Remote

Actions:

(o) Accept Deletion
( ) Restore Remotely
```

### Accept Deletion

User confirms:

```text
Yes, this deletion was intentional.
```

Result:

```text
Item marked RESOLVED.
```

No immediate data mutation is performed.

### Restore Remotely

User confirms:

```text
No, this item should remain.
```

Result:

The dashboard performs:

```text
Local → Remote restore
```

using Rclone RC operations.

After successful restoration:

```text
Deletion candidate disappears.
```

---

# User Story 3: Review File Conflicts

## Scenario

The same file was modified on both sides.

Example:

```text
Report.xlsx
```

Local:

```text
Modified: 10:12
Size: 420 KB
```

Remote:

```text
Modified: 10:18
Size: 427 KB
```

## Acceptance Criteria

The dashboard displays:

```text
Report.xlsx

LOCAL
Modified: 10:12
Size: 420 KB

REMOTE
Modified: 10:18
Size: 427 KB
```

Available actions:

```text
[Keep Local]
[Keep Remote]
[Keep Both]
```

### Keep Local

System performs:

```text
Local → Remote copy
```

Result:

```text
Both locations contain Local version.
```

Conflict removed.

### Keep Remote

System performs:

```text
Remote → Local copy
```

Result:

```text
Both locations contain Remote version.
```

Conflict removed.

### Keep Both

System duplicates one copy.

Examples:

```text
Report.xlsx
Report.local.xlsx
```

or:

```text
Report.remote.xlsx
```

Conflict removed.

Manual review can occur later.

---

# User Story 4: Explorer Visualization

## Scenario

The user opens the File Explorer during a Review Session.

## Acceptance Criteria

Affected items are visually highlighted.

### Red

```text
Pending deletion
```

### Amber

```text
Conflict requiring review
```

### Green

```text
New file
```

Example:

```text
LOCAL                    REMOTE

FolderA                  FolderA
Report.xlsx              Report.xlsx

FolderB                  FolderB

FolderC                  [deleted]
```

Color overlays provide immediate visibility into sync decisions.

---

# User Story 5: Resolution Queue

## Scenario

The user reviews multiple items before execution.

## Acceptance Criteria

Selections are stored as pending resolutions.

Example:

```text
✓ Accept deletion FolderA
✓ Restore FolderB
✓ Keep Local Report.xlsx
```

Synchronization does not execute immediately.

The user must explicitly apply decisions.

---

# User Story 6: Preview Planned Actions

## Scenario

The user finishes making decisions.

## Acceptance Criteria

The dashboard displays a final preview.

Example:

```text
Planned Actions

Restore:
  FolderB

Copy Local → Remote:
  Report.xlsx

Accept Deletions:
  FolderA
```

Options:

```text
[Apply Resolutions]
[Back]
```

No changes occur before confirmation.

---

# User Story 7: Transactional Resolution Application

## Scenario

The user applies resolutions.

## Acceptance Criteria

Actions execute sequentially.

Workflow:

```text
Backup
↓
Execute
↓
Verify
↓
Mark Complete
```

Example:

```text
1/3 Report.xlsx copied
2/3 FolderB restored
3/3 FolderA accepted
```

Progress is visible in the UI.

---

# User Story 8: Automatic Backup Before Mutation

## Scenario

A resolution will modify data.

## Acceptance Criteria

Before any operation:

- Keep Local
- Keep Remote
- Restore Item

the system automatically creates a targeted backup.

Example:

```text
cache/review-session-123/

Report.xlsx.remote.backup
FolderB.tar
```

Only affected items are backed up.

Entire repositories are never copied.

---

# User Story 9: Rollback

## Scenario

Resolution application succeeds but synchronization has not yet started.

## Acceptance Criteria

The user can roll back all applied resolutions.

Example:

```text
[Rollback Session]
```

The system restores backups.

Verification is performed after restoration.

Successful rollback returns the session to:

```text
REVIEWING
```

state.

---

# User Story 10: Failure Handling

## Scenario

One resolution action fails.

Example:

```text
403 Authentication Failed
```

or:

```text
Network Timeout
```

## Acceptance Criteria

Synchronization is immediately blocked.

The dashboard displays:

```text
Review Application Failed

Applied: 1
Failed: 1
Pending: 2
```

Options:

```text
[Retry Failed]
[Rollback Applied]
[Cancel Session]
```

Bisync must not execute.

---

# User Story 11: Crash Recovery

## Scenario

The browser, daemon, or host crashes while applying resolutions.

## Acceptance Criteria

The session state is persisted on disk.

Example:

```json
{
  "sessionId": "abc123",
  "status": "APPLYING",
  "completed": ["Report.xlsx"],
  "pending": ["FolderB"]
}
```

After restart:

```text
Interrupted Sync Review Detected

[Resume]
[Rollback]
```

No action is executed twice.

---

# User Story 12: Resolution Audit Journal

## Scenario

The user wants traceability.

## Acceptance Criteria

Every action is recorded.

Example:

```json
{
  "sessionId": "abc123",
  "item": "Report.xlsx",
  "action": "keep-local",
  "status": "completed",
  "timestamp": "2026-10-22T10:15:00Z"
}
```

Stored in:

```text
state/review-sessions/
```

or:

```text
state/review-journal.jsonl
```

---

# User Story 13: Final Synchronization

## Scenario

All review items have been resolved.

## Acceptance Criteria

The system performs a validation scan.

If unresolved items remain:

```text
Return to Review Session
```

If all issues are resolved:

```text
Ready for Synchronization
```

User executes:

```text
[Start Synchronization]
```

Only then is Bisync launched.

---

# Session Lifecycle

```text
CREATED
    │
    ▼
REVIEWING
    │
    ▼
APPLYING
    │
    ├─────────────┐
    ▼             │
READY             │
    │             │
    ▼             │
RUNNING_BISYNC    │
    │             │
    ▼             │
COMPLETE          │
                  │
                  ▼
                FAILED
                  │
                  ▼
               ROLLBACK
                  │
                  ▼
               REVIEWING
```

---

# Architecture Components

## Backend

### New Package

```text
internal/engine/review/
```

Suggested structure:

```text
internal/engine/review/
├── session.go
├── resolver.go
├── conflicts.go
├── deletions.go
├── backup.go
├── rollback.go
├── journal.go
└── dto.go
```

### Core Responsibilities

- Review session management
- Conflict discovery
- Deletion review handling
- Resolution execution
- Backup creation
- Rollback processing
- Journal persistence
- Crash recovery

---

## Review Session Model

```go
type ReviewSession struct {
    ID         string
    CreatedAt  time.Time
    Status     string

    Deletions  []DeletionCandidate
    Conflicts  []ConflictCandidate
    Actions    []ResolutionAction
}
```

---

## Resolution Actions

```go
type ResolutionAction struct {
    ID         string
    Type       string
    Item       string
    Status     string
    BackupPath string
}
```

Example action types:

```text
keep-local
keep-remote
keep-both
accept-delete
restore-item
```

---

## Audit Trail

Every mutation is persisted before execution.

Example:

```json
{
  "sessionId":"abc123",
  "action":"keep-local",
  "path":"Report.xlsx",
  "status":"completed"
}
```

This log is also used for crash recovery.

---

# Explorer Enhancements

## Visual States

### Red

```text
Deletion candidate
```

### Amber

```text
Conflict candidate
```

### Green

```text
New item
```

### Blue (Optional)

```text
Selected user decision
```

---

# Security & Safety Requirements

## Required

- All resolution actions must be journaled.
- All file mutations must be preceded by backup creation.
- Resolution execution must be idempotent.
- Bisync cannot start if unresolved items exist.
- Bisync cannot start if resolution execution partially fails.
- Rollback remains available until synchronization begins.
- Crash recovery must survive daemon restarts.

---

# MVP Scope

## Included

- Per-file conflict resolution
- Per-folder deletion approval
- Keep Local
- Keep Remote
- Keep Both
- Restore deleted file/folder
- Explorer highlighting
- Persistent Review Sessions
- Action journal
- Backup and rollback
- Crash recovery
- Final validation prior to Bisync

---

# Future Enhancements

## V2

- Batch resolution actions
- Search and filtering
- Folder-level resolution shortcuts
- Resolution templates
- Detailed action previews

## V3

- Text file diffs
- Office document previews
- Session history explorer
- Undo previous review session
- Snapshot retention management

## Explicitly Out of Scope

- Office document merge engine
- Binary file merging
- Git-style three-way merge
- Full filesystem snapshot rollback after Bisync execution

---

# Success Criteria

Users can:

- Approve some deletions while rejecting others.
- Resolve file conflicts individually.
- Keep local versions of selected files.
- Keep remote versions of selected files.
- Restore accidentally deleted content.
- Preview all actions before execution.
- Roll back decisions before synchronization starts.
- Recover cleanly from crashes or interruptions.

The feature elevates the dashboard from a monitoring tool into an **Interactive Sync Review Engine**, providing file-level control over Bisync outcomes while preserving transactional safety, auditability, and rollback guarantees.
