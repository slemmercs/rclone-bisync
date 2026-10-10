"""
Rclone Bisync Package Configuration Bridge
Loads and exposes configuration from user-facing bisync_config.py
"""

import sys
import os

# Ensure project root is in sys.path to load bisync_config
ROOT_DIR = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
if ROOT_DIR not in sys.path:
    sys.path.insert(0, ROOT_DIR)

try:
    import bisync_config as _user_config
    # Expose all configuration attributes into package namespace
    for _k, _v in _user_config.__dict__.items():
        if not _k.startswith("__"):
            globals()[_k] = _v
except ImportError:
    pass
