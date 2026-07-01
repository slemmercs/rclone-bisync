#!/usr/bin/env python3
import os
import sys
import json
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

@app.get("/api/telemetry", response_model=TelemetryPayload)
def get_telemetry():
    # 1. Parse Daemon Health File
    daemon_data = {"status": "unknown", "response_time_ms": 0.0, "last_check": 0.0}
    if os.path.exists(config.DAEMON_HEALTH_FILE):
        try:
            with open(config.DAEMON_HEALTH_FILE, 'r') as f:
                daemon_data = json.load(f)
        except Exception:
            pass

    # 2. Parse Notification State File
    notification_data = {"last_success_timestamp": None, "last_error_type": None, "last_error_timestamp": None}
    if os.path.exists(config.NOTIFICATION_STATE_FILE):
        try:
            with open(config.NOTIFICATION_STATE_FILE, 'r') as f:
                notification_data = json.load(f)
        except Exception:
            pass

    # 3. Parse Metrics Log
    metrics_data = []
    if os.path.exists(config.METRICS_FILE):
        try:
            with open(config.METRICS_FILE, 'r') as f:
                lines = f.readlines()
                for line in lines[-30:]:
                    if line.strip():
                        metrics_data.append(json.loads(line.strip()))
        except Exception:
            pass

    # 4. Pull fresh streams for log files
    bisync_logs = tail_file(config.LOG_FILE, lines_count=40)
    
    # Resolve daemon log location path
    daemon_log_path = os.path.join(config.BASE_DIR, "rclone-daemon.log")
    daemon_logs = tail_file(daemon_log_path, lines_count=40)

    # 5. Harvest configuration variables dynamically
    runtime_config = {}
    for key in dir(config):
        if key.isupper():
            value = getattr(config, key)
            if isinstance(value, (str, int, float, bool, list, dict)) or value is None:
                runtime_config[key] = value

    return {
        "daemon_health": daemon_data,
        "notification_state": notification_data,
        "metrics": metrics_data,
        "bisync_logs": bisync_logs,
        "daemon_logs": daemon_logs,
        "config": runtime_config
    }

# Mount static files cleanly
FRONTEND_DIR = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "frontend")
app.mount("/static", StaticFiles(directory=FRONTEND_DIR), name="static")

@app.get("/")
def read_index():
    return FileResponse(os.path.join(FRONTEND_DIR, "index.html"))