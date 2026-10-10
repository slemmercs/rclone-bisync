"""
Synchronization Engine Runner
Orchestrates lifecycle, automated recovery, and telemetry recording.
"""

import os
import glob
import time
import logging
from typing import Tuple, Optional, Dict, Any

from rclone_bisync.core.lock import LockManager
from rclone_bisync.core.client import RcloneClient
from rclone_bisync.core.compiler import compile_bisync_payload
from rclone_bisync.core.state import (
    load_notification_state,
    write_notification_state,
    check_rate_limit_backoff,
    set_rate_limit_backoff,
    write_json_file,
    log_metrics_entry
)
from rclone_bisync.engine.classifier import classify_error
from rclone_bisync.engine.notifier import NotificationDispatcher


class SyncRunner:
    """Orchestrates bidirectional sync transactions with automated recovery."""

    def __init__(self, cfg):
        self.config = cfg
        self.lock_mgr = LockManager(
            lock_file_path=cfg.LOCK_FILE,
            stale_timeout_seconds=cfg.LOCK_STALE_TIMEOUT_SECONDS,
            check_stale=cfg.FEATURES.get("stale_lock_detection", True) and cfg.CHECK_STALE_LOCKS
        )
        self.client = RcloneClient(
            api_url=cfg.RC_API_URL,
            user=cfg.RC_USER if cfg.RC_AUTH_ENABLED else None,
            password=cfg.RC_PASS if cfg.RC_AUTH_ENABLED else None
        )
        self.notifier = NotificationDispatcher(
            app_name=cfg.NOTIFICATION_APP_NAME,
            webhook_url=cfg.WEBHOOK_URL if cfg.FEATURES.get("webhook_alerts") else None,
            webhook_timeout=cfg.WEBHOOK_TIMEOUT,
            enabled_smart_notifications=cfg.FEATURES.get("smart_notifications", True),
            error_repeat_hours=cfg.ERROR_NOTIFICATION_REPEAT_HOURS
        )

    def validate_configuration(self) -> bool:
        """Validates filesystem paths and daemon connectivity."""
        if not self.config.VALIDATE_CONFIG_ON_STARTUP:
            return True

        if self.config.REQUIRE_LOCAL_PATH_EXISTS:
            if not os.path.exists(self.config.LOCAL_PATH) or not os.path.isdir(self.config.LOCAL_PATH):
                logging.error(f"Local path does not exist or is not a directory: {self.config.LOCAL_PATH}")
                return False

        os.makedirs(self.config.BASE_DIR, mode=0o700, exist_ok=True)
        healthy, latency, err = self.client.check_health(
            retries=self.config.DAEMON_HEALTH_CHECK_RETRIES,
            retry_delay=self.config.RETRY_DELAY
        )
        # Record health status
        health_data = {
            "last_check": time.time(),
            "status": "healthy" if healthy else "offline",
            "response_time_ms": latency if healthy else 0.0,
            "error": err
        }
        write_json_file(self.config.DAEMON_HEALTH_FILE, health_data)

        if not healthy:
            logging.error(f"Daemon health check failed at {self.config.RC_API_URL}: {err}")
            return False

        return True

    def cleanup_on_success(self):
        """Cleans configured file patterns post-sync."""
        for pattern in self.config.FILES_TO_CLEANUP:
            try:
                for file_path in glob.glob(os.path.join(self.config.LOCAL_PATH, pattern), recursive=True):
                    if os.path.isfile(file_path):
                        os.remove(file_path)
                        logging.info(f"Cleaned up: {file_path}")
            except Exception as e:
                logging.warning(f"Cleanup failed for {pattern}: {e}")
    def execute_transaction(
        self,
        resync: bool = False,
        force: bool = False,
        dry_run: bool = False
    ) -> Tuple[bool, Optional[str], Optional[str], int]:
        """Compiles arguments, dispatches to daemon, and monitors execution."""
        try:
            params = compile_bisync_payload(
                args_list=getattr(self.config, "RCLONE_COMMON_ARGS", []),
                local_path=self.config.LOCAL_PATH,
                remote_path=self.config.REMOTE_PATH,
                resync_attempt=resync,
                force_attempt=force,
                dry_run_attempt=dry_run
            )
        except FileNotFoundError as e:
            logging.error(f"Canary safety guard abort: {e}")
            return False, "CHECK_ACCESS_FAILED", str(e), 0

        try:
            logging.info(f"Triggering bisync job on daemon (resync={resync}, force={force}, dry_run={dry_run})...")
            job_id = self.client.trigger_bisync(params)
            logging.info(f"Bisync job started (Job ID: {job_id})")

            success, err_type, err_msg, files = self.client.monitor_job(
                job_id=job_id,
                timeout_seconds=self.config.JOB_TIMEOUT_SECONDS,
                initial_sleep=self.config.JOB_POLL_INITIAL_SLEEP,
                max_sleep=self.config.JOB_POLL_MAX_SLEEP,
                backoff_multiplier=self.config.JOB_POLL_BACKOFF_MULTIPLIER
            )
            if not success and not err_type and err_msg:
                err_type = classify_error(err_msg)
            return success, err_type, err_msg, files
        except Exception as e:
            err_str = str(e)
            return False, classify_error(err_str), err_str, 0

    def run(
        self,
        resync: bool = False,
        force: bool = False,
        dry_run: bool = False,
        origin: str = "cli",
        run_id: str = "unknown",
        no_notify: bool = False
    ) -> bool:
        """Main entry point orchestrating safety gates, execution, and telemetry."""
        start_time = time.time()
        notification_state = load_notification_state(self.config.NOTIFICATION_STATE_FILE)

        # 1. Rate Limit Gate
        if not force and self.config.FEATURES.get("rate_limit_backoff", True):
            if check_rate_limit_backoff(self.config.RATE_LIMIT_BACKOFF_FILE, self.config.RATE_LIMIT_BACKOFF_HOURS):
                logging.info("Exiting due to active rate limit backoff. Use --force to override.")
                return True

        # 2. Lock Gate
        if not self.lock_mgr.acquire():
            logging.info("Exiting: another sync instance holds the lock.")
            return False

        try:
            # 3. Validation Gate
            if not self.validate_configuration():
                return False

            # 4. Execute Transaction
            success, err_type, err_msg, files = self.execute_transaction(resync=resync, force=force, dry_run=dry_run)
            duration = time.time() - start_time

            if dry_run:
                logging.info(f"Dry-run simulation finished: {'SUCCESS' if success else 'FAILURE'} ({duration:.1f}s)")
                return success

            if success:
                self.cleanup_on_success()
                notification_state["last_success_timestamp"] = time.time()
                notification_state["last_error_type"] = None
                write_notification_state(self.config.NOTIFICATION_STATE_FILE, notification_state)

                if self.config.ENABLE_METRICS:
                    log_metrics_entry(
                        self.config.METRICS_FILE,
                        {
                            "timestamp": time.time(),
                            "datetime": time.strftime("%Y-%m-%dT%H:%M:%S"),
                            "run_id": run_id,
                            "origin": origin,
                            "success": True,
                            "duration_seconds": round(duration, 2),
                            "error_type": None,
                            "files_synced": files
                        },
                        max_entries=self.config.METRICS_MAX_ENTRIES,
                        max_backups=self.config.METRICS_MAX_BACKUPS
                    )

                if not no_notify:
                    self.notifier.send_desktop("Sync Success", f"Google Drive synchronized cleanly ({duration:.1f}s)")
                return True
            else:
                return self._handle_failure(err_type, err_msg, duration, files, resync, force, no_notify, notification_state, run_id, origin, start_time)
        finally:
            self.lock_mgr.release()

    def _handle_failure(
        self,
        err_type: Optional[str],
        err_msg: Optional[str],
        duration: float,
        files: int,
        resync: bool,
        force: bool,
        no_notify: bool,
        notification_state: Dict[str, Any],
        run_id: str,
        origin: str,
        start_time: float
    ) -> bool:
        """Handles failure path, auto-recovery resync, and notifications."""
        logging.error(f"Sync failed ({err_type}): {err_msg}")

        # Automated Recovery on NO_LISTINGS
        if err_type == "NO_LISTINGS":
            if resync:
                logging.error("Sync failed during an explicit resync execution. Manual intervention required.")
            else:
                logging.warning("⚠️ Cache tracking missing or corrupted. Triggering automated self-healing resync...")
                if not no_notify:
                    self.notifier.send_desktop(
                        "Sync Auto-Recovery",
                        "Cache tracking missing. Rebuilding synchronization baseline states...",
                        urgency="normal"
                    )

                retry_success, retry_err, retry_detail, retry_files = self.execute_transaction(resync=True, force=force)
                if retry_success:
                    retry_duration = time.time() - start_time
                    logging.info("✅ Auto-recovery successful! Baseline sync state rebuild complete.")
                    self.cleanup_on_success()
                    notification_state["last_error_type"] = None
                    notification_state["last_success_timestamp"] = time.time()
                    write_notification_state(self.config.NOTIFICATION_STATE_FILE, notification_state)

                    if self.config.ENABLE_METRICS:
                        log_metrics_entry(
                            self.config.METRICS_FILE,
                            {
                                "timestamp": time.time(),
                                "datetime": time.strftime("%Y-%m-%dT%H:%M:%S"),
                                "run_id": run_id,
                                "origin": origin,
                                "success": True,
                                "duration_seconds": round(retry_duration, 2),
                                "error_type": None,
                                "files_synced": retry_files
                            },
                            max_entries=self.config.METRICS_MAX_ENTRIES,
                            max_backups=self.config.METRICS_MAX_BACKUPS
                        )
                    if not no_notify:
                        self.notifier.send_desktop("Sync Auto-Recovery Complete", f"Baseline state rebuilt ({retry_duration:.1f}s)")
                    return True
                else:
                    err_type, err_msg = retry_err, retry_detail

        # Record failure metric
        if self.config.ENABLE_METRICS:
            log_metrics_entry(
                self.config.METRICS_FILE,
                {
                    "timestamp": time.time(),
                    "datetime": time.strftime("%Y-%m-%dT%H:%M:%S"),
                    "run_id": run_id,
                    "origin": origin,
                    "success": False,
                    "duration_seconds": round(duration, 2),
                    "error_type": err_type,
                    "files_synced": files
                },
                max_entries=self.config.METRICS_MAX_ENTRIES,
                max_backups=self.config.METRICS_MAX_BACKUPS
            )

        if err_type == "RATE_LIMIT":
            set_rate_limit_backoff(self.config.RATE_LIMIT_BACKOFF_FILE)

        # Notify if applicable
        if not no_notify and self.notifier.should_notify_error(err_type, notification_state):
            urgency = "critical" if err_type in ("CHECK_ACCESS_FAILED", "TOO_MANY_DELETES", "PERMISSION") else "normal"
            self.notifier.send_desktop(f"Sync Failure - {err_type}", f"{err_msg[:200]}", urgency=urgency)
            notification_state["last_error_type"] = err_type
            notification_state["last_error_timestamp"] = time.time()
            write_notification_state(self.config.NOTIFICATION_STATE_FILE, notification_state)

        return False

