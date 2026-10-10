"""
Error Classification & Taxonomy
"""

from typing import Optional


def classify_error(error_msg: Optional[str]) -> str:
    """Classify error message into categories for targeted automated recovery and alerting."""
    if not error_msg:
        return "UNKNOWN"

    msg_lower = error_msg.lower()

    if "prior lock file found" in msg_lower or "lock" in msg_lower:
        return "STALE_LOCK"

    # Missing listings baseline (requires resync)
    if "cannot find prior" in msg_lower:
        return "NO_LISTINGS"

    # Rate limiting
    if any(x in msg_lower for x in ["rate", "429", "quota exceeded"]):
        return "RATE_LIMIT"

    # Permissions and OAuth failures
    if any(x in msg_lower for x in [
        "permission denied", "401", "403", "unauthorized",
        "oauth", "invalid_grant", "token expired", "token revoked"
    ]):
        return "PERMISSION"

    # Network issues
    if any(x in msg_lower for x in ["timeout", "connection", "network", "unreachable", "handshake"]):
        return "NETWORK"

    # File conflicts
    if "conflict" in msg_lower:
        return "CONFLICT"

    # Daemon issues
    if "daemon" in msg_lower or "rcd" in msg_lower:
        return "DAEMON_ERROR"

    # Safety deletion thresholds
    if "too many deletes" in msg_lower:
        return "TOO_MANY_DELETES"

    # Canary check access failure
    if "rclone_test" in msg_lower or "checkaccess" in msg_lower:
        return "CHECK_ACCESS_FAILED"

    return "UNKNOWN"
