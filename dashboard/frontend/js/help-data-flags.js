/**
 * In-App Reference Manual: CLI Flags Catalog Data
 * Zero-build vanilla JS module exposing window.HelpCatalog.flags
 */
window.HelpCatalog = window.HelpCatalog || {};

window.HelpCatalog.flags = [
    {
        flag: "--track-renames",
        name: "Rename & Move Detection",
        category: "Detection",
        backends: ["all"],
        contexts: ["modal_profiles", "view_explorer"],
        description: "Detects file moves and renames using hashes and modtimes, avoiding destructive delete-and-reupload cycles.",
        example: "rclone bisync local: remote: --track-renames",
        docUrl: "https://rclone.org/bisync/#track-renames"
    },
    {
        flag: "--track-renames-strategy",
        name: "Rename Matching Strategy",
        category: "Detection",
        backends: ["all"],
        contexts: ["modal_profiles"],
        description: "Priority order for matching renamed files: 'modtime', 'leaf', or 'hash' (default: modtime,leaf).",
        example: "rclone bisync local: remote: --track-renames --track-renames-strategy modtime,leaf",
        docUrl: "https://rclone.org/bisync/#track-renames-strategy"
    },
    {
        flag: "--fast-list",
        name: "Recursive Directory Listing",
        category: "Performance",
        backends: ["all"],
        contexts: ["modal_profiles", "view_remotes"],
        description: "Uses recursive listing calls to reduce cloud API transactions on providers supporting it (e.g., GDrive, S3, OneDrive).",
        caveat: "On OneDrive, --fast-list requires recursive listing and errors if shortcuts or Personal Vault are in the sync path. Delta enumeration is preferred.",
        example: "rclone bisync local: remote: --fast-list",
        docUrl: "https://rclone.org/flags/#fast-list"
    },
    {
        flag: "--drive-shared-with-me",
        name: "Google Drive Shared Access",
        category: "Provider",
        backends: ["drive"],
        contexts: ["modal_profiles", "view_remotes"],
        description: "Instructs Google Drive backend to list and sync files shared with the user rather than root 'My Drive'.",
        example: "rclone bisync local: gdrive: --drive-shared-with-me",
        docUrl: "https://rclone.org/drive/#drive-shared-with-me"
    },
    {
        flag: "--checksum",
        name: "Checksum Hash Verification",
        category: "Verification",
        backends: ["all"],
        contexts: ["modal_profiles", "modal_review"],
        description: "Checks hashes and sizes instead of modification times during synchronization diffing.",
        example: "rclone bisync local: remote: --checksum",
        docUrl: "https://rclone.org/flags/#c-checksum"
    },
    {
        flag: "--ignore-checksum",
        name: "Bypass Checksum Validation",
        category: "Verification",
        backends: ["all"],
        contexts: ["modal_profiles"],
        description: "Ignores cryptographic checksum comparisons during sync diffing, relying on modtime and size.",
        example: "rclone bisync local: remote: --ignore-checksum",
        docUrl: "https://rclone.org/flags/#ignore-checksum"
    },
    {
        flag: "--size-only",
        name: "Size-Only Diff Comparison",
        category: "Performance",
        backends: ["all"],
        contexts: ["modal_profiles"],
        description: "Compares file sizes only, skipping modification times. Ideal for coarse filesystem timestamp resolutions.",
        example: "rclone bisync local: remote: --size-only",
        docUrl: "https://rclone.org/flags/#size-only"
    },
    {
        flag: "--resilient",
        name: "Resilient Error Tolerance",
        category: "Resilience",
        backends: ["all"],
        contexts: ["modal_profiles", "view_telemetry"],
        description: "Allows bisync to continue processing non-fatal errors without aborting the synchronization run immediately.",
        example: "rclone bisync local: remote: --resilient",
        docUrl: "https://rclone.org/bisync/#resilient"
    },
    {
        flag: "--recover",
        name: "Interrupted Run Auto-Recovery",
        category: "Recovery",
        backends: ["all"],
        contexts: ["modal_profiles", "view_telemetry"],
        description: "Automatically recovers from interrupted or failed sync runs without requiring a manual resync.",
        example: "rclone bisync local: remote: --recover",
        docUrl: "https://rclone.org/bisync/#recover"
    },
    {
        flag: "--checkers",
        name: "Parallel File Checkers",
        category: "Concurrency",
        backends: ["all"],
        contexts: ["modal_profiles", "modal_global_settings"],
        description: "Number of parallel directory and file comparison threads (default: 8). Increase for high latency links.",
        example: "rclone bisync local: remote: --checkers 16",
        docUrl: "https://rclone.org/flags/#checkers-n"
    },
    {
        flag: "--transfers",
        name: "Concurrent File Transfers",
        category: "Concurrency",
        backends: ["all"],
        contexts: ["modal_profiles", "modal_global_settings"],
        description: "Number of concurrent file uploads/downloads executed simultaneously (default: 4).",
        example: "rclone bisync local: remote: --transfers 8",
        docUrl: "https://rclone.org/flags/#transfers-n"
    },
    {
        flag: "--bwlimit",
        name: "Bandwidth Transfer Limit",
        category: "Rate Limiting",
        backends: ["all"],
        contexts: ["modal_profiles", "modal_global_settings"],
        description: "Caps upload/download bandwidth rate (e.g., 10M, 500k, or timetable 08:00,5M 18:00,off).",
        example: "rclone bisync local: remote: --bwlimit 10M",
        docUrl: "https://rclone.org/flags/#bwlimit-bandwidth-spec"
    },
    {
        flag: "--tpslimit",
        name: "Transaction Rate Limiting",
        category: "Rate Limiting",
        backends: ["all"],
        contexts: ["modal_profiles", "modal_global_settings", "view_remotes"],
        description: "Restricts transactions-per-second to prevent cloud API rate limit HTTP 429 throttling (e.g., Google Drive, Box).",
        caveat: "Crucial for Microsoft Graph API (OneDrive/SharePoint) and Google Drive to avert aggressive HTTP 429 rate limit throttling. Recommended: 10 TPS.",
        example: "rclone bisync local: remote: --tpslimit 10",
        docUrl: "https://rclone.org/flags/#tpslimit-float"
    },
    {
        flag: "--tpslimit-burst",
        name: "Transaction Burst Allowance",
        category: "Rate Limiting",
        backends: ["all"],
        contexts: ["modal_profiles", "modal_global_settings"],
        description: "Max number of HTTP requests allowed in a sudden burst when --tpslimit is active (default: 1).",
        example: "rclone bisync local: remote: --tpslimit 10 --tpslimit-burst 5",
        docUrl: "https://rclone.org/flags/#tpslimit-burst-n"
    },
    {
        flag: "--buffer-size",
        name: "In-Memory Read Buffer",
        category: "Performance",
        backends: ["all"],
        contexts: ["modal_profiles", "modal_global_settings"],
        description: "Memory buffer size per active file transfer (e.g. 16M, 64M). Balances upload throughput with daemon RAM.",
        example: "rclone bisync local: remote: --buffer-size 32M",
        docUrl: "https://rclone.org/flags/#buffer-size-size"
    },
    {
        flag: "--retries",
        name: "Sync Attempt Retry Ceiling",
        category: "Resilience",
        backends: ["all"],
        contexts: ["modal_profiles", "modal_global_settings"],
        description: "Number of full sync cycle retry attempts when recoverable errors occur during execution (default: 3).",
        example: "rclone bisync local: remote: --retries 5",
        docUrl: "https://rclone.org/flags/#retries-int"
    },
    {
        flag: "--retries-sleep",
        name: "Retry Exponential Sleep",
        category: "Resilience",
        backends: ["all"],
        contexts: ["modal_profiles", "modal_global_settings"],
        description: "Interval to sleep between retries, scaling with exponential backoff on subsequent failures (e.g., 500ms, 2s).",
        example: "rclone bisync local: remote: --retries-sleep 2s",
        docUrl: "https://rclone.org/flags/#retries-sleep-duration"
    },
    {
        flag: "--low-level-retries",
        name: "Low-Level HTTP Retries",
        category: "Resilience",
        backends: ["all"],
        contexts: ["modal_profiles", "modal_global_settings"],
        description: "Number of low-level transport retries executed on individual HTTP chunk requests (default: 10).",
        example: "rclone bisync local: remote: --low-level-retries 20",
        docUrl: "https://rclone.org/flags/#low-level-retries-int"
    },
    {
        flag: "--resync",
        name: "Full Listing Cache Rebuild",
        category: "Recovery",
        backends: ["all"],
        contexts: ["modal_profiles", "modal_dedupe", "view_telemetry"],
        description: "Re-initializes the bisync cache by building fresh file listings from both endpoints without propagating deletions.",
        example: "rclone bisync local: remote: --resync --verbose",
        docUrl: "https://rclone.org/bisync/#resync"
    },
    {
        flag: "--dry-run",
        name: "Simulation / Trial Execution",
        category: "Safety",
        backends: ["all"],
        contexts: ["modal_profiles", "modal_review", "modal_global_settings", "view_telemetry"],
        description: "Simulates the synchronization cycle without modifying or deleting files on either endpoint.",
        example: "rclone bisync local: remote: --dry-run -v",
        docUrl: "https://rclone.org/flags/#n-dry-run"
    },
    {
        flag: "--max-delete",
        name: "Deletion Threshold Ceiling",
        category: "Safety",
        backends: ["all"],
        contexts: ["modal_profiles", "modal_global_settings", "modal_review"],
        description: "Maximum number or percentage of files permitted to be deleted during a single sync before aborting.",
        example: "rclone bisync local: remote: --max-delete 50",
        docUrl: "https://rclone.org/bisync/#max-delete"
    },
    {
        flag: "--conflict-resolve",
        name: "Automatic Conflict Resolution",
        category: "Resolution",
        backends: ["all"],
        contexts: ["modal_profiles", "modal_global_settings", "modal_review"],
        description: "Strategy for resolving conflicting modifications: path1 (prefer local), path2 (prefer remote), newer, older, larger, or smaller.",
        example: "rclone bisync local: remote: --conflict-resolve newer",
        docUrl: "https://rclone.org/bisync/#conflict-resolve"
    },
    {
        flag: "--compare",
        name: "File Comparison Criteria",
        category: "Performance",
        backends: ["all"],
        contexts: ["modal_profiles", "modal_global_settings"],
        description: "Comma-separated comparison attributes: size,modtime (standard), size,modtime,checksum (thorough), or checksum only.",
        example: "rclone bisync local: remote: --compare size,modtime,checksum",
        docUrl: "https://rclone.org/bisync/#compare"
    },
    {
        flag: "--check-access",
        name: "Canary Access Verification",
        category: "Safety",
        backends: ["all"],
        contexts: ["modal_profiles", "modal_global_settings", "view_remotes"],
        description: "Verifies the presence of the canary file (default RCLONE_TEST) on both Path1 and Path2 before executing sync.",
        example: "rclone bisync local: remote: --check-access",
        docUrl: "https://rclone.org/bisync/#check-access"
    },
    {
        flag: "--filters-file",
        name: "Include / Exclude Rule File",
        category: "Filtering",
        backends: ["all"],
        contexts: ["modal_profiles", "view_explorer"],
        description: "Loads file filtering rules (+ include, - exclude) from a specified text file for ignoring artifacts and OS files.",
        example: "rclone bisync local: remote: --filters-file /path/to/filters.txt",
        docUrl: "https://rclone.org/filtering/#filter-from-filter-file"
    },
    {
        flag: "--slow-hash",
        name: "Slow Hash Provider Handling",
        category: "Performance",
        backends: ["all"],
        contexts: ["modal_profiles", "view_remotes"],
        description: "Allows checksum comparisons on cloud providers that compute hashes slowly or on-demand without timing out.",
        example: "rclone bisync local: remote: --compare checksum --slow-hash",
        docUrl: "https://rclone.org/bisync/#slow-hash"
    },
    {
        flag: "--onedrive-no-versions",
        name: "OneDrive Prevent Version Duplication",
        category: "Provider",
        backends: ["onedrive"],
        contexts: ["modal_profiles", "view_remotes"],
        description: "Prevents OneDrive from creating unnecessary historical file versions on upload, saving cloud storage quota and preventing version buildup.",
        caveat: "Essential for OneDrive sync pairs that update frequently to prevent quota exhaustion from hidden version histories.",
        example: "rclone bisync local: onedrive: --onedrive-no-versions",
        docUrl: "https://rclone.org/onedrive/#onedrive-no-versions"
    },
    {
        flag: "--onedrive-chunk-size",
        name: "OneDrive Upload Chunk Size",
        category: "Performance",
        backends: ["onedrive"],
        contexts: ["modal_profiles", "view_remotes"],
        description: "Upload chunk size for large files (must be multiple of 320k, e.g. 10M, 50M, 100M). Larger chunks increase upload throughput on reliable connections.",
        caveat: "Must be an exact multiple of 320k. Larger values consume more daemon RAM per transfer but significantly boost upload speeds.",
        example: "rclone bisync local: onedrive: --onedrive-chunk-size 10M",
        docUrl: "https://rclone.org/onedrive/#onedrive-chunk-size"
    },
    {
        flag: "--onedrive-delta",
        name: "OneDrive Delta Enumeration",
        category: "Performance",
        backends: ["onedrive"],
        contexts: ["modal_profiles", "view_remotes"],
        description: "Enables delta change enumeration on OneDrive via Microsoft Graph API for rapid incremental change detection.",
        caveat: "Incompatible with Personal Vault and cannot traverse shortcut links. Recommended when fast directory delta scanning is desired.",
        example: "rclone bisync local: onedrive: --onedrive-delta",
        docUrl: "https://rclone.org/onedrive/#onedrive-delta"
    }
];

