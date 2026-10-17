/**
 * Rclone Bisync Dashboard — Main Application Orchestrator
 * Modularized Alpine.js architecture. Zero build step, offline-ready.
 */

document.addEventListener("alpine:init", () => {
    Alpine.data("dashboard", () => ({
        currentTab: "telemetry", // 'telemetry' | 'remotes' | 'explorer'
        remotesData: {
            remotes: [],
            summary: {},
            count: 0,
            loading: false,
            error: null,
            last_checked: null
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
        actionLoading: null,
        configExpanded: false,

        showDeletionsModal: false,
        deletionsData: { total_count: 0, source: "", target_action: "", files: [] },
        deletionsSearch: "",

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


        get parsedBisyncLogs() {
            return (this.telemetry.bisync_logs || []).map(parseJsonLogLine).filter(Boolean);
        },

        get parsedDaemonLogs() {
            return (this.telemetry.daemon_logs || []).map(parseJsonLogLine).filter(Boolean);
        },

        get filteredDeletions() {
            const files = this.deletionsData.files || [];
            const q = (this.deletionsSearch || "").toLowerCase().trim();
            if (!q) return files;
            return files.filter(f => f.toLowerCase().includes(q));
        },

        get isTooManyDeletesActive() {
            const notif = this.telemetry.notification_state || {};
            const lastErr = notif.last_error_type;
            const lastErrTs = notif.last_error_timestamp || 0;
            const lastSuccessTs = notif.last_success_timestamp || 0;
            const metrics = this.telemetry.metrics || [];
            const lastMetric = metrics.length > 0 ? metrics[metrics.length - 1] : null;

            return (
                (lastMetric && !lastMetric.success && lastMetric.error_type === "TOO_MANY_DELETES") ||
                (lastErr === "TOO_MANY_DELETES" && lastErrTs > lastSuccessTs)
            );
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
            if (this.telemetry.is_syncing) return "SYNC IN PROGRESS";
            const notif = this.telemetry.notification_state || {};
            const successTs = notif.last_success_timestamp || 0;
            const errorTs = notif.last_error_timestamp || 0;
            if (errorTs > successTs && notif.last_error_type) return `FAULT (${notif.last_error_type})`;
            if (successTs > 0) return "OPERATIONAL";
            return "UNKNOWN";
        },

        get executionStatusClass() {
            if (this.telemetry.is_syncing) return "text-amber-400 font-bold tracking-wide";
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
            const exclude = new Set(["LOCAL_PATH", "REMOTE_PATH", "BASE_DIR"]);
            return Object.entries(cfg).filter(([k]) => !exclude.has(k));
        },

        formatConfig(key, val) {
            return formatConfigValue(key, val);
        },

        // Module Compositions (Remotes & Dual-Pane File Explorer)
        ...(window.RemotesModule || {}),
        ...(window.ExplorerModule || {}),

        // Lifecycle & Navigation
        async initDashboard() {
            initPerformanceChart();
            await this.poll();
            this.fetchRemotes(false);
            setInterval(() => this.poll(), 3000);
            this.setupLogScroll("log-bisync");
            this.setupLogScroll("log-daemon");
        },

        async poll() {
            try {
                const res = await fetch("/api/telemetry?limit=" + this.currentLogLimit);
                if (!res.ok) throw new Error("HTTP error: " + res.status);
                const data = await res.json();
                this.telemetry = data;
                updatePerformanceChart(data.metrics || []);
            } catch (err) {
                console.error("Telemetry poll failed:", err);
            }
        },

        async triggerAction(actionStr) {
            this.actionLoading = actionStr;
            try {
                const res = await fetch("/api/action", {
                    method: "POST",
                    headers: {
                        "Content-Type": "application/json",
                        "X-Dashboard-Request": "1"
                    },
                    body: JSON.stringify({ action: actionStr })
                });
                const data = await res.json();
                if (!res.ok || data.status === "error") {
                    alert("Action Error: " + (data.msg || "Request failed"));
                }
                await this.poll();
            } catch (err) {
                alert("Network error: " + err.message);
            } finally {
                setTimeout(() => { this.actionLoading = null; }, 1200);
            }
        },

        async openDeletionsModal() {
            this.deletionsSearch = "";
            this.showDeletionsModal = true;
            this.deletionsData = { total_count: 0, source: "Scanning...", target_action: "Reading daemon logs...", files: [] };
            try {
                const res = await fetch("/api/deletions/pending");
                const data = await res.json();
                this.deletionsData = data;
            } catch (err) {
                this.deletionsData = { total_count: 0, source: "Error", target_action: err.message, files: [] };
            }
        },

        confirmResync() {
            if (confirm("⚠ FORCE RESYNC ⚠\n\nAre you sure you want to rebuild baseline sync states?\nThis will take significantly longer.")) {
                this.triggerAction("force_resync");
            }
        },

        setupLogScroll(containerId) {
            const container = document.getElementById(containerId);
            if (!container) return;
            container.addEventListener("scroll", () => {
                if (container.scrollTop <= 40 && this.currentLogLimit < this.maxLogLimit) {
                    const oldHeight = container.scrollHeight;
                    this.currentLogLimit += 60;
                    this.poll().then(() => {
                        this.$nextTick(() => {
                            container.scrollTop = container.scrollHeight - oldHeight;
                        });
                    });
                }
            });
        },

        switchTab(tab) {
            this.currentTab = tab;
            if (tab === "remotes") {
                this.fetchRemotes(true);
            } else if (tab === "explorer") {
                if (typeof this.initExplorer === "function") {
                    this.initExplorer();
                }
            }
        },
    }));
});
