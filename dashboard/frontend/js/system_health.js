/**
 * System Health & Diagnostics Center Module for Bisync Hub Dashboard
 * Unifies Daemon telemetry, MCP agent server status, and background Poller scheduler.
 */

window.SystemHealthModule = {
    systemHealthModal: {
        show: false,
        tab: "daemon", // "daemon" | "mcp" | "poller"
        copiedConfig: false,
        copiedLogs: false,
        searchQuery: "",
        logLevel: "ALL"
    },
    quickCreateOpen: false,

    openSystemHealthModal(tab = "daemon") {
        this.systemHealthModal.show = true;
        this.systemHealthModal.tab = tab;
        this.quickCreateOpen = false;

        // If daemon tab opened, ensure daemon logs are fetched
        if (tab === "daemon" && typeof this.fetchDaemonLogs === "function") {
            this.fetchDaemonLogs();
        }
    },

    closeSystemHealthModal() {
        this.systemHealthModal.show = false;
    },

    setSystemHealthTab(tab) {
        this.systemHealthModal.tab = tab;
        if (tab === "daemon" && typeof this.fetchDaemonLogs === "function") {
            this.fetchDaemonLogs();
        }
    },

    async triggerManualPoll() {
        if (typeof this.poll === "function") {
            await this.poll();
            if (typeof this.showToast === "function") {
                this.showToast("Telemetry metrics refreshed successfully", "success");
            }
        }
    },

    get isSystemHealthy() {
        return Boolean(this.telemetry && this.telemetry.daemon_health && this.telemetry.daemon_health.status === "healthy");
    },

    get systemHealthBadgeClass() {
        if (!this.isSystemHealthy) {
            return "border-rose-500/40 bg-rose-950/20 text-rose-300";
        }
        if (this.isPollingPaused) {
            return "border-amber-500/40 bg-amber-950/20 text-amber-300";
        }
        return "border-slate-800 bg-slate-900/90 text-slate-300 hover:border-slate-700";
    },

    get pollerActivityText() {
        if (this.telemetry && this.telemetry.is_syncing) {
            return "Syncing";
        }
        if (this.isPollingPaused) {
            return "Paused";
        }
        return "Idle";
    },

    get pollerActivityBadgeClass() {
        if (this.telemetry && this.telemetry.is_syncing) {
            return "bg-indigo-500/20 text-indigo-300 border border-indigo-500/30";
        }
        if (this.isPollingPaused) {
            return "bg-amber-500/20 text-amber-300 border border-amber-500/30";
        }
        return "bg-slate-800 text-slate-400 border border-slate-700/60";
    }
};


// Global Keyboard Shortcuts for Quick Create, Profile, and Remote Wizards
window.addEventListener("keydown", (e) => {
    const tag = (e.target && e.target.tagName) ? e.target.tagName.toLowerCase() : "";
    if (tag === "input" || tag === "textarea" || tag === "select" || (e.target && e.target.isContentEditable)) {
        return;
    }
    const rootEl = document.querySelector("[x-data]");
    if (!rootEl || !window.Alpine) return;
    const app = Alpine.$data(rootEl);
    if (!app) return;

    if ((e.key === "c" || e.key === "C") && !e.ctrlKey && !e.metaKey && !e.altKey) {
        app.quickCreateOpen = !app.quickCreateOpen;
    } else if ((e.key === "k" || e.key === "K") && (e.ctrlKey || e.metaKey)) {
        e.preventDefault();
        app.quickCreateOpen = !app.quickCreateOpen;
    } else if ((e.key === "n" || e.key === "N") && !e.ctrlKey && !e.metaKey && !e.altKey) {
        if (typeof app.startNewProfile === "function") {
            app.quickCreateOpen = false;
            app.startNewProfile();
        }
    } else if ((e.key === "r" || e.key === "R") && !e.ctrlKey && !e.metaKey && !e.altKey) {
        if (typeof app.openCreateRemoteWizard === "function") {
            app.quickCreateOpen = false;
            app.openCreateRemoteWizard();
        }
    }
});