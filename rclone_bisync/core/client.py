"""
Rclone Remote Control (RC) HTTP Client
"""

import time
import logging
import requests
from typing import Tuple, Optional, Dict, Any, List


class RcloneClient:
    """Client for Rclone Remote Control (RC) HTTP API."""

    def __init__(self, api_url: str, user: Optional[str] = None, password: Optional[str] = None):
        self.api_url = api_url.rstrip("/")
        self.auth = (user, password) if (user and password) else None

    def check_health(self, retries: int = 2, retry_delay: float = 1.0) -> Tuple[bool, float, Optional[str]]:
        """Pings /core/stats. Returns: (is_healthy, response_time_ms, error_message)."""
        last_error = None
        for attempt in range(retries):
            try:
                response = requests.post(f"{self.api_url}/core/stats", timeout=5, auth=self.auth)
                if response.status_code == 200:
                    latency = round(response.elapsed.total_seconds() * 1000, 2)
                    return True, latency, None
                last_error = f"Daemon returned HTTP status {response.status_code}"
            except requests.exceptions.ConnectionError:
                last_error = f"Cannot connect to daemon at {self.api_url}. Is rclone-daemon.service running?"
            except Exception as e:
                last_error = str(e)
            if attempt < retries - 1:
                time.sleep(retry_delay)
        return False, 0.0, last_error

    def trigger_bisync(self, params: Dict[str, Any]) -> int:
        """Calls /sync/bisync?_async=true. Returns job ID integer."""
        url = f"{self.api_url}/sync/bisync?_async=true"
        response = requests.post(url, json=params, timeout=30, auth=self.auth)
        response.raise_for_status()
        data = response.json()
        job_id = data.get("jobid")
        if job_id is None:
            raise RuntimeError(f"Rclone RC did not return a job ID: {data}")
        return int(job_id)

    def get_job_status(self, job_id: int) -> Dict[str, Any]:
        """Queries /job/status for a specific job."""
        response = requests.post(f"{self.api_url}/job/status", json={"jobid": job_id}, timeout=10, auth=self.auth)
        response.raise_for_status()
        return response.json()

    def stop_job(self, job_id: int) -> bool:
        """Stops a running job via /job/stop."""
        try:
            res = requests.post(f"{self.api_url}/job/stop", json={"jobid": job_id}, timeout=10, auth=self.auth)
            return res.status_code == 200
        except Exception as e:
            logging.warning(f"Failed to stop job {job_id}: {e}")
            return False

    def list_jobs(self) -> List[int]:
        """Returns list of active job IDs via /job/list."""
        try:
            res = requests.post(f"{self.api_url}/job/list", timeout=5, auth=self.auth)
            if res.status_code == 200:
                return res.json().get("jobids", [])
        except Exception:
            pass
        return []

    def stop_all_active_jobs(self) -> int:
        """Stops all running jobs on the daemon. Returns count of stopped jobs."""
        stopped = 0
        for jid in self.list_jobs():
            if self.stop_job(jid):
                stopped += 1
        return stopped
    def monitor_job(
        self,
        job_id: int,
        timeout_seconds: float = 1800,
        initial_sleep: float = 2.0,
        max_sleep: float = 10.0,
        backoff_multiplier: float = 1.5,
        max_consecutive_poll_errors: int = 3
    ) -> Tuple[bool, Optional[str], Optional[str], int]:
        """
        Polls /job/status with exponential backoff and transient failure tolerance.
        Returns: (success, error_type, error_message, files_synced)
        """
        start_time = time.time()
        sleep_duration = initial_sleep
        consecutive_poll_errors = 0

        logging.info(f"Monitoring bisync job {job_id}...")

        while True:
            elapsed = time.time() - start_time
            if elapsed > timeout_seconds:
                logging.error(f"Job {job_id} exceeded timeout ({timeout_seconds}s)")
                self.stop_job(job_id)
                return False, "TIMEOUT", f"Job exceeded {timeout_seconds}s timeout", 0

            try:
                status = self.get_job_status(job_id)
                consecutive_poll_errors = 0

                if status.get("finished"):
                    success = bool(status.get("success"))
                    error_msg = status.get("error", "")

                    files_synced = 0
                    try:
                        output_data = status.get("output") or {}
                        if output_data.get("transfers") is not None:
                            files_synced = int(output_data["transfers"])
                        elif output_data.get("stats", {}).get("transfers") is not None:
                            files_synced = int(output_data["stats"]["transfers"])
                    except (TypeError, ValueError):
                        pass

                    if success:
                        duration = status.get("duration", elapsed)
                        logging.info(
                            f"Job {job_id} completed successfully in {duration:.1f}s "
                            f"({files_synced} file(s) transferred)"
                        )
                        return True, None, None, files_synced

                    return False, None, error_msg, files_synced

                logging.debug(f"Job {job_id} running... ({elapsed:.1f}s elapsed)")
                time.sleep(min(sleep_duration, max_sleep))
                sleep_duration *= backoff_multiplier

            except requests.exceptions.RequestException as e:
                consecutive_poll_errors += 1
                logging.warning(
                    f"Temporary error polling job {job_id} "
                    f"({consecutive_poll_errors}/{max_consecutive_poll_errors}): {e}"
                )
                if consecutive_poll_errors >= max_consecutive_poll_errors:
                    logging.error(f"Job {job_id} polling failed after {consecutive_poll_errors} consecutive errors.")
                    return False, "MONITOR_ERROR", str(e), 0
                time.sleep(min(sleep_duration, max_sleep))

            except Exception as e:
                logging.error(f"Unexpected error monitoring job {job_id}: {e}")
                return False, "MONITOR_ERROR", str(e), 0

    def list_remotes(self) -> List[str]:
        """Queries /config/listremotes to discover configured cloud providers."""
        try:
            res = requests.post(f"{self.api_url}/config/listremotes", timeout=5, auth=self.auth)
            if res.status_code == 200:
                return res.json().get("remotes", [])
        except Exception as e:
            logging.warning(f"Failed to query remotes: {e}")
        return []
    def get_remote_config(self, remote_name: str) -> Dict[str, Any]:
        """Queries /config/get and sanitizes credentials, returning only safe metadata (type, scope)."""
        clean_name = remote_name.rstrip(":")
        try:
            res = requests.post(f"{self.api_url}/config/get", json={"name": clean_name}, timeout=5, auth=self.auth)
            if res.status_code == 200:
                raw = res.json()
                safe_keys = {"type", "scope", "endpoint", "region", "env_auth", "team_drive"}
                return {k: v for k, v in raw.items() if k in safe_keys}
        except Exception as e:
            logging.warning(f"Failed to query config for {clean_name}: {e}")
        return {}


    def get_remote_about(self, remote: str) -> Dict[str, Any]:
        """Queries /operations/about to retrieve storage quota (total, used, free, trashed)."""
        fs = remote if remote.endswith(":") else f"{remote}:"
        try:
            res = requests.post(f"{self.api_url}/operations/about", json={"fs": fs}, timeout=5, auth=self.auth)
            if res.status_code == 200:
                return res.json()
        except Exception as e:
            logging.warning(f"Failed to query storage quota for {fs}: {e}")
        return {}

    def list_directory(self, remote: str, path: str = "") -> Dict[str, Any]:
        """
        Queries /operations/list to explore directory contents.
        Returns dictionary with items list and has_canary flag.
        """
        fs = remote if remote.endswith(":") else f"{remote}:"
        clean_path = path.strip("/")
        payload = {"fs": fs, "remote": clean_path, "opt": {"recurse": False}}
        try:
            res = requests.post(f"{self.api_url}/operations/list", json=payload, timeout=10, auth=self.auth)
            if res.status_code == 200:
                items = res.json().get("list", [])
                has_canary = any(item.get("Name") == "RCLONE_TEST" for item in items)
                return {"items": items, "has_canary": has_canary, "remote": fs, "path": clean_path}
        except Exception as e:
            logging.warning(f"Failed to list directory {fs}{clean_path}: {e}")
            return {"items": [], "has_canary": False, "error": str(e), "remote": fs, "path": clean_path}
        return {"items": [], "has_canary": False, "remote": fs, "path": clean_path}


