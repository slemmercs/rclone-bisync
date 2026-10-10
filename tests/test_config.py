import os
import pytest
import warnings
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


def test_deprecated_bisync_config_shim():
    with pytest.deprecated_call():
        import bisync_config
        # Ensure shim exports core attributes
        assert hasattr(bisync_config, "LOCAL_PATH")
        assert hasattr(bisync_config, "REMOTE_PATH")
        assert hasattr(bisync_config, "RCLONE_COMMON_ARGS")
        assert hasattr(bisync_config, "FEATURES")
