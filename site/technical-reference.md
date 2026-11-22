---
layout: default
title: Technical Reference
nav_order: 8
description: "Auto-generated technical reference for settings, API models, and MCP tools."
has_toc: true
---

# Technical & Settings Reference
{: .no_toc }

*Auto-generated directly from source code AST (`internal/models/`, `internal/profiles/`, `internal/mcp/`).*
{: .fs-3 .text-grey-dk-000 }

<div class="hero-badges">
  <span class="badge-online">● Source AST Synchronized</span>
  <span class="badge-indigo">Go 1.24+</span>
  <span class="badge-indigo">Zero Drift CI Guaranteed</span>
</div>

---

## Table of Contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Global System Configuration

Global settings define defaults and engine behavior across all synchronization workloads.

### `GlobalDefaults`

> GlobalDefaults represents baseline operational settings applied across all profiles.

| Setting Key | Type | Description |
| :--- | :--- | :--- |
| `max_delete` | `int` | Safety deletion threshold percentage (0–100%). Aborts sync if pending deletions exceed this ratio. |
| `canary_validation` | `bool` | Verifies presence of RCLONE_TEST canary token files before proceeding. |
| `conflict_resolve` | `string` | Heuristic resolution mode: path1, path2, newer, older, larger, smaller. |
| `compare_mode` | `string` | Divergence detection: size,modtime, size,modtime,checksum, checksum, modtime. |
| `remove_empty_dirs` | `bool` | Cleans empty directories on target after file deletions. |
| `create_empty_dirs` | `bool` | Replicates empty directory trees between replicas. |
| `lock_explorer_remote` | `bool` | Restricts browser file explorer to profile root directory paths. |
| `custom_flags` | `string` | Raw CLI flags appended to Rclone bisync subprocess executions. |
| `default_local_path` | `string` | Suggested default base directory when creating local replica endpoints. |
| `default_remote_path` | `string` | Suggested default remote bucket or folder for new profiles. |

---

### `ConfigDaemonSettings`

> ConfigDaemonSettings represents daemon scheduling, timeout, and backoff settings.

| Setting Key | Type | Description |
| :--- | :--- | :--- |
| `job_poll_initial_sleep_sec` | `float64` | - |
| `job_poll_max_sleep_sec` | `float64` | - |
| `job_poll_backoff_multiplier` | `float64` | - |
| `job_timeout_seconds` | `int` | - |
| `lock_stale_timeout_seconds` | `int` | - |
| `rate_limit_backoff_hours` | `float64` | - |
| `rate_limit_retry_after_backoff` | `bool` | - |
| `check_stale_locks` | `bool` | - |

---

### `ConfigLoggingSettings`

> ConfigLoggingSettings represents log file size and backup retention.

| Setting Key | Type | Description |
| :--- | :--- | :--- |
| `max_log_size_mb` | `int` | - |
| `log_backup_count` | `int` | - |

---

### `ConfigCleanupSettings`

> ConfigCleanupSettings represents post-sync file cleanup globs.

| Setting Key | Type | Description |
| :--- | :--- | :--- |
| `files_to_cleanup` | `string` | - |

---

### `ConfigTelemetrySettings`

> ConfigTelemetrySettings represents telemetry and metrics collection settings.

| Setting Key | Type | Description |
| :--- | :--- | :--- |
| `enable_metrics` | `bool` | - |
| `metrics_max_entries` | `int` | - |
| `metrics_max_backups` | `int` | - |

---

### `ConfigFeatureSettings`

> ConfigFeatureSettings represents subsystem feature flags and debug toggles.

| Setting Key | Type | Description |
| :--- | :--- | :--- |
| `debug_mode` | `bool` | - |
| `mcp_read_only` | `bool` | - |
| `stale_lock_detection` | `bool` | - |
| `rate_limit_backoff` | `bool` | - |
| `daemon_health_check` | `bool` | - |
| `metrics_collection` | `bool` | - |
| `smart_notifications` | `bool` | - |
| `webhook_alerts` | `bool` | - |

---

### `ConfigNotificationSettings`

> ConfigNotificationSettings represents external alerting webhook endpoints and throttling.

| Setting Key | Type | Description |
| :--- | :--- | :--- |
| `app_name` | `string` | - |
| `success_interval_min` | `float64` | - |
| `error_repeat_hours` | `float64` | - |
| `webhook_url` | `string` | - |
| `webhook_timeout_sec` | `int` | - |
| `watchdog_ping_url` | `string` | - |
| `watchdog_timeout_sec` | `int` | - |

---


## Profile Workload Policies

Profile models govern path mappings, schedules, flag overrides, and failure thresholds.

### `SafetySettings`

> SafetySettings defines thresholds that trigger safeguards.

| Property Key | Type | Description |
| :--- | :--- | :--- |
| `delete_threshold_pct` | `float64` | e.g. 10.0 for 10% |
| `failure_threshold` | `int` | e.g. 3 consecutive failures |
| `canary_validation` | `bool` | - |

---

### `ProfileFlags`

> ProfileFlags defines profile-level operational flag overrides.
Pointer fields represent explicit overrides; nil indicates inheritance from GlobalDefaults.

| Property Key | Type | Description |
| :--- | :--- | :--- |
| `conflict_resolve` | `*string` | - |
| `max_delete` | `*MaxDeleteValue` | - |
| `canary_validation` | `*bool` | - |
| `compare_mode` | `*string` | - |
| `remove_empty_dirs` | `*bool` | - |
| `create_empty_dirs` | `*bool` | - |
| `filters_file_path` | `string` | - |
| `custom_flags` | `string` | - |
| `provider_options` | `ProviderOptionsMap` | - |

---

### `ResolvedFlags`

> ResolvedFlags represents effective execution settings after evaluating overrides and globals.

| Property Key | Type | Description |
| :--- | :--- | :--- |
| `dry_run` | `bool` | - |
| `resync` | `bool` | - |
| `force` | `bool` | - |
| `conflict_resolve` | `string` | - |
| `max_delete` | `MaxDeleteValue` | - |
| `canary_validation` | `bool` | - |
| `compare_mode` | `string` | - |
| `remove_empty_dirs` | `bool` | - |
| `create_empty_dirs` | `bool` | - |
| `exclude_patterns` | `[]string` | - |
| `filters_file_path` | `string` | - |
| `custom_flags` | `string` | - |
| `provider_options` | `map[string]string` | - |

---

### `Profile`

> Profile represents a managed synchronization workload.

| Property Key | Type | Description |
| :--- | :--- | :--- |
| `id` | `string` | - |
| `name` | `string` | - |
| `description` | `string` | - |
| `enabled` | `bool` | - |
| `archived` | `bool` | - |
| `remote` | `string` | - |
| `remote_path` | `string` | - |
| `local_path` | `string` | - |
| `schedule` | `string` | e.g. "15m", "1h", "daily", "manual" |
| `review_mode` | `ReviewMode` | - |
| `notification_pref` | `NotificationPreference` | - |
| `safety` | `SafetySettings` | - |
| `flags` | `ProfileFlags` | - |
| `exclude_patterns` | `[]string` | - |
| `tags` | `[]string` | - |
| `version` | `int` | - |
| `created_at` | `time.Time` | - |
| `updated_at` | `time.Time` | - |
| `state` | `OperationalState` | - |

---


## Model Context Protocol (MCP) AI Tools

Bisync Hub exposes native operational tools for AI agents (Claude, Cursor, Cline):

| Tool Identifier | Blast-Radius Guard | Description |
| :--- | :---: | :--- |
| `bisync_copy_item` | Jail-Confined | Copy a file or directory between remote and local storage targets |
| `bisync_create_crypt_remote` | Read-Only | Wrap an existing remote with client-side Crypt encryption overlay |
| `bisync_create_profile` | Read-Only | Create a new bidirectional synchronization profile with path validation |
| `bisync_create_remote` | Read-Only | Provision a new Rclone remote target with optional automated sync profile provisioning bridge |
| `bisync_dedupe_by_hash` | **SHA-256 Ticket** | Execute Rclone deduplication on a remote or profile path |
| `bisync_delete_item` | **SHA-256 Ticket** | Safely delete a file or directory with canary protection and confirmation safeguards |
| `bisync_delete_profile` | **SHA-256 Ticket** | Permanently delete a sync profile with confirmation token protection |
| `bisync_delete_remote` | **SHA-256 Ticket** | Delete an existing Rclone remote with blast-radius protection and profile-link safeguards |
| `bisync_explore_path` | Jail-Confined | Safely explore files and directories under a configured profile or local path with jail containment |
| `bisync_get_logs` | Read-Only | Retrieve recent bisync execution logs with tail limit, severity level, profile, and run ID filtering |
| `bisync_get_status` | Read-Only | Get compact system health summary (status, active syncs, daemon health, canary status) |
| `bisync_get_sync_history` | Read-Only | Query historical sync execution records and audit ledger |
| `bisync_hasher_command` | Read-Only | Execute Hasher cache maintenance command (dump, fulldump, drop, import, stickyimport) |
| `bisync_inspect_conflict_session` | Read-Only | Retrieve pending conflict review session details, file differences, and candidate resolutions |
| `bisync_list_profiles` | Read-Only | List all configured bisync profiles with current status, schedules, and path mappings |
| `bisync_list_providers` | Read-Only | List supported Rclone cloud and object storage providers categorized by type |
| `bisync_list_remotes` | Read-Only | List configured Rclone remotes, their storage types, and connectivity status |
| `bisync_mkdir` | Read-Only | Create a directory under local or remote workspace path |
| `bisync_move_item` | Jail-Confined | Move or rename a file or directory between remote and local storage targets |
| `bisync_preview_file` | Read-Only | Inspect safe text or image preview metadata for a file under remote or local path |
| `bisync_purge_dir` | **SHA-256 Ticket** | Recursively delete a directory and all contents with high blast-radius safeguards |
| `bisync_resolve_review` | **SHA-256 Ticket** | Stage or apply conflict resolution decisions on a review session |
| `bisync_rollback_profile` | **SHA-256 Ticket** | Roll back a sync profile to a previous version snapshot from configuration history |
| `bisync_trigger_sync` | **SHA-256 Ticket** | Trigger synchronization on a profile with blast-radius protection and dry-run safety |
| `bisync_update_profile` | Read-Only | Update configuration settings on an existing sync profile |
| `bisync_validate_flags` | Read-Only | Validate CLI flags and preview compiled Rclone bisync parameters and engine config |

### Tool Parameter Specifications

#### `bisync_copy_item`

| Parameter | Type | Required | Description |
| :--- | :--- | :---: | :--- |
| `src_remote` | `string` | **Yes** | Source remote name or 'local' |
| `src_path` | `string` | **Yes** | Source relative path |
| `dst_remote` | `string` | **Yes** | Destination remote name or 'local' |
| `dst_path` | `string` | **Yes** | Destination relative path |
| `is_dir` | `boolean` | No | True if copying a directory |
| `profile_id` | `string` | No | Optional profile ID to infer local jail root and excludes |

#### `bisync_create_crypt_remote`

| Parameter | Type | Required | Description |
| :--- | :--- | :---: | :--- |
| `name` | `string` | **Yes** | Unique name for the encrypted remote (alphanumeric, dash, underscore) |
| `remote` | `string` | **Yes** | Base remote target to wrap (e.g. 'myremote:path') |
| `password` | `string` | **Yes** | Primary encryption password |
| `password2` | `string` | No | Optional secondary password / salt |
| `filename_encryption` | `string` | No | Filename encryption mode: 'standard', 'obfuscate', 'off' (default: 'standard') |
| `create_profile` | `boolean` | No | Whether to automatically create a linked sync profile |
| `local_path` | `string` | No | Local filesystem path if creating a linked profile |
| `schedule` | `string` | No | Sync schedule interval preset: 'manual', '15m', '30m', '1h', '4h', 'daily' (default: 'manual') |

#### `bisync_create_profile`

| Parameter | Type | Required | Description |
| :--- | :--- | :---: | :--- |
| `name` | `string` | **Yes** | Human-readable profile name |
| `remote` | `string` | **Yes** | Configured remote target name (or 'local', e.g. 'drive', 'backup') |
| `local_path` | `string` | **Yes** | Local filesystem directory path |
| `remote_path` | `string` | No | Optional remote subdirectory path |
| `schedule` | `string` | No | Sync schedule interval preset: 'manual', '15m', '30m', '1h', '4h', 'daily' (default: 'manual') |
| `conflict_mode` | `string` | No | Conflict resolution mode: 'path1', 'path2', 'newer', 'older', 'larger', 'smaller' |
| `review_mode` | `string` | No | Review mode for operations: 'disabled', 'deletions', 'conflicts', 'all' (default: 'disabled') |
| `compare_mode` | `string` | No | Comparison mode: 'size,modtime', 'size,modtime,checksum', 'checksum', 'modtime' |
| `max_delete` | `number` | No | Max allowable deletion percentage before triggering safety abort |

#### `bisync_create_remote`

| Parameter | Type | Required | Description |
| :--- | :--- | :---: | :--- |
| `name` | `string` | **Yes** | Unique name for the remote (alphanumeric, dash, underscore; colons stripped) |
| `type` | `string` | **Yes** | Provider backend type (e.g. s3, drive, b2, sftp) |
| `create_profile` | `boolean` | No | Whether to automatically create a linked sync profile |
| `local_path` | `string` | No | Local filesystem path if creating a linked profile |
| `remote_subpath` | `string` | No | Optional subdirectory path within the remote |
| `profile_name` | `string` | No | Optional display name for the linked profile |
| `schedule` | `string` | No | Sync schedule interval preset: 'manual', '15m', '30m', '1h', '4h', 'daily' (default: 'manual') |

#### `bisync_dedupe_by_hash`

| Parameter | Type | Required | Description |
| :--- | :--- | :---: | :--- |
| `remote` | `string` | **Yes** | Target remote or local path |
| `path` | `string` | No | Subdirectory path within the remote |
| `mode` | `string` | No | Deduplication resolution strategy: 'list', 'oldest', or 'newest' |
| `by_hash` | `boolean` | No | Find duplicates by cryptographic hash instead of name/size only |
| `confirmation_token` | `string` | No | Confirmation token required when mode != 'list' |

#### `bisync_delete_item`

| Parameter | Type | Required | Description |
| :--- | :--- | :---: | :--- |
| `remote` | `string` | **Yes** | Target remote name or 'local' |
| `path` | `string` | **Yes** | Relative path to delete |
| `is_dir` | `boolean` | No | True if deleting a directory |
| `profile_id` | `string` | No | Optional profile ID to infer local jail root |
| `confirmation_token` | `string` | No | Confirmation token required for deletion |

#### `bisync_delete_profile`

| Parameter | Type | Required | Description |
| :--- | :--- | :---: | :--- |
| `profile_id` | `string` | **Yes** | Profile ID to permanently delete |
| `confirmation_token` | `string` | No | Confirmation token required for profile deletion |

#### `bisync_delete_remote`

| Parameter | Type | Required | Description |
| :--- | :--- | :---: | :--- |
| `name` | `string` | **Yes** | Target remote name to delete |
| `force` | `boolean` | No | Force deletion even if active profiles reference this remote |
| `confirmation_token` | `string` | No | Confirmation token required for destructive remote deletion |

#### `bisync_explore_path`

| Parameter | Type | Required | Description |
| :--- | :--- | :---: | :--- |
| `remote` | `string` | **Yes** | Storage target, e.g. 'local' or 'remote:' |
| `path` | `string` | No | Relative path within the remote to explore |
| `profile_id` | `string` | No | Profile ID to infer local jail root from |
| `limit` | `number` | No | Max items to return (default: 50, max: 200) |
| `offset` | `number` | No | Item offset for pagination |
| `filter` | `string` | No | Optional substring to filter item names/paths |

#### `bisync_get_logs`

| Parameter | Type | Required | Description |
| :--- | :--- | :---: | :--- |
| `limit` | `number` | No | Number of recent log entries to retrieve (max 100) |
| `level` | `string` | No | Optional severity filter (e.g. ERROR, NOTICE, INFO, DEBUG) |
| `profile_id` | `string` | No | Optional profile ID filter to scope log output to a specific profile |
| `run_id` | `string` | No | Optional run ID filter to inspect logs for a specific sync execution |

#### `bisync_get_status`

| Parameter | Type | Required | Description |
| :--- | :--- | :---: | :--- |
| `include_logs` | `boolean` | No | Include recent bisync and daemon logs |
| `include_config` | `boolean` | No | Include sanitized runtime configuration |
| `include_metrics` | `boolean` | No | Include execution metrics timeseries |

#### `bisync_get_sync_history`

| Parameter | Type | Required | Description |
| :--- | :--- | :---: | :--- |
| `profile_id` | `string` | No | Optional profile ID to filter history |
| `limit` | `number` | No | Maximum number of records to return (default: 20, max: 100) |
| `include_hashes` | `boolean` | No | Include cryptographic prev_hash and hash fields |

#### `bisync_hasher_command`

| Parameter | Type | Required | Description |
| :--- | :--- | :---: | :--- |
| `remote` | `string` | **Yes** | Hasher remote target name |
| `command` | `string` | **Yes** | Hasher backend command: dump, fulldump, drop, import, stickyimport |
| `args` | `string` | No | Optional positional arguments or space-separated string (e.g. hash types) |

#### `bisync_inspect_conflict_session`

| Parameter | Type | Required | Description |
| :--- | :--- | :---: | :--- |
| `session_id` | `string` | No | Optional conflict session ID; returns active session if omitted |
| `unresolved_only` | `boolean` | No | Only return unresolved conflict and deletion candidates |

#### `bisync_list_profiles`

| Parameter | Type | Required | Description |
| :--- | :--- | :---: | :--- |
| `include_archived` | `boolean` | No | Include archived profiles |

#### `bisync_list_providers`

| Parameter | Type | Required | Description |
| :--- | :--- | :---: | :--- |
| `category` | `string` | No | Optional filter by category: cloud, object, protocol, virtual, other |
| `popular_only` | `boolean` | No | Filter to only popular providers (e.g., drive, onedrive, s3, b2, dropbox) |
| `force_refresh` | `boolean` | No | Bypass internal 5-minute cache and query rclone RC fresh |

#### `bisync_list_remotes`

| Parameter | Type | Required | Description |
| :--- | :--- | :---: | :--- |
| `refresh` | `boolean` | No | Force fresh probe of remote status |

#### `bisync_mkdir`

| Parameter | Type | Required | Description |
| :--- | :--- | :---: | :--- |
| `remote` | `string` | **Yes** | Target remote name or 'local' |
| `path` | `string` | **Yes** | Directory path to create |
| `profile_id` | `string` | No | Optional profile ID to infer local jail root |

#### `bisync_move_item`

| Parameter | Type | Required | Description |
| :--- | :--- | :---: | :--- |
| `src_remote` | `string` | **Yes** | Source remote name or 'local' |
| `src_path` | `string` | **Yes** | Source relative path |
| `dst_remote` | `string` | **Yes** | Destination remote name or 'local' |
| `dst_path` | `string` | **Yes** | Destination relative path |
| `is_dir` | `boolean` | No | True if moving a directory |
| `profile_id` | `string` | No | Optional profile ID to infer local jail root and excludes |

#### `bisync_preview_file`

| Parameter | Type | Required | Description |
| :--- | :--- | :---: | :--- |
| `remote` | `string` | **Yes** | Target remote name or 'local' |
| `path` | `string` | **Yes** | Relative file path to preview |
| `profile_id` | `string` | No | Optional profile ID to infer local jail root |

#### `bisync_purge_dir`

| Parameter | Type | Required | Description |
| :--- | :--- | :---: | :--- |
| `remote` | `string` | **Yes** | Target remote name or 'local' |
| `path` | `string` | **Yes** | Directory relative path to purge |
| `profile_id` | `string` | No | Optional profile ID to infer local jail root |
| `confirmation_token` | `string` | No | Confirmation token required for directory purge |

#### `bisync_resolve_review`

| Parameter | Type | Required | Description |
| :--- | :--- | :---: | :--- |
| `session_id` | `string` | No | Conflict review session ID (defaults to active session) |
| `candidate_id` | `string` | No | Optional specific candidate file ID to stage decision for |
| `decision` | `string` | **Yes** | Resolution decision: 'keep-local', 'keep-remote', 'keep-both', 'accept-delete', 'restore-item', 'skip', or 'dismiss' |
| `execute` | `boolean` | No | Apply and execute staged decisions to disk (requires confirmation) |
| `confirmation_token` | `string` | No | Confirmation token required when execute=true |

#### `bisync_rollback_profile`

| Parameter | Type | Required | Description |
| :--- | :--- | :---: | :--- |
| `profile_id` | `string` | **Yes** | Profile ID to roll back |
| `version` | `number` | **Yes** | Target historical version number to restore |
| `confirmation_token` | `string` | No | Confirmation token required for profile rollback |

#### `bisync_trigger_sync`

| Parameter | Type | Required | Description |
| :--- | :--- | :---: | :--- |
| `profile_id` | `string` | **Yes** | Target profile ID |
| `dry_run` | `boolean` | No | Perform a dry-run without writing changes |
| `resync` | `boolean` | No | Force full resync (rebuilds bisync cache; requires confirmation) |
| `force` | `boolean` | No | Bypass lock checks and override conflict safety (requires confirmation) |
| `confirmation_token` | `string` | No | Confirmation token returned by previous invocation for high blast-radius operations |

#### `bisync_update_profile`

| Parameter | Type | Required | Description |
| :--- | :--- | :---: | :--- |
| `profile_id` | `string` | **Yes** | ID of the profile to update |
| `name` | `string` | No | Updated human-readable name |
| `remote` | `string` | No | Updated remote target name |
| `local_path` | `string` | No | Updated local directory path |
| `remote_path` | `string` | No | Updated remote subdirectory path |
| `schedule` | `string` | No | Updated sync schedule interval preset: 'manual', '15m', '30m', '1h', '4h', 'daily' |
| `conflict_mode` | `string` | No | Updated conflict resolution mode: 'path1', 'path2', 'newer', 'older', 'larger', 'smaller' |
| `review_mode` | `string` | No | Updated review mode: 'disabled', 'deletions', 'conflicts', 'all' |
| `compare_mode` | `string` | No | Updated comparison mode: 'size,modtime', 'size,modtime,checksum', 'checksum', 'modtime' |
| `max_delete` | `number` | No | Updated max delete percentage threshold |
| `enabled` | `boolean` | No | Enable or disable scheduled execution |

#### `bisync_validate_flags`

| Parameter | Type | Required | Description |
| :--- | :--- | :---: | :--- |
| `flags` | `string` | No | CLI flags to compile and validate (space-separated string or array) |
| `profile_id` | `string` | No | Optional profile ID to compile and validate its effective configured flags |

