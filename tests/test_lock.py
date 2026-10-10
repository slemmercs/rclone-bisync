import os
import subprocess
import pytest
from rclone_bisync.core.lock import LockManager, check_pid_exists


def test_lock_acquire_and_release(tmp_path):
    lock_file = str(tmp_path / "test.lock")
    mgr = LockManager(lock_file)

    assert mgr.acquire() is True

    # PID is stamped in lockfile; read through existing handle so we don't trigger POSIX lock release on close
    mgr.handle.seek(0)
    content = mgr.handle.read().strip()
    assert content == str(os.getpid())

    # In a separate process, probing is_locked must return True
    probe_cmd = f"python3 -c 'from rclone_bisync.core.lock import LockManager; exit(0 if LockManager.is_locked(\"{lock_file}\") else 1)'"
    res = subprocess.run(probe_cmd, shell=True, env=dict(os.environ, PYTHONPATH="."))
    assert res.returncode == 0

    mgr.release()

    # After release, probe must return False (exit code 1)
    res_after = subprocess.run(probe_cmd, shell=True, env=dict(os.environ, PYTHONPATH="."))
    assert res_after.returncode == 1


def test_stale_lock_recovery_with_dead_pid(tmp_path):
    lock_file = str(tmp_path / "stale.lock")
    # Write a dead PID (e.g. 999999999)
    with open(lock_file, "w") as f:
        f.write("999999999")

    mgr = LockManager(lock_file, check_stale=True)
    # Dead PID should be cleaned up and new lock acquired
    assert mgr.acquire() is True

    probe_cmd = f"python3 -c 'from rclone_bisync.core.lock import LockManager; exit(0 if LockManager.is_locked(\"{lock_file}\") else 1)'"
    res = subprocess.run(probe_cmd, shell=True, env=dict(os.environ, PYTHONPATH="."))
    assert res.returncode == 0

    mgr.release()


def test_check_pid_exists():
    assert check_pid_exists(os.getpid()) is True
    assert check_pid_exists(999999999) is False
    assert check_pid_exists(-1) is False

