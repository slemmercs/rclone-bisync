import pytest
from rclone_bisync.engine.classifier import classify_error


def test_classify_no_listings():
    assert classify_error("Fatal: cannot find prior path1 or path2 listings") == "NO_LISTINGS"


def test_classify_rate_limit():
    assert classify_error("HTTP 429 Too Many Requests: user rate limit exceeded") == "RATE_LIMIT"
    assert classify_error("Google drive quota exceeded for user") == "RATE_LIMIT"


def test_classify_permissions_and_oauth():
    assert classify_error("401 Unauthorized") == "PERMISSION"
    assert classify_error("Permission Denied: access to remote refused") == "PERMISSION"
    # OAuth token expiration / invalid grant test
    assert classify_error("oauth2: cannot fetch token: 400 Bad Request Response: invalid_grant") == "PERMISSION"
    assert classify_error("Token expired or revoked by user") == "PERMISSION"


def test_classify_stale_lock():
    assert classify_error("Prior lock file found on remote path") == "STALE_LOCK"


def test_classify_too_many_deletions():
    assert classify_error("Safety abort: too many deletes found in path2") == "TOO_MANY_DELETES"


def test_classify_check_access():
    assert classify_error("RCLONE_TEST missing from local folder") == "CHECK_ACCESS_FAILED"


def test_classify_network():
    assert classify_error("Connection timed out after 30 seconds") == "NETWORK"
    assert classify_error("TLS handshake error: remote host unreachable") == "NETWORK"


def test_classify_unknown():
    assert classify_error("Some random unexpected error string") == "UNKNOWN"
    assert classify_error(None) == "UNKNOWN"
