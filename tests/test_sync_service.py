import os
import json
import pytest
from unittest import mock
from rclone_bisync.services.sync_service import SyncService


class DummyConfig:
    BASE_DIR = "/fake/base"
    LOG_FILE = "/fake/base/bisync.log"
    LOCK_FILE = "/fake/base/test.lock"
    METRICS_FILE = "/fake/base/metrics.jsonl"
    DAEMON_HEALTH_FILE = "/fake/base/daemon_health.json"
    NOTIFICATION_STATE_FILE = "/fake/base/notification_state.json"
    PENDING_DELETIONS_FILE = "/fake/base/pending_deletions.json"
    RATE_LIMIT_BACKOFF_FILE = "/fake/base/rate_limit.flag"
    RC_API_URL = "http://localhost:5572"
    RC_AUTH_ENABLED = False
    RC_USER = "rclone"
    RC_PASS = "test_pass"
    REMOTE_PATH = "gdrive:"
    SECRET_TOKEN = "super_secret"
    NORMAL_PARAM = "normal_val"


def test_sync_service_init():
    cfg = DummyConfig()
    mock_client = mock.Mock()
    service = SyncService(config=cfg, rc_client=mock_client)
    assert service.config == cfg
    assert service.client == mock_client


def test_is_syncing():
    cfg = DummyConfig()
    service = SyncService(config=cfg)
    with mock.patch("rclone_bisync.core.lock.LockManager.is_locked", return_value=True):
        assert service.is_syncing() is True
    with mock.patch("rclone_bisync.core.lock.LockManager.is_locked", return_value=False):
        assert service.is_syncing() is False


def test_handle_action_start_sync():
    cfg = DummyConfig()
    service = SyncService(config=cfg)
    with mock.patch.object(service, "is_syncing", return_value=False), \
         mock.patch.object(service, "run_bisync_systemd") as mock_run:
        res = service.handle_action("start_sync")
        assert res["status"] == "success"
        mock_run.assert_called_once_with([], "bisync-gdrive-manual")

    with mock.patch.object(service, "is_syncing", return_value=True):
        res = service.handle_action("start_sync")
        assert res["status"] == "error"
        assert "already in progress" in res["msg"]


def test_handle_action_stop_sync():
    cfg = DummyConfig()
    mock_client = mock.Mock()
    mock_client.stop_all_active_jobs.return_value = 2
    service = SyncService(config=cfg, rc_client=mock_client)

    with mock.patch("subprocess.run"):
        res = service.handle_action("stop_sync")
        assert res["status"] == "success"
        assert "2 daemon jobs halted" in res["msg"]
        mock_client.stop_all_active_jobs.assert_called_once()


def test_handle_action_force_resync():
    cfg = DummyConfig()
    service = SyncService(config=cfg)
    with mock.patch.object(service, "is_syncing", return_value=False), \
         mock.patch.object(service, "run_bisync_systemd") as mock_run:
        res = service.handle_action("force_resync")
        assert res["status"] == "success"
        mock_run.assert_called_once_with(["--resync"], "bisync-gdrive-resync")


def test_handle_action_force_sync():
    cfg = DummyConfig()
    service = SyncService(config=cfg)
    with mock.patch.object(service, "is_syncing", return_value=False), \
         mock.patch.object(service, "run_bisync_systemd") as mock_run:
        res = service.handle_action("force_sync")
        assert res["status"] == "success"
        mock_run.assert_called_once_with(["--force"], "bisync-gdrive-force")


def test_handle_action_dry_run():
    cfg = DummyConfig()
    service = SyncService(config=cfg)
    with mock.patch.object(service, "is_syncing", return_value=False), \
         mock.patch.object(service, "run_bisync_systemd") as mock_run:
        res = service.handle_action("dry_run")
        assert res["status"] == "success"
        mock_run.assert_called_once_with(["--dry-run"], "bisync-gdrive-dryrun")


def test_handle_action_remove_lock(tmp_path):
    lock_file = str(tmp_path / "bisync.lock")
    with open(lock_file, "w") as f:
        f.write("1234")

    cfg = DummyConfig()
    cfg.LOCK_FILE = lock_file
    service = SyncService(config=cfg)

    # Refuse when syncing
    with mock.patch.object(service, "is_syncing", return_value=True):
        res = service.handle_action("remove_lock")
        assert res["status"] == "error"

    # Allow when not syncing
    with mock.patch.object(service, "is_syncing", return_value=False):
        res = service.handle_action("remove_lock")
        assert res["status"] == "success"
        assert not os.path.exists(lock_file)


def test_handle_action_unknown():
    cfg = DummyConfig()
    service = SyncService(config=cfg)
    res = service.handle_action("nonexistent_action")
    assert res["status"] == "error"
    assert "Unknown action" in res["msg"]


def test_is_sensitive_key():
    cfg = DummyConfig()
    service = SyncService(config=cfg)
    assert service.is_sensitive_key("RC_PASS") is True
    assert service.is_sensitive_key("MY_SECRET_TOKEN") is True
    assert service.is_sensitive_key("LOCAL_PATH") is False
    assert service.is_sensitive_key("NORMAL_PARAM") is False


def test_get_runtime_config_masks_secrets():
    cfg = DummyConfig()
    service = SyncService(config=cfg)
    runtime = service.get_runtime_config()
    assert runtime["NORMAL_PARAM"] == "normal_val"
    assert runtime["RC_PASS"] == "********"
    assert runtime["SECRET_TOKEN"] == "********"


def test_tail_file_and_tail_json_log(tmp_path):
    log_file = str(tmp_path / "daemon.log")
    with open(log_file, "w") as f:
        f.write('{"time":"2026-10-11T12:00:00Z","level":"info","msg":"line 1"}\n')
        # Nested JSON log inside wrapper
        f.write('{"time":"2026-10-11T12:00:01Z","level":"info","msg":"{\\"time\\":\\"2026-10-11T12:00:01Z\\",\\"msg\\":\\"nested line\\"}"}\n')

    cfg = DummyConfig()
    service = SyncService(config=cfg)

    lines = service.tail_file(log_file, lines_count=10)
    assert len(lines) == 2

    json_entries = service.tail_json_log(log_file, entries_count=10)
    assert len(json_entries) == 2
    assert "nested line" in json_entries[1]


def test_get_remotes_summary():
    cfg = DummyConfig()
    mock_client = mock.Mock()
    mock_client.list_remotes.return_value = ["gdrive", "onedrive"]
    mock_client.get_remote_config.side_effect = lambda r: {"type": "drive" if r == "gdrive" else "onedrive"}
    mock_client.get_remote_about.side_effect = lambda r: {"total": 1000, "used": 400, "free": 600}

    service = SyncService(config=cfg, rc_client=mock_client)
    res = service.get_remotes_summary()

    assert res["count"] == 2
    assert res["summary"]["total_capacity_bytes"] == 2000
    assert res["summary"]["total_used_bytes"] == 800
    assert res["summary"]["total_free_bytes"] == 1200
    assert res["remotes"][0]["is_active_target"] is True
    assert res["remotes"][1]["is_active_target"] is False



def test_browse_path_remote_delegation():
    cfg = DummyConfig()
    mock_client = mock.Mock()
    mock_client.list_directory.return_value = {
        "items": [{"Name": "test.txt", "IsDir": False}],
        "has_canary": False,
        "remote": "gdrive:",
        "path": "docs"
    }
    service = SyncService(config=cfg, rc_client=mock_client)
    res = service.browse_path(remote="gdrive", path="docs")
    mock_client.list_directory.assert_called_once_with(remote="gdrive", path="docs")
    assert res["remote"] == "gdrive:"
    assert len(res["items"]) == 1


def test_browse_path_local_success_and_canary(tmp_path):
    # Setup test directory structure
    base = tmp_path / "gdrive"
    base.mkdir()
    (base / "file1.txt").write_text("hello")
    (base / "RCLONE_TEST").write_text("canary")
    sub = base / "SubDir"
    sub.mkdir()
    (sub / "nested.txt").write_text("world")

    cfg = DummyConfig()
    cfg.LOCAL_PATH = str(base)
    service = SyncService(config=cfg, rc_client=mock.Mock())

    # Root browse
    res = service.browse_path(remote="local", path="")
    assert res["remote"] == "local"
    assert res["path"] == ""
    assert res["has_canary"] is True
    # Verify subfolder is sorted first
    names = [i["Name"] for i in res["items"]]
    assert names[0] == "SubDir"
    assert "file1.txt" in names
    assert "RCLONE_TEST" in names

    # SubDir browse
    sub_res = service.browse_path(remote="local", path="SubDir")
    assert sub_res["remote"] == "local"
    assert sub_res["path"] == "SubDir"
    assert sub_res["has_canary"] is False
    assert len(sub_res["items"]) == 1
    assert sub_res["items"][0]["Name"] == "nested.txt"
    assert sub_res["items"][0]["IsDir"] is False


def test_browse_path_local_security_jail(tmp_path):
    base = tmp_path / "gdrive"
    base.mkdir()
    cfg = DummyConfig()
    cfg.LOCAL_PATH = str(base)
    service = SyncService(config=cfg, rc_client=mock.Mock())

    # Attempt path traversal
    res = service.browse_path(remote="local", path="../../etc")
    assert "forbidden" in res.get("error", "").lower()
    assert res["items"] == []

    # Nonexistent folder inside base
    res_nonexistent = service.browse_path(remote="local", path="does_not_exist")
    assert "does not exist" in res_nonexistent.get("error", "").lower()
    assert res_nonexistent["items"] == []


def test_create_canary_local_and_remote(tmp_path):
    base = tmp_path / "gdrive"
    base.mkdir()
    cfg = DummyConfig()
    cfg.LOCAL_PATH = str(base)
    mock_client = mock.Mock()
    mock_client.touch_file.return_value = {"status": "success", "remote": "gdrive:", "path": "RCLONE_TEST"}
    service = SyncService(config=cfg, rc_client=mock_client)

    # 1. Local canary creation
    canary_path = base / "RCLONE_TEST"
    assert not canary_path.exists()
    res = service.create_canary(remote="local", path="")
    assert res["status"] == "success"
    assert canary_path.exists()

    # 2. Local jail violation
    res_jail = service.create_canary(remote="local", path="../../etc")
    assert res_jail["status"] == "error"
    assert "forbidden" in res_jail["error"].lower()

    # 3. Remote canary creation
    res_remote = service.create_canary(remote="gdrive", path="subfolder")
    mock_client.touch_file.assert_called_once_with(remote="gdrive", path="subfolder/RCLONE_TEST")
    assert res_remote["status"] == "success"



def test_copy_item_local_to_remote(tmp_path):
    base = tmp_path / "gdrive"
    base.mkdir()
    sample = base / "notes.txt"
    sample.write_text("my notes")

    cfg = DummyConfig()
    cfg.LOCAL_PATH = str(base)
    mock_client = mock.Mock()
    mock_client.copy_file.return_value = {"status": "success"}
    mock_client.copy_dir.return_value = {"status": "success"}
    service = SyncService(config=cfg, rc_client=mock_client)

    # 1. Copy file local to remote
    res = service.copy_item(src_remote="local", src_path="notes.txt", dst_remote="gdrive", dst_path="docs")
    assert res["status"] == "success"
    mock_client.copy_file.assert_called_once_with(
        src_fs=str(base),
        src_remote="notes.txt",
        dst_fs="gdrive:docs",
        dst_remote="notes.txt"
    )

    # 2. Local jail escape attempt
    res_jail = service.copy_item(src_remote="local", src_path="../../etc/passwd", dst_remote="gdrive", dst_path="pwn")
    assert res_jail["status"] == "error"
    assert "forbidden" in res_jail["error"].lower()

