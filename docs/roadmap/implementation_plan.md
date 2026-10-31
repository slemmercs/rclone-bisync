# Implementation Plan: Decommission Legacy Google Drive Systemd Timer & Generalize Deployment

## Proposed Changes

Bisync Hub features a built-in concurrent multi-profile scheduling subsystem (`internal/profiles/scheduler.go`) executed within `rclone-dashboard.service`. Each profile maintains independent cron or interval recurrence, advisory flock isolation, and notification rules.

The legacy `deploy/systemd/bisync-gdrive.timer` and `bisync-gdrive.service` units are redundant single-remote artifacts from early versions. They fire every 15 minutes against the root `bisync.env` regardless of user-configured profile schedules, creating lock contention and unpredictable background execution.

This plan removes the obsolete systemd units, updates `deploy/install.sh` to safely decommission any existing running timers on host machines, and generalizes transient unit naming in `internal/services/sync_actions.go`.

### Affected Components & Packages

- `deploy/systemd/bisync-gdrive.service`: Delete obsolete systemd service unit.
- `deploy/systemd/bisync-gdrive.timer`: Delete obsolete systemd timer unit.
- `deploy/install.sh`:
  - Remove `bisync-gdrive.*` from required file checks and copy lists.
  - Remove `bisync-gdrive.timer` from `systemctl --user enable` and `start` targets.
  - Add explicit decommissioning commands to stop, disable, and clean up lingering `bisync-gdrive.service` and `bisync-gdrive.timer` files on target systems during installation/upgrade.
- `internal/services/sync_actions.go`:
  - Generalize transient unit names from `bisync-gdrive-*` to `bisync-sync-*`.
  - Update `SyslogIdentifier` to `bisync-hub`.
  - In `stop_sync`, halt both new unit targets (`bisync-sync-*`) and legacy unit names (`bisync-gdrive*`) to ensure backward-compatible cancellation.
- `docs/architecture/code-review.md`:
  - Note retirement of `bisync-gdrive.*` systemd units in favor of `ProfileScheduler`.

### Technical Strategy & Data Flow

```
[ System Boot / User Session ]
       │
       ├── rclone-daemon.service (Rclone RC API daemon)
       └── rclone-dashboard.service (Bisync Hub Server & Engine)
                    │
                    ▼
           [ ProfileScheduler ]
                    │
                    ├── Worker: Profile "Google Drive" (cron/interval, e.g., @every 1h)
                    ├── Worker: Profile "OneDrive"    (cron/interval, e.g., @every 30m)
                    └── Worker: Profile "Nextcloud"   (manual / event-driven)
```

1. **Clean Decommissioning**: When `install.sh` runs, it stops and disables `bisync-gdrive.timer` and `bisync-gdrive.service` if present in `$SYSTEMD_USER_DIR/`, preventing dual-scheduler collisions.
2. **Unified Systemd Management**: Only `rclone-daemon.service` and `rclone-dashboard.service` remain as systemd user units.
3. **Execution Safety**: Profile execution remains fully managed by `ProfileScheduler` with flock isolation (`.bisync-<profile>.lock`).

### Safety & Error Handling

- **Idempotent Decommissioning**: Stopping or disabling nonexistent units in `install.sh` will ignore non-zero exits (`2>/dev/null || true`) to prevent script aborts under `set -e`.
- **Backward-Compatible Stop**: `stop_sync` in `sync_actions.go` includes legacy unit patterns in the stop list to guarantee that any running legacy systemd unit will be halted if an upgrade occurs while a job is running.

## Verification Plan

- Validate `deploy/install.sh` syntax and execution flow via shellcheck and test run.
- Verify unit tests across `internal/services/...` pass with updated unit identifiers.
- Execute full pre-flight quality gate (`./scripts/check.sh all`) to confirm zero regressions.


