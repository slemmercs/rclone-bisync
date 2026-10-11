"""
Sync Application Service
Encapsulates systemd transaction dispatch, process locking, log aggregation,
health polling, and storage remote discovery.
"""

import os
import sys
import json
import time
import requests
import subprocess
from typing import List, Dict, Any, Optional

import rclone_bisync.config as default_config
from rclone_bisync.core.lock import LockManager
from rclone_bisync.core.client import RcloneClient
from rclone_bisync.core.state import clear_rate_limit_backoff, read_pending_deletions

PROVIDER_LABELS = {
    "drive": "Google Drive",
    "onedrive": "Microsoft OneDrive",
    "s3": "Amazon S3 / S3-Compatible",
    "dropbox": "Dropbox",
    "box": "Box",
    "mega": "Mega",
    "b2": "Backblaze B2",
    "webdav": "WebDAV",
    "smb": "Samba / SMB",
    "ftp": "FTP",
    "sftp": "SFTP",
    "alias": "Alias Remote",
    "crypt": "Encrypted Remote"
}

SENSITIVE_CONFIG_KEYS = {"RC_PASS", "WEBHOOK_URL"}
SENSITIVE_WORDS = {"PASS", "PASSWORD", "SECRET", "TOKEN", "KEY", "CREDENTIAL", "CREDENTIALS"}


class SyncService:
    """Core domain service orchestrating bisync execution, systemd, and telemetry."""

    def __init__(self, config=None, rc_client: Optional[RcloneClient] = None):
        self.config = config or default_config
        if rc_client:
            self.client = rc_client
        else:
            self.client = RcloneClient(
                self.config.RC_API_URL,
                user=self.config.RC_USER if getattr(self.config, "RC_AUTH_ENABLED", False) else None,
                password=self.config.RC_PASS if getattr(self.config, "RC_AUTH_ENABLED", False) else None
            )
        self._file_cache: Dict[str, Any] = {}
        self._metrics_cache: Dict[str, Any] = {"mtime": 0, "size": 0, "data": []}
        self._live_health_cache: Dict[str, Any] = {"time": 0.0, "data": None}

    def is_syncing(self) -> bool:
        """Checks if a sync is actively in progress using the fcntl lockfile."""
        return LockManager.is_locked(self.config.LOCK_FILE)

    def run_bisync_systemd(self, extra_args: list, unit_name: str, origin: str = "dashboard"):
        """Executes bisync with specific flags under a systemd transient unit."""
        subprocess.run(["systemctl", "--user", "reset-failed", unit_name], check=False)
        cmd = [
            "systemd-run", "--user",
            f"--unit={unit_name}",
            "--service-type=oneshot",
            f"--working-directory={self.config.BASE_DIR}",
            f"--property=EnvironmentFile=-{os.path.expanduser('~')}/.config/dbus-vars",
            f"--property=EnvironmentFile={os.path.join(self.config.BASE_DIR, 'rc-secrets.env')}",
            f"--property=EnvironmentFile={os.path.join(self.config.BASE_DIR, 'bisync.env')}",
            "--property=MemoryMax=1G",
            "--property=CPUQuota=80%",
            "--property=NoNewPrivileges=true",
            "--property=PrivateTmp=true",
            "--property=StandardOutput=journal",
            "--property=StandardError=journal",
            "--property=SyslogIdentifier=bisync-gdrive",
            sys.executable, "-u", os.path.join(self.config.BASE_DIR, "bisync_gdrive_advanced.py"),
            f"--origin={origin}"
        ]
        if extra_args:
            cmd.extend(extra_args)
        subprocess.run(cmd, check=True)

    def handle_action(self, action: str) -> Dict[str, Any]:
        """Dispatches maintenance and operational commands."""
        try:
            if action == "start_sync":
                if self.is_syncing():
                    return {"status": "error", "msg": "Sync already in progress."}
                self.run_bisync_systemd([], "bisync-gdrive-manual")
                return {"status": "success", "msg": "Sync started via systemd (origin: dashboard)"}

            elif action == "stop_sync":
                stopped_jobs = self.client.stop_all_active_jobs()
                subprocess.run(
                    ["systemctl", "--user", "stop", "bisync-gdrive.service", "bisync-gdrive-manual.service",
                     "bisync-gdrive-resync.service", "bisync-gdrive-force.service", "bisync-gdrive-dryrun.service"],
                    check=False
                )
                return {"status": "success", "msg": f"Sync stopped ({stopped_jobs} daemon jobs halted)"}

            elif action == "force_resync":
                if self.is_syncing():
                    return {"status": "error", "msg": "Cannot resync: another sync is actively running."}
                self.run_bisync_systemd(["--resync"], "bisync-gdrive-resync")
                return {"status": "success", "msg": "Resync initiated via systemd (origin: dashboard)"}

            elif action == "force_sync":
                if self.is_syncing():
                    return {"status": "error", "msg": "Cannot force sync: another sync is actively running."}
                self.run_bisync_systemd(["--force"], "bisync-gdrive-force")
                return {"status": "success", "msg": "Force sync initiated via systemd (deletions authorized)"}

            elif action == "dry_run":
                if self.is_syncing():
                    return {"status": "error", "msg": "Cannot start dry run: another sync is actively running."}
                self.run_bisync_systemd(["--dry-run"], "bisync-gdrive-dryrun")
                return {"status": "success", "msg": "Dry-run simulation started via systemd (origin: dashboard)"}

            elif action == "remove_lock":
                if self.is_syncing():
                    return {"status": "error", "msg": "Cannot remove lock: a synchronization run is actively in progress."}
                if os.path.exists(self.config.LOCK_FILE):
                    os.remove(self.config.LOCK_FILE)
                    return {"status": "success", "msg": "Lock file removed"}
                return {"status": "info", "msg": "No lock file found"}

            elif action == "clear_rate_limit":
                if clear_rate_limit_backoff(self.config.RATE_LIMIT_BACKOFF_FILE):
                    return {"status": "success", "msg": "Rate limit backoff cleared"}
                return {"status": "info", "msg": "No active rate limit flag found"}

            else:
                return {"status": "error", "msg": f"Unknown action: '{action}'"}
        except Exception as e:
            return {"status": "error", "msg": str(e)}

    def get_daemon_health(self) -> Dict[str, Any]:
        """Returns daemon health metrics with disk and live in-memory caching."""
        now = time.time()
        cached = self.get_cached_json(self.config.DAEMON_HEALTH_FILE, default=None)
        if cached and (now - cached.get("last_check", 0)) <= 10:
            return cached

        if self._live_health_cache["data"] and (now - self._live_health_cache["time"]) <= 5:
            return self._live_health_cache["data"]

        healthy, latency, err = self.client.check_health(retries=1)
        health = {
            "status": "healthy" if healthy else "offline",
            "response_time_ms": latency if healthy else 0.0,
            "last_check": now
        }

        self._live_health_cache["time"] = now
        self._live_health_cache["data"] = health
        return health

    def get_transfer_stats(self) -> Optional[Dict[str, Any]]:
        """Queries /core/stats on Rclone RC when sync is active."""
        auth = (self.config.RC_USER, self.config.RC_PASS) if getattr(self.config, "RC_AUTH_ENABLED", False) else None
        try:
            res = requests.post(f"{self.config.RC_API_URL}/core/stats", timeout=1.5, auth=auth)
            if res.status_code == 200:
                data = res.json()
                total_bytes = data.get("totalBytes", 0)
                bytes_transferred = data.get("bytes", 0)
                percentage = (bytes_transferred / total_bytes * 100) if total_bytes > 0 else 0.0
                speed = data.get("speed", 0.0)
                speed_mb = round(speed / 1024 / 1024, 2)
                return {
                    "bytes": bytes_transferred,
                    "total_bytes": total_bytes,
                    "speed": speed,
                    "speed_str": f"{speed_mb} MB/s",
                    "eta": data.get("eta"),
                    "percentage": round(percentage, 1),
                    "transferring": data.get("transferring", [])
                }
        except Exception:
            pass
        return None


    def tail_file(self, filepath: str, lines_count: int = 40) -> List[str]:
        """Accurately reads the last N whole lines of a file by seeking backward from EOF."""
        if not os.path.exists(filepath):
            return [f"Log file target not found at: {filepath}"]

        lines = []
        buffer_size = 8192

        with open(filepath, 'rb') as f:
            f.seek(0, os.SEEK_END)
            position = f.tell()
            remainder = b""

            while position > 0 and len(lines) < lines_count:
                to_read = min(buffer_size, position)
                position -= to_read
                f.seek(position, os.SEEK_SET)
                chunk = f.read(to_read) + remainder

                chunk_lines = chunk.split(b'\n')
                remainder = chunk_lines[0]

                for line_bytes in reversed(chunk_lines[1:]):
                    if len(lines) < lines_count:
                        line_str = line_bytes.decode('utf-8', errors='replace').rstrip('\r')
                        if line_str:
                            lines.append(line_str)
                    else:
                        break

            if remainder and len(lines) < lines_count:
                line_str = remainder.decode('utf-8', errors='replace').rstrip('\r')
                if line_str:
                    lines.append(line_str)

        return list(reversed(lines))

    def tail_json_log(self, filepath: str, entries_count: int = 40) -> List[str]:
        """Reads log file, unwraps nested JSON logs, and returns single-line JSON strings."""
        if not os.path.exists(filepath):
            return [f"Log file target not found at: {filepath}"]

        raw_lines = self.tail_file(filepath, lines_count=entries_count * 2)
        entries = []

        for line in raw_lines:
            line_str = line.strip()
            if not line_str:
                continue
            try:
                parsed = json.loads(line_str)
                msg = parsed.get("msg")

                if isinstance(msg, str) and ("{\"" in msg or "{'" in msg):
                    sub_lines = msg.splitlines()
                    unwrapped_any = False
                    for sub_line in sub_lines:
                        sub_line = sub_line.strip()
                        if sub_line.startswith("{") and sub_line.endswith("}"):
                            try:
                                inner_json = json.loads(sub_line)
                                entries.append(json.dumps(inner_json))
                                unwrapped_any = True
                            except Exception:
                                pass
                    if not unwrapped_any:
                        entries.append(json.dumps(parsed))
                else:
                    entries.append(json.dumps(parsed))
            except Exception:
                entries.append(line_str)

        return entries[-entries_count:]

    def get_cached_json(self, filepath: str, default=None):
        """Loads JSON file with mtime/size caching to prevent redundant disk I/O."""
        if default is None:
            default = {}
        if not os.path.exists(filepath):
            return default
        try:
            stat = os.stat(filepath)
            cached = self._file_cache.get(filepath)
            if cached and cached["mtime"] == stat.st_mtime and cached["size"] == stat.st_size:
                return cached["data"]
            with open(filepath, 'r') as f:
                data = json.load(f)
            self._file_cache[filepath] = {
                "mtime": stat.st_mtime,
                "size": stat.st_size,
                "data": data
            }
            return data
        except Exception:
            return default

    def get_cached_metrics(self, lines_count: int = 30) -> List[Dict[str, Any]]:
        """Tails and caches metrics.jsonl."""
        filepath = self.config.METRICS_FILE
        if not os.path.exists(filepath):
            return []
        try:
            stat = os.stat(filepath)
            if self._metrics_cache["mtime"] == stat.st_mtime and self._metrics_cache["size"] == stat.st_size:
                return self._metrics_cache["data"]

            raw_lines = self.tail_file(filepath, lines_count=lines_count)
            parsed = []
            for line in raw_lines:
                line_str = line.strip()
                if line_str:
                    try:
                        parsed.append(json.loads(line_str))
                    except (json.JSONDecodeError, ValueError):
                        continue

            self._metrics_cache["mtime"] = stat.st_mtime
            self._metrics_cache["size"] = stat.st_size
            self._metrics_cache["data"] = parsed
            return parsed
        except Exception:
            return self._metrics_cache.get("data", [])

    def is_sensitive_key(self, key: str) -> bool:
        """Determines if a configuration attribute is sensitive."""
        key_upper = key.upper()
        if key_upper in SENSITIVE_CONFIG_KEYS:
            return True
        parts = set(key_upper.split("_"))
        return bool(parts & SENSITIVE_WORDS)

    def get_runtime_config(self) -> Dict[str, Any]:
        """Harvests configuration parameters with credential masking."""
        runtime_config = {}
        for key in dir(self.config):
            if key.isupper():
                value = getattr(self.config, key)
                if isinstance(value, (str, int, float, bool, list, dict)) or value is None:
                    if self.is_sensitive_key(key):
                        runtime_config[key] = "********" if (value is not None and value != "") else value
                    else:
                        runtime_config[key] = value
        return runtime_config


    def get_telemetry_payload(self, limit: int = 40) -> Dict[str, Any]:
        """Aggregates unified telemetry vectors for the UI dashboard."""
        daemon_data = self.get_daemon_health()
        is_syncing = self.is_syncing()
        transfer_stats = self.get_transfer_stats() if is_syncing else None

        notification_data = dict(self.get_cached_json(
            self.config.NOTIFICATION_STATE_FILE,
            default={"last_success_timestamp": None, "last_error_type": None, "last_error_timestamp": None}
        ))

        metrics_data = self.get_cached_metrics(lines_count=30)
        if metrics_data:
            successful_runs = [m.get("timestamp") for m in metrics_data if m.get("success") and m.get("timestamp")]
            if successful_runs:
                latest_metric_success = max(successful_runs)
                current_notif_success = notification_data.get("last_success_timestamp") or 0
                if latest_metric_success > current_notif_success:
                    notification_data["last_success_timestamp"] = latest_metric_success

        bisync_logs = self.tail_json_log(self.config.LOG_FILE, entries_count=limit)
        daemon_log_path = os.path.join(self.config.BASE_DIR, "rclone-daemon.log")
        daemon_logs = self.tail_json_log(daemon_log_path, entries_count=limit)

        return {
            "daemon_health": daemon_data,
            "notification_state": notification_data,
            "metrics": metrics_data,
            "bisync_logs": bisync_logs,
            "daemon_logs": daemon_logs,
            "config": self.get_runtime_config(),
            "is_syncing": is_syncing,
            "transfer_stats": transfer_stats
        }

    def parse_pending_deletions_from_log(self) -> Dict[str, Any]:
        """Parses aborted deletions from rclone-daemon.log as a fallback."""
        daemon_log = os.path.join(self.config.BASE_DIR, "rclone-daemon.log")
        if not os.path.exists(daemon_log):
            return {"has_deletions": False, "total_count": 0, "files": [], "source": "unknown"}

        deleted_files = []
        abort_msg = None
        source = "Google Drive (remote)"
        target_action = "Will be deleted from local disk (~/gdrive)"

        try:
            lines = self.tail_file(daemon_log, lines_count=2000)
            for line in reversed(lines):
                line_str = line.strip()
                if not line_str:
                    continue
                try:
                    data = json.loads(line_str)
                    raw_msg = data.get("msg", "")
                    sublines = raw_msg.splitlines() if isinstance(raw_msg, str) else [raw_msg]
                    for sub in reversed(sublines):
                        sub = sub.strip()
                        if sub.startswith("{") and sub.endswith("}"):
                            try:
                                inner = json.loads(sub)
                                imsg = inner.get("msg", "")
                            except Exception:
                                imsg = sub
                        else:
                            imsg = sub

                        if "too many deletes" in imsg.lower():
                            abort_msg = imsg
                            if "path2" in imsg.lower():
                                source = "Local (~/gdrive)"
                                target_action = "Will be deleted from Google Drive"
                            else:
                                source = "Google Drive (remote)"
                                target_action = "Will be deleted from local disk (~/gdrive)"

                        if "File was deleted" in imsg:
                            parts = imsg.split("File was deleted", 1)
                            if len(parts) == 2:
                                fp = parts[1].strip().lstrip("-").strip()
                                if fp and fp not in deleted_files:
                                    deleted_files.append(fp)

                    if abort_msg and "Synching Path1" in line_str:
                        break
                except Exception:
                    continue

            if deleted_files or abort_msg:
                return {
                    "has_deletions": True,
                    "total_count": len(deleted_files),
                    "source": source,
                    "target_action": target_action,
                    "message": abort_msg or f"{len(deleted_files)} files deleted",
                    "files": list(reversed(deleted_files))
                }
        except Exception as e:
            return {"has_deletions": False, "total_count": 0, "files": [], "error": str(e)}

        return {"has_deletions": False, "total_count": 0, "files": []}


    def get_pending_deletions(self) -> Dict[str, Any]:
        """Returns structured pending deletions or falls back to log parsing."""
        structured = read_pending_deletions(getattr(self.config, "PENDING_DELETIONS_FILE", ""))
        if structured:
            return structured
        return self.parse_pending_deletions_from_log()

    def get_remotes_summary(self) -> Dict[str, Any]:
        """Discovers configured cloud remotes, provider types, and quotas."""
        remotes = self.client.list_remotes()
        remote_data = []
        total_cap = 0
        total_used = 0
        total_free = 0

        active_remote = getattr(self.config, "REMOTE_PATH", "").rstrip(":")

        for r in remotes:
            clean_name = r.rstrip(":")
            cfg = self.client.get_remote_config(clean_name)
            about = self.client.get_remote_about(clean_name)
            rtype = cfg.get("type", "unknown")
            type_label = PROVIDER_LABELS.get(rtype, rtype.upper())
            is_active = (clean_name == active_remote)

            if about:
                total_cap += about.get("total", 0)
                total_used += about.get("used", 0)
                total_free += about.get("free", 0)

            remote_data.append({
                "name": clean_name,
                "fs": f"{clean_name}:",
                "type": rtype,
                "type_label": type_label,
                "is_active_target": is_active,
                "about": about or {}
            })

        summary = {
            "total_capacity_bytes": total_cap,
            "total_used_bytes": total_used,
            "total_free_bytes": total_free,
            "provider_count": len(remote_data)
        }

        return {
            "remotes": remote_data,
            "summary": summary,
            "active_remote": f"{active_remote}:" if active_remote else None,
            "count": len(remote_data)
        }

    def browse_path(self, remote: str, path: str = "") -> Dict[str, Any]:
        """Lists directory contents and checks canary presence."""
        return self.client.list_directory(remote=remote, path=path or "")

