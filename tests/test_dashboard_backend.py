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
    dummy_deletions = [
        {"remote": "gdrive", "path": "old.txt", "size": 123}
    ]
    with mock.patch("dashboard.backend.main.sync_service.get_pending_deletions", return_value=dummy_deletions):
        res = client.get("/api/deletions/pending")
        assert res.status_code == 200
        assert len(res.json()) == 1

