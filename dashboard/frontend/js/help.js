/**
 * Interactive Diagnostic Manual & In-App User Guidance Module
 * Manages the slide-over drawer, curated safety policies, CLI flag catalog,
 * operational runbooks, and real-time self-diagnostic audit execution.
 */

window.HelpModule = {
    helpDrawer: {
        open: false,
        tab: "diagnostics", // 'diagnostics' | 'safety' | 'manual' | 'runbooks'
        search: "",
        diagnostics: null,
        loading: false,
        error: null,
        contextId: "global",
        contextTitle: "Fleet Overview",
        contextOnly: false,
    },

    detectActiveContext() {
        if (this.profileModal && this.profileModal.show) {
            return { id: "modal_profiles", title: "Profile Configuration", defaultTab: "manual" };
        }
        if (this.reviewState && this.reviewState.showModal) {
            return { id: "modal_review", title: "Conflict Review", defaultTab: "safety" };
        }
        if (this.globalSettingsModal && this.globalSettingsModal.show) {
            return { id: "modal_global_settings", title: "Global Settings", defaultTab: "safety" };
        }
        if (this.dedupeModal && this.dedupeModal.show) {
            return { id: "modal_dedupe", title: "Deduplication", defaultTab: "manual" };
        }
        if (this.showMcpModal) {
            return { id: "modal_mcp", title: "MCP Inspector", defaultTab: "diagnostics" };
        }
        if (this.showDaemonModal) {
            return { id: "modal_daemon", title: "Daemon Logs", defaultTab: "diagnostics" };
        }
        if (this.showDeletionsModal) {
            return { id: "modal_deletions", title: "Pending Deletions", defaultTab: "safety" };
        }

        switch (this.currentTab) {
            case "telemetry":
                return { id: "view_telemetry", title: "Engine & Telemetry", defaultTab: "diagnostics" };
            case "remotes":
                return { id: "view_remotes", title: "Storage Remotes", defaultTab: "runbooks" };
            case "explorer":
                return { id: "view_explorer", title: "File Explorer", defaultTab: "safety" };
            case "audit":
                return { id: "view_audit", title: "Audit Ledger", defaultTab: "safety" };
            case "fleet":
            default:
                return { id: "global", title: "Fleet Overview", defaultTab: "diagnostics" };
        }
    },

    openHelpDrawer(tab = null, topic = "") {
        let ctx = this.detectActiveContext();
        const validTabs = ["diagnostics", "safety", "manual", "runbooks"];
        const contextMap = {
            modal_profiles: { id: "modal_profiles", title: "Profile Configuration", defaultTab: "manual" },
            modal_review: { id: "modal_review", title: "Conflict Review", defaultTab: "safety" },
            modal_global_settings: { id: "modal_global_settings", title: "Global Settings", defaultTab: "safety" },
        };
        if (tab && contextMap[tab]) {
            ctx = contextMap[tab];
            tab = ctx.defaultTab;
        }
        this.helpDrawer.contextId = ctx.id;
        this.helpDrawer.contextTitle = ctx.title;
        this.helpDrawer.contextOnly = (ctx.id !== "global");
        this.helpDrawer.tab = (tab && validTabs.includes(tab)) ? tab : ctx.defaultTab;
        this.helpDrawer.search = topic || "";
        this.helpDrawer.open = true;

        if (this.helpDrawer.tab === "diagnostics" && !this.helpDrawer.diagnostics && !this.helpDrawer.loading) {
            this.fetchDiagnostics();
        }
    },

    toggleContextFilter() {
        this.helpDrawer.contextOnly = !this.helpDrawer.contextOnly;
    },

    closeHelpDrawer() {
        this.helpDrawer.open = false;
    },

    setHelpTab(tab) {
        this.helpDrawer.tab = tab;
        if (tab === "diagnostics" && !this.helpDrawer.diagnostics && !this.helpDrawer.loading) {
            this.fetchDiagnostics();
        }
    },

    isDiagnosticsLoading() {
        return Boolean(this.helpDrawer && this.helpDrawer.loading);
    },

    async fetchDiagnostics() {
        this.helpDrawer.loading = true;
        this.helpDrawer.error = null;
        try {
            const data = await window.API.get("/api/diagnostics");
            this.helpDrawer.diagnostics = data;
            if (data.status === "error") {
                this.showToast("System diagnostics reported issues", "error");
            } else if (data.status === "warning") {
                this.showToast("System diagnostics reported warnings", "warning");
            } else {
                this.showToast("All diagnostic checks passed", "success");
            }
        } catch (err) {
            this.helpDrawer.error = err.message || "Failed to load diagnostics";
            this.showToast("Diagnostic audit failed: " + this.helpDrawer.error, "error");
        } finally {
            this.helpDrawer.loading = false;
        }
    },

    async copyToClipboard(text, message = "Copied to clipboard") {
        try {
            await navigator.clipboard.writeText(text);
            this.showToast(message, "success");
        } catch (_) {
            const el = document.createElement("textarea");
            el.value = text;
            document.body.appendChild(el);
            el.select();
            document.execCommand("copy");
            document.body.removeChild(el);
            this.showToast(message, "success");
        }
    },

    filteredDiagnosticChecks() {
        const checks = this.helpDrawer.diagnostics?.checks || [];
        const q = (this.helpDrawer.search || "").trim().toLowerCase();
        let list = checks;
        if (this.helpDrawer.contextOnly && this.helpDrawer.contextId !== "global") {
            const ctxId = this.helpDrawer.contextId;
            list = checks.filter(c => {
                if (ctxId === "view_remotes" && c.category === "remotes") return true;
                if ((ctxId === "view_explorer" || ctxId === "modal_profiles") && (c.category === "storage" || c.category === "safety")) return true;
                if (ctxId === "view_telemetry" && (c.category === "daemon" || c.category === "system")) return true;
                if (ctxId === "modal_mcp" && c.category === "daemon") return true;
                return false;
            });
            if (list.length === 0) list = checks;
        }
        if (!q) return list;
        return list.filter(c =>
            (c.name && c.name.toLowerCase().includes(q)) ||
            (c.message && c.message.toLowerCase().includes(q)) ||
            (c.category && c.category.toLowerCase().includes(q)) ||
            (c.remedy && c.remedy.toLowerCase().includes(q))
        );
    },
    policiesList: [
        {
            id: "too_many_deletes",
            title: "TOO_MANY_DELETES Circuit Breaker",
            tag: "Circuit Breaker",
            contexts: ["modal_profiles", "modal_review", "modal_global_settings", "modal_deletions", "view_telemetry"],
            description: "Halts synchronization immediately when proposed deletion count or percentage exceeds configured safety thresholds. Prevents accidental mass file deletion due to empty mount points, corrupted drives, or misconfigured sync pairs.",
            action: "Inspect pending deletions in the Sync Review modal, or run dry-run with --verbose. If deletions are legitimate, raise the Max Delete threshold or approve them via Interactive Review."
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
            id: "canary_protection",
            title: "RCLONE_TEST Canary Validation",
            tag: "Canary Probe",
            contexts: ["modal_profiles", "modal_global_settings", "view_remotes", "view_explorer"],
            description: "Places a zero-byte token (RCLONE_TEST) on both endpoints. Bisync verifies this file before proceeding; if missing (e.g., remote unmounted, network drive disconnected), sync terminates immediately.",
            action: "Check that both endpoints are mounted and reachable. Ensure RCLONE_TEST is present on both sides, or run initial setup to create it."
        },
        {
            id: "flock_advisory_lock",
            title: "POSIX flock Concurrency Locking",
            tag: "Process Isolation",
            contexts: ["modal_profiles", "modal_global_settings", "view_telemetry", "modal_mcp"],
            description: "Acquires exclusive OS-level POSIX file locks (flock) before executing bisync jobs. Guarantees that multiple background schedulers, webhooks, or manual triggers cannot execute concurrently on the same profile.",
            action: "Wait for active run to finish. If process crashed, stale locks are reclaimed automatically after lock_stale_timeout_seconds (default 3600s)."
        },
        {
            id: "heuristic_conflict_scoring",
            title: "Multi-Analyzer Heuristic Conflict Engine",
            tag: "Conflict Engine",
            contexts: ["modal_review", "modal_profiles"],
            description: "Evaluates conflict candidates across 4 parallel dimensions: size divergence (30%), deletion volume (30%), modification volatility (20%), and path depth (20%) to produce safety risk scores.",
            action: "Review candidates in the Sync Review modal. Safe candidates can be auto-staged using Auto-Apply Recommendations. High-risk items require manual decision."
        }
    ],

    filteredPolicies() {
        const q = (this.helpDrawer.search || "").trim().toLowerCase();
        let list = this.policiesList;
        if (this.helpDrawer.contextOnly && this.helpDrawer.contextId !== "global") {
            const ctxId = this.helpDrawer.contextId;
            const contextual = this.policiesList.filter(p => p.contexts && p.contexts.includes(ctxId));
            if (contextual.length > 0) list = contextual;
        }
        if (!q) return list;
        return list.filter(p =>
            p.title.toLowerCase().includes(q) ||
            p.tag.toLowerCase().includes(q) ||
            p.description.toLowerCase().includes(q) ||
            p.action.toLowerCase().includes(q)
        );
    },
    flagsList: [
        {
            flag: "--resync",
            name: "Full Listing Cache Rebuild",
            category: "Recovery",
            contexts: ["modal_profiles", "modal_dedupe", "view_telemetry"],
            description: "Re-initializes the bisync cache by building fresh file listings from both endpoints without propagating deletions. Mandatory on initial run or after path/compare mode changes.",
            example: "rclone bisync local: remote: --resync --verbose"
        },
        {
            flag: "--dry-run",
            name: "Simulation / Trial Execution",
            category: "Safety",
            contexts: ["modal_profiles", "modal_review", "modal_global_settings", "view_telemetry"],
            description: "Simulates the synchronization cycle without modifying or deleting files on either endpoint. Generates a preview diff of all additions, modifications, and deletions.",
            example: "rclone bisync local: remote: --dry-run -v"
        },
        {
            flag: "--max-delete",
            name: "Deletion Threshold Ceiling",
            category: "Safety",
            contexts: ["modal_profiles", "modal_global_settings", "modal_review"],
            description: "Maximum number or percentage of files permitted to be deleted during a single sync. If exceeded, bisync aborts before performing any deletions.",
            example: "rclone bisync local: remote: --max-delete 50"
        },
        {
            flag: "--conflict-resolve",
            name: "Automatic Conflict Resolution",
            category: "Resolution",
            contexts: ["modal_profiles", "modal_global_settings", "modal_review"],
            description: "Strategy for resolving conflicting modifications: path1 (prefer local), path2 (prefer remote), newer (latest modtime), older, larger, or smaller.",
            example: "rclone bisync local: remote: --conflict-resolve newer"
        },
        {
            flag: "--compare",
            name: "File Comparison Criteria",
            category: "Performance",
            contexts: ["modal_profiles", "modal_global_settings"],
            description: "Comma-separated comparison attributes: size,modtime (fastest, standard), size,modtime,checksum (thorough hash check), or checksum only.",
            example: "rclone bisync local: remote: --compare size,modtime,checksum"
        },
        {
            flag: "--check-access",
            name: "Canary Access Verification",
            category: "Safety",
            contexts: ["modal_profiles", "modal_global_settings", "view_remotes"],
            description: "Verifies the presence of the canary file (default RCLONE_TEST) on both Path1 and Path2 before executing the synchronization cycle.",
            example: "rclone bisync local: remote: --check-access"
        },
        {
            flag: "--filters-file",
            name: "Include / Exclude Rule File",
            category: "Filtering",
            contexts: ["modal_profiles", "view_explorer"],
            description: "Loads file filtering rules (+ include, - exclude) from a specified text file. Useful for ignoring build artifacts, caches, and OS metadata files.",
            example: "rclone bisync local: remote: --filters-file /path/to/filters.txt"
        },
        {
            flag: "--slow-hash",
            name: "Slow Hash Provider Handling",
            category: "Performance",
            contexts: ["modal_profiles", "view_remotes"],
            description: "Allows checksum comparisons on cloud providers that compute hashes slowly or on-demand without timing out operations.",
            example: "rclone bisync local: remote: --compare checksum --slow-hash"
        }
    ],

    filteredFlags() {
        const q = (this.helpDrawer.search || "").trim().toLowerCase();
        let list = this.flagsList;
        if (this.helpDrawer.contextOnly && this.helpDrawer.contextId !== "global") {
            const ctxId = this.helpDrawer.contextId;
            const contextual = this.flagsList.filter(f => f.contexts && f.contexts.includes(ctxId));
            if (contextual.length > 0) list = contextual;
        }
        if (!q) return list;
        return list.filter(f =>
            f.flag.toLowerCase().includes(q) ||
            f.name.toLowerCase().includes(q) ||
            f.category.toLowerCase().includes(q) ||
            f.description.toLowerCase().includes(q)
        );
    },
    runbooksList: [
        {
            id: "too_many_deletes",
            title: "Resolving TOO_MANY_DELETES Circuit Breakers",
            category: "Circuit Breaker",
            contexts: ["modal_review", "modal_deletions", "view_telemetry"],
            summary: "When a sync cycle trips the deletion threshold, rclone bisync halts immediately to prevent mass data destruction. Follow these steps to triage and recover.",
            steps: [
                "Open the Sync Review modal from the Fleet view or the top banner to inspect proposed deletions.",
                "Verify whether the deletions originated from intentional user action or an unmounted/empty folder.",
                "If deletions are accidental, click Preserve All Files or select individual items to restore.",
                "If deletions are intended, click Accept All Deletions to authorize and stage the changes.",
                "Execute the review session or re-run the sync with adjusted --max-delete threshold."
            ],
            command: "rclone bisync local: remote: --dry-run -v"
        },
        {
            id: "oauth_expired",
            title: "Re-authenticating Cloud OAuth Tokens",
            category: "Authentication",
            contexts: ["view_remotes", "modal_auth_audit", "modal_oauth_migration"],
            summary: "Cloud providers (Google Drive, OneDrive, Dropbox, Box) periodically expire refresh tokens, resulting in HTTP 401/403 errors during sync.",
            steps: [
                "Navigate to the Remotes tab in the Bisync Hub dashboard.",
                "Locate the affected remote and click Re-authenticate or Auth.",
                "Follow the browser redirect to authorize Rclone Bisync with your cloud provider.",
                "Alternatively, run 'rclone config reconnect remote:' from your terminal.",
                "Verify the remote status shows OK in the Diagnostics panel."
            ],
            command: "rclone config reconnect myremote:"
        },
        {
            id: "stale_lock",
            title: "Clearing Stuck POSIX flock Locks",
            category: "Locks",
            contexts: ["view_telemetry", "modal_profiles", "modal_global_settings"],
            summary: "If the daemon or system shuts down ungracefully during an active sync, an advisory lock file (.flock) may remain on disk.",
            steps: [
                "Confirm in Fleet view that no active runner process is currently executing on the profile.",
                "Check system processes using 'ps aux | grep rclone' to verify the old process is terminated.",
                "Wait for lock_stale_timeout_seconds (default 1 hour) for automatic lease expiration, or click Clear Lock in Profile Settings.",
                "Alternatively, manually inspect and remove the .flock file in the profile directory."
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
            command: "rclone bisync local: remote: --resync -v"
        }
    ],

    filteredRunbooks() {
        const q = (this.helpDrawer.search || "").trim().toLowerCase();
        let list = this.runbooksList;
        if (this.helpDrawer.contextOnly && this.helpDrawer.contextId !== "global") {
            const ctxId = this.helpDrawer.contextId;
            const contextual = this.runbooksList.filter(r => r.contexts && r.contexts.includes(ctxId));
            if (contextual.length > 0) list = contextual;
        }
        if (!q) return list;
        return list.filter(r =>
            r.title.toLowerCase().includes(q) ||
            r.category.toLowerCase().includes(q) ||
            r.summary.toLowerCase().includes(q) ||
            (r.command && r.command.toLowerCase().includes(q))
        );
    }
};

// Global Keyboard Shortcuts (? / F1) for Help Drawer
window.addEventListener("keydown", (e) => {
    const tag = (e.target && e.target.tagName) ? e.target.tagName.toLowerCase() : "";
    if (tag === "input" || tag === "textarea" || tag === "select" || (e.target && e.target.isContentEditable)) {
        return;
    }
    if (e.key === "?" || e.key === "F1") {
        e.preventDefault();
        if (typeof window.openHelpDrawer === "function") {
            window.openHelpDrawer();
        }
    }
});



