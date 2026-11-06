/**
 * In-App Reference Manual: Safety Policies & Runbooks Catalog Data
 * Zero-build vanilla JS module exposing window.HelpCatalog
 */
window.HelpCatalog = window.HelpCatalog || {};

window.HelpCatalog.policies = [
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
        id: "mandatory_subdir",
        title: "MANDATORY_SUBDIR Isolation",
        tag: "Path Confinement",
        contexts: ["modal_profiles", "view_explorer", "modal_dedupe"],
        description: "Direct synchronization of root directories (e.g. / or C:\\) is prohibited by rclone bisync to prevent accidental total system wipes. Every synchronization pair must target a designated subfolder.",
        action: "Specify an explicit subfolder path for both local and remote endpoints (e.g., /home/user/data or remote:backup/subfolder)."
    },
    {
        id: "too_many_deletes",
        title: "Circuit Breaker: TOO_MANY_DELETES",
        tag: "Threshold Guard",
        contexts: ["modal_profiles", "modal_global_settings", "modal_review", "modal_deletions", "view_telemetry"],
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
    },
    {
        id: "heuristic_risk_score",
        title: "Heuristic Risk Scoring Engine",
        tag: "Conflict Engine",
        contexts: ["modal_review", "modal_profiles"],
        description: "Evaluates conflict candidates across 4 parallel dimensions: size divergence (30%), deletion volume (30%), modification volatility (20%), and path depth (20%) to produce safety risk scores.",
        action: "Review candidates in the Sync Review modal. Safe candidates can be auto-staged using Auto-Apply Recommendations. High-risk items require manual decision."
    }
];

window.HelpCatalog.runbooks = [
    {
        id: "too_many_deletes",
        title: "Resolving TOO_MANY_DELETES Circuit Breakers",
        category: "Deletions",
        contexts: ["modal_profiles", "modal_global_settings", "modal_review"],
        summary: "Bisync halts execution when planned deletions exceed the safety threshold (default 50% or configured value).",
        steps: [
            "Open Sync Review & Resolution engine to inspect staged candidates and diff preview.",
            "Verify whether mass deletion is legitimate (e.g. deliberate folder removal) or accidental.",
            "If valid, review candidates and click 'Apply Selected Decisions' with confirmation.",
            "If unexpected, dismiss review session and run dry-run sync to examine diffs under blast-shield protection."
        ],
        command: "rclone bisync local: remote: --dry-run -v"
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
        ],
        command: "touch /path/to/local/RCLONE_TEST"
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
        ],
        command: "rclone config reconnect myremote:"
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
        ],
        command: "rclone-bisync status --include-config"
    },
    {
        id: "resync_cache",
        title: "Rebuilding Desynchronized Listing Caches",
        category: "Cache",
        contexts: ["modal_profiles", "modal_dedupe", "view_explorer"],
        summary: "If listing caches fall out of sync due to external modifications or compare mode changes, a one-time --resync rebuilds the baseline.",
        steps: [
            "Ensure both local and remote folders are online, accessible, and in the desired initial state.",
            "In the Bisync Hub, select the profile, click Run Sync, and enable the Resync option.",
            "Review the dry-run simulation output to verify that files will not be overwritten inappropriately.",
            "Confirm and execute. Bisync will rebuild both path listings and establish a clean sync baseline."
        ],
        command: "rclone bisync local: remote: --resync --verbose"
    }
];
