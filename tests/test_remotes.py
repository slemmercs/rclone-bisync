import pytest
from unittest import mock
from rclone_bisync.core.client import RcloneClient
from dashboard.backend.main import get_remotes


def test_get_remote_config_sanitization():
    client = RcloneClient("http://localhost:5572")
    with mock.patch("requests.post") as mock_post:
        mock_resp = mock.Mock()
        mock_resp.status_code = 200
        mock_resp.json.return_value = {
            "type": "drive",
            "scope": "drive",
            "client_id": "345384492444.apps.googleusercontent.com",
            "client_secret": "SUPER_SECRET_LEAK_ME_NOT",
            "token": '{"access_token":"secret_token","refresh_token":"secret_refresh"}',
            "password": "secret_password"
        }
        mock_post.return_value = mock_resp

        sanitized = client.get_remote_config("gdrive")
        assert sanitized.get("type") == "drive"
        assert sanitized.get("scope") == "drive"
        # Must not contain private credentials
        assert "client_secret" not in sanitized
        assert "token" not in sanitized
        assert "password" not in sanitized


def test_get_remotes_function():
    with mock.patch("dashboard.backend.main.rc_client.list_remotes", return_value=["gdrive"]):
        with mock.patch("dashboard.backend.main.rc_client.get_remote_config", return_value={"type": "drive"}):
            with mock.patch("dashboard.backend.main.rc_client.get_remote_about", return_value={
                "total": 400000000000,
                "used": 40000000000,
                "free": 360000000000
            }):
                data = get_remotes()
                assert data["count"] == 1
                assert len(data["remotes"]) == 1
                remote = data["remotes"][0]
                assert remote["name"] == "gdrive"
                assert remote["type"] == "drive"
                assert remote["type_label"] == "Google Drive"
                assert data["summary"]["total_capacity_bytes"] == 400000000000
                assert data["summary"]["total_used_bytes"] == 40000000000

