/**
 * Telemetry & Correlated Daemon Logs Module for Bisync Hub Dashboard
 * Handles bisync/daemon log filtering, 50-line on-demand pagination,
 * inline log deduplication/folding, correlated run modals, and deletion state.
 */

function normalizeLogMessage(msg) {
    if (!msg) return "";
    return msg
        .replace(/\[cid=[a-zA-Z0-9_-]+\]/g, "")
        .replace(/\b\d+(\.\d+)?\s*(s|ms|m|h|B|KB|MB|GB|TB|PB)\b/gi, "")
        .replace(/\b\d{4}-\d{2}-\d{2}[T\s]\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:?\d{2})?\b/g, "")
        .replace(/\b\d+\s+(transferred|checked|deleted|matching|errors)\b/gi, "")
        .replace(/\s+/g, " ")
        .trim();
}

function clusterLogEntries(entries, foldEnabled, expandedMap) {
    if (!Array.isArray(entries) || entries.length === 0) return [];
    if (!foldEnabled) {
        return entries.map((entry, idx) => ({
            id: "entry-" + idx,
            entry: entry,
            count: 1,
            isCluster: false,
            expanded: false,
            entries: [entry]
        }));
    }

    const clusters = [];
    let currentCluster = null;

    for (let i = 0; i < entries.length; i++) {
        const item = entries[i];
        const norm = normalizeLogMessage(item.msg || "");
        const prof = item.profile_id || item.profile_name || "";
        const obj = item.object || "";
        const act = item.action || "";
        const key = `${item.level || ""}::${item.type || ""}::${prof}::${obj}::${act}::${norm}`;

        if (currentCluster && currentCluster.key === key && norm.length > 5) {
            currentCluster.count++;
            currentCluster.entries.push(item);
        } else {
            if (currentCluster) {
                currentCluster.isCluster = currentCluster.count > 1;
                clusters.push(currentCluster);
            }
            const clusterId = "cluster-" + clusters.length + "-" + i;
            currentCluster = {
                id: clusterId,
                key: key,
                entry: item,
                count: 1,
                isCluster: false,
                expanded: !!(expandedMap && expandedMap[clusterId]),
                entries: [item]
            };
        }
    }
    if (currentCluster) {
        currentCluster.isCluster = currentCluster.count > 1;
        clusters.push(currentCluster);
    }
    return clusters;
}

window.LogsModule = {
    bisyncLogPage: 1,
    bisyncLogLimit: 50,
    bisyncLogTotal: 0,
    bisyncLogTotalPages: 1,
    bisyncPaginatedLogs: [],
    bisyncLogLoading: false,
    foldRepetitive: true,
    telemetryProfileFilter: "all",
    expandedClusters: {},

    get parsedBisyncLogs() {
        let logs = [];
        if (this.bisyncPaginatedLogs && this.bisyncPaginatedLogs.length > 0) {
            logs = [...this.bisyncPaginatedLogs];
        } else {
            logs = (this.telemetry.bisync_logs || []).map(parseJsonLogLine).filter(Boolean);
        }
        // Unified ledger: reverse order to display newest entries first
        return logs.reverse();
    },

    get filteredBisyncLogs() {
        let logs = this.parsedBisyncLogs;
        logs = logs.filter(l => !l.is_read_only_access && l.type !== "http_access");
        if (this.telemetryProfileFilter && this.telemetryProfileFilter !== "all") {
            const selected = (this.telemetryProfileFilter || "").toLowerCase().trim();
            const profObj = (this.profilesData?.list || []).find(p => (p.id || "").toLowerCase() === selected || (p.name || "").toLowerCase() === selected);
            const targetId = (profObj?.id || selected).toLowerCase();
            const targetName = (profObj?.name || "").toLowerCase();

            logs = logs.filter(l => {
                const pid = (l.profile_id || "").toLowerCase().trim();
                const pname = (l.profile_name || "").toLowerCase().trim();
                const msg = (l.msg || "").toLowerCase();
                if (pid && (pid === targetId || pid === targetName)) return true;
                if (pname && (pname === targetId || pname === targetName)) return true;
                if (targetName && msg.includes(`[${targetName}]`)) return true;
                if (targetId && msg.includes(`[${targetId}]`)) return true;
                return false;
            });
        }
        if (this.telemetryLogFilter === "sync") return logs.filter(l => l.type !== "audit");
        if (this.telemetryLogFilter === "audit") return logs.filter(l => l.type === "audit");
        if (this.telemetryLogFilter === "file_ops" || this.telemetryLogFilter === "mutations") {
            return logs.filter(l => typeof this.isFileOperationLog === "function" ? this.isFileOperationLog(l) : Boolean(this.getEntryPath(l)));
        }
        return logs;
    },

    get clusteredBisyncLogs() {
        return clusterLogEntries(this.filteredBisyncLogs, this.foldRepetitive, this.expandedClusters);
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

    get clusteredDaemonModalLogs() {
        return clusterLogEntries(this.filteredDaemonModalLogs, this.foldRepetitive, this.expandedClusters);
    },

    get daemonModalCounts() {
        const logs = this.daemonModalLogs || [];
        let errors = 0, warnings = 0, info = 0;
        for (let i = 0; i < logs.length; i++) {
            const lvl = (logs[i].level || "").toLowerCase().trim();
            if (lvl === "error") errors++;
            else if (lvl === "warning" || lvl === "warn") warnings++;
            else if (lvl === "info" || lvl === "notice") info++;
        }
        return { total: logs.length, errors, warnings, info };
    },

    async fetchBisyncLogs(page = 1, silent = false) {
        if (!silent) this.bisyncLogLoading = true;
        try {
            const data = await window.API.get(`/api/logs/bisync?page=${page}&limit=${this.bisyncLogLimit}`);
            this.bisyncPaginatedLogs = (data.logs || []).map(parseJsonLogLine).filter(Boolean);
            this.bisyncLogPage = data.page || 1;
            this.bisyncLogTotal = data.total || 0;
            this.bisyncLogTotalPages = data.total_pages || 1;
        } catch (err) {
            console.error("Failed to fetch paginated bisync logs:", err);
        } finally {
            if (!silent) this.bisyncLogLoading = false;
        }
    },

    nextBisyncPage() { if (this.bisyncLogPage < this.bisyncLogTotalPages) this.fetchBisyncLogs(this.bisyncLogPage + 1); },
    prevBisyncPage() { if (this.bisyncLogPage > 1) this.fetchBisyncLogs(this.bisyncLogPage - 1); },

    toggleCluster(clusterId) { this.expandedClusters[clusterId] = !this.expandedClusters[clusterId]; },
    expandAllClusters() {
        const map = {};
        this.clusteredBisyncLogs.forEach(c => { if (c.isCluster) map[c.id] = true; });
        this.clusteredDaemonModalLogs.forEach(c => { if (c.isCluster) map[c.id] = true; });
        this.expandedClusters = map;
    },
    collapseAllClusters() { this.expandedClusters = {}; },
    get areAllClustersExpanded() {
        const clusters = this.clusteredBisyncLogs.filter(c => c.isCluster);
        return clusters.length > 0 && clusters.every(c => !!this.expandedClusters[c.id]);
    },
    toggleAllClusters() {
        if (this.areAllClustersExpanded) this.collapseAllClusters();
        else this.expandAllClusters();
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
            const data = await window.API.get("/api/deletions/pending");
            this.deletionsData = data;
        } catch (err) {
            this.deletionsData = { total_count: 0, source: "Error", target_action: err.message, files: [] };
        }
    },

    async fetchDaemonLogs(entry = null) {
        this.daemonModalLoading = true;
        this.daemonModalRun = entry || null;
        this.daemonModalError = null;

        let url = "/api/logs/daemon?limit=500";
        if (entry) {
            if (entry.run_id) url += `&run_id=${encodeURIComponent(entry.run_id)}`;
            if (entry.job_id) url += `&job_id=${encodeURIComponent(entry.job_id)}`;
            if (entry.start_time) url += `&start_time=${encodeURIComponent(entry.start_time)}`;
            if (entry.end_time) url += `&end_time=${encodeURIComponent(entry.end_time)}`;
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
        this.daemonModalCopied = false;
        this.daemonModalCopiedJSON = false;
        await this.fetchDaemonLogs(entry);
    },

    closeDaemonModal() {
        this.showDaemonModal = false;
        this.daemonModalRun = null;
        this.daemonModalLogs = [];
        this.daemonModalError = null;
    },

    copyDaemonLogs() {
        const text = this.daemonModalLogs.map(l => l.raw || `[${l.time}] ${l.level} ${l.msg}`).join("\n");
        navigator.clipboard.writeText(text).then(() => {
            this.daemonModalCopied = true;
            setTimeout(() => { this.daemonModalCopied = false; }, 2000);
        });
    },

    copyDaemonLogsJSON() {
        const json = JSON.stringify(this.daemonModalLogs, null, 2);
        navigator.clipboard.writeText(json).then(() => {
            this.daemonModalCopiedJSON = true;
            setTimeout(() => { this.daemonModalCopiedJSON = false; }, 2000);
        });
    },

    confirmResync() {
        const proceed = () => this.triggerAction("force_resync");
        if (typeof this.askConfirmation === "function") {
            this.askConfirmation({
                title: "Force Resync Confirmation",
                message: "Are you sure you want to rebuild baseline sync states?\nThis will recreate listings and take significantly longer.",
                confirmText: "Rebuild & Resync",
                confirmClass: "bg-amber-600 hover:bg-amber-500 shadow-amber-950/40",
                icon: "warning",
                onConfirm: proceed
            });
        } else if (confirm("⚠ FORCE RESYNC ⚠\n\nAre you sure you want to rebuild baseline sync states?\nThis will take significantly longer.")) {
            proceed();
        }
    },

    getEntryPath(entry) {
        if (!entry) return null;
        if (entry.object) return entry.object;
        if (entry.details) {
            if (entry.details.item_path) return entry.details.item_path;
            if (entry.details.path && entry.details.filename) {
                return `${entry.details.path}/${entry.details.filename}`.replace(/^\/+/, "");
            }
            if (entry.details.path) return entry.details.path;
            if (entry.details.object) return entry.details.object;
            if (entry.details.file) return entry.details.file;
        }
        if (entry.msg && typeof extractPathFromLogMsg === "function") {
            return extractPathFromLogMsg(entry.msg);
        }
        return null;
    },

    async jumpToExplorerPath(targetPath, profileId = null) {
        if (!targetPath) return;
        const clean = String(targetPath).trim().replace(/^["'`\s]+|["'`\s]+$/g, "").replace(/\\/g, "/").replace(/^\/+/, "");
        if (!clean) return;

        const lastSlash = clean.lastIndexOf("/");
        const dir = lastSlash >= 0 ? clean.substring(0, lastSlash) : "";
        const filename = lastSlash >= 0 ? clean.substring(lastSlash + 1) : clean;

        if (typeof this.closeDaemonModal === "function") this.closeDaemonModal();
        else this.showDaemonModal = false;

        const targetProfileId = (profileId && typeof profileId === "object") ? (profileId.profile_id || null) : profileId;
        if (targetProfileId && this.profilesData && this.profilesData.activeId !== targetProfileId && typeof this.selectProfile === "function") {
            await this.selectProfile(targetProfileId, dir);
        }

        if (typeof this.switchTab === "function") this.switchTab("explorer");
        else this.currentTab = "explorer";

        if (this.explorer) {
            this.explorer.highlightPath = clean;
            this.explorer.highlightName = filename;
            setTimeout(() => {
                if (this.explorer && this.explorer.highlightName === filename) this.explorer.highlightName = null;
            }, 6000);

            if (this.explorer.left) this.explorer.left.path = dir;
            if (this.explorer.right) this.explorer.right.path = dir;

            const p1 = (this.explorer.left && typeof this.fetchExplorerDirectory === "function")
                ? this.fetchExplorerDirectory("left", this.explorer.left.target, dir) : Promise.resolve();
            const p2 = (this.explorer.right && typeof this.fetchExplorerDirectory === "function")
                ? this.fetchExplorerDirectory("right", this.explorer.right.target, dir) : Promise.resolve();
            await Promise.allSettled([p1, p2]);
        }

        const notifyOrPulse = () => this.scrollToHighlightedRow(filename);
        if (typeof this.$nextTick === "function") this.$nextTick(notifyOrPulse);
        else setTimeout(notifyOrPulse, 50);
    },

    scrollToHighlightedRow(filename) {
        if (!filename) return;
        const selector = `[data-item-name="${CSS.escape ? CSS.escape(filename) : filename}"]`;
        const els = document.querySelectorAll(selector);
        els.forEach(el => {
            el.scrollIntoView({ behavior: "smooth", block: "center" });
            el.classList.remove("row-pulse-ring");
            void el.offsetWidth;
            el.classList.add("row-pulse-ring");
        });
        if (els.length === 0 && typeof this.showToast === "function") {
            this.showToast(`Navigated to folder. Item "${filename}" may be remote-only or excluded.`, "info");
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
