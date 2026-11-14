/**
 * Telemetry & Correlated Daemon Logs Module for Bisync Hub Dashboard
 * Handles bisync/daemon log filtering, 50-line on-demand pagination,
 * inline log deduplication/folding, correlated run modals, and deletion state.
 */

window.LogsModule = {
    clusterLogEntries(entries, fold = this.foldRepetitive, map = this.expandedClusters) {
        const clusterFn = typeof clusterLogEntries === "function" ? clusterLogEntries : window.clusterLogEntries;
        return typeof clusterFn === "function" ? clusterFn(entries, fold, map) : entries;
    },
    bisyncLogPage: 1,
    bisyncLogLimit: 50,
    bisyncLogTotal: 0,
    bisyncLogTotalPages: 1,
    bisyncPaginatedLogs: [],
    bisyncLogLoading: false,
    foldRepetitive: true,
    includeFileTransfers: false,
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
        if (!this.includeFileTransfers && this.telemetryLogFilter !== "file_ops" && this.telemetryLogFilter !== "mutations") {
            logs = logs.filter(l => l.type !== "sync_file" && l.type !== "sync_overflow");
        }
        if (this.telemetryProfileFilter && this.telemetryProfileFilter !== "all") {
            const selected = (this.telemetryProfileFilter || "").toLowerCase().trim();
            const profObj = (this.profilesData?.list || []).find(p => (p.id || "").toLowerCase() === selected || (p.name || "").toLowerCase() === selected);
            const targetId = (profObj?.id || selected).toLowerCase();
            const targetName = (profObj?.name || "").toLowerCase();

            logs = logs.filter(l => {
                const pid = (l.profile_id || "").toLowerCase().trim();
                const pname = (l.profile_name || "").toLowerCase().trim();
                const msg = (l.msg || "").toLowerCase();
                return (pid && (pid === targetId || pid === targetName)) ||
                       (pname && (pname === targetId || pname === targetName)) ||
                       (targetName && msg.includes(`[${targetName}]`)) ||
                       (targetId && msg.includes(`[${targetId}]`));
            });
        }
        if (this.telemetryLogFilter === "sync") return logs.filter(l => l.type !== "audit");
        if (this.telemetryLogFilter === "audit") return logs.filter(l => l.type === "audit");
        if (this.telemetryLogFilter === "file_ops" || this.telemetryLogFilter === "mutations") {
            return logs.filter(l => typeof this.isFileOperationLog === "function" ? this.isFileOperationLog(l) : Boolean(this.getEntryPath(l)));
        }
        const consolidateFn = typeof consolidateRunEntries === "function" ? consolidateRunEntries : window.consolidateRunEntries;
        return typeof consolidateFn === "function" ? consolidateFn(logs, this.profilesData?.list || []) : logs;
    },

    get clusteredBisyncLogs() {
        const clusterFn = typeof clusterLogEntries === "function" ? clusterLogEntries : window.clusterLogEntries;
        return typeof clusterFn === "function" ? clusterFn(this.filteredBisyncLogs, this.foldRepetitive, this.expandedClusters) : this.filteredBisyncLogs;
    },

    async fetchBisyncLogs(page = 1, silent = false) {
        if (!silent) this.bisyncLogLoading = true;
        try {
            let url = `/api/logs/bisync?page=${page}&limit=${this.bisyncLogLimit}`;
            if (!this.includeFileTransfers && this.telemetryLogFilter !== "file_ops" && this.telemetryLogFilter !== "mutations") {
                url += "&exclude_types=sync_file,sync_overflow";
            }
            if (this.telemetryProfileFilter && this.telemetryProfileFilter !== "all") {
                url += `&profile_id=${encodeURIComponent(this.telemetryProfileFilter)}`;
            }
            const data = await window.API.get(url);
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
        const d = entry.details;
        if (d) {
            if (d.item_path) return d.item_path;
            if (d.path && d.filename) return `${d.path}/${d.filename}`.replace(/^\/+/, "");
            if (d.path || d.object || d.file) return d.path || d.object || d.file;
        }
        if (entry.msg && typeof extractPathFromLogMsg === "function") return extractPathFromLogMsg(entry.msg);
        return null;
    },

    async jumpToExplorerPath(targetPath, profileId = null) {
        if (!targetPath) return;
        const clean = String(targetPath).trim().replace(/^["'`\s]+|["'`\s]+$/g, "").replace(/\\/g, "/").replace(/^\/+/, "");
        if (!clean) return;

        const lastSlash = clean.lastIndexOf("/");
        const dir = lastSlash >= 0 ? clean.substring(0, lastSlash) : "";
        const filename = lastSlash >= 0 ? clean.substring(lastSlash + 1) : clean;

        if (typeof this.closeDaemonModal === "function") this.closeDaemonModal(); else this.showDaemonModal = false;

        const targetProfileId = (profileId && typeof profileId === "object") ? (profileId.profile_id || null) : profileId;
        if (targetProfileId && this.profilesData && this.profilesData.activeId !== targetProfileId && typeof this.selectProfile === "function") {
            await this.selectProfile(targetProfileId, dir);
        }

        if (this.currentTab !== "explorer") {
            if (typeof this.switchTab === "function") this.switchTab("explorer"); else this.currentTab = "explorer";
        }

        if (this.explorer) {
            this.explorer.highlightPath = clean;
            this.explorer.highlightName = filename;
            setTimeout(() => { if (this.explorer && this.explorer.highlightName === filename) this.explorer.highlightName = null; }, 6000);

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
