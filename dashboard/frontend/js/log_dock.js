/**
 * Rclone Bisync Dashboard — Pinned Log Audit Dock Engine
 * Collapsible bottom tray anchored beneath Dual-Pane File Explorer for rapid log review & step-through.
 */

window.LogDockModule = {
    logDock: {
        open: false,
        heightPreset: "docked", // 'collapsed' | 'docked' | 'expanded'
        activeFilter: "all",    // 'all' | 'mutations' | 'errors'
        selectedIdx: -1,
        autoScroll: true,
        limit: 100
    },

    _logDockInitialized: false,

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

    getFilteredDockLogs() {
        let sourceLogs = [];
        if (Array.isArray(this.filteredBisyncLogs) && this.filteredBisyncLogs.length > 0) {
            sourceLogs = this.filteredBisyncLogs;
        } else if (Array.isArray(this.parsedBisyncLogs) && this.parsedBisyncLogs.length > 0) {
            sourceLogs = this.parsedBisyncLogs;
        } else if (this.telemetry && Array.isArray(this.telemetry.bisync_logs)) {
            const mapper = typeof parseJsonLogLine === "function" ? parseJsonLogLine : (l) => ({ msg: l });
            sourceLogs = (this.telemetry.bisync_logs || []).map(l => typeof l === "string" ? mapper(l) : l).filter(Boolean);
        }

        let logs = sourceLogs;
        if (this.logDock.activeFilter === "mutations") {
            logs = sourceLogs.filter(l => {
                if (!l) return false;
                const lvl = String(l.level || "").trim().toUpperCase();
                if (lvl === "ERROR" || lvl === "NOTICE") return true;
                const path = typeof this.getEntryPath === "function" ? this.getEntryPath(l) : (l.object || null);
                if (path) return true;
                const msg = String(l.msg || "").toLowerCase();
                return msg.includes("transfer") || msg.includes("delete") || msg.includes("conflict") ||
                       msg.includes("copy") || msg.includes("move") || msg.includes("updated");
            });
        } else if (this.logDock.activeFilter === "errors") {
            logs = sourceLogs.filter(l => String(l?.level || "").trim().toUpperCase() === "ERROR");
        }
        return logs.slice(0, this.logDock.limit || 100);
    },

    logDockStep(direction) {
        const logs = this.getFilteredDockLogs();
        if (!logs || logs.length === 0) return;
        let newIdx = this.logDock.selectedIdx + direction;
        if (newIdx < 0) newIdx = 0;
        if (newIdx >= logs.length) newIdx = logs.length - 1;
        this.logDock.selectedIdx = newIdx;
        const entry = logs[newIdx];
        if (!entry) return;
        const path = typeof this.getEntryPath === "function" ? this.getEntryPath(entry) : null;
        if (path && typeof this.jumpToExplorerPath === "function") {
            this.jumpToExplorerPath(path, entry, { silentToast: true });
        }
        this._scrollDockItemIntoView(newIdx);
    },

    logDockSelectAndJump(log, idx) {
        this.logDock.selectedIdx = idx;
        const path = typeof this.getEntryPath === "function" ? this.getEntryPath(log) : null;
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
