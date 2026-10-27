/**
 * Cryptographic Immutable Audit Ledger Module for Bisync Hub Dashboard
 * Manages ledger query state, cryptographic chain verification, filtering,
 * pagination, and forensic details inspection.
 */

window.AuditModule = {
    auditData: {
        records: [],
        total: 0,
        page: 1,
        limit: 50,
        totalPages: 1,
        loading: false,
        error: null,
        filterAction: "",
        filterProfile: "",
        filterActor: "",
        verification: {
            valid: true,
            total_records: 0,
            checked_at: null,
            head_hash: "",
            error: null,
            checking: false
        },
        selectedRecord: null,
        copiedHash: false
    },

    async initAudit() {
        await Promise.all([
            this.fetchAuditRecords(),
            this.verifyAuditLedger()
        ]);
    },

    async fetchAuditRecords() {
        this.auditData.loading = true;
        this.auditData.error = null;
        try {
            const params = new URLSearchParams({
                limit: this.auditData.limit,
                page: this.auditData.page,
                action: this.auditData.filterAction,
                profile_id: this.auditData.filterProfile,
                actor: this.auditData.filterActor
            });

            const data = await window.API.get(`/api/audit/records?${params.toString()}`);
            if (data.status === "error") throw new Error(data.error || "Query failed");

            this.auditData.records = data.records || [];
            this.auditData.total = data.total || 0;
            this.auditData.totalPages = Math.max(1, Math.ceil(this.auditData.total / this.auditData.limit));
        } catch (err) {
            this.auditData.error = err.message;
            if (typeof this.showToast === "function") {
                this.showToast(`Audit ledger query failed: ${err.message}`, "error");
            }
        } finally {
            this.auditData.loading = false;
        }
    },

    async verifyAuditLedger() {
        this.auditData.verification.checking = true;
        try {
            const data = await window.API.get("/api/audit/verify");
            this.auditData.verification.valid = !!data.valid;
            this.auditData.verification.total_records = data.total_records || 0;
            this.auditData.verification.checked_at = data.checked_at ? new Date(data.checked_at).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit", hour12: false }) : new Date().toLocaleTimeString();
            this.auditData.verification.head_hash = data.head_hash || "";
            this.auditData.verification.error = data.error || null;
            if (typeof this.showToast === "function") {
                if (data.valid) {
                    this.showToast(`Cryptographic chain verified (${data.total_records} blocks)`, "success");
                } else {
                    this.showToast(`Cryptographic integrity failure: ${data.error}`, "error");
                }
            }
        } catch (err) {
            this.auditData.verification.valid = false;
            this.auditData.verification.error = err.message;
            if (typeof this.showToast === "function") {
                this.showToast(`Verification check error: ${err.message}`, "error");
            }
        } finally {
            this.auditData.verification.checking = false;
        }
    },

    changeAuditPage(newPage) {
        if (newPage < 1 || newPage > this.auditData.totalPages) return;
        this.auditData.page = newPage;
        this.fetchAuditRecords();
    },

    applyAuditFilters() {
        this.auditData.page = 1;
        this.fetchAuditRecords();
    },

    resetAuditFilters() {
        this.auditData.filterAction = "";
        this.auditData.filterProfile = "";
        this.auditData.filterActor = "";
        this.auditData.page = 1;
        this.fetchAuditRecords();
    },

    inspectAuditRecord(rec) {
        this.auditData.selectedRecord = rec;
    },

    closeAuditDetailsModal() {
        this.auditData.selectedRecord = null;
    },

    copyHeadHash(hash) {
        if (!hash) return;
        navigator.clipboard.writeText(hash).then(() => {
            this.auditData.copiedHash = true;
            setTimeout(() => {
                this.auditData.copiedHash = false;
            }, 2000);
        });
    },

    getAuditActionBadgeClass(action) {
        if (!action) return "bg-slate-800 text-slate-300 border-slate-700";
        if (action.includes("SYNC_EXECUTED")) return "bg-emerald-950/80 text-emerald-300 border-emerald-800/60";
        if (action.includes("CONFLICT_RESOLVED")) return "bg-indigo-950/80 text-indigo-300 border-indigo-800/60";
        if (action.includes("DELETION_ACCEPTED")) return "bg-cyan-950/80 text-cyan-300 border-cyan-800/60";
        if (action.includes("DELETION_RESTORED")) return "bg-teal-950/80 text-teal-300 border-teal-800/60";
        if (action.includes("ROLLBACK")) return "bg-rose-950/80 text-rose-300 border-rose-800/60";
        if (action.includes("DISMISSED")) return "bg-slate-900 text-slate-400 border-slate-700/60";
        if (action.includes("CONFIG") || action.includes("SECRET")) return "bg-purple-950/80 text-purple-300 border-purple-800/60";
        return "bg-slate-800/80 text-slate-300 border-slate-700/50";
    },

    formatAuditTimestamp(ts) {
        if (!ts) return "--";
        try {
            const d = new Date(ts);
            if (isNaN(d)) return ts;
            return d.toLocaleDateString() + " " + d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit", hour12: false });
        } catch (e) {
            return ts;
        }
    },
    getProfileDisplayName(profileId, record) {
        if (record && record.profile_name) {
            return record.profile_name;
        }
        if (!profileId) return "--";
        if (this.profilesData && Array.isArray(this.profilesData.list)) {
            const found = this.profilesData.list.find(p => p.id === profileId);
            if (found && found.name) return found.name;
        }
        if (profileId === "default") return "Default";
        return profileId;
    },


    isAuditVerifying() {
        return Boolean(this.auditData?.verification?.checking);
    },

    isFirstAuditPage() {
        return Boolean(this.auditData?.page <= 1);
    },

    isLastAuditPage() {
        return Boolean(this.auditData?.page >= this.auditData?.totalPages);
    }
};
