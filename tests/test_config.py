import os
import shutil
import subprocess
import sys
import pytest
from rclone_bisync.config import _str2bool, load_env_file


def test_str2bool():
    assert _str2bool("true") is True
    assert _str2bool("TRUE") is True
    assert _str2bool("1") is True
    assert _str2bool("yes") is True
    assert _str2bool("on") is True
    assert _str2bool(True) is True

    assert _str2bool("false") is False
    assert _str2bool("0") is False
    assert _str2bool("no") is False
    assert _str2bool("off") is False
    assert _str2bool("") is False
    assert _str2bool(False) is False
    assert _str2bool(None, default=True) is True


def test_load_env_file(tmp_path):
    env_file = str(tmp_path / "test.env")
    with open(env_file, "w") as f:
        f.write("""
# Comment line
TEST_PARAM_A=alpha
TEST_PARAM_B="beta quoted"
TEST_PARAM_C='single quoted'
""")

    load_env_file(env_file)
    assert os.environ.get("TEST_PARAM_A") == "alpha"
    assert os.environ.get("TEST_PARAM_B") == "beta quoted"
    assert os.environ.get("TEST_PARAM_C") == "single quoted"


def test_pure_env_config_exports():
    import rclone_bisync.config as config
    assert hasattr(config, "LOCAL_PATH")
    assert hasattr(config, "REMOTE_PATH")
    assert hasattr(config, "RCLONE_COMMON_ARGS")
    assert hasattr(config, "FEATURES")
    assert isinstance(config.RCLONE_COMMON_ARGS, list)
    assert isinstance(config.FEATURES, dict)
    assert "stale_lock_detection" in config.FEATURES


def test_missing_rc_pass_raises_error(tmp_path):
    # Copy package to an isolated root with no rc-secrets.env present
    repo_root = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
    pkg_src = os.path.join(repo_root, "rclone_bisync")
    pkg_dst = os.path.join(str(tmp_path), "rclone_bisync")
    shutil.copytree(pkg_src, pkg_dst)

    code = """
import os, sys
os.environ["HOME"] = sys.argv[1]
try:
    import rclone_bisync.config
    sys.exit(0)
except ValueError as e:
    if "RC_PASS is not set" in str(e):
        sys.exit(42)
    sys.exit(1)
"""
    res = subprocess.run(
        [sys.executable, "-c", code, str(tmp_path)],
        cwd=str(tmp_path),
        capture_output=True,
        text=True,
        env={"HOME": str(tmp_path), "PATH": os.environ["PATH"]}
    )
    assert res.returncode == 42


