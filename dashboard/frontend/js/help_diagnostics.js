/**
 * In-App Diagnostic Audit Execution Module
 * Provides live telemetry probing and diagnostic checks filtering.
 */
window.HelpDiagnosticsModule = {
    isDiagnosticsLoading() {
        return Boolean(this.helpDrawer && this.helpDrawer.loading);
    },

    async fetchDiagnostics() {
        this.helpDrawer.loading = true;
        this.helpDrawer.error = null;
        try {
            const data = await window.API.get("/api/diagnostics");
            this.helpDrawer.diagnostics = data;
            if (data.status === "error") {
                this.showToast("System diagnostics reported issues", "error");
            } else if (data.status === "warning") {
                this.showToast("System diagnostics reported warnings", "warning");
            } else {
                this.showToast("All diagnostic checks passed", "success");
            }
        } catch (err) {
            this.helpDrawer.error = err.message || "Failed to load diagnostics";
            this.showToast("Diagnostic audit failed: " + this.helpDrawer.error, "error");
        } finally {
            this.helpDrawer.loading = false;
        }
    },

    filteredDiagnosticChecks() {
        const checks = this.helpDrawer.diagnostics?.checks || [];
        const q = (this.helpDrawer.search || "").trim().toLowerCase();
        let list = checks;
        if (this.helpDrawer.contextOnly && this.helpDrawer.contextId !== "global") {
            const ctxId = this.helpDrawer.contextId;
            list = checks.filter(c => {
                if (ctxId === "view_remotes" && c.category === "remotes") return true;
                if ((ctxId === "view_explorer" || ctxId === "modal_profiles") && (c.category === "storage" || c.category === "safety")) return true;
                if (ctxId === "view_telemetry" && (c.category === "daemon" || c.category === "system")) return true;
                if (ctxId === "modal_mcp" && c.category === "daemon") return true;
                return false;
            });
            if (list.length === 0) list = checks;
        }
        if (!q) return list;
        return list.filter(c =>
            (c.name && c.name.toLowerCase().includes(q)) ||
            (c.message && c.message.toLowerCase().includes(q)) ||
            (c.category && c.category.toLowerCase().includes(q)) ||
            (c.remedy && c.remedy.toLowerCase().includes(q))
        );
    }
};
