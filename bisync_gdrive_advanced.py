#!/usr/bin/env python3
"""
Rclone Bisync Advanced Script (Compatibility Entry Point)
Delegates execution to the modular rclone_bisync engine package.
"""

import os
import sys

# Ensure package directory is in sys.path when invoked directly
CURRENT_DIR = os.path.dirname(os.path.abspath(__file__))
if CURRENT_DIR not in sys.path:
    sys.path.insert(0, CURRENT_DIR)

from rclone_bisync.cli import main

if __name__ == "__main__":
    main()
