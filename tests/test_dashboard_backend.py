import pytest
from fastapi.testclient import TestClient
from unittest import mock

# Ensure test secrets env is loaded
with mock.patch.dict("os.environ", {"RC_PASS": "test_password"}):
    from dashboard.backend.main import app

client = TestClient(app)


def test_root_index_html():
    res = client.get("/")
    assert res.status_code == 200
    assert "text/html" in res.headers["content-type"]


def test_csrf_middleware_blocks_external_origin():
    # Attempt POST with an untrusted external origin
    res = client.post(
        "/api/action",
        json={"action": "start_sync"},
        headers={
            "Origin": "https://malicious-site.com",
            "X-Dashboard-Request": "1"
        }
    )
    assert res.status_code == 403
    assert res.json()["status"] == "error"
    assert "Untrusted request origin" in res.json()["msg"]


def test_csrf_middleware_blocks_missing_custom_header():
    # Attempt POST without X-Dashboard-Request
    res = client.post(
        "/api/action",
        json={"action": "start_sync"}
    )
    assert res.status_code == 403
    assert res.json()["status"] == "error"
    assert "Missing required X-Dashboard-Request" in res.json()["msg"]


def test_csrf_middleware_permits_valid_loopback_and_header():
    with mock.patch("dashboard.backend.main.sync_service.handle_action", return_value={"status": "success", "msg": "Started"}):
        res = client.post(
            "/api/action",
            json={"action": "start_sync"},
            headers={
                "Origin": "http://127.0.0.1:8000",
                "X-Dashboard-Request": "1"
            }
        )
        assert res.status_code == 200
        assert res.json()["status"] == "success"


def test_get_telemetry_endpoint():
    dummy_payload = {
        "daemon_health": {"status": "ok", "response_time_ms": 12.5, "last_check": 1000.0},
        "notification_state": {"last_success_timestamp": 1000.0, "last_error_type": None, "last_error_timestamp": None},
        "metrics": [],
        "bisync_logs": ["line 1", "line 2"],
        "daemon_logs": ["daemon line"],
        "config": {"LOCAL_PATH": "/home/user/gdrive"},
        "is_syncing": False,
        "transfer_stats": None
    }
    with mock.patch("dashboard.backend.main.sync_service.get_telemetry_payload", return_value=dummy_payload):
        res = client.get("/api/telemetry?limit=20")
        assert res.status_code == 200
        data = res.json()
        assert data["daemon_health"]["status"] == "ok"
        assert data["is_syncing"] is False
        assert data["config"]["LOCAL_PATH"] == "/home/user/gdrive"
        assert len(data["bisync_logs"]) == 2


def test_browse_endpoint():
    dummy_browse = {
        "remote": "gdrive:",
        "path": "photos",
        "has_canary": True,
        "items": [
            {"Name": "pic1.jpg", "IsDir": False, "Size": 1024, "ModTime": "2026-10-17T10:00:00Z"}
        ]
    }
    with mock.patch("dashboard.backend.main.sync_service.browse_path", return_value=dummy_browse):
        res = client.post(
            "/api/browse",
            json={"remote": "gdrive", "path": "photos"},
            headers={"X-Dashboard-Request": "1"}
        )
        assert res.status_code == 200
        assert res.json()["has_canary"] is True
        assert len(res.json()["items"]) == 1


def test_get_remotes_endpoint():
    dummy_remotes = {
        "remotes": [
            {"name": "gdrive", "type": "drive", "is_active_target": True, "total_bytes": 1000, "used_bytes": 500}
        ],
        "count": 1,
        "active_remote": "gdrive",
        "summary": {"total_capacity_bytes": 1000, "total_used_bytes": 500, "total_free_bytes": 500}
    }
    with mock.patch("dashboard.backend.main.sync_service.get_remotes_summary", return_value=dummy_remotes):
        res = client.get("/api/remotes")
        assert res.status_code == 200
        assert res.json()["count"] == 1
        assert res.json()["remotes"][0]["name"] == "gdrive"


def test_get_pending_deletions_endpoint():
    dummy_deletions = {
        "has_deletions": True,
        "total_count": 1,
        "source": "Google Drive (remote)",
        "target_action": "Will delete locally",
        "message": "Too many deletes",
        "files": ["old.txt"]
    }
    with mock.patch("dashboard.backend.main.sync_service.get_pending_deletions", return_value=dummy_deletions):
        res = client.get("/api/deletions/pending")
        assert res.status_code == 200
        data = res.json()
        assert data["has_deletions"] is True
        assert data["total_count"] == 1
        assert len(data["files"]) == 1


def test_browse_canary_endpoint():
    dummy_res = {"status": "success", "remote": "gdrive:", "path": "photos"}
    with mock.patch("dashboard.backend.main.sync_service.create_canary", return_value=dummy_res):
        res = client.post(
            "/api/browse/canary",
            json={"remote": "gdrive", "path": "photos"},
            headers={"X-Dashboard-Request": "1"}
        )
        assert res.status_code == 200
        data = res.json()
        assert data["status"] == "success"
        assert data["remote"] == "gdrive:"
        assert data["path"] == "photos"


def test_browse_copy_endpoint():
    dummy_res = {"status": "success", "msg": "Copied photo.jpg"}
    with mock.patch("dashboard.backend.main.sync_service.copy_item", return_value=dummy_res):
        res = client.post(
            "/api/browse/copy",
            json={
                "src_remote": "local",
                "src_path": "photo.jpg",
                "dst_remote": "gdrive",
                "dst_path": "photos/photo.jpg",
                "is_dir": False
            },
            headers={"X-Dashboard-Request": "1"}
        )
        assert res.status_code == 200
        data = res.json()
        assert data["status"] == "success"
        assert "photo.jpg" in data["msg"]


def test_browse_delete_endpoint():
    dummy_res = {"status": "success", "msg": "Deleted old.txt", "remote": "local", "path": "old.txt"}
    with mock.patch("dashboard.backend.main.sync_service.delete_item", return_value=dummy_res):
        res = client.post(
            "/api/browse/delete",
            json={"remote": "local", "path": "old.txt", "is_dir": False},
            headers={"X-Dashboard-Request": "1"}
        )
        assert res.status_code == 200
        data = res.json()
        assert data["status"] == "success"
        assert data["path"] == "old.txt"


def test_recursive_redaction_on_api_endpoints():
    # Verify that if an endpoint returns a payload containing sensitive keys,
    # the response is automatically redacted before transmission.
    leaky_remotes = {
        "remotes": [
            {
                "name": "gdrive",
                "type": "drive",
                "client_id": "public_id",
                "client_secret": "SUPER_SECRET_LEAK",
                "about": {
                    "access_token": "ya29.leaked_token",
                    "total": 500,
                    "used": 100,
                    "free": 400
                }
            }
        ],
        "count": 1,
        "active_remote": "gdrive",
        "summary": {"total_capacity_bytes": 500, "total_used_bytes": 100, "total_free_bytes": 400}
    }
    with mock.patch("dashboard.backend.main.sync_service.get_remotes_summary", return_value=leaky_remotes):
        res = client.get("/api/remotes")
        assert res.status_code == 200
        data = res.json()
        assert data["remotes"][0]["about"]["access_token"] == "********"


