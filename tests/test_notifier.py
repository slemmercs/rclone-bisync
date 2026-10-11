import pytest
from unittest import mock
import requests
from rclone_bisync.engine.notifier import NotificationDispatcher


def test_watchdog_ping_success():
    dispatcher = NotificationDispatcher()
    with mock.patch("requests.get") as mock_get:
        mock_response = mock.Mock()
        mock_response.status_code = 200
        mock_get.return_value = mock_response

        res = dispatcher.send_watchdog_ping("https://hc-ping.com/fake-uuid")
        assert res is True
        mock_get.assert_called_once_with("https://hc-ping.com/fake-uuid", timeout=5)


def test_watchdog_ping_failure():
    dispatcher = NotificationDispatcher()
    with mock.patch("requests.get", side_effect=requests.exceptions.Timeout("Connection timed out")):
        res = dispatcher.send_watchdog_ping("https://hc-ping.com/fake-uuid")
        assert res is False


def test_watchdog_ping_none_skipped():
    dispatcher = NotificationDispatcher()
    res = dispatcher.send_watchdog_ping(None)
    assert res is False


def test_send_desktop_invokes_notify_send_with_icon():
    dispatcher = NotificationDispatcher(icon_path="/custom/path/cloud-sync-light.svg")
    with mock.patch("shutil.which", return_value="/usr/bin/notify-send"), \
         mock.patch("subprocess.run") as mock_subproc:
        dispatcher.send_desktop("Sync Complete", "All files synced", urgency="normal")

        mock_subproc.assert_called_once()
        cmd = mock_subproc.call_args[0][0]
        assert cmd[0] == "/usr/bin/notify-send"
        assert "-a" in cmd and cmd[cmd.index("-a") + 1] == "Rclone Bisync"
        assert "-i" in cmd and cmd[cmd.index("-i") + 1] == "/custom/path/cloud-sync-light.svg"
        assert "Sync Complete" in cmd
        assert "All files synced" in cmd


def test_resolve_icon_fallback_when_missing():
    dispatcher = NotificationDispatcher()
    with mock.patch("os.path.exists", return_value=False):
        assert dispatcher._resolve_icon() == "sync-synchronizing"

