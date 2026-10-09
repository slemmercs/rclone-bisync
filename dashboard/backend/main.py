#!/usr/bin/env python3
import os
import sys
import json
import fcntl
import time
import requests
import subprocess
from fastapi import FastAPI
from fastapi.staticfiles import StaticFiles
from fastapi.responses import FileResponse
from pydantic import BaseModel
from typing import List, Optional

# Get the absolute path of backend/main.py
CURRENT_DIR = os.path.dirname(os.path.abspath(__file__))  # ~/dashboard/backend
# Go up two levels to reach the core rclone-bisync root
PROJECT_ROOT = os.path.abspath(os.path.join(CURRENT_DIR, "..", ".."))

sys.path.append(PROJECT_ROOT)

try:
    import bisync_config as config
except ImportError:
    print(f"ERROR: bisync_config.py could not be resolved from root path: {PROJECT_ROOT}")
    sys.exit(1)

app = FastAPI(title="Rclone Bisync Telemetry API")

class DaemonHealth(BaseModel):
    status: str
    response_time_ms: float
    last_check: float

class NotificationState(BaseModel):
    last_success_timestamp: Optional[float] = None
    last_error_type: Optional[str] = None
    last_error_timestamp: Optional[float] = None

class MetricEntry(BaseModel):
    timestamp: float
    datetime: str
    success: bool
    duration_seconds: float
    error_type: Optional[str] = None
    files_synced: int

class TransferStats(BaseModel):
    bytes: int = 0
    total_bytes: int = 0
    speed: float = 0.0
    speed_str: str = "0 B/s"
    eta: Optional[int] = None
    percentage: float = 0.0
    transferring: List[dict] = []

class TelemetryPayload(BaseModel):
    daemon_health: DaemonHealth
    notification_state: NotificationState
    metrics: List[MetricEntry]
    bisync_logs: List[str]
    daemon_logs: List[str]
    config: dict
    is_syncing: bool = False
    transfer_stats: Optional[TransferStats] = None

def get_transfer_stats() -> Optional[dict]:
    auth = (config.RC_USER, config.RC_PASS) if getattr(config, "RC_AUTH_ENABLED", False) else None
    try:
        res = requests.post(f"{config.RC_API_URL}/core/stats", timeout=1.5, auth=auth)
        if res.status_code == 200:
            data = res.json()
            total_bytes = data.get("totalBytes", 0)
            bytes_transferred = data.get("bytes", 0)
            percentage = (bytes_transferred / total_bytes * 100) if total_bytes > 0 else 0.0
            
            # Format speed nicely
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

@app.get("/api/logs/bisync/raw")
def get_raw_bisync_log():
    """Serves the raw bisync.log file as plain text."""
    if os.path.exists(config.LOG_FILE):
        return FileResponse(config.LOG_FILE, media_type="text/plain", filename="bisync.log")
    return {"error": "Log file not found"}

@app.get("/api/logs/daemon/raw")
def get_raw_daemon_log():
    """Serves the raw rclone-daemon.log file as plain text."""
    daemon_log_path = os.path.join(config.BASE_DIR, "rclone-daemon.log")
    if os.path.exists(daemon_log_path):
        return FileResponse(daemon_log_path, media_type="text/plain", filename="rclone-daemon.log")
    return {"error": "Log file not found"}

def tail_file(filepath: str, lines_count: int = 40) -> List[str]:
    """Accurately reads the last N whole lines of a file by seeking backward from the end."""
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
            # The first element might be an incomplete line segment; store for next iteration
            remainder = chunk_lines[0]
            
            # Walk backward through complete lines in this chunk
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

# Keys and word patterns that should be masked in telemetry responses
SENSITIVE_CONFIG_KEYS = {"RC_PASS", "WEBHOOK_URL"}
SENSITIVE_WORDS = {"PASS", "PASSWORD", "SECRET", "TOKEN", "KEY", "CREDENTIAL", "CREDENTIALS"}

def is_sensitive_key(key: str) -> bool:
    key_upper = key.upper()
    if key_upper in SENSITIVE_CONFIG_KEYS:
        return True
    parts = set(key_upper.split("_"))
    return bool(parts & SENSITIVE_WORDS)

# In-memory cache for JSON files based on mtime and file size
_file_cache = {}

def get_cached_json(filepath: str, default=None):
    """Loads JSON file with mtime/size caching to prevent redundant disk I/O."""
    if default is None:
        default = {}
    if not os.path.exists(filepath):
        return default
    try:
        stat = os.stat(filepath)
        cached = _file_cache.get(filepath)
        if cached and cached["mtime"] == stat.st_mtime and cached["size"] == stat.st_size:
            return cached["data"]
        with open(filepath, 'r') as f:
            data = json.load(f)
        _file_cache[filepath] = {
            "mtime": stat.st_mtime,
            "size": stat.st_size,
            "data": data
        }
        return data
    except Exception:
        return default

_metrics_cache = {"mtime": 0, "size": 0, "data": []}

def get_cached_metrics(filepath: str, lines_count: int = 30) -> List[dict]:
    """Tails only the last N lines of metrics.jsonl and caches result by mtime and file size."""
    if not os.path.exists(filepath):
        return []
    try:
        stat = os.stat(filepath)
        if _metrics_cache["mtime"] == stat.st_mtime and _metrics_cache["size"] == stat.st_size:
            return _metrics_cache["data"]

        # Efficient backward tail instead of reading entire file into memory
        raw_lines = tail_file(filepath, lines_count=lines_count)
        parsed = []
        for line in raw_lines:
            line_str = line.strip()
            if line_str:
                try:
                    parsed.append(json.loads(line_str))
                except (json.JSONDecodeError, ValueError):
                    continue

        _metrics_cache["mtime"] = stat.st_mtime
        _metrics_cache["size"] = stat.st_size
        _metrics_cache["data"] = parsed
        return parsed
    except Exception:
        return _metrics_cache.get("data", [])

def check_is_syncing() -> bool:
    """Check if a synchronization run is actively in progress using the file lock."""
    if not os.path.exists(config.LOCK_FILE):
        return False

    # Test non-blockingly if the lock is held by a running process
    try:
        with open(config.LOCK_FILE, 'r') as f:
            try:
                fcntl.lockf(f, fcntl.LOCK_SH | fcntl.LOCK_NB)
                # Lock acquired successfully -> no active process is holding it
                fcntl.lockf(f, fcntl.LOCK_UN)
                return False
            except (BlockingIOError, IOError, OSError) as e:
                # EAGAIN (11) or EACCES (13) indicates lock is held by another process
                if getattr(e, "errno", None) in (11, 13):
                    return True
    except Exception:
        return False


_live_health_cache = {"time": 0.0, "data": None}

def get_daemon_health() -> dict:
    """Returns daemon health metrics.

    Uses DAEMON_HEALTH_FILE if updated within the last 10s. Otherwise performs
    a direct lightweight health check (with 5s in-memory throttle) so the
    dashboard shows live ONLINE/OFFLINE status and latency even between syncs.
    """
    now = time.time()
    cached = get_cached_json(config.DAEMON_HEALTH_FILE, default=None)
    if cached and (now - cached.get("last_check", 0)) <= 10:
        return cached

    if _live_health_cache["data"] and (now - _live_health_cache["time"]) <= 5:
        return _live_health_cache["data"]

    auth = (config.RC_USER, config.RC_PASS) if getattr(config, "RC_AUTH_ENABLED", False) else None
    try:
        res = requests.post(f"{config.RC_API_URL}/core/stats", timeout=1.5, auth=auth)
        if res.status_code == 200:
            health = {
                "status": "healthy",
                "response_time_ms": round(res.elapsed.total_seconds() * 1000, 2),
                "last_check": now
            }
        else:
            health = {
                "status": "offline",
                "response_time_ms": 0.0,
                "last_check": now
            }
    except Exception:
        health = {
            "status": "offline",
            "response_time_ms": 0.0,
            "last_check": now
        }

    _live_health_cache["time"] = now
    _live_health_cache["data"] = health
    return health


@app.get("/api/telemetry", response_model=TelemetryPayload)
def get_telemetry(limit: int = 40):
    # 1. Daemon Health
    daemon_data = get_daemon_health()

    # 2. Check active sync state early so transfer stats can use it
    is_syncing = check_is_syncing()
    transfer_stats = get_transfer_stats() if is_syncing else None

    # 3. Parse Notification State File
    notification_data = dict(get_cached_json(
        config.NOTIFICATION_STATE_FILE,
        default={"last_success_timestamp": None, "last_error_type": None, "last_error_timestamp": None}
    ))

    # 4. Parse Metrics Log
    metrics_data = get_cached_metrics(config.METRICS_FILE, lines_count=30)

    if metrics_data:
        successful_runs = [m.get("timestamp") for m in metrics_data if m.get("success") and m.get("timestamp")]
        if successful_runs:
            latest_metric_success = max(successful_runs)
            current_notif_success = notification_data.get("last_success_timestamp") or 0
            if latest_metric_success > current_notif_success:
                notification_data["last_success_timestamp"] = latest_metric_success

    # 5. Pull fresh streams for log files using the dynamic limit
    bisync_logs = tail_file(config.LOG_FILE, lines_count=limit)

    daemon_log_path = os.path.join(config.BASE_DIR, "rclone-daemon.log")
    daemon_logs = tail_json_log(daemon_log_path, entries_count=limit)

    # 6. Harvest configuration variables dynamically
    runtime_config = {}
    for key in dir(config):
        if key.isupper():
            value = getattr(config, key)
            if isinstance(value, (str, int, float, bool, list, dict)) or value is None:
                if is_sensitive_key(key):
                    runtime_config[key] = "********" if (value is not None and value != "") else value
                else:
                    runtime_config[key] = value

    return {
        "daemon_health": daemon_data,
        "notification_state": notification_data,
        "metrics": metrics_data,
        "bisync_logs": bisync_logs,
        "daemon_logs": daemon_logs,
        "config": runtime_config,
        "is_syncing": is_syncing,
        "transfer_stats": transfer_stats
    }

class ActionRequest(BaseModel):
    action: str

def run_bisync_systemd(flag: str, unit_name: str):
    """Executes bisync with specific flags under a systemd transient unit."""
    subprocess.run(["systemctl", "--user", "reset-failed", unit_name], check=False)
    cmd = [
        "systemd-run", "--user",
        f"--unit={unit_name}",
        "--service-type=oneshot",
        f"--working-directory={config.BASE_DIR}",
        f"--property=EnvironmentFile=-{os.path.expanduser('~')}/.config/dbus-vars",
        f"--property=EnvironmentFile=-{os.path.join(config.BASE_DIR, 'rc-secrets.env')}",
        "--property=MemoryMax=1G",
        "--property=CPUQuota=80%",
        "--property=NoNewPrivileges=true",
        "--property=PrivateTmp=true",
        "--property=StandardOutput=journal",
        "--property=StandardError=journal",
        "--property=SyslogIdentifier=bisync-gdrive",
        sys.executable, "-u", os.path.join(config.BASE_DIR, "bisync_gdrive_advanced.py"),
        flag
    ]
    subprocess.run(cmd, check=True)

@app.post("/api/action")
def perform_action(req: ActionRequest):
    action = req.action
    try:
        if action == "start_sync":
            subprocess.run(["systemctl", "--user", "start", "bisync-gdrive.service"], check=True)
            return {"status": "success", "msg": "Sync started via systemd"}
            
        elif action == "stop_sync":
            subprocess.run(["systemctl", "--user", "stop", "bisync-gdrive.service", "bisync-gdrive-resync.service", "bisync-gdrive-force.service"], check=False)
            return {"status": "success", "msg": "Sync stopped"}
            
        elif action == "force_resync":
            if check_is_syncing():
                return {"status": "error", "msg": "Cannot resync: another sync is actively running."}
            run_bisync_systemd("--resync", "bisync-gdrive-resync")
            return {"status": "success", "msg": "Resync initiated via systemd"}

        elif action == "force_sync":
            if check_is_syncing():
                return {"status": "error", "msg": "Cannot force sync: another sync is actively running."}
            run_bisync_systemd("--force", "bisync-gdrive-force")
            return {"status": "success", "msg": "Force sync initiated via systemd (deletions authorized)"}
            
        elif action == "remove_lock":
            if check_is_syncing():
                return {"status": "error", "msg": "Cannot remove lock: a synchronization run is actively in progress."}
            if os.path.exists(config.LOCK_FILE):
                os.remove(config.LOCK_FILE)
                return {"status": "success", "msg": "Lock file removed"}
            return {"status": "info", "msg": "No lock file found"}
            
        else:
            return {"status": "error", "msg": "Unknown action"}
    except Exception as e:
        return {"status": "error", "msg": str(e)}

def tail_json_log(filepath: str, entries_count: int = 40) -> List[str]:
    """Reads log file, unwraps nested JSON logs from daemon wrapper entries, and returns valid single-line JSON strings."""
    if not os.path.exists(filepath):
        return [f"Log file target not found at: {filepath}"]

    raw_lines = tail_file(filepath, lines_count=entries_count * 2)
    entries = []

    for line in raw_lines:
        line_str = line.strip()
        if not line_str:
            continue
        try:
            parsed = json.loads(line_str)
            msg = parsed.get("msg")

            # Unwrap nested JSON logs stored inside daemon wrapper msg fields
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

# Mount static files cleanly
FRONTEND_DIR = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "frontend")
app.mount("/static", StaticFiles(directory=FRONTEND_DIR), name="static")

@app.get("/")
def read_index():
    return FileResponse(os.path.join(FRONTEND_DIR, "index.html"))
