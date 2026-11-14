/**
 * Rclone Bisync Dashboard — Application State & Computed Status Module
 * Provides entity drawer, toast notifications, confirmation modals,
 * and computed telemetry status properties.
 */

window.AppStateModule = {
    entityDrawer: {
        open: false,
        title: "",
        subtitle: "",
        type: "audit",
        data: null
    },

    openAuditDrawer(record) {
        if (!record) return;
        this.entityDrawer.open = true;
        this.entityDrawer.title = `Audit Block #${record.seq_id} — ${record.action}`;
        this.entityDrawer.subtitle = record.timestamp || "";
        this.entityDrawer.type = "audit";
        let rec = record;
        if (record.action === "DEDUPE_EXECUTED" && record.details?.output && (record.details.collisions == null || record.details.deleted == null)) {
            const parsed = typeof parseDedupeOutput === "function" ? parseDedupeOutput(record.details.output) : null;
            if (parsed) {
                rec = { ...record, details: { ...record.details, collisions: record.details.collisions ?? parsed.totalCollisions, deleted: record.details.deleted ?? parsed.deletedCopies } };
            }
        }
        this.entityDrawer.data = rec;
    },

    toast: {
        show: false,
        message: "",
        type: "info",
        timeout: null
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

    confirmModal: {
        show: false,
        title: "",
        message: "",
        confirmText: "Confirm",
        confirmClass: "bg-indigo-600 hover:bg-indigo-500",
        icon: "warning",
        transfer: null,
        onConfirm: null
    },

    askConfirmation({ title, message, confirmText = "Confirm", confirmClass = "bg-indigo-600 hover:bg-indigo-500", icon = "warning", transfer = null, onConfirm }) {
        this.confirmModal.title = title;
        this.confirmModal.message = message;
        this.confirmModal.confirmText = confirmText;
        this.confirmModal.confirmClass = confirmClass;
        this.confirmModal.icon = icon;
        this.confirmModal.transfer = transfer;
        this.confirmModal.onConfirm = onConfirm;
        this.confirmModal.show = true;
    },

    handleConfirmModal() {
        const fn = this.confirmModal.onConfirm;
        this.confirmModal.show = false;
        this.confirmModal.onConfirm = null;
        this.confirmModal.transfer = null;
        if (typeof fn === "function") {
            fn();
        }
    },

    cancelConfirmModal() {
        this.confirmModal.show = false;
        this.confirmModal.onConfirm = null;
        this.confirmModal.transfer = null;
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
        const status = this.telemetry?.daemon_health?.status;
        return status === "healthy" ? "ONLINE" : "OFFLINE";
    },

    get daemonStatusClass() {
        const status = this.telemetry?.daemon_health?.status;
        return status === "healthy" ? "text-emerald-400 tracking-wide" : "text-rose-500 tracking-wide";
    },

    get daemonLatencyText() {
        const lat = this.telemetry?.daemon_health?.response_time_ms;
        return typeof lat === "number" ? lat.toFixed(2) : "--";
    },

    get mcpStatusText() {
        const m = this.telemetry?.mcp_status;
        if (m && m.status === "active") {
            const mode = m.read_only ? "read-only" : "read/write";
            return `MCP: ACTIVE [${mode}]`;
        }
        return "MCP: OFFLINE";
    },

    get mcpStatusClass() {
        const m = this.telemetry?.mcp_status;
        if (m && m.status === "active") {
            return "border-violet-800/60 bg-violet-950/40 text-violet-300";
        }
        return "border-slate-800 bg-slate-900/60 text-slate-500";
    },

    get isMcpActive() {
        return Boolean(this.telemetry?.mcp_status?.status === "active");
    },

    get executionStatusText() {
        if (this.isEngineSyncing()) return "SYNC IN PROGRESS";
        const notif = this.telemetry?.notification_state || {};
        const successTs = notif.last_success_timestamp || 0;
        const errorTs = notif.last_error_timestamp || 0;
        if (errorTs > successTs && notif.last_error_type) return `FAULT (${notif.last_error_type})`;
        if (successTs > 0) return "OPERATIONAL";
        return "UNKNOWN";
    },

    get executionStatusClass() {
        if (this.isEngineSyncing()) return "text-amber-400 font-bold tracking-wide";
        const notif = this.telemetry?.notification_state || {};
        const successTs = notif.last_success_timestamp || 0;
        const errorTs = notif.last_error_timestamp || 0;
        if (errorTs > successTs && notif.last_error_type) return "text-rose-400 text-lg";
        if (successTs > 0) return "text-emerald-400";
        return "text-slate-400";
    },

    get lastUpdatedText() {
        const notif = this.telemetry?.notification_state || {};
        const ts = Math.max(notif.last_success_timestamp || 0, notif.last_error_timestamp || 0);
        if (!ts) return "--";
        return new Date(ts * 1000).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit", hour12: false });
    },

    get isDevMode() {
        const params = new URLSearchParams(window.location.search);
        const dev = params.get("dev");
        return dev === "true" || dev === "1";
    },

    get transferStats() {
        const stats = this.telemetry?.transfer_stats || {};
        const isSyncing = Boolean(this.telemetry?.is_syncing);
        const isListing = isSyncing && (!stats.bytes || stats.bytes === 0);
        const currentFile = (stats.transferring && stats.transferring.length > 0)
            ? stats.transferring[0].name
            : (isSyncing ? (isListing ? "Listing & Diffing..." : "Syncing in progress...") : "Idle");
        let etaStr = "--";
        if (typeof stats.eta === "number") {
            const mins = Math.floor(stats.eta / 60);
            const secs = stats.eta % 60;
            etaStr = `${mins}m ${secs}s`;
        } else if (isListing) {
            etaStr = "Diffing";
        }
        return {
            speed_str: isListing ? "Listing & Diffing" : (stats.speed_str || "0 B/s"),
            eta_str: etaStr,
            percentage: typeof stats.percentage === "number" ? stats.percentage : 0,
            current_file: currentFile,
            is_listing: isListing
        };
    },

    normalizeStatus(status) {
        return typeof normalizeStatus === "function" ? normalizeStatus(status) : status;
    },

    getStatusBadgeClass(status) {
        return typeof getStatusBadgeClass === "function" ? getStatusBadgeClass(status) : "";
    }
};
