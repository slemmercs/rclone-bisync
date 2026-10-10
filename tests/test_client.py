import pytest
from unittest import mock
import requests
from rclone_bisync.core.client import RcloneClient


def test_client_init():
    client = RcloneClient("http://localhost:5572", user="user", password="pwd")
    assert client.api_url == "http://localhost:5572"
    assert client.auth == ("user", "pwd")


def test_monitor_job_resilience_to_transient_errors():
    client = RcloneClient("http://localhost:5572")

    # Simulate: 1st poll succeeds (not finished), 2nd poll fails with ConnectionError, 3rd poll succeeds (finished)
    poll_count = 0

    def mock_get_status(job_id):
        nonlocal poll_count
        poll_count += 1
        if poll_count == 1:
            return {"finished": False}
        elif poll_count == 2:
            raise requests.exceptions.ConnectionError("Temporary network blip")
        else:
            return {"finished": True, "success": True, "output": {"transfers": 3}}

    with mock.patch.object(client, "get_job_status", side_effect=mock_get_status):
        with mock.patch("time.sleep", return_value=None):
            success, err_type, err_msg, transfers = client.monitor_job(
                job_id=42,
                timeout_seconds=30,
                initial_sleep=0.01,
                max_sleep=0.05,
                max_consecutive_poll_errors=3
            )

    assert success is True
    assert transfers == 3
    assert poll_count == 3


def test_monitor_job_aborts_after_max_consecutive_errors():
    client = RcloneClient("http://localhost:5572")

    def mock_always_fails(job_id):
        raise requests.exceptions.ConnectionError("Persistent connection failure")

    with mock.patch.object(client, "get_job_status", side_effect=mock_always_fails):
        with mock.patch("time.sleep", return_value=None):
            success, err_type, err_msg, transfers = client.monitor_job(
                job_id=42,
                timeout_seconds=30,
                initial_sleep=0.01,
                max_sleep=0.05,
                max_consecutive_poll_errors=3
            )

    assert success is False
    assert err_type == "MONITOR_ERROR"
    assert "Persistent connection failure" in err_msg


def test_list_remotes():
    client = RcloneClient("http://localhost:5572")
    with mock.patch("requests.post") as mock_post:
        mock_resp = mock.Mock()
        mock_resp.status_code = 200
        mock_resp.json.return_value = {"remotes": ["gdrive", "onedrive"]}
        mock_post.return_value = mock_resp

        remotes = client.list_remotes()
        assert remotes == ["gdrive", "onedrive"]


def test_get_remote_about():
    client = RcloneClient("http://localhost:5572")
    with mock.patch("requests.post") as mock_post:
        mock_resp = mock.Mock()
        mock_resp.status_code = 200
        mock_resp.json.return_value = {"total": 1000000000, "used": 500000000, "free": 500000000}
        mock_post.return_value = mock_resp

        about = client.get_remote_about("gdrive:")
        assert about["total"] == 1000000000
        assert about["used"] == 500000000


def test_list_directory_canary_detection():
    client = RcloneClient("http://localhost:5572")
    with mock.patch("requests.post") as mock_post:
        mock_resp = mock.Mock()
        mock_resp.status_code = 200
        mock_resp.json.return_value = {
            "list": [
                {"Name": "Documents", "IsDir": True},
                {"Name": "RCLONE_TEST", "IsDir": False, "Size": 0}
            ]
        }
        mock_post.return_value = mock_resp

        data = client.list_directory("gdrive:", path="root")
        assert len(data["items"]) == 2
        assert data["has_canary"] is True
        assert data["remote"] == "gdrive:"

