#!/usr/bin/env python3
import os
import sys
import json
import fcntl
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

class TelemetryPayload(BaseModel):
    daemon_health: DaemonHealth
    notification_state: NotificationState
    metrics: List[MetricEntry]
    bisync_logs: List[str]
    daemon_logs: List[str]
    config: dict
    is_syncing: bool = False

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
                return False
    except Exception:
        return False

@app.get("/api/telemetry", response_model=TelemetryPayload)
def get_telemetry():
    # 1. Parse Daemon Health File (cached by mtime/size)
    daemon_data = get_cached_json(
        config.DAEMON_HEALTH_FILE,
        default={"status": "unknown", "response_time_ms": 0.0, "last_check": 0.0}
    )

    # 2. Parse Notification State File (cached by mtime/size)
    notification_data = get_cached_json(
        config.NOTIFICATION_STATE_FILE,
        default={"last_success_timestamp": None, "last_error_type": None, "last_error_timestamp": None}
    )

    # 3. Parse Metrics Log (cached tail read without loading entire file)
    metrics_data = get_cached_metrics(config.METRICS_FILE, lines_count=30)

    # 4. Pull fresh streams for log files
    bisync_logs = tail_file(config.LOG_FILE, lines_count=40)
    
    # Resolve daemon log location path
    daemon_log_path = os.path.join(config.BASE_DIR, "rclone-daemon.log")
    daemon_logs = tail_file(daemon_log_path, lines_count=40)

    # 5. Harvest configuration variables dynamically, sanitizing sensitive keys
    runtime_config = {}
    for key in dir(config):
        if key.isupper():
            value = getattr(config, key)
            if isinstance(value, (str, int, float, bool, list, dict)) or value is None:
                if is_sensitive_key(key):
                    runtime_config[key] = "********" if (value is not None and value != "") else value
                else:
                    runtime_config[key] = value

    # 6. Check active sync state
    is_syncing = check_is_syncing()

    return {
        "daemon_health": daemon_data,
        "notification_state": notification_data,
        "metrics": metrics_data,
        "bisync_logs": bisync_logs,
        "daemon_logs": daemon_logs,
        "config": runtime_config,
        "is_syncing": is_syncing
    }

# Mount static files cleanly
FRONTEND_DIR = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "frontend")
app.mount("/static", StaticFiles(directory=FRONTEND_DIR), name="static")

@app.get("/")
def read_index():
    return FileResponse(os.path.join(FRONTEND_DIR, "index.html"))