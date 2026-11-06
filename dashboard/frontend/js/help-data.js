/**
 * In-App Reference Manual & Diagnostics Catalog Data
 * Zero-build vanilla JS module exposing window.HelpCatalog
 */
window.HelpCatalog = {
    policies: [
        {
            id: "posix_locking",
            title: "POSIX Advisory Locking & Blast Shield",
            tag: "Lock Safety",
            contexts: ["modal_profiles", "view_fleet", "view_explorer"],
            description: "Enforces atomic non-blocking POSIX flock isolation on daemon operations. Prevents concurrent bisync invocations on the same workload.",
            action: "Clear stale locks only if PID verification confirms no active sync process is running."
        },
        {
            id: "canary_protection",
            title: "RCLONE_TEST Canary Validation",
            tag: "Canary Probe",
            contexts: ["modal_profiles", "modal_global_settings", "view_remotes", "view_explorer"],
            description: "Places a zero-byte token (RCLONE_TEST) on both endpoints. Bisync verifies this file before proceeding; if missing, sync terminates immediately.",
            action: "Check that both endpoints are mounted and reachable. Ensure RCLONE_TEST is present on both sides."
        },
        {
            id: "too_many_deletes",
            title: "Circuit Breaker: TOO_MANY_DELETES",
            tag: "Threshold Guard",
            contexts: ["modal_profiles", "modal_global_settings", "modal_review"],
            description: "Restricts mass file deletions per execution cycle. Triggers critical safety abort whenever planned deletions exceed configured threshold percentage.",
            action: "Inspect review session candidates to verify deletions are intentional, or adjust threshold in profile settings."
        },
        {
            id: "rate_limit_backoff",
            title: "Exponential Backoff & API Quota Protection",
            tag: "Quota Protection",
            contexts: ["modal_profiles", "modal_global_settings", "view_remotes"],
            description: "Applies automated delay intervals and exponential backoff on HTTP 429 and rate-limiting responses from cloud backends.",
            action: "Configure --tpslimit or check cloud provider API quotas."
        },
        {
            id: "isolated_jail",
            title: "Local Filesystem Jail Confinement",
            tag: "Path Jail",
            contexts: ["view_explorer", "modal_review"],
            description: "Restricts all filesystem operations strictly within the configured profile local root directory to prevent directory traversal.",
            action: "Verify relative path arguments and ensure directory symlinks do not target paths outside the local sync root."
        }
    ],

    flags: [
        {
            flag: "--track-renames",
            name: "Rename & Move Detection",
            category: "Detection",
            contexts: ["modal_profiles", "view_explorer"],
            description: "Detects file moves and renames using hashes and modtimes, avoiding destructive delete-and-reupload cycles.",
            example: "rclone bisync local: remote: --track-renames",
            docUrl: "https://rclone.org/bisync/#track-renames"
        },
        {
            flag: "--track-renames-strategy",
            name: "Rename Matching Strategy",
            category: "Detection",
            contexts: ["modal_profiles"],
            description: "Priority order for matching renamed files: 'modtime', 'leaf', or 'hash' (default: modtime,leaf).",
            example: "rclone bisync local: remote: --track-renames --track-renames-strategy modtime,leaf",
            docUrl: "https://rclone.org/bisync/#track-renames-strategy"
        },
        {
            flag: "--fast-list",
            name: "Recursive Directory Listing",
            category: "Performance",
            contexts: ["modal_profiles", "view_remotes"],
            description: "Uses recursive listing calls to reduce cloud API transactions on providers supporting it (e.g., GDrive, S3, OneDrive).",
            example: "rclone bisync local: remote: --fast-list",
            docUrl: "https://rclone.org/flags/#fast-list"
        },
        {
            flag: "--drive-shared-with-me",
            name: "Google Drive Shared Access",
            category: "Provider",
            contexts: ["modal_profiles", "view_remotes"],
            description: "Instructs Google Drive backend to list and sync files shared with the user rather than root 'My Drive'.",
            example: "rclone bisync local: gdrive: --drive-shared-with-me",
            docUrl: "https://rclone.org/drive/#drive-shared-with-me"
        },
        {
            flag: "--checksum",
            name: "Checksum Hash Verification",
            category: "Verification",
            contexts: ["modal_profiles", "modal_review"],
            description: "Checks hashes and sizes instead of modification times during synchronization diffing.",
            example: "rclone bisync local: remote: --checksum",
            docUrl: "https://rclone.org/flags/#c-checksum"
        },
        {
            flag: "--ignore-checksum",
            name: "Bypass Checksum Validation",
            category: "Verification",
            contexts: ["modal_profiles"],
            description: "Ignores cryptographic checksum comparisons during sync diffing, relying on modtime and size.",
            example: "rclone bisync local: remote: --ignore-checksum",
            docUrl: "https://rclone.org/flags/#ignore-checksum"
        },
        {
            flag: "--size-only",
            name: "Size-Only Diff Comparison",
            category: "Performance",
            contexts: ["modal_profiles"],
            description: "Compares file sizes only, skipping modification times. Ideal for coarse filesystem timestamp resolutions.",
            example: "rclone bisync local: remote: --size-only",
            docUrl: "https://rclone.org/flags/#size-only"
        },
        {
            flag: "--resilient",
            name: "Resilient Error Tolerance",
            category: "Resilience",
            contexts: ["modal_profiles", "view_telemetry"],
            description: "Allows bisync to continue processing non-fatal errors without aborting the synchronization run immediately.",
            example: "rclone bisync local: remote: --resilient",
            docUrl: "https://rclone.org/bisync/#resilient"
        },
        {
            flag: "--recover",
            name: "Interrupted Run Auto-Recovery",
            category: "Recovery",
            contexts: ["modal_profiles", "view_telemetry"],
            description: "Automatically recovers from interrupted or failed sync runs without requiring a full manual --resync.",
            example: "rclone bisync local: remote: --recover",
            docUrl: "https://rclone.org/bisync/#recover"
        },
        {
            flag: "--checkers",
            name: "Parallel File Checkers",
            category: "Concurrency",
            contexts: ["modal_profiles", "modal_global_settings"],
            description: "Number of parallel directory and file comparison threads (default: 8). Increase for high latency links.",
            example: "rclone bisync local: remote: --checkers 16",
            docUrl: "https://rclone.org/flags/#checkers-n"
        },
        {
            flag: "--transfers",
            name: "Concurrent File Transfers",
            category: "Concurrency",
            contexts: ["modal_profiles", "modal_global_settings"],
            description: "Number of concurrent file uploads/downloads executed simultaneously (default: 4).",
            example: "rclone bisync local: remote: --transfers 8",
            docUrl: "https://rclone.org/flags/#transfers-n"
        },
        {
            flag: "--bwlimit",
            name: "Bandwidth Transfer Limit",
            category: "Rate Limiting",
            contexts: ["modal_profiles", "modal_global_settings"],
            description: "Caps upload/download bandwidth rate (e.g., 10M, 500k, or timetable 08:00,5M 18:00,off).",
            example: "rclone bisync local: remote: --bwlimit 10M",
            docUrl: "https://rclone.org/flags/#bwlimit-bandwidth-spec"
        },
        {
            flag: "--tpslimit",
            name: "Transaction Rate Limiting",
            category: "Rate Limiting",
            contexts: ["modal_profiles", "modal_global_settings", "view_remotes"],
            description: "Restricts transactions-per-second to prevent cloud API rate limit HTTP 429 throttling (e.g., Google Drive, Box).",
            example: "rclone bisync local: remote: --tpslimit 10",
            docUrl: "https://rclone.org/flags/#tpslimit-float"
        },
        {
            flag: "--tpslimit-burst",
            name: "Transaction Burst Allowance",
            category: "Rate Limiting",
            contexts: ["modal_profiles", "modal_global_settings"],
            description: "Max number of HTTP requests allowed in a sudden burst when --tpslimit is active (default: 1).",
            example: "rclone bisync local: remote: --tpslimit 10 --tpslimit-burst 5",
            docUrl: "https://rclone.org/flags/#tpslimit-burst-n"
        },
        {
            flag: "--buffer-size",
            name: "In-Memory Read Buffer",
            category: "Performance",
            contexts: ["modal_profiles", "modal_global_settings"],
            description: "Memory buffer size per active file transfer (e.g. 16M, 64M). Balances upload throughput with daemon RAM.",
            example: "rclone bisync local: remote: --buffer-size 32M",
            docUrl: "https://rclone.org/flags/#buffer-size-size"
        },
        {
            flag: "--retries",
            name: "Sync Attempt Retry Ceiling",
            category: "Resilience",
            contexts: ["modal_profiles", "modal_global_settings"],
            description: "Number of full sync cycle retry attempts when recoverable errors occur during execution (default: 3).",
            example: "rclone bisync local: remote: --retries 5",
            docUrl: "https://rclone.org/flags/#retries-int"
        },
        {
            flag: "--retries-sleep",
            name: "Retry Exponential Sleep",
            category: "Resilience",
            contexts: ["modal_profiles", "modal_global_settings"],
            description: "Interval to sleep between retries, scaling with exponential backoff on subsequent failures (e.g., 500ms, 2s).",
            example: "rclone bisync local: remote: --retries-sleep 2s",
            docUrl: "https://rclone.org/flags/#retries-sleep-duration"
        },
        {
            flag: "--low-level-retries",
            name: "Low-Level HTTP Retries",
            category: "Resilience",
            contexts: ["modal_profiles", "modal_global_settings"],
            description: "Number of low-level transport retries executed on individual HTTP chunk requests (default: 10).",
            example: "rclone bisync local: remote: --low-level-retries 20",
            docUrl: "https://rclone.org/flags/#low-level-retries-int"
        },
        {
            flag: "--resync",
            name: "Full Listing Cache Rebuild",
            category: "Recovery",
            contexts: ["modal_profiles", "modal_dedupe", "view_telemetry"],
            description: "Re-initializes the bisync cache by building fresh file listings from both endpoints without propagating deletions.",
            example: "rclone bisync local: remote: --resync --verbose",
            docUrl: "https://rclone.org/bisync/#resync"
        },
        {
            flag: "--dry-run",
            name: "Simulation / Trial Execution",
            category: "Safety",
            contexts: ["modal_profiles", "modal_review", "modal_global_settings", "view_telemetry"],
            description: "Simulates the synchronization cycle without modifying or deleting files on either endpoint.",
            example: "rclone bisync local: remote: --dry-run -v",
            docUrl: "https://rclone.org/flags/#n-dry-run"
        },
        {
            flag: "--max-delete",
            name: "Deletion Threshold Ceiling",
            category: "Safety",
            contexts: ["modal_profiles", "modal_global_settings", "modal_review"],
            description: "Maximum number or percentage of files permitted to be deleted during a single sync before aborting.",
            example: "rclone bisync local: remote: --max-delete 50",
            docUrl: "https://rclone.org/bisync/#max-delete"
        },
        {
            flag: "--conflict-resolve",
            name: "Automatic Conflict Resolution",
            category: "Resolution",
            contexts: ["modal_profiles", "modal_global_settings", "modal_review"],
            description: "Strategy for resolving conflicting modifications: path1 (prefer local), path2 (prefer remote), newer, older, larger, or smaller.",
            example: "rclone bisync local: remote: --conflict-resolve newer",
            docUrl: "https://rclone.org/bisync/#conflict-resolve"
        },
        {
            flag: "--compare",
            name: "File Comparison Criteria",
            category: "Performance",
            contexts: ["modal_profiles", "modal_global_settings"],
            description: "Comma-separated comparison attributes: size,modtime (standard), size,modtime,checksum (thorough), or checksum only.",
            example: "rclone bisync local: remote: --compare size,modtime,checksum",
            docUrl: "https://rclone.org/bisync/#compare"
        },
        {
            flag: "--check-access",
            name: "Canary Access Verification",
            category: "Safety",
            contexts: ["modal_profiles", "modal_global_settings", "view_remotes"],
            description: "Verifies the presence of the canary file (default RCLONE_TEST) on both Path1 and Path2 before executing sync.",
            example: "rclone bisync local: remote: --check-access",
            docUrl: "https://rclone.org/bisync/#check-access"
        },
        {
            flag: "--filters-file",
            name: "Include / Exclude Rule File",
            category: "Filtering",
            contexts: ["modal_profiles", "view_explorer"],
            description: "Loads file filtering rules (+ include, - exclude) from a specified text file for ignoring artifacts and OS files.",
            example: "rclone bisync local: remote: --filters-file /path/to/filters.txt",
            docUrl: "https://rclone.org/filtering/#filter-from-filter-file"
        },
        {
            flag: "--slow-hash",
            name: "Slow Hash Provider Handling",
            category: "Performance",
            contexts: ["modal_profiles", "view_remotes"],
            description: "Allows checksum comparisons on cloud providers that compute hashes slowly or on-demand without timing out.",
            example: "rclone bisync local: remote: --compare checksum --slow-hash",
            docUrl: "https://rclone.org/bisync/#slow-hash"
        }
    ],

    runbooks: [
        {
            id: "too_many_deletes",
            title: "Resolving TOO_MANY_DELETES Circuit Breakers",
            category: "Deletions",
            contexts: ["modal_profiles", "modal_global_settings", "modal_review"],
            summary: "Bisync halts execution when planned deletions exceed the safety threshold (default 50% or configured value).",
            steps: [
                "Open Sync Review & Resolution engine to inspect staged candidates and diff preview.",
                "Verify whether deletions were intentional (e.g. bulk cleanup) or accidental (e.g. drive dismounted).",
                "If legitimate, select 'Force Deletions' or increase the threshold in Profile Settings.",
                "Trigger sync to apply approved deletions under blast-shield protection."
            ]
        },
        {
            id: "canary_missing",
            title: "Remediating Missing RCLONE_TEST Canary Tokens",
            category: "Canary Guard",
            contexts: ["modal_profiles", "modal_global_settings", "view_remotes", "view_explorer"],
            summary: "Bisync aborts immediately if the zero-byte canary file (RCLONE_TEST) is absent on either endpoint.",
            steps: [
                "Verify that both local filesystem and cloud remote endpoints are accessible and mounted.",
                "Check whether the canary file was accidentally moved, deleted, or excluded by filter rules.",
                "In Storage Remotes view, click 'Provision Canary' to re-create the missing zero-byte token.",
                "Confirm canary presence using File Explorer before restarting the workload."
            ]
        },
        {
            id: "stale_lock",
            title: "Clearing Stale Daemon Locks",
            category: "Concurrency",
            contexts: ["modal_profiles", "view_fleet", "view_telemetry"],
            summary: "If a prior bisync run crashed or terminated abruptly, an orphaned flock lock file may prevent new sync cycles.",
            steps: [
                "Inspect Engine & Telemetry tab to check active sync status and process PID.",
                "Ensure no active rclone or bisync process is currently writing data.",
                "Click 'Clear Stale Lock' in the fleet header or profile operational menu.",
                "Trigger a dry run to verify execution lock acquisition and engine responsiveness."
            ]
        },
        {
            id: "oauth_expired",
            title: "Re-authenticating Cloud OAuth Tokens",
            category: "Authentication",
            contexts: ["view_remotes", "modal_auth_audit", "modal_oauth_migration"],
            summary: "Cloud providers (Google Drive, OneDrive, Dropbox, Box) periodically expire refresh tokens, resulting in HTTP 401/403 errors.",
            steps: [
                "Navigate to Storage Remotes tab and identify remotes flagged with warning or expired status.",
                "Click 'Re-authenticate' to initiate provider token renewal or browser consent flow.",
                "Verify remote connectivity with 'Test Remote Connection'.",
                "Run a dry-run sync to confirm token permissions."
            ]
        }
    ]
};
