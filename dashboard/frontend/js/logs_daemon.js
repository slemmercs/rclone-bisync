/**
 * Rclone Bisync Dashboard — Correlated Daemon Logs Module
 * Handles fetching, filtering, copying, and exporting correlated daemon logs.
 */

window.LogsDaemonModule = {
    get parsedDaemonLogs() {
        return typeof unwrapAndParseLogLines === "function" ? unwrapAndParseLogLines(this.telemetry?.daemon_logs || []) : [];
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

    get clusteredDaemonModalLogs() {
        const clusterFn = typeof clusterLogEntries === "function" ? clusterLogEntries : window.clusterLogEntries;
        if (typeof clusterFn === "function") {
            return clusterFn(this.filteredDaemonModalLogs, this.foldRepetitive, this.expandedClusters);
        }
        return this.filteredDaemonModalLogs.map((l, i) => ({ id: "d-" + i, entry: l, count: 1, isCluster: false, entries: [l] }));
    },

    get daemonModalCounts() {
        const logs = this.daemonModalLogs || [];
        let errors = 0, warnings = 0, info = 0;
        for (const l of logs) {
            const lvl = (l.level || "").toLowerCase().trim();
            if (lvl === "error") errors++;
            else if (lvl === "warning" || lvl === "warn") warnings++;
            else if (lvl === "info" || lvl === "notice") info++;
        }
        return { total: logs.length, errors, warnings, info };
    },

    async fetchDaemonLogs(entry = null) {
        this.daemonModalLoading = true;
        this.daemonModalRun = entry || null;
        this.daemonModalError = null;

        let url = "/api/logs/daemon?limit=500";
        if (entry) {
            ["run_id", "job_id", "start_time", "end_time"].forEach(k => {
                if (entry[k]) url += `&${k}=${encodeURIComponent(entry[k])}`;
            });
        }

        try {
            const data = await window.API.get(url);
            if (data && data.error) this.daemonModalError = data.error;
            this.daemonModalLogs = (data && Array.isArray(data.logs)) ? unwrapAndParseLogLines(data.logs) : [];
        } catch (err) {
            console.error("Failed to fetch daemon logs:", err);
            const msg = err && err.message ? err.message : "Failed to connect to daemon log endpoint";
            this.daemonModalError = `Unable to retrieve daemon logs: ${msg}`;
            this.daemonModalLogs = [];
        } finally {
            this.daemonModalLoading = false;
        }
    },

    async openDaemonModal(entry) {
        this.showDaemonModal = true;
        this.daemonModalSearch = "";
        this.daemonModalLevelFilter = "all";
        this.daemonModalCopied = this.daemonModalCopiedJSON = false;
        await this.fetchDaemonLogs(entry);
    },

    closeDaemonModal() {
        this.showDaemonModal = false;
        this.daemonModalRun = this.daemonModalError = null;
        this.daemonModalLogs = [];
    },

    copyDaemonLogs() {
        const text = this.daemonModalLogs.map(l => l.raw || `[${l.time}] ${l.level} ${l.msg}`).join("\n");
        navigator.clipboard.writeText(text).then(() => {
            this.daemonModalCopied = true;
            setTimeout(() => this.daemonModalCopied = false, 2000);
        });
    },

    copyDaemonLogsJSON() {
        const text = JSON.stringify(this.daemonModalLogs, null, 2);
        navigator.clipboard.writeText(text).then(() => {
            this.daemonModalCopiedJSON = true;
            setTimeout(() => this.daemonModalCopiedJSON = false, 2000);
        });
    },

    exportDaemonLogsJSON() {
        const dataStr = "data:text/json;charset=utf-8," + encodeURIComponent(JSON.stringify(this.daemonModalLogs, null, 2));
        const downloadAnchor = document.createElement("a");
        downloadAnchor.setAttribute("href", dataStr);
        const filename = this.daemonModalRun?.run_id ? `daemon-${this.daemonModalRun.run_id}.json` : "daemon-logs.json";
        downloadAnchor.setAttribute("download", filename);
        document.body.appendChild(downloadAnchor);
        downloadAnchor.click();
        downloadAnchor.remove();
    },

    jumpToDaemonRun(runId, jobId = null) {
        const entry = { run_id: runId, job_id: jobId };
        this.openDaemonModal(entry);
    }
};
