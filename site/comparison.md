---
layout: default
title: Comparison Matrix
nav_order: 5
description: "How Bisync Hub compares against custom cron scripts, desktop sync GUIs, and raw Rclone CLI commands."
---

# Capability Comparison Matrix

Understanding where Bisync Hub fits in your storage infrastructure:

| Capability / Feature | Generic Cron Script | Sync Desktop GUI | Raw Rclone CLI | Bisync Hub |
| :--- | :---: | :---: | :---: | :---: |
| **Interactive Deletion Review** | ❌ None | ❌ Rare | ❌ None | ✅ **Full Diff Modal** |
| **Multi-Signal Conflict Heuristics** | ❌ Timestamps only | ⚠️ Basic rename | ⚠️ Timestamp or manual | ✅ **4-Analyzer Engine** |
| **Canary Mount Protection** | ⚠️ Custom script only | ❌ None | ❌ None | ✅ **Built-in `RCLONE_TEST`** |
| **Cryptographic Audit Ledger** | ❌ Plain text logs | ❌ Local state | ❌ Stdout logs | ✅ **Immutable SHA-256 Ledger** |
| **Zero External Runtime Deps** | ⚠️ Bash / Shell | ❌ Electron / Qt | ✅ Go binary | ✅ **Single Go Binary** |
| **Embedded Web Dashboard** | ❌ None | ⚠️ Desktop only | ❌ RC webgui only | ✅ **Lightweight Embedded** |
| **Webhook Alerts (Discord/Slack)** | ⚠️ Manual curl | ❌ OS toasts only | ❌ None | ✅ **Built-in Webhooks** |
| **Multi-Profile Fleet Hierarchy** | ❌ None | ⚠️ Basic profiles | ❌ None | ✅ **Fleet Command Deck** |
| **Remote Hash Deduplication** | ❌ None | ❌ None | ⚠️ Manual CLI | ✅ **Safe Dry-Run & Lock** |
| **Cloud Clock Skew Compensation** | ❌ None | ⚠️ Basic fuzzing | ⚠️ `--modify-window` | ✅ **Automatic Skew Matrix** |
| **POSIX Advisory Flock Locks** | ⚠️ Fragile PID files | ❌ OS locks | ❌ None | ✅ **Race-Free POSIX Flock** |
| **Path Traversal Jail Confinement** | ❌ None | ❌ None | ❌ None | ✅ **Enforced Local Jails** |
| **Native AI Agent Protocol (MCP)** | ❌ None | ❌ None | ❌ None | ✅ **Zero-Dep stdio/SSE** |

---

## Why Not Just Run a Cron Script?

Cron scripts are ubiquitous, but they fail silently when edge cases occur:

1. **The Unmounted Mount Disaster**: If an NFS/SMB share or rclone mount disconnects, an automated cron script sees an empty folder and issues deletions for thousands of files across the remote replica.
2. **The Cloud Timestamp Shift**: Google Drive and Microsoft OneDrive report timestamps differently than local ext4/APFS filesystems. A simple script relying on `test fileA -nt fileB` frequently overwrites valid changes.
3. **Overlapping Job Corruption**: If a sync run takes longer than the cron interval (e.g., due to large video uploads), concurrent sync instances access the cache simultaneously, corrupting `.bisync` lock state.

Bisync Hub eliminates every one of these failure modes out of the box.
