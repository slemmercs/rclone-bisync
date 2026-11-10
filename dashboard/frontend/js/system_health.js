/**
 * System Health & Diagnostics Center Module for Bisync Hub Dashboard
 * Unifies Daemon telemetry, MCP agent server status, and background Poller scheduler.
 */

window.SystemHealthModule = {
    systemHealthModal: {
        show: false,
        tab: "daemon", // "daemon" | "mcp" | "poller"
        copiedConfig: false,
        copiedLogs: false,
        searchQuery: "",
        logLevel: "ALL",
        mcpSearch: "",
        mcpFilter: "all", // "all" | "read" | "write"
        mcpExpandedTool: null
    },
    quickCreateOpen: false,

    mcpTools: [
        { name: "bisync_list_providers", type: "read", risk: "read-only", desc: "List supported cloud storage providers categorized by type with OAuth detection.", params: ["category", "force_refresh", "popular_only"] },
        { name: "bisync_list_profiles", type: "read", risk: "read-only", desc: "List configured bisync profiles with current status, schedules, and remote path mappings.", params: ["include_archived"] },
        { name: "bisync_get_status", type: "read", risk: "read-only", desc: "Compact system health summary, active syncs, daemon health, and canary status.", params: ["include_logs", "include_config", "include_metrics"] },
        { name: "bisync_get_logs", type: "read", risk: "read-only", desc: "Retrieve recent execution logs with severity level, profile ID, and run ID filtering.", params: ["level", "limit", "profile_id", "run_id"] },
        { name: "bisync_list_remotes", type: "read", risk: "read-only", desc: "List configured Rclone remotes, storage types, and connectivity health probe.", params: ["refresh"] },
        { name: "bisync_get_sync_history", type: "read", risk: "read-only", desc: "Query historical sync execution records and cryptographic audit trail ledger.", params: ["profile_id", "limit", "include_hashes"] },
        { name: "bisync_inspect_conflict_session", type: "read", risk: "read-only", desc: "Inspect pending conflict review sessions, file differences, and candidate resolutions.", params: ["session_id", "unresolved_only"] },
        { name: "bisync_explore_path", type: "read", risk: "read-only", desc: "Safely explore files and directories under configured profile or jail root.", params: ["remote", "path", "profile_id", "limit", "offset", "filter"] },
        { name: "bisync_preview_file", type: "read", risk: "read-only", desc: "Inspect text or image preview metadata for a file under remote or local path.", params: ["remote", "path", "profile_id"] },
        { name: "bisync_validate_flags", type: "read", risk: "read-only", desc: "Validate CLI flags and preview compiled Rclone bisync parameters and engine config.", params: ["flags", "profile_id"] },
        { name: "bisync_trigger_sync", type: "write", risk: "high", desc: "Trigger synchronization with blast-radius protection and dry-run safety.", params: ["profile_id", "dry_run", "resync", "force", "confirmation_token"] },
        { name: "bisync_resolve_review", type: "write", risk: "high", desc: "Stage or apply conflict resolution decisions on a review session.", params: ["decision", "session_id", "candidate_id", "execute", "confirmation_token"] },
        { name: "bisync_dedupe_by_hash", type: "write", risk: "high", desc: "Execute Rclone deduplication on a remote or profile path.", params: ["remote", "path", "mode", "by_hash", "confirmation_token"] },
        { name: "bisync_create_profile", type: "write", risk: "standard", desc: "Create a new bidirectional synchronization profile with path validation.", params: ["name", "remote", "local_path", "remote_path", "schedule", "conflict_mode", "max_delete"] },
        { name: "bisync_update_profile", type: "write", risk: "standard", desc: "Update configuration settings, schedules, or activation on an existing profile.", params: ["profile_id", "name", "remote", "local_path", "remote_path", "schedule", "conflict_mode", "max_delete", "enabled"] },
        { name: "bisync_delete_profile", type: "write", risk: "high", desc: "Permanently delete a sync profile with confirmation token protection.", params: ["profile_id", "confirmation_token"] },
        { name: "bisync_rollback_profile", type: "write", risk: "high", desc: "Roll back a sync profile to a previous version snapshot from configuration history.", params: ["profile_id", "version", "confirmation_token"] },
        { name: "bisync_create_remote", type: "write", risk: "standard", desc: "Provision a new Rclone remote target with optional sync profile bridge.", params: ["name", "type", "parameters", "create_profile", "profile_name", "local_path", "remote_subpath", "schedule"] },
        { name: "bisync_delete_remote", type: "write", risk: "high", desc: "Delete an existing Rclone remote with blast-radius safeguards.", params: ["name", "confirmation_token", "force"] },
        { name: "bisync_mkdir", type: "write", risk: "standard", desc: "Create a directory under local jail or remote workspace path.", params: ["remote", "path", "profile_id"] },
        { name: "bisync_copy_item", type: "write", risk: "standard", desc: "Copy a file or directory between remote and local storage targets.", params: ["src_remote", "src_path", "dst_remote", "dst_path", "is_dir", "profile_id"] },
        { name: "bisync_move_item", type: "write", risk: "standard", desc: "Move or rename a file or directory between remote and local storage targets.", params: ["src_remote", "src_path", "dst_remote", "dst_path", "is_dir", "profile_id"] },
        { name: "bisync_delete_item", type: "write", risk: "high", desc: "Safely delete a file or directory with canary protection and confirmation safeguards.", params: ["remote", "path", "is_dir", "profile_id", "confirmation_token"] },
        { name: "bisync_purge_dir", type: "write", risk: "high", desc: "Recursively delete a directory and all contents with high blast-radius safeguards.", params: ["remote", "path", "profile_id", "confirmation_token"] }
    ],

    toggleMcpTool(name) {
        if (this.systemHealthModal.mcpExpandedTool === name) {
            this.systemHealthModal.mcpExpandedTool = null;
        } else {
            this.systemHealthModal.mcpExpandedTool = name;
        }
    },

    get filteredMcpTools() {
        const q = (this.systemHealthModal.mcpSearch || "").toLowerCase().trim();
        const f = this.systemHealthModal.mcpFilter || "all";
        return (this.mcpTools || []).filter(t => {
            if (f === "read" && t.type !== "read") return false;
            if (f === "write" && t.type !== "write") return false;
            if (!q) return true;
            return t.name.toLowerCase().includes(q) || t.desc.toLowerCase().includes(q);
        });
    },

    openSystemHealthModal(tab = "daemon") {
        this.systemHealthModal.show = true;
        this.systemHealthModal.tab = tab;
        this.quickCreateOpen = false;

        // If daemon tab opened, ensure daemon logs are fetched
        if (tab === "daemon" && typeof this.fetchDaemonLogs === "function") {
            this.fetchDaemonLogs();
        }
    },

    closeSystemHealthModal() {
        this.systemHealthModal.show = false;
    },

    setSystemHealthTab(tab) {
        this.systemHealthModal.tab = tab;
        if (tab === "daemon" && typeof this.fetchDaemonLogs === "function") {
            this.fetchDaemonLogs();
        }
    },

    async triggerManualPoll() {
        if (typeof this.poll === "function") {
            await this.poll();
            if (typeof this.showToast === "function") {
                this.showToast("Telemetry metrics refreshed successfully", "success");
            }
        }
    },

    get isSystemHealthy() {
        return Boolean(this.telemetry && this.telemetry.daemon_health && this.telemetry.daemon_health.status === "healthy");
    },

    get systemHealthBadgeClass() {
        if (!this.isSystemHealthy) {
            return "border-rose-500/40 bg-rose-950/20 text-rose-300";
        }
        if (this.isPollingPaused) {
            return "border-amber-500/40 bg-amber-950/20 text-amber-300";
        }
        return "border-slate-800 bg-slate-900/90 text-slate-300 hover:border-slate-700";
    },

    get pollerActivityText() {
        if (this.telemetry && this.telemetry.is_syncing) {
            return "Syncing";
        }
        if (this.isPollingPaused) {
            return "Paused";
        }
        return "Idle";
    },

    get pollerActivityBadgeClass() {
        if (this.telemetry && this.telemetry.is_syncing) {
            return "bg-indigo-500/20 text-indigo-300 border border-indigo-500/30";
        }
        if (this.isPollingPaused) {
            return "bg-amber-500/20 text-amber-300 border border-amber-500/30";
        }
        return "bg-slate-800 text-slate-400 border border-slate-700/60";
    }
};


// Global Keyboard Shortcuts for Quick Create, Profile, and Remote Wizards
window.addEventListener("keydown", (e) => {
    const tag = (e.target && e.target.tagName) ? e.target.tagName.toLowerCase() : "";
    if (tag === "input" || tag === "textarea" || tag === "select" || (e.target && e.target.isContentEditable)) {
        return;
    }
    const rootEl = document.querySelector("[x-data]");
    if (!rootEl || !window.Alpine) return;
    const app = Alpine.$data(rootEl);
    if (!app) return;

    if ((e.key === "c" || e.key === "C") && !e.ctrlKey && !e.metaKey && !e.altKey) {
        app.quickCreateOpen = !app.quickCreateOpen;
    } else if ((e.key === "k" || e.key === "K") && (e.ctrlKey || e.metaKey)) {
        e.preventDefault();
        app.quickCreateOpen = !app.quickCreateOpen;
    } else if ((e.key === "n" || e.key === "N") && !e.ctrlKey && !e.metaKey && !e.altKey) {
        if (typeof app.startNewProfile === "function") {
            app.quickCreateOpen = false;
            app.startNewProfile();
        }
    } else if ((e.key === "r" || e.key === "R") && !e.ctrlKey && !e.metaKey && !e.altKey) {
        if (typeof app.openCreateRemoteWizard === "function") {
            app.quickCreateOpen = false;
            app.openCreateRemoteWizard();
        }
    }
});