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
