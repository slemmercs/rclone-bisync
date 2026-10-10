"""
Atomic State Storage & Metrics Management
"""

import os
import glob
import json
import time
import logging
from typing import Any, Dict, Optional


def read_json_file(filepath: str, default: Any = None) -> Any:
    """Read a JSON file safely, returning default on missing or corrupt file."""
    if default is None:
        default = {}
    if not os.path.exists(filepath):
        return default
    try:
        with open(filepath, "r", encoding="utf-8") as f:
            return json.load(f)
    except (json.JSONDecodeError, IOError) as e:
        logging.warning(f"Failed to parse JSON file {filepath}: {e}")
        return default


def write_json_file(filepath: str, data: Any) -> bool:
    """Write a JSON file atomically using tmp file, flush, fsync, and os.replace."""
    os.makedirs(os.path.dirname(filepath), exist_ok=True)
    tmp_path = f"{filepath}.tmp.{os.getpid()}"
    try:
        with open(tmp_path, "w", encoding="utf-8") as f:
            json.dump(data, f, indent=2)
            f.flush()
            os.fsync(f.fileno())
        os.replace(tmp_path, filepath)
        return True
    except IOError as e:
        logging.error(f"Failed to atomically write {filepath}: {e}")
        if os.path.exists(tmp_path):
            try:
                os.remove(tmp_path)
            except OSError:
                pass
        return False


def load_notification_state(filepath: str) -> Dict[str, Any]:
    """Load notification state timestamp metadata."""
    return read_json_file(filepath, {
        "last_success_timestamp": 0,
        "last_error_type": None,
        "last_error_timestamp": 0
    })


def write_notification_state(filepath: str, state: Dict[str, Any]) -> bool:
    """Save notification state atomically."""
    return write_json_file(filepath, state)


def check_rate_limit_backoff(flag_path: str, backoff_hours: float) -> bool:
    """Returns True if within active rate limit backoff period."""
    if not os.path.exists(flag_path):
        return False
    try:
        with open(flag_path, "r", encoding="utf-8") as f:
            backoff_start = float(f.read().strip())
        elapsed_hours = (time.time() - backoff_start) / 3600
        if elapsed_hours < backoff_hours:
            remaining = backoff_hours - elapsed_hours
            logging.info(f"In rate limit backoff period. Remaining: {remaining:.1f} hour(s)")
            return True
        else:
            try:
                os.remove(flag_path)
            except OSError:
                pass
            logging.info("Rate limit backoff expired. Resuming sync.")
            return False
    except (ValueError, IOError):
        try:
            os.remove(flag_path)
        except OSError:
            pass
        return False


def set_rate_limit_backoff(flag_path: str) -> bool:
    """Write current timestamp to rate limit flag file."""
    try:
        os.makedirs(os.path.dirname(flag_path), exist_ok=True)
        with open(flag_path, "w", encoding="utf-8") as f:
            f.write(str(time.time()))
        return True
    except IOError as e:
        logging.error(f"Failed to record rate limit flag: {e}")
        return False


def clear_rate_limit_backoff(flag_path: str) -> bool:
    """Remove rate limit flag file."""
    if os.path.exists(flag_path):
        try:
            os.remove(flag_path)
            return True
        except OSError as e:
            logging.warning(f"Failed to clear rate limit flag: {e}")
            return False
    return False


def log_metrics_entry(
    metrics_file: str,
    entry: Dict[str, Any],
    max_entries: int = 1000,
    max_backups: int = 5
) -> bool:
    """Append a metric record to JSONL and prune rotated archives to max_backups."""
    try:
        os.makedirs(os.path.dirname(metrics_file), exist_ok=True)
        with open(metrics_file, "a", encoding="utf-8") as f:
            json.dump(entry, f)
            f.write("\n")

        # Rotate if entry threshold exceeded
        if os.path.exists(metrics_file):
            line_count = sum(1 for _ in open(metrics_file, "r", encoding="utf-8"))
            if line_count > max_entries:
                rotated_file = f"{metrics_file}.{int(time.time())}"
                os.rename(metrics_file, rotated_file)
                logging.info(f"Metrics file rotated to {rotated_file}")

                # Prune older rotated metric archives
                existing_backups = sorted(glob.glob(f"{metrics_file}.*"))
                if len(existing_backups) > max_backups:
                    for old_backup in existing_backups[:-max_backups]:
                        try:
                            os.remove(old_backup)
                            logging.debug(f"Pruned old metrics backup: {old_backup}")
                        except OSError:
                            pass
        return True
    except IOError as e:
        logging.warning(f"Failed to log metrics: {e}")
        return False
