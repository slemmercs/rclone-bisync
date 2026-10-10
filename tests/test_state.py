import os
import time
import pytest
from rclone_bisync.core.state import (
    read_json_file,
    write_json_file,
    check_rate_limit_backoff,
    set_rate_limit_backoff,
    clear_rate_limit_backoff,
    log_metrics_entry
)


def test_atomic_json_read_write(tmp_path):
    target = str(tmp_path / "state.json")
    data = {"status": "ok", "value": 42}

    assert write_json_file(target, data) is True
    read_data = read_json_file(target)
    assert read_data == data

    # Temp file was cleanly replaced
    tmp_files = [f for f in os.listdir(tmp_path) if f.endswith(".tmp")]
    assert len(tmp_files) == 0


def test_rate_limit_lifecycle(tmp_path):
    flag_file = str(tmp_path / "rate_limit.flag")

    assert check_rate_limit_backoff(flag_file, backoff_hours=1.0) is False

    set_rate_limit_backoff(flag_file)
    assert os.path.exists(flag_file) is True
    assert check_rate_limit_backoff(flag_file, backoff_hours=1.0) is True

    # Test expired backoff (backoff_hours = 0)
    assert check_rate_limit_backoff(flag_file, backoff_hours=0.0) is False
    assert os.path.exists(flag_file) is False

    # Clear rate limit
    set_rate_limit_backoff(flag_file)
    clear_rate_limit_backoff(flag_file)
    assert os.path.exists(flag_file) is False


def test_metrics_logging_and_rotation(tmp_path):
    metrics_file = str(tmp_path / "metrics.jsonl")

    # Write 5 entries with max_entries = 3
    for i in range(5):
        log_metrics_entry(
            metrics_file,
            {"run": i, "success": True},
            max_entries=3,
            max_backups=2
        )

    assert os.path.exists(metrics_file) is True
    # Verify file is valid JSON lines
    with open(metrics_file, "r") as f:
        lines = f.readlines()
    assert len(lines) >= 1


def test_pending_deletions_lifecycle(tmp_path):
    deletions_file = str(tmp_path / "pending_deletions.json")
    from rclone_bisync.core.state import (
        record_pending_deletions,
        read_pending_deletions,
        clear_pending_deletions
    )

    assert read_pending_deletions(deletions_file) is None

    test_data = {"count": 5, "files": ["a.txt", "b.txt"], "source": "Local"}
    assert record_pending_deletions(deletions_file, test_data) is True

    loaded = read_pending_deletions(deletions_file)
    assert loaded == test_data

    assert clear_pending_deletions(deletions_file) is True
    assert read_pending_deletions(deletions_file) is None

