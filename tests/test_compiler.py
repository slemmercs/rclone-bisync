import os
import pytest
from unittest import mock
from rclone_bisync.core.compiler import compile_bisync_payload


def test_compile_basic_parameters():
    args = [
        "--resilient",
        "--check-first",
        "--track-renames",
        "--remove-empty-dirs",
        "--buffer-size", "32M",
        "--retries", "4",
        "--tpslimit", "10"
    ]
    payload = compile_bisync_payload(
        args_list=args,
        local_path="/tmp/test_local",
        remote_path="gdrive:",
        resync_attempt=False,
        force_attempt=True,
        dry_run_attempt=False
    )

    assert payload["path1"] == "gdrive:"
    assert payload["path2"] == "/tmp/test_local"
    assert payload["force"] is True
    assert payload["resync"] is False
    assert payload["checkFirst"] is True
    assert payload["trackRenames"] is True
    assert payload["removeEmptyDirs"] is True
    assert payload["_config"]["BufferSize"] == "32M"
    assert payload["_config"]["Retries"] == 4
    assert payload["_config"]["TPSLimit"] == 10.0


def test_compile_backend_connection_string():
    args = [
        "--drive-use-trash",
        "--drive-skip-gdocs",
        "--drive-export-formats", "docx,xlsx"
    ]
    payload = compile_bisync_payload(
        args_list=args,
        local_path="/tmp/test_local",
        remote_path="gdrive:"
    )

    assert payload["path1"].startswith("gdrive,")
    assert "use_trash=true" in payload["path1"]
    assert "skip_gdocs=true" in payload["path1"]
    assert 'export_formats="docx,xlsx"' in payload["path1"]
    assert payload["path1"].endswith(":")


def test_canary_guard_aborts_when_rclone_test_missing(tmp_path):
    local_dir = str(tmp_path / "gdrive")
    os.makedirs(local_dir, exist_ok=True)
    args = ["--check-access"]

    with pytest.raises(FileNotFoundError) as exc_info:
        compile_bisync_payload(args_list=args, local_path=local_dir, remote_path="gdrive:")

    assert "Safety guard triggered: --check-access is enabled" in str(exc_info.value)
    assert "RCLONE_TEST" in str(exc_info.value)


def test_canary_guard_passes_when_rclone_test_present(tmp_path):
    local_dir = str(tmp_path / "gdrive")
    os.makedirs(local_dir, exist_ok=True)
    canary = os.path.join(local_dir, "RCLONE_TEST")
    with open(canary, "w") as f:
        f.write("canary")

    args = ["--check-access"]
    payload = compile_bisync_payload(args_list=args, local_path=local_dir, remote_path="gdrive:")
    assert payload["checkAccess"] is True
