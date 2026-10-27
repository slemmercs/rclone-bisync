/**
 * Rclone Bisync Dashboard — Main Application Orchestrator
 * Modularized Alpine.js architecture. Zero build step, offline-ready.
 */

document.addEventListener("alpine:init", () => {
    function composeDashboard(base, ...modules) {
        for (const mod of modules) {
            if (!mod) continue;
            Object.defineProperties(base, Object.getOwnPropertyDescriptors(mod));
        }
        return base;
    }

    Alpine.data("dashboard", () => composeDashboard({
        currentTab: "telemetry", // 'telemetry' | 'remotes' | 'explorer'
        remotesData: {
            remotes: [],
            summary: {},
            count: 0,
            loading: false,
            error: null,
            last_checked: null,
            authSummary: { total_remotes: 0, oauth_remotes: 0, connected: 0, migration_required: 0, authorization_required: 0, configuration_required: 0, errors: 0 },
            reconnecting: {},
            validating: {},
            validated: {}
        },
        migrationModal: {
            show: false,
            remote: "",
            type: "",
            typeLabel: "",
            clientId: "",
            clientSecret: "",
            showSecret: false,
            loading: false,
            error: null,
            success: null
        },
        reconnectModal: {
            show: false,
            remote: "",
            loading: false,
            authUrl: null,
            msg: null,
            error: null
        },
        auditModal: {
            show: false,
            loading: false,
            events: []
        },
        telemetry: {
            daemon_health: { status: "evaluating", response_time_ms: 0 },
            notification_state: {},
            metrics: [],
            bisync_logs: [],
            daemon_logs: [],
            config: {},
            is_syncing: false,
            transfer_stats: null
        },
        currentLogLimit: 40,
        maxLogLimit: 300,
        telemetryLogFilter: "all", // 'all' | 'sync' | 'audit'
        actionLoading: null,
        configExpanded: false,

        showDeletionsModal: false,
        deletionsData: { total_count: 0, source: "", target_action: "", files: [] },
        deletionsSearch: "",

        // Correlated Daemon Log Modal State (Branch 6)
        showDaemonModal: false,
        daemonModalLoading: false,
        daemonModalRun: null,
        daemonModalLogs: [],
        daemonModalSearch: "",
        daemonModalLevelFilter: "all",
        daemonModalCopied: false,

        // Dual-Pane File Explorer State
        explorer: {
            left: {
                target: "local",
                path: "",
                items: [],
                loading: false,
                error: null,
                hasCanary: false,
                totalSize: 0,
                itemCount: 0,
                copied: false,
                search: "",
                sortBy: "name",
                sortAsc: true,
                foldersFirst: true
            },
            right: {
                target: "gdrive",
                path: "",
                items: [],
                loading: false,
                error: null,
                hasCanary: false,
                totalSize: 0,
                itemCount: 0,
                copied: false,
                search: "",
                sortBy: "name",
                sortAsc: true,
                foldersFirst: true
            },
            draggedItem: null,
            draggedSourcePane: null,
            isDragOverLeft: false,
            isDragOverRight: false,
            isSwapped: false
        },
        toast: {
            show: false,
            message: "",
            type: "info",
            timeout: null
        },

        deleteConfirmMode: localStorage.getItem("rclone_delete_confirm_mode") || "double",
        deleteModal: {
            show: false,
            step: 1,
            paneKey: null,
            item: null,
            deleting: false
        },

        // Reusable Confirmation Modal State
        confirmModal: {
            show: false,
            title: "",
            message: "",
            confirmText: "Confirm",
            confirmClass: "bg-indigo-600 hover:bg-indigo-500",
            icon: "warning",
            onConfirm: null
        },

        askConfirmation({ title, message, confirmText = "Confirm", confirmClass = "bg-indigo-600 hover:bg-indigo-500", icon = "warning", onConfirm }) {
            this.confirmModal.title = title;
            this.confirmModal.message = message;
            this.confirmModal.confirmText = confirmText;
            this.confirmModal.confirmClass = confirmClass;
            this.confirmModal.icon = icon;
            this.confirmModal.onConfirm = onConfirm;
            this.confirmModal.show = true;
        },

        handleConfirmModal() {
            const fn = this.confirmModal.onConfirm;
            this.confirmModal.show = false;
            this.confirmModal.onConfirm = null;
            if (typeof fn === "function") {
                fn();
            }
        },

        cancelConfirmModal() {
            this.confirmModal.show = false;
            this.confirmModal.onConfirm = null;
        },

        // Interactive Sync Review State
        reviewState: {
            active: false,
            loading: false,
            error: null,
            session: null,
            executing: false,
            rollingBack: false,
            dismissing: false,
            filter: "all", // 'all' | 'deletions' | 'conflicts'
            statusFilter: "all", // 'all' | 'pending' | 'decided'
            search: "",
            showModal: false,
            lastResult: null
        },

        showToast(message, type = "info") {
            if (!this.toast) return;
            if (this.toast.timeout) clearTimeout(this.toast.timeout);
            this.toast.message = message;
            this.toast.type = type;
            this.toast.show = true;
            this.toast.timeout = setTimeout(() => {
                if (this.toast) this.toast.show = false;
            }, 4500);
        },

        isEngineSyncing() {
            if (this.telemetry && (this.telemetry.is_syncing || this.telemetry.execution_status === "SYNCING")) {
                return true;
            }
            if (this.activeProfile && this.activeProfile.state && this.activeProfile.state.status === "running") {
                return true;
            }
            if (typeof this.isProfileSyncing === "function" && this.activeProfile && this.isProfileSyncing(this.activeProfile)) {
                return true;
            }
            return false;
        },

        getEngineStatus() {
            return this.isEngineSyncing() ? "SYNCING" : (this.telemetry?.execution_status || "IDLE");
        },

        get daemonStatusText() {
            const status = this.telemetry.daemon_health?.status;
            return status === "healthy" ? "ONLINE" : "OFFLINE";
        },

        get daemonStatusClass() {
            const status = this.telemetry.daemon_health?.status;
            return status === "healthy" ? "text-emerald-400 tracking-wide" : "text-rose-500 tracking-wide";
        },

        get daemonLatencyText() {
            const lat = this.telemetry.daemon_health?.response_time_ms;
            return typeof lat === "number" ? lat.toFixed(2) : "--";
        },

        get executionStatusText() {
            if (this.isEngineSyncing()) return "SYNC IN PROGRESS";
            const notif = this.telemetry.notification_state || {};
            const successTs = notif.last_success_timestamp || 0;
            const errorTs = notif.last_error_timestamp || 0;
            if (errorTs > successTs && notif.last_error_type) return `FAULT (${notif.last_error_type})`;
            if (successTs > 0) return "OPERATIONAL";
            return "UNKNOWN";
        },

        get executionStatusClass() {
            if (this.isEngineSyncing()) return "text-amber-400 font-bold tracking-wide";
            const notif = this.telemetry.notification_state || {};
            const successTs = notif.last_success_timestamp || 0;
            const errorTs = notif.last_error_timestamp || 0;
            if (errorTs > successTs && notif.last_error_type) return "text-rose-400 text-lg";
            if (successTs > 0) return "text-emerald-400";
            return "text-slate-400";
        },

        get lastUpdatedText() {
            const notif = this.telemetry.notification_state || {};
            const ts = Math.max(notif.last_success_timestamp || 0, notif.last_error_timestamp || 0);
            if (!ts) return "--";
            return new Date(ts * 1000).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit", hour12: false });
        },

        get transferStats() {
            const stats = this.telemetry.transfer_stats || {};
            const currentFile = (stats.transferring && stats.transferring.length > 0)
                ? stats.transferring[0].name
                : (this.telemetry.is_syncing ? "Scanning differences & checking access..." : "Idle");
            let etaStr = "--";
            if (typeof stats.eta === "number") {
                const mins = Math.floor(stats.eta / 60);
                const secs = stats.eta % 60;
                etaStr = `${mins}m ${secs}s`;
            }
            return {
                speed_str: stats.speed_str || "0 B/s",
                eta_str: etaStr,
                percentage: typeof stats.percentage === "number" ? stats.percentage : 0,
                current_file: currentFile
            };
        },

        get filteredConfigEntries() {
            const cfg = this.telemetry.config || {};
            const exclude = new Set(["LOCAL_PATH", "REMOTE_PATH", "BASE_DIR", "LOCALPATH", "REMOTEPATH", "BASEDIR"]);
            return Object.entries(cfg).filter(([k]) => {
                if (exclude.has(k)) return false;
                // Exclude flat uppercase aliases when snake_case version is present
                const snake = k.replace(/([a-z])([A-Z])/g, '$1_$2').toUpperCase();
                if (!k.includes("_") && cfg[snake] !== undefined && snake !== k) return false;
                return true;
            });
        },

        formatConfig(key, val) {
            return formatConfigValue(key, val);
        },

        // Lifecycle & Navigation
        async initDashboard() {
            initPerformanceChart();
            await this.poll();
            if (typeof this.fetchProfiles === "function") {
                this.fetchProfiles();
            }
            this.fetchRemotes(false);
            this.fetchReviewSession(false);
            setInterval(() => this.poll(), 3000);
            this.setupLogScroll("log-bisync");
        },

        async poll() {
            try {
                const data = await window.API.get("/api/telemetry?limit=" + this.currentLogLimit);
                this.telemetry = data;
                updatePerformanceChart(data.metrics || []);
                this.fetchReviewSession(false);
                if (typeof this.fetchProfiles === "function") {
                    this.fetchProfiles();
                }
            } catch (err) {
                console.error("Telemetry poll failed:", err);
            }
        },

        async triggerAction(actionStr) {
            this.actionLoading = actionStr;
            try {
                const payload = { action: actionStr };
                if (this.profilesData && this.profilesData.activeId) {
                    payload.profile_id = this.profilesData.activeId;
                }
                const data = await window.API.post("/api/action", payload);
                if (data.status === "error") {
                    const errDetail = data.msg || data.error || "Request failed";
                    this.showToast("Action Error: " + errDetail, "error");
                } else {
                    this.showToast(data.msg || `Action "${actionStr}" completed successfully.`, "success");
                }
                await this.poll();
            } catch (err) {
                this.showToast("Action failed: " + err.message, "error");
            } finally {
                setTimeout(() => { this.actionLoading = null; }, 1200);
            }
        },

        switchTab(tab) {
            this.currentTab = tab;
            if (tab === "remotes") {
                this.fetchRemotes(true);
            } else if (tab === "explorer") {
                if (typeof this.initExplorer === "function") {
                    this.initExplorer();
                }
            } else if (tab === "audit") {
                if (typeof this.initAudit === "function") {
                    this.initAudit();
                }
            }
        },
    },
        window.ProfilesModule,
        window.RemotesModule,
        window.ExplorerModule,
        window.ExplorerActionsModule,
        window.ReviewModule,
        window.LogsModule,
        window.AuditModule
    ));
});
