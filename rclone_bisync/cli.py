"""
Command-Line Interface for Rclone Bisync
"""

import os
import sys
import json
import time
import signal
import argparse
import logging
import logging.handlers
from datetime import datetime

import rclone_bisync.config as config
from rclone_bisync.engine.runner import SyncRunner


class JsonLogFormatter(logging.Formatter):
    """Formats log records as single-line JSON matching Rclone's schema."""

    def __init__(self, run_id: str, origin: str):
        super().__init__()
        self.run_id = run_id
        self.origin = origin

    def format(self, record):
        entry = {
            "time": datetime.fromtimestamp(record.created).astimezone().isoformat(),
            "level": record.levelname.lower(),
            "msg": record.getMessage(),
            "source": f"{record.filename}:{record.lineno}",
            "origin": getattr(record, "origin", self.origin),
            "run_id": getattr(record, "run_id", self.run_id)
        }
        if record.exc_info:
            entry["error"] = self.formatException(record.exc_info)
        return json.dumps(entry, ensure_ascii=False)


def setup_logging(verbose: bool, run_id: str, origin: str):
    """Configure unified rotating JSON log for file, and readable output for stdout."""
    os.makedirs(os.path.dirname(config.LOG_FILE), mode=0o700, exist_ok=True)
    root_logger = logging.getLogger()
    root_logger.setLevel(logging.DEBUG if (verbose or config.DEBUG_MODE) else logging.INFO)
    root_logger.handlers = []

    file_handler = logging.handlers.RotatingFileHandler(
        config.LOG_FILE,
        maxBytes=config.MAX_LOG_SIZE_MB * 1024 * 1024,
        backupCount=config.LOG_BACKUP_COUNT,
        encoding="utf-8"
    )
    file_handler.setFormatter(JsonLogFormatter(run_id=run_id, origin=origin))
    root_logger.addHandler(file_handler)

    console_handler = logging.StreamHandler()
    console_handler.setFormatter(logging.Formatter("%(asctime)s [%(levelname)s] %(message)s", datefmt="%Y-%m-%d %H:%M:%S"))
    root_logger.addHandler(console_handler)


def signal_handler(signum, frame):
    """Exits with code 143 on SIGTERM for systemd convention."""
    logging.warning("Process received termination signal (SIGTERM). Aborting.")
    sys.exit(143)


def main():
    parser = argparse.ArgumentParser(description="Bidirectional synchronization with Google Drive via Rclone RC API")
    parser.add_argument("--resync", action="store_true", help="Rebuild synchronization cache tracking from baseline")
    parser.add_argument("--force", action="store_true", help="Authorize safety overrides including mass deletions")
    parser.add_argument("--dry-run", action="store_true", help="Simulate synchronization without applying filesystem changes")
    parser.add_argument("--no-notify", action="store_true", help="Suppress desktop notifications")
    parser.add_argument("--origin", type=str, default="cli", choices=["cli", "timer", "dashboard"], help="Origin caller context")
    parser.add_argument("--verbose", action="store_true", help="Enable verbose debug logging")

    args = parser.parse_args()

    run_id = f"sync_{datetime.now().strftime('%y%m%d_%H%M%S')}_{os.urandom(2).hex()}"
    setup_logging(verbose=args.verbose, run_id=run_id, origin=args.origin)

    signal.signal(signal.SIGTERM, signal_handler)

    logging.info("=" * 60)
    logging.info(f"Rclone Bisync Engine Started [Run ID: {run_id} | Origin: {args.origin}]")
    logging.info("=" * 60)

    runner = SyncRunner(config)

    try:
        success = runner.run(
            resync=args.resync,
            force=args.force,
            dry_run=args.dry_run,
            origin=args.origin,
            run_id=run_id,
            no_notify=args.no_notify
        )
        sys.exit(0 if success else 1)
    except KeyboardInterrupt:
        logging.warning("Sync interrupted by user (Ctrl+C)")
        sys.exit(130)
    except Exception as e:
        logging.exception(f"Unhandled sync engine exception: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()
