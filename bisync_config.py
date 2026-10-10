#!/usr/bin/env python3
"""
DEPRECATED: bisync_config.py has been sundowned.
Configuration is now loaded dynamically from 'bisync.env' and 'rc-secrets.env'.
This shim provides backward compatibility for legacy imports.
"""

import warnings

warnings.warn(
    "bisync_config.py is deprecated and will be removed in a future release. "
    "Please manage configuration using bisync.env and rc-secrets.env.",
    DeprecationWarning,
    stacklevel=2
)

from rclone_bisync.config import *
