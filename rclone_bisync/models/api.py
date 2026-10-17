"""
Pydantic API schemas and response DTOs for dashboard and operational endpoints.
"""

from typing import List, Optional, Dict, Any, Union
from pydantic import BaseModel, Field


# ---------------------------------------------------------------------------
# Common / Operational Schemas
# ---------------------------------------------------------------------------

class ActionResponse(BaseModel):
    """Standard response model for operational triggers and actions."""
    status: str = Field(..., description="Action outcome: 'success', 'error', or 'skipped'")
    msg: Optional[str] = Field(None, description="Human-readable status summary or diagnostic message")
    error: Optional[str] = Field(None, description="Detailed error description when status is 'error'")


class ActionRequest(BaseModel):
    """Request model to execute an engine operation."""
    action: str = Field(..., description="Action identifier: start_sync, stop_sync, force_sync, force_resync, dry_run, remove_lock")


# ---------------------------------------------------------------------------
# File Explorer & Path Schemas
# ---------------------------------------------------------------------------

class BrowseRequest(BaseModel):
    """Request model for listing filesystem contents or provisioning canaries."""
    remote: Optional[str] = Field("local", description="Remote identifier ('local' or configured cloud remote)")
    path: Optional[str] = Field("", description="Subdirectory path relative to remote/local root")


class BrowseItem(BaseModel):
    """Directory entry representation."""
    Path: Optional[str] = Field(None, description="Item relative path")
    Name: str = Field(..., description="File or directory name")
    Size: int = Field(0, description="Size in bytes (-1 or 0 for directories)")
    Mtime: Optional[str] = Field(None, description="ISO-8601 modification timestamp")
    ModTime: Optional[str] = Field(None, description="ISO-8601 modification timestamp alias from Rclone")
    MimeType: Optional[str] = Field(None, description="MIME type if available")
    IsDir: bool = Field(False, description="True if entry is a directory")



class BrowseResponse(BaseModel):
    """Response payload for directory traversal."""
    remote: str = Field(..., description="Active remote identifier (e.g. 'local' or 'gdrive:')")
    path: str = Field("", description="Resolved relative directory path")
    items: List[BrowseItem] = Field(default_factory=list, description="List of entries in directory")
    has_canary: bool = Field(False, description="Whether RCLONE_TEST is present in this path")
    error: Optional[str] = Field(None, description="Traversal error message if aborted or forbidden")


class CanaryResponse(BaseModel):
    """Response model for canary provisioning."""
    status: str = Field(..., description="'success' or 'error'")
    remote: Optional[str] = Field(None, description="Target remote")
    path: Optional[str] = Field(None, description="Path where canary was created")
    msg: Optional[str] = Field(None, description="Status details")
    error: Optional[str] = Field(None, description="Error reason if provisioning failed")


class CopyRequest(BaseModel):
    """Request model for copying files or directories between remotes."""
    src_remote: str = Field("local", description="Source remote identifier")
    src_path: str = Field(..., description="Source path")
    dst_remote: str = Field(..., description="Destination remote identifier")
    dst_path: str = Field("", description="Destination path")
    is_dir: bool = Field(False, description="Whether transferring a directory")


class CopyResponse(BaseModel):
    """Response model for copy operations."""
    status: str = Field(..., description="'success' or 'error'")
    msg: Optional[str] = Field(None, description="Success message")
    error: Optional[str] = Field(None, description="Error message if copy failed")


class DeleteRequest(BaseModel):
    """Request model for deleting files or directories."""
    remote: str = Field("local", description="Target remote identifier")
    path: str = Field(..., description="Target file or directory path")
    is_dir: bool = Field(False, description="True if deleting a directory")


class DeleteResponse(BaseModel):
    """Response model for delete operations."""
    status: str = Field(..., description="'success' or 'error'")
    msg: Optional[str] = Field(None, description="Success message")
    remote: Optional[str] = Field(None, description="Remote from which target was deleted")
    path: Optional[str] = Field(None, description="Path that was deleted")
    error: Optional[str] = Field(None, description="Error message if deletion failed")


# ---------------------------------------------------------------------------
# Storage Remotes & Pending Deletions Schemas
# ---------------------------------------------------------------------------

class RemoteCardInfo(BaseModel):
    """Detailed summary card for a configured remote."""
    name: str
    type: str
    fs: Optional[str] = None
    type_label: Optional[str] = None
    is_active_target: bool = False
    about: Dict[str, Any] = Field(default_factory=dict)
    total_bytes: Optional[int] = None
    used_bytes: Optional[int] = None
    free_bytes: Optional[int] = None
    trashed_bytes: Optional[int] = None
    other_bytes: Optional[int] = None
    quota_error: Optional[str] = None


class RemotesSummary(BaseModel):
    """Aggregate capacity statistics across discovered remotes."""
    total_capacity_bytes: int = 0
    total_used_bytes: int = 0
    total_free_bytes: int = 0
    provider_count: Optional[int] = None



class RemotesResponse(BaseModel):
    """Response model for discovered remotes and overall storage summary."""
    remotes: List[RemoteCardInfo] = Field(default_factory=list)
    count: int = 0
    active_remote: Optional[str] = None
    summary: RemotesSummary = Field(default_factory=RemotesSummary)


class PendingDeletionEntry(BaseModel):
    """Record of a pending file deletion blocked by safety limits."""
    path: str
    remote: Optional[str] = None
    size: Optional[int] = None
    modified: Optional[str] = None


class PendingDeletionsResponse(BaseModel):
    """Structured response model for pending deletions review."""
    has_deletions: bool = False
    total_count: int = 0
    source: Optional[str] = None
    target_action: Optional[str] = None
    message: Optional[str] = None
    files: List[Any] = Field(default_factory=list)
    error: Optional[str] = None
    aborted_at: Optional[float] = None



# ---------------------------------------------------------------------------
# Telemetry Schemas
# ---------------------------------------------------------------------------

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
    run_id: Optional[str] = None
    job_id: Optional[int] = None
    origin: Optional[str] = "unknown"
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
    transferring: List[Dict[str, Any]] = Field(default_factory=list)


class TelemetryPayload(BaseModel):
    daemon_health: DaemonHealth
    notification_state: NotificationState
    metrics: List[MetricEntry] = Field(default_factory=list)
    bisync_logs: List[str] = Field(default_factory=list)
    daemon_logs: List[str] = Field(default_factory=list)
    config: Dict[str, Any] = Field(default_factory=dict)
    is_syncing: bool = False
    transfer_stats: Optional[TransferStats] = None

