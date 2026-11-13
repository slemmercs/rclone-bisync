/**
 * Rclone Bisync Dashboard — Pinned Log Audit Dock Engine
 * Collapsible bottom tray anchored beneath Dual-Pane File Explorer for rapid log review & step-through.
 */

window.LogDockModule = {
    logDock: {
        open: false,
        heightPreset: "docked", // 'collapsed' | 'docked' | 'expanded'
        activeFilter: "all",    // 'all' | 'file_ops' | 'mutations' | 'errors'
        selectedIdx: -1,
        autoScroll: true,
        limit: 100,
        page: 1,
        pageSize: 50
    },

    _logDockInitialized: false,
    _lastActiveFilter: "all",

    initLogDock() {
        if (this._logDockInitialized) return;
        this._logDockInitialized = true;
        window.addEventListener("keydown", (e) => {
            if (!this.logDock || !this.logDock.open) return;
            const tag = (e.target && e.target.tagName) || "";
            if (tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT" || e.target?.isContentEditable) return;
            if (e.key === "n" || e.key === "N") {
                e.preventDefault();
                this.logDockStep(1);
            } else if (e.key === "p" || e.key === "P") {
                e.preventDefault();
                this.logDockStep(-1);
            }
        });
    },

    toggleLogDock(openState = null) {
        if (typeof this.initLogDock === "function") this.initLogDock();
        if (openState !== null) {
            this.logDock.open = Boolean(openState);
        } else {
            this.logDock.open = !this.logDock.open;
        }
        if (this.logDock.open && this.logDock.heightPreset === "collapsed") {
            this.logDock.heightPreset = "docked";
        }
    },

    setLogDockHeight(preset) {
        if (["collapsed", "docked", "expanded"].includes(preset)) {
            this.logDock.heightPreset = preset;
            if (!this.logDock.open) this.logDock.open = true;
        }
    },

    setLogDockFilter(filter) {
        this.logDock.activeFilter = filter;
        this._lastActiveFilter = filter;
        this.logDock.page = 1;
        this.logDock.selectedIdx = -1;
    },

    isFileOperationLog(entry) {
        if (!entry) return false;
        const msg = String(entry.msg || "").trim().toLowerCase();
        if (msg.includes("0 file(s) transferred") || msg.includes("0 files transferred") ||
            msg.includes("dry-run banner") || msg.includes("starting rclone") || msg.includes("daemon started")) {
            return false;
        }
        if (entry.type === "sync_file" || entry.type === "sync_overflow") return true;
        const fileActions = [
            "WORKSPACE_FILE_UPLOADED", "WORKSPACE_ITEM_COPIED", "WORKSPACE_ITEM_MOVED",
            "WORKSPACE_ITEM_DELETED", "WORKSPACE_DIR_CREATED", "WORKSPACE_DIR_PURGED",
            "CONFLICT_RESOLVED", "DELETION_ACCEPTED", "DELETION_RESTORED",
            "ITEM_RESTORED", "FILE_DELETED", "FILE_TRANSFER", "FILE_DIFF"
        ];
        if (entry.action && fileActions.includes(String(entry.action).toUpperCase())) return true;
        const path = typeof this.getEntryPath === "function" ? this.getEntryPath(entry) : (entry.object || null);
        if (path) return true;
        return (msg.includes("transfer") || msg.includes("delet") || msg.includes("conflict") ||
                msg.includes("copi") || msg.includes("copy") || msg.includes("mov") ||
                msg.includes("upload") || msg.includes("purged") || msg.includes("restor"));
    },

    getFilteredDockLogsRaw() {
        let sourceLogs = [];
        if (Array.isArray(this.filteredBisyncLogs) && this.filteredBisyncLogs.length > 0) {
            sourceLogs = this.filteredBisyncLogs;
        } else if (Array.isArray(this.parsedBisyncLogs) && this.parsedBisyncLogs.length > 0) {
            sourceLogs = this.parsedBisyncLogs;
        } else if (this.telemetry && Array.isArray(this.telemetry.bisync_logs)) {
            const mapper = typeof parseJsonLogLine === "function" ? parseJsonLogLine : (l) => ({ msg: l });
            sourceLogs = (this.telemetry.bisync_logs || []).map(l => typeof l === "string" ? mapper(l) : l).filter(Boolean);
        }

        if (this.logDock.activeFilter !== this._lastActiveFilter) {
            this._lastActiveFilter = this.logDock.activeFilter;
            this.logDock.page = 1;
            this.logDock.selectedIdx = -1;
        }

        if (this.logDock.activeFilter === "file_ops") {
            return sourceLogs.filter(l => this.isFileOperationLog(l));
        } else if (this.logDock.activeFilter === "mutations") {
            return sourceLogs.filter(l => {
                if (!l) return false;
                const lvl = String(l.level || "").trim().toUpperCase();
                if (lvl === "ERROR" || lvl === "NOTICE") return true;
                return this.isFileOperationLog(l);
            });
        } else if (this.logDock.activeFilter === "errors") {
            return sourceLogs.filter(l => String(l?.level || "").trim().toUpperCase() === "ERROR");
        }
        return sourceLogs;
    },

    getFilteredDockLogs() {
        const logs = this.getFilteredDockLogsRaw();
        const pageSize = this.logDock.pageSize || 50;
        const totalPages = Math.max(1, Math.ceil(logs.length / pageSize));
        if (this.logDock.page > totalPages) this.logDock.page = totalPages;
        if (this.logDock.page < 1) this.logDock.page = 1;
        const start = (this.logDock.page - 1) * pageSize;
        return logs.slice(start, start + pageSize);
    },

    getDockTotalEntries() {
        return this.getFilteredDockLogsRaw().length;
    },

    getDockTotalPages() {
        const total = this.getFilteredDockLogsRaw().length;
        const size = this.logDock.pageSize || 50;
        return Math.max(1, Math.ceil(total / size));
    },

    loadDockNextPage() {
        if (!this.isDockNextDisabled()) {
            this.logDock.page = (this.logDock.page || 1) + 1;
            this.logDock.selectedIdx = -1;
        }
    },

    loadDockPrevPage() {
        if (!this.isDockPrevDisabled()) {
            this.logDock.page = Math.max(1, (this.logDock.page || 1) - 1);
            this.logDock.selectedIdx = -1;
        }
    },

    isDockPrevDisabled() {
        return (this.logDock?.page || 1) <= 1;
    },

    isDockNextDisabled() {
        return (this.logDock?.page || 1) >= this.getDockTotalPages();
    },

    navigateToExplorer(path, profileId = null) {
        if (typeof this.jumpToExplorerPath === "function") {
            return this.jumpToExplorerPath(path, profileId);
        }
    },

    logDockStep(direction) {
        const logs = this.getFilteredDockLogs();
        if (!logs || logs.length === 0) return;
        let newIdx = this.logDock.selectedIdx + direction;
        if (newIdx < 0) {
            if ((this.logDock.page || 1) > 1) {
                this.loadDockPrevPage();
                const prevLogs = this.getFilteredDockLogs();
                this.logDock.selectedIdx = Math.max(0, prevLogs.length - 1);
                this._scrollDockItemIntoView(this.logDock.selectedIdx);
                const entry = prevLogs[this.logDock.selectedIdx];
                const path = typeof this.getEntryPath === "function" ? this.getEntryPath(entry) : (entry?.object || null);
                if (path && typeof this.jumpToExplorerPath === "function") {
                    this.jumpToExplorerPath(path, entry, { silentToast: true });
                }
                return;
            }
            newIdx = 0;
        } else if (newIdx >= logs.length) {
            if ((this.logDock.page || 1) < this.getDockTotalPages()) {
                this.loadDockNextPage();
                this.logDock.selectedIdx = 0;
                this._scrollDockItemIntoView(0);
                const nextLogs = this.getFilteredDockLogs();
                const entry = nextLogs[0];
                const path = typeof this.getEntryPath === "function" ? this.getEntryPath(entry) : (entry?.object || null);
                if (path && typeof this.jumpToExplorerPath === "function") {
                    this.jumpToExplorerPath(path, entry, { silentToast: true });
                }
                return;
            }
            newIdx = logs.length - 1;
        }
        this.logDock.selectedIdx = newIdx;
        const entry = logs[newIdx];
        if (!entry) return;
        const path = typeof this.getEntryPath === "function" ? this.getEntryPath(entry) : (entry.object || null);
        if (path && typeof this.jumpToExplorerPath === "function") {
            this.jumpToExplorerPath(path, entry, { silentToast: true });
        }
        this._scrollDockItemIntoView(newIdx);
    },

    logDockSelectAndJump(log, idx) {
        this.logDock.selectedIdx = idx;
        const path = typeof this.getEntryPath === "function" ? this.getEntryPath(log) : (log?.object || null);
        if (path && typeof this.jumpToExplorerPath === "function") {
            this.jumpToExplorerPath(path, log);
        }
    },

    _scrollDockItemIntoView(idx) {
        const container = document.getElementById("log-dock-container");
        if (!container) return;
        const rows = container.children;
        if (rows && rows[idx]) {
            rows[idx].scrollIntoView({ behavior: "smooth", block: "nearest" });
        }
    },

    getDockLogLevelClass(level) {
        const lvl = String(level || "INFO").trim().toUpperCase();
        if (lvl === "ERROR") return "bg-rose-500/20 text-rose-300 border border-rose-500/30";
        if (lvl === "NOTICE" || lvl === "WARN" || lvl === "WARNING") return "bg-amber-500/20 text-amber-300 border border-amber-500/30";
        return "bg-slate-800 text-slate-400 border border-slate-700/60";
    },

    getDockActionBadgeClass(log) {
        if (!log) return "bg-slate-800 text-slate-300 border-slate-700";
        const action = String(log.action || "").toUpperCase();
        if (action === "DELETED" || action.includes("DELETE") || action.includes("PURGE")) {
            return "bg-rose-500/20 text-rose-300 border border-rose-500/40";
        }
        if (action === "RENAMED" || action === "MOVED" || action.includes("MOVE") || action.includes("RENAME")) {
            return "bg-purple-500/20 text-purple-300 border border-purple-500/40";
        }
        if (action === "COPIED" || action === "REPLACED" || action.includes("COPY") || action.includes("UPLOAD") || log.type === "sync_file") {
            return "bg-emerald-500/20 text-emerald-300 border border-emerald-500/40";
        }
        return "bg-slate-800 text-slate-300 border-slate-700";
    },

    getDockActionLabel(log) {
        if (!log) return "SYNC";
        if (log.action) {
            return log.type === "sync_file" ? `SYNC: ${log.action}` : String(log.action);
        }
        return log.type === "sync_file" ? "SYNC" : "LOG";
    },

    formatDockLogTime(ts) {
        if (!ts) return "";
        try {
            const d = new Date(ts);
            if (isNaN(d.getTime())) return String(ts);
            return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit", hour12: false });
        } catch (_) {
            return String(ts);
        }
    }
};
