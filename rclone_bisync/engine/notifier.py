"""
Desktop & Webhook Notification Dispatcher
"""

import os
import time
import shutil
import logging
import subprocess
import requests
from typing import Optional, Dict, Any

try:
    from plyer import notification as plyer_notification
    PLYER_AVAILABLE = True
except ImportError:
    PLYER_AVAILABLE = False


class NotificationDispatcher:
    """Dispatches notifications to Linux desktop and optional webhooks."""

    def __init__(
        self,
        app_name: str = "Rclone Bisync",
        webhook_url: Optional[str] = None,
        webhook_timeout: int = 5,
        enabled_smart_notifications: bool = True,
        error_repeat_hours: float = 6.0
    ):
        self.app_name = app_name
        self.webhook_url = webhook_url
        self.webhook_timeout = webhook_timeout
        self.enabled_smart_notifications = enabled_smart_notifications
        self.error_repeat_hours = error_repeat_hours

    def send_desktop(self, title: str, message: str, urgency: str = "normal"):
        """Sends native notify-send or plyer notification."""
        notify_send_bin = shutil.which("notify-send")
        if notify_send_bin:
            try:
                u = "critical" if urgency == "critical" else "normal"
                subprocess.run(
                    [notify_send_bin, "-a", self.app_name, "-u", u, "-i", "sync-synchronizing", title, message],
                    check=False,
                    timeout=3
                )
                logging.debug(f"Native notification sent via notify-send: {title}")
                return
            except Exception as e:
                logging.debug(f"notify-send call failed: {e}")

        if PLYER_AVAILABLE:
            try:
                plyer_notification.notify(
                    title=title,
                    message=message,
                    app_name=self.app_name,
                    timeout=10,
                    hints={"urgency": 2 if urgency == "critical" else 1}
                )
                logging.debug(f"Notification sent via plyer: {title}")
            except Exception as e:
                logging.error(f"Failed to send notification via plyer: {e}")
        else:
            logging.debug(f"Desktop notification skipped (no provider): {title}")

    def should_notify_error(self, err_type: str, notification_state: Dict[str, Any]) -> bool:
        """Determines if an error should trigger an alert based on state changes and interval."""
        if not self.enabled_smart_notifications:
            return True

        last_error = notification_state.get("last_error_type")
        last_error_time = notification_state.get("last_error_timestamp", 0)

        # Notify on new error state
        if err_type != last_error:
            return True

        # Re-notify persistent error after interval
        hours_since = (time.time() - last_error_time) / 3600
        return hours_since >= self.error_repeat_hours

    def send_webhook(self, event_type: str, data: Dict[str, Any]):
        """Dispatches event to configured webhook endpoint."""
        if not self.webhook_url:
            return
        payload = {
            "event": event_type,
            "timestamp": time.time(),
            "hostname": os.uname().nodename,
            "data": data
        }
        try:
            response = requests.post(self.webhook_url, json=payload, timeout=self.webhook_timeout)
            response.raise_for_status()
            logging.debug(f"Webhook dispatched: {event_type}")
        except Exception as e:
            logging.warning(f"Failed to send webhook: {e}")

    def send_watchdog_ping(self, ping_url: Optional[str] = None, timeout: int = 5) -> bool:
        """Dispatches external watchdog / dead-man's snitch heartbeat ping on successful sync."""
        target_url = ping_url or self.webhook_url
        if not target_url:
            return False
        try:
            res = requests.get(target_url, timeout=timeout)
            logging.debug(f"Watchdog heartbeat ping sent to {target_url} (HTTP {res.status_code})")
            return res.status_code < 400
        except Exception as e:
            logging.warning(f"Failed to dispatch watchdog heartbeat ping to {target_url}: {e}")
            return False

