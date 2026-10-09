# Honest Code Review — rclone-bisync

> Reviewed after all recent fixes applied. Focus is on the layer that sits on top of rclone itself.

---

## What This Actually Is

A personal automation system with three layers:
1. **rclone daemon** — kept alive by systemd, exposes an HTTP RC API
2. **bisync_gdrive_advanced.py** — orchestrator that speaks to the RC API, handles errors, writes metrics, sends notifications
3. **Dashboard** — FastAPI backend + vanilla JS frontend polling every 3s

This is a well-scoped tool for one person. Judging it by enterprise standards would be unfair. The right question is: *is it reliable, safe, and easy to operate?*

---

## What's Genuinely Good

### Architecture
- **RC API over CLI subprocess** — the right call. No shell injection surface, no parsing stdout, proper async job tracking with `_async=true`. Most rclone wrappers never get here.
- **Daemon + timer separation** — rclone runs persistently, bisync is triggered by a timer. Clean. No wasted reconnections.
- **Config is a single flat file** — `bisync_config.py` is straightforward to read and modify. The cross-references to service file values (RC_PORT, rc-addr) are well documented inline.
- **Feature flags** — `FEATURES = {...}` dict lets you surgically disable subsystems without touching logic. Underrated.

### Reliability features (post-fixes)
- Lock file with stale detection — now race-free after the `O_RDWR | O_CREAT` fix
- Rate limit backoff with persistent flag file — survives process restarts
- `classify_error()` is now precise — `NO_LISTINGS` only fires on the real Rclone message
- Auto-resync has a guard against infinite loops (`if args.resync: stop`)
- Signal handling is now correct — SIGTERM→143, SIGINT→KeyboardInterrupt→130

### Dashboard
- `tail_file()` backward-seeking implementation is genuinely clever — O(lines_wanted) not O(file_size)
- mtime+size cache prevents redundant disk I/O on every 3s poll
- `is_syncing` via `LOCK_SH | LOCK_NB` is the right way to check — no PIDs, no races
- Sensitive key masking is heuristic-based (checks word fragments) — catches things you wouldn't have thought to list
- XSS protection is correct and complete (`escapeHtml` everywhere user data touches the DOM)
- Bundled Tailwind + Chart.js means it works offline/on-plane

---

## Real Problems That Remain

### 🔴 High — Still broken or risky

**1. `RCLONE_COMMON_ARGS` is only half-translated**

[`bisync_config.py` L53-90](file:///home/csaba/repos/tuxedo-aura15-gen3/rclone-bisync/bisync_config.py#L53-L90) has `--color NEVER` and `--use-json-log` in the list. `--color` has no entry in `CLI_TO_RC_MAP` and will be silently dropped. `--use-json-log` maps to `config_bool → UseJSONLog` but the RC API uses `LogJSON` not `UseJSONLog` — so the daemon is probably not sending JSON logs, meaning the dashboard's JSON log parser falls back to plaintext for everything.

**2. Daemon health file is never written**

`bisync_config.py` defines `DAEMON_HEALTH_FILE` and the dashboard reads it via `get_cached_json()`, but nothing in `bisync_gdrive_advanced.py` ever writes to it. `check_daemon_health()` makes HTTP calls but throws the result away. The dashboard's daemon health tile will always show `status: "unknown"` and `response_time_ms: 0`.

**3. `check_daemon_health()` retry loop burns 5 × RETRY_DELAY seconds on startup**

[`check_daemon_health()`](file:///home/csaba/repos/tuxedo-aura15-gen3/rclone-bisync/bisync_gdrive_advanced.py) retries 5 times with `RETRY_DELAY=1s` between attempts. On a healthy system this adds 0s overhead, but every cold start (daemon slow to come up) hangs the first sync job for up to 5 seconds with no feedback to the user. The daemon service already has `ExecStartPost` that waits for the daemon to answer — so by the time the timer fires, the daemon should already be healthy. The retry loop is likely never needed.

**4. `rc-no-auth` in rclone-daemon.service**

The daemon runs with `--rc-no-auth`, which means any local process or user on the machine can trigger a sync, delete files, or call `core/quit`. The dashboard has auth for its own API but the underlying rclone endpoint it proxies to is completely open. If you ever share this machine, or if something compromised runs as your user, it can call `sync/bisync` with arbitrary paths. Easy fix: enable `--rc-user / --rc-pass` in the service file and set `RC_AUTH_ENABLED = True` in config.

---

### 🟡 Medium — Correctness or operational issues

**5. `notification_state` is only updated on *notification send*, not on every sync outcome**

`notification_state["last_success_timestamp"]` is only written when a desktop notification is actually sent (throttled to 30 min). The dashboard reads this file to determine the last sync time. So for the majority of successful syncs, the "last sync" tile on the dashboard is wrong — it shows the last time a notification was fired, not the last time a sync ran. The metrics JSONL has the right data; the dashboard should use that instead for the "last run" display.

**6. `write_json_file()` is not atomic**

`open(filepath, 'w')` truncates before writing. If the process crashes mid-write (unlikely but possible for `notification_state.json` which is written after every error), the dashboard reads a truncated/empty JSON file and breaks. Fix: write to `filepath.tmp`, then `os.replace()`.

**7. Timer has no `RandomizedDelaySec`**

`bisync-gdrive.timer` fires at exact wall-clock minutes (`:0/15`). If you have other services also firing at :00/:15/:30/:45, they all hammer the network simultaneously after a resume from suspend. Adding `RandomizedDelaySec=60s` would spread them out with no user-visible impact.

**8. `psutil` used for PID checking but listed as optional**

If `psutil` isn't installed, `check_pid_exists()` presumably falls back to something weaker (or errors). The install script doesn't install `psutil`. Stale lock detection silently degrades without it, but the user gets no indication of this unless they see the startup `print()` warning — which goes to stdout, not the log file.

**9. Log format mismatch**

The Python logger writes `"%(asctime)s [%(levelname)s] %(message)s"` (plain text). The dashboard's `formatLogLine()` first tries to parse JSON (rclone daemon format), then falls back to checking for `[ERROR]`/`[WARNING]` strings. This works, but the two log streams (bisync + daemon) have fundamentally different formats and are displayed in identical UI panels, which is confusing.

---

### 🟢 Low — Polish and maintainability

**10. `STATE_FILE` is defined in config but never used**

`bisync_config.py` defines `STATE_FILE` but nothing reads or writes it. Dead config.

**11. `RCLONE_REMOTE` and `GDRIVE_FOLDER` defined but unused**

Same — defined but never referenced in the main script.

**12. `re` is imported but never used**

[`bisync_gdrive_advanced.py` L14](file:///home/csaba/repos/tuxedo-aura15-gen3/rclone-bisync/bisync_gdrive_advanced.py#L14) imports `re`. Nothing in the file uses it.

**13. Dashboard config panel reloads only once (`configLoaded = true`)**

If the config changes (you edit `bisync_config.py` and restart the dashboard), the config panel won't update until you hard-refresh the browser. Fine for personal use but worth knowing.

**14. `ExecStop` in rclone-daemon.service uses `core/quit`**

`/usr/bin/curl -sf -X POST http://localhost:5572/core/quit` — this is correct and clean. But if the daemon is unhealthy/crashed, `systemctl stop` will fail with a non-zero ExecStop and systemd will log an error. Low impact, but `ExecStop` failures don't prevent the unit from stopping.

---

## Summary Table

| Area | Grade | Notes |
|---|---|---|
| Architecture | **A** | RC API, daemon separation, config file — all solid |
| Safety (bisync) | **B+** | Lock race fixed, classify_error fixed. Auth still open |
| Safety (dashboard) | **A-** | XSS clean, secret masking good. No CSRF (localhost only so fine) |
| Reliability | **B** | Good retry logic, but notification_state timestamp drift, non-atomic writes |
| Observability | **C+** | Daemon health tile broken, log format mismatch, files_synced now fixed |
| Systemd integration | **B** | Service files clean. Timer missing jitter. cgroup v2 now fixed |
| Code cleanliness | **B-** | Dead imports/config vars, half-translated args list |

**Overall: B.** For a personal tool this is genuinely above average — the architecture decisions are right, the failure modes are mostly handled, and the dashboard is a nice touch. The gap between B and A is mostly the daemon health file never being written, the auth gap on the RC endpoint, and the `notification_state` timestamp drift affecting the dashboard's most prominent display value.
