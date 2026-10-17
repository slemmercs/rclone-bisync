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
from fastapi.responses import FileResponse, JSONResponse
from starlette.middleware.base import BaseHTTPMiddleware
from pydantic import BaseModel
from typing import List, Optional
from urllib.parse import urlparse

# Get the absolute path of backend/main.py
CURRENT_DIR = os.path.dirname(os.path.abspath(__file__))  # ~/dashboard/backend
# Go up two levels to reach the core rclone-bisync root
PROJECT_ROOT = os.path.abspath(os.path.join(CURRENT_DIR, "..", ".."))

if PROJECT_ROOT not in sys.path:
    sys.path.insert(0, PROJECT_ROOT)

try:
    import rclone_bisync.config as config
    from rclone_bisync.services.sync_service import SyncService
    from rclone_bisync.core.security import redact_secrets
    from rclone_bisync.models.api import (
        ActionRequest,
        ActionResponse,
        BrowseRequest,
        BrowseResponse,
        CanaryResponse,
        CopyRequest,
        CopyResponse,
        DeleteRequest,
        DeleteResponse,
        RemotesResponse,
        PendingDeletionsResponse,
        TelemetryPayload,
        DaemonHealth,
        NotificationState,
        MetricEntry,
        TransferStats,
    )
except ImportError as e:
    print(f"ERROR: Failed to import rclone_bisync modules: {e}")
    sys.exit(1)

sync_service = SyncService(config=config)
rc_client = sync_service.client

app = FastAPI(title="Rclone Bisync Telemetry API")

def is_allowed_origin(url_str: Optional[str]) -> bool:
    if not url_str:
        return True
    try:
        parsed = urlparse(url_str)
        # Allow requests originating from local loopback (any port)
        return parsed.hostname in ("localhost", "127.0.0.1")
    except Exception:
        return False

class CSRFProtectionMiddleware(BaseHTTPMiddleware):
    """Enforces origin validation and custom header checks on state-changing API requests."""
    async def dispatch(self, request, call_next):
        if request.method in ("POST", "PUT", "DELETE", "PATCH") and request.url.path.startswith("/api/"):
            # 1. Require custom header X-Dashboard-Request
            custom_header = request.headers.get("X-Dashboard-Request")
            if custom_header != "1":
                return JSONResponse(
                    status_code=403,
                    content={"status": "error", "msg": "Security policy violation: Missing required X-Dashboard-Request header."}
                )

            # 2. Origin & Referer checks
            origin = request.headers.get("origin")
            if origin and not is_allowed_origin(origin):
                return JSONResponse(
                    status_code=403,
                    content={"status": "error", "msg": f"Security policy violation: Untrusted request origin '{origin}'."}
                )

            referer = request.headers.get("referer")
            if referer and not is_allowed_origin(referer):
                return JSONResponse(
                    status_code=403,
                    content={"status": "error", "msg": "Security policy violation: Untrusted request referer."}
                )

        return await call_next(request)

app.add_middleware(CSRFProtectionMiddleware)


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

@app.get("/api/telemetry", response_model=TelemetryPayload)
def get_telemetry(limit: int = 40):
    payload = sync_service.get_telemetry_payload(limit=limit)
    return redact_secrets(payload)

@app.post("/api/action", response_model=ActionResponse)
def perform_action(req: ActionRequest):
    res = sync_service.handle_action(req.action)
    return redact_secrets(res)

@app.get("/api/deletions/pending", response_model=PendingDeletionsResponse)
def get_pending_deletions():
    """Returns the list of pending deletions detected during the last safety abort."""
    raw = sync_service.get_pending_deletions()
    if isinstance(raw, list):
        payload = {
            "has_deletions": len(raw) > 0,
            "total_count": len(raw),
            "files": raw,
        }
    elif isinstance(raw, dict):
        payload = raw
    else:
        payload = {"has_deletions": False, "total_count": 0, "files": []}
    return redact_secrets(payload)

@app.get("/api/remotes", response_model=RemotesResponse)
def get_remotes():
    """Discovers configured Rclone remotes, provider types, and capacity quotas."""
    summary = sync_service.get_remotes_summary()
    return redact_secrets(summary)

@app.post("/api/browse", response_model=BrowseResponse)
def browse_files(req: BrowseRequest):
    """Browses remote directories and checks for canary RCLONE_TEST presence."""
    res = sync_service.browse_path(remote=req.remote or "local", path=req.path or "")
    return redact_secrets(res)

@app.post("/api/browse/canary", response_model=CanaryResponse)
def provision_canary(req: BrowseRequest):
    """Provisions RCLONE_TEST canary file on target local or remote filesystem."""
    res = sync_service.create_canary(remote=req.remote or "local", path=req.path or "")
    return redact_secrets(res)

@app.post("/api/browse/copy", response_model=CopyResponse)
def copy_item(req: CopyRequest):
    """Copies a file or folder between local and remote directories."""
    res = sync_service.copy_item(
        src_remote=req.src_remote,
        src_path=req.src_path,
        dst_remote=req.dst_remote,
        dst_path=req.dst_path,
        is_dir=req.is_dir
    )
    return redact_secrets(res)

@app.post("/api/browse/delete", response_model=DeleteResponse)
def delete_item(req: DeleteRequest):
    """Deletes a file or directory from local or remote filesystem."""
    res = sync_service.delete_item(
        remote=req.remote,
        path=req.path,
        is_dir=req.is_dir
    )
    return redact_secrets(res)




# Mount static files cleanly
FRONTEND_DIR = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "frontend")
app.mount("/static", StaticFiles(directory=FRONTEND_DIR), name="static")

@app.get("/")
def read_index():
    return FileResponse(os.path.join(FRONTEND_DIR, "index.html"))

@app.get("/favicon.ico", include_in_schema=False)
def get_favicon():
    favicon_path = os.path.join(FRONTEND_DIR, "favicon.svg")
    if os.path.exists(favicon_path):
        return FileResponse(favicon_path, media_type="image/svg+xml")
    return JSONResponse(status_code=404, content={"detail": "Favicon not found"})
