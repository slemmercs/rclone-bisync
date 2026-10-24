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


        get parsedBisyncLogs() {
            return (this.telemetry.bisync_logs || []).map(parseJsonLogLine).filter(Boolean);
        },

        get filteredBisyncLogs() {
            const logs = this.parsedBisyncLogs;
            if (this.telemetryLogFilter === "sync") {
                return logs.filter(l => l.type !== "audit");
            }
            if (this.telemetryLogFilter === "audit") {
                return logs.filter(l => l.type === "audit");
            }
            return logs;
        },

        get parsedDaemonLogs() {
            return unwrapAndParseLogLines(this.telemetry.daemon_logs || []);
        },

        get filteredDaemonModalLogs() {
            let logs = this.daemonModalLogs || [];
            const q = (this.daemonModalSearch || "").toLowerCase().trim();
            const filter = this.daemonModalLevelFilter;

            if (filter !== "all") {
                logs = logs.filter(l => {
                    const lvl = (l.level || "").toLowerCase().trim();
                    if (filter === "error") return lvl === "error";
                    if (filter === "warning") return lvl === "warning" || lvl === "warn";
                    if (filter === "info") return lvl === "info" || lvl === "notice";
                    return true;
                });
            }

            if (q) {
                logs = logs.filter(l => (l.msg || "").toLowerCase().includes(q) || (l.object || "").toLowerCase().includes(q) || (l.source || "").toLowerCase().includes(q));
            }
            return logs;
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

        get reviewCandidates() {
            const sess = this.reviewState?.session;
            if (!sess) return [];
            const dels = (sess.deletions || []).map(d => ({
                ...d,
                type: "deletion",
                candidateSide: d.side || "Path1",
                path: d.path
            }));
            const cnfs = (sess.conflicts || []).map(c => ({
                ...c,
                type: "conflict",
                candidateSide: "Both",
                path: c.path
            }));
            return [...dels, ...cnfs];
        },

        get filteredReviewCandidates() {
            let list = this.reviewCandidates;
            const q = (this.reviewState.search || "").toLowerCase().trim();
            const typeFilter = this.reviewState.filter;
            const statusFilter = this.reviewState.statusFilter;

            if (typeFilter === "deletions") {
                list = list.filter(c => c.type === "deletion");
            } else if (typeFilter === "conflicts") {
                list = list.filter(c => c.type === "conflict");
            }

            if (statusFilter === "pending") {
                list = list.filter(c => !c.decision || c.decision === "pending");
            } else if (statusFilter === "decided") {
                list = list.filter(c => c.decision && c.decision !== "pending");
            }

            if (q) {
                list = list.filter(c => (c.path || "").toLowerCase().includes(q) || (c.reason || "").toLowerCase().includes(q));
            }

            return list;
        },

        get reviewStats() {
            const all = this.reviewCandidates;
            const dels = all.filter(c => c.type === "deletion");
            const cnfs = all.filter(c => c.type === "conflict");
            const decided = all.filter(c => c.decision && c.decision !== "pending");
            const pending = all.filter(c => !c.decision || c.decision === "pending");

            return {
                total: all.length,
                deletions: dels.length,
                conflicts: cnfs.length,
                decided: decided.length,
                pending: pending.length
            };
        },

        get hasPendingReviewDecisions() {
            return this.reviewStats.pending > 0;
        },

        get canApplyReview() {
            const s = this.reviewState;
            if (!s.session || s.executing || s.rollingBack) return false;
            return this.reviewStats.decided > 0 || s.session.status === "STAGED";
        },

        get canRollbackReview() {
            const sess = this.reviewState.session;
            if (!sess || this.reviewState.executing || this.reviewState.rollingBack) return false;
            return sess.status === "COMPLETED" || sess.status === "APPLIED" || (sess.actions && sess.actions.length > 0);
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

        // Module Compositions (Remotes, Dual-Pane File Explorer, Interactive Review & Audit Ledger)
        ...(window.RemotesModule || {}),
        ...(window.ExplorerModule || {}),
        ...(window.ReviewModule || {}),
        ...(window.AuditModule || {}),

        // Lifecycle & Navigation
        async initDashboard() {
            initPerformanceChart();
            await this.poll();
            this.fetchRemotes(false);
            this.fetchReviewSession(false);
            setInterval(() => this.poll(), 3000);
            this.setupLogScroll("log-bisync");
        },

        async poll() {
            try {
                const res = await fetch("/api/telemetry?limit=" + this.currentLogLimit);
                if (!res.ok) throw new Error("HTTP error: " + res.status);
                const data = await res.json();
                this.telemetry = data;
                updatePerformanceChart(data.metrics || []);
                this.fetchReviewSession(false);
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

        async openDaemonModal(entry) {
            this.showDaemonModal = true;
            this.daemonModalLoading = true;
            this.daemonModalRun = entry || null;
            this.daemonModalLogs = [];
            this.daemonModalSearch = "";
            this.daemonModalLevelFilter = "all";
            this.daemonModalCopied = false;

            let url = "/api/logs/daemon?limit=500";
            if (entry) {
                if (entry.run_id) url += `&run_id=${encodeURIComponent(entry.run_id)}`;
                if (entry.job_id) url += `&job_id=${encodeURIComponent(entry.job_id)}`;
                if (entry.start_time) url += `&start_time=${encodeURIComponent(entry.start_time)}`;
                if (entry.end_time) url += `&end_time=${encodeURIComponent(entry.end_time)}`;
            }

            try {
                const res = await fetch(url);
                const data = await res.json();
                if (data && Array.isArray(data.logs)) {
                    this.daemonModalLogs = unwrapAndParseLogLines(data.logs);
                } else {
                    this.daemonModalLogs = [];
                }
            } catch (err) {
                console.error("Failed to fetch daemon logs:", err);
                this.daemonModalLogs = [];
            } finally {
                this.daemonModalLoading = false;
            }
        },

        closeDaemonModal() {
            this.showDaemonModal = false;
            this.daemonModalRun = null;
            this.daemonModalLogs = [];
        },

        copyDaemonLogs() {
            const text = this.daemonModalLogs.map(l => l.raw || `[${l.time}] ${l.level} ${l.msg}`).join("\n");
            navigator.clipboard.writeText(text).then(() => {
                this.daemonModalCopied = true;
                setTimeout(() => { this.daemonModalCopied = false; }, 2000);
            });
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
            } else if (tab === "audit") {
                if (typeof this.initAudit === "function") {
                    this.initAudit();
                }
            }
        },
    }));
});
