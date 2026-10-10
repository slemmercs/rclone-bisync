"""
Race-Free Process Locking & Stale Lock Detection
"""

import os
import time
import fcntl
import logging

try:
    import psutil
    PSUTIL_AVAILABLE = True
except ImportError:
    PSUTIL_AVAILABLE = False


def check_pid_exists(pid: int) -> bool:
    """Check if a process with given PID exists."""
    if pid <= 0:
        return False
    if PSUTIL_AVAILABLE:
        try:
            return psutil.pid_exists(pid)
        except Exception:
            pass
    try:
        os.kill(pid, 0)
        return True
    except OSError:
        return False


class LockManager:
    """Manages an exclusive process lockfile using fcntl without race conditions."""

    def __init__(self, lock_file_path: str, stale_timeout_seconds: int = 3600, check_stale: bool = True):
        self.lock_file_path = lock_file_path
        self.stale_timeout_seconds = stale_timeout_seconds
        self.check_stale = check_stale
        self.handle = None

    def acquire(self) -> bool:
        """
        Acquire the lock.
        Opens with O_RDWR | O_CREAT without truncation.
        Truncates and stamps PID ONLY after obtaining LOCK_EX.
        """
        os.makedirs(os.path.dirname(self.lock_file_path), mode=0o700, exist_ok=True)
        fd = os.open(self.lock_file_path, os.O_RDWR | os.O_CREAT, 0o600)
        self.handle = os.fdopen(fd, "r+")

        try:
            fcntl.lockf(self.handle, fcntl.LOCK_EX | fcntl.LOCK_NB)
            # Acquired lock successfully: safe to truncate and write PID
            self.handle.seek(0)
            self.handle.truncate()
            self.handle.write(str(os.getpid()))
            self.handle.flush()
            logging.debug(f"Lock acquired (PID: {os.getpid()})")
            return True
        except (IOError, BlockingIOError):
            if not self.check_stale:
                logging.warning("Another instance is running (lock held).")
                self._close_handle()
                return False

            return self._handle_stale_lock()

    def _handle_stale_lock(self) -> bool:
        """Inspects existing lock for stale PID or timeout."""
        if os.path.exists(self.lock_file_path):
            try:
                with open(self.lock_file_path, "r") as f:
                    content = f.read().strip()
                if content:
                    old_pid = int(content)
                    if not check_pid_exists(old_pid):
                        logging.warning(f"Removing stale lock (PID {old_pid} not running)")
                        self._close_handle()
                        try:
                            os.remove(self.lock_file_path)
                        except OSError:
                            pass
                        return self.acquire()

                    lock_age = time.time() - os.path.getmtime(self.lock_file_path)
                    if lock_age > self.stale_timeout_seconds:
                        logging.warning(
                            f"Lock is {lock_age / 3600:.1f} hours old (PID {old_pid}). "
                            "Process may be hung. Removing stale lock."
                        )
                        self._close_handle()
                        try:
                            os.remove(self.lock_file_path)
                        except OSError:
                            pass
                        return self.acquire()
                    else:
                        logging.warning(f"Another instance is running (PID {old_pid})")
                        self._close_handle()
                        return False
            except (ValueError, IOError) as e:
                logging.warning(f"Could not read lock file: {e}")

        self._close_handle()
        logging.warning("Another instance is running.")
        return False

    def release(self):
        """Release lock and close file descriptor."""
        if self.handle:
            try:
                fcntl.lockf(self.handle, fcntl.LOCK_UN)
            except Exception as e:
                logging.warning(f"Error releasing lockf: {e}")
            self._close_handle()
            logging.debug("Lock released")

    def _close_handle(self):
        if self.handle:
            try:
                self.handle.close()
            except Exception:
                pass
            self.handle = None

    @staticmethod
    def is_locked(lock_file_path: str) -> bool:
        """
        Non-blocking probe to verify if lock is actively held.
        Returns True if locked by a running process, False otherwise.
        """
        if not os.path.exists(lock_file_path):
            return False
        try:
            with open(lock_file_path, "r") as f:
                try:
                    fcntl.lockf(f, fcntl.LOCK_SH | fcntl.LOCK_NB)
                    fcntl.lockf(f, fcntl.LOCK_UN)
                    return False
                except (BlockingIOError, IOError, OSError) as e:
                    if getattr(e, "errno", None) in (11, 13):
                        return True
        except Exception:
            return False
        return False
