/**
 * Telemetry & Correlated Daemon Logs Module for Bisync Hub Dashboard
 * Handles bisync/daemon log filtering, correlated run log modals,
 * and deletion modal state interactions.
 */

window.LogsModule = {
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
    }
};
