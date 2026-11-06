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
    },

    openHelpDrawer(tab = "diagnostics", topic = "") {
        this.helpDrawer.tab = tab;
        this.helpDrawer.search = topic || "";
        this.helpDrawer.open = true;

        if (tab === "diagnostics" && !this.helpDrawer.diagnostics && !this.helpDrawer.loading) {
            this.fetchDiagnostics();
        }
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
        if (!q) return checks;
        return checks.filter(c =>
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
            description: "Halts synchronization immediately when proposed deletion count or percentage exceeds configured safety thresholds. Prevents accidental mass file deletion due to empty mount points, corrupted drives, or misconfigured sync pairs.",
            action: "Inspect pending deletions in the Sync Review modal, or run dry-run with --verbose. If deletions are legitimate, raise the Max Delete threshold or approve them via Interactive Review."
        },
        {
            id: "mandatory_subdir",
            title: "MANDATORY_SUBDIR Isolation",
            tag: "Path Confinement",
            description: "Direct synchronization of root directories (e.g. / or C:\\) is prohibited by rclone bisync to prevent accidental total system wipes. Every synchronization pair must target a designated subfolder.",
            action: "Specify an explicit subfolder path for both local and remote endpoints (e.g., /home/user/data or remote:backup/subfolder)."
        },
        {
            id: "canary_protection",
            title: "RCLONE_TEST Canary Validation",
            tag: "Canary Probe",
            description: "Places a zero-byte token (RCLONE_TEST) on both endpoints. Bisync verifies this file before proceeding; if missing (e.g., remote unmounted, network drive disconnected), sync terminates immediately.",
            action: "Check that both endpoints are mounted and reachable. Ensure RCLONE_TEST is present on both sides, or run initial setup to create it."
        },
        {
            id: "flock_advisory_lock",
            title: "POSIX flock Concurrency Locking",
            tag: "Process Isolation",
            description: "Acquires exclusive OS-level POSIX file locks (flock) before executing bisync jobs. Guarantees that multiple background schedulers, webhooks, or manual triggers cannot execute concurrently on the same profile.",
            action: "Wait for active run to finish. If process crashed, stale locks are reclaimed automatically after lock_stale_timeout_seconds (default 3600s)."
        },
        {
            id: "heuristic_conflict_scoring",
            title: "Multi-Analyzer Heuristic Conflict Engine",
            tag: "Conflict Engine",
            description: "Evaluates conflict candidates across 4 parallel dimensions: size divergence (30%), deletion volume (30%), modification volatility (20%), and path depth (20%) to produce safety risk scores.",
            action: "Review candidates in the Sync Review modal. Safe candidates can be auto-staged using Auto-Apply Recommendations. High-risk items require manual decision."
        }
    ],

    filteredPolicies() {
        const q = (this.helpDrawer.search || "").trim().toLowerCase();
        if (!q) return this.policiesList;
        return this.policiesList.filter(p =>
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
            description: "Re-initializes the bisync cache by building fresh file listings from both endpoints without propagating deletions. Mandatory on initial run or after path/compare mode changes.",
            example: "rclone bisync local: remote: --resync --verbose"
        },
        {
            flag: "--dry-run",
            name: "Simulation / Trial Execution",
            category: "Safety",
            description: "Simulates the synchronization cycle without modifying or deleting files on either endpoint. Generates a preview diff of all additions, modifications, and deletions.",
            example: "rclone bisync local: remote: --dry-run -v"
        },
        {
            flag: "--max-delete",
            name: "Deletion Threshold Ceiling",
            category: "Safety",
            description: "Maximum number or percentage of files permitted to be deleted during a single sync. If exceeded, bisync aborts before performing any deletions.",
            example: "rclone bisync local: remote: --max-delete 50"
        },
        {
            flag: "--conflict-resolve",
            name: "Automatic Conflict Resolution",
            category: "Resolution",
            description: "Strategy for resolving conflicting modifications: path1 (prefer local), path2 (prefer remote), newer (latest modtime), older, larger, or smaller.",
            example: "rclone bisync local: remote: --conflict-resolve newer"
        },
        {
            flag: "--compare",
            name: "File Comparison Criteria",
            category: "Performance",
            description: "Comma-separated comparison attributes: size,modtime (fastest, standard), size,modtime,checksum (thorough hash check), or checksum only.",
            example: "rclone bisync local: remote: --compare size,modtime,checksum"
        },
        {
            flag: "--check-access",
            name: "Canary Access Verification",
            category: "Safety",
            description: "Verifies the presence of the canary file (default RCLONE_TEST) on both Path1 and Path2 before executing the synchronization cycle.",
            example: "rclone bisync local: remote: --check-access"
        },
        {
            flag: "--filters-file",
            name: "Include / Exclude Rule File",
            category: "Filtering",
            description: "Loads file filtering rules (+ include, - exclude) from a specified text file. Useful for ignoring build artifacts, caches, and OS metadata files.",
            example: "rclone bisync local: remote: --filters-file /path/to/filters.txt"
        },
        {
            flag: "--slow-hash",
            name: "Slow Hash Provider Handling",
            category: "Performance",
            description: "Allows checksum comparisons on cloud providers that compute hashes slowly or on-demand without timing out operations.",
            example: "rclone bisync local: remote: --compare checksum --slow-hash"
        }
    ],

    filteredFlags() {
        const q = (this.helpDrawer.search || "").trim().toLowerCase();
        if (!q) return this.flagsList;
        return this.flagsList.filter(f =>
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
        if (!q) return this.runbooksList;
        return this.runbooksList.filter(r =>
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



