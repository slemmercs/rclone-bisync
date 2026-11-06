/**
 * Interactive Diagnostic Manual & In-App User Guidance Module
 * Manages the slide-over drawer, curated safety policies, CLI flag catalog,
 * operational runbooks, and real-time self-diagnostic audit execution.
 */

window.HelpModule = {
    helpDrawer: {
        open: false,
        tab: "diagnostics", // 'diagnostics' | 'safety' | 'manual' | 'runbooks'
        search: "",
        diagnostics: null,
        loading: false,
        error: null,
        contextId: "global",
        contextTitle: "Fleet Overview",
        contextOnly: false,
    },

    detectActiveContext() {
        if (this.profileModal && this.profileModal.show) {
            return { id: "modal_profiles", title: "Profile Configuration", defaultTab: "manual" };
        }
        if (this.reviewState && this.reviewState.showModal) {
            return { id: "modal_review", title: "Conflict Review", defaultTab: "safety" };
        }
        if (this.globalSettingsModal && this.globalSettingsModal.show) {
            return { id: "modal_global_settings", title: "Global Settings", defaultTab: "safety" };
        }
        if (this.dedupeModal && this.dedupeModal.show) {
            return { id: "modal_dedupe", title: "Deduplication", defaultTab: "manual" };
        }
        if (this.showMcpModal) {
            return { id: "modal_mcp", title: "MCP Inspector", defaultTab: "diagnostics" };
        }
        if (this.showDaemonModal) {
            return { id: "modal_daemon", title: "Daemon Logs", defaultTab: "diagnostics" };
        }
        if (this.showDeletionsModal) {
            return { id: "modal_deletions", title: "Pending Deletions", defaultTab: "safety" };
        }

        switch (this.currentTab) {
            case "telemetry":
                return { id: "view_telemetry", title: "Engine & Telemetry", defaultTab: "diagnostics" };
            case "remotes":
                return { id: "view_remotes", title: "Storage Remotes", defaultTab: "runbooks" };
            case "explorer":
                return { id: "view_explorer", title: "File Explorer", defaultTab: "safety" };
            case "audit":
                return { id: "view_audit", title: "Audit Ledger", defaultTab: "safety" };
            case "fleet":
            default:
                return { id: "global", title: "Fleet Overview", defaultTab: "diagnostics" };
        }
    },

    openHelpDrawer(tab = null, topic = "") {
        let ctx = this.detectActiveContext();
        const validTabs = ["diagnostics", "safety", "manual", "runbooks"];
        const contextMap = {
            modal_profiles: { id: "modal_profiles", title: "Profile Configuration", defaultTab: "manual" },
            modal_review: { id: "modal_review", title: "Conflict Review", defaultTab: "safety" },
            modal_global_settings: { id: "modal_global_settings", title: "Global Settings", defaultTab: "safety" },
        };
        if (tab && contextMap[tab]) {
            ctx = contextMap[tab];
            tab = ctx.defaultTab;
        }
        this.helpDrawer.contextId = ctx.id;
        this.helpDrawer.contextTitle = ctx.title;
        this.helpDrawer.contextOnly = (ctx.id !== "global");
        this.helpDrawer.tab = (tab && validTabs.includes(tab)) ? tab : ctx.defaultTab;
        this.helpDrawer.search = topic || "";
        this.helpDrawer.open = true;

        if (this.helpDrawer.tab === "diagnostics" && !this.helpDrawer.diagnostics && !this.helpDrawer.loading) {
            this.fetchDiagnostics();
        }
    },

    toggleContextFilter() {
        this.helpDrawer.contextOnly = !this.helpDrawer.contextOnly;
    },

    closeHelpDrawer() {
        this.helpDrawer.open = false;
    },

    setHelpTab(tab) {
        this.helpDrawer.tab = tab;
        if (tab === "diagnostics" && !this.helpDrawer.diagnostics && !this.helpDrawer.loading) {
            this.fetchDiagnostics();
        }
    },

    async copyToClipboard(text, message = "Copied to clipboard") {
        try {
            await navigator.clipboard.writeText(text);
            this.showToast(message, "success");
        } catch (_) {
            const el = document.createElement("textarea");
            el.value = text;
            document.body.appendChild(el);
            el.select();
            document.execCommand("copy");
            document.body.removeChild(el);
            this.showToast(message, "success");
        }
    },

    get policiesList() {
        return (window.HelpCatalog && window.HelpCatalog.policies) ? window.HelpCatalog.policies : [];
    },

    filteredPolicies() {
        const q = (this.helpDrawer.search || "").trim().toLowerCase();
        let list = this.policiesList;
        if (this.helpDrawer.contextOnly && this.helpDrawer.contextId !== "global") {
            const ctxId = this.helpDrawer.contextId;
            const contextual = this.policiesList.filter(p => p.contexts && p.contexts.includes(ctxId));
            if (contextual.length > 0) list = contextual;
        }
        if (!q) return list;
        return list.filter(p =>
            p.title.toLowerCase().includes(q) ||
            p.tag.toLowerCase().includes(q) ||
            p.description.toLowerCase().includes(q) ||
            p.action.toLowerCase().includes(q)
        );
    },

    get flagsList() {
        return (window.HelpCatalog && window.HelpCatalog.flags) ? window.HelpCatalog.flags : [];
    },

    filteredFlags() {
        const q = (this.helpDrawer.search || "").trim().toLowerCase();
        let list = this.flagsList;
        if (this.helpDrawer.contextOnly && this.helpDrawer.contextId !== "global") {
            const ctxId = this.helpDrawer.contextId;
            const contextual = this.flagsList.filter(f => f.contexts && f.contexts.includes(ctxId));
            if (contextual.length > 0) list = contextual;
        }
        if (!q) return list;
        return list.filter(f =>
            f.flag.toLowerCase().includes(q) ||
            f.name.toLowerCase().includes(q) ||
            f.category.toLowerCase().includes(q) ||
            f.description.toLowerCase().includes(q)
        );
    },

    get runbooksList() {
        return (window.HelpCatalog && window.HelpCatalog.runbooks) ? window.HelpCatalog.runbooks : [];
    },

    filteredRunbooks() {
        const q = (this.helpDrawer.search || "").trim().toLowerCase();
        let list = this.runbooksList;
        if (this.helpDrawer.contextOnly && this.helpDrawer.contextId !== "global") {
            const ctxId = this.helpDrawer.contextId;
            const contextual = this.runbooksList.filter(r => r.contexts && r.contexts.includes(ctxId));
            if (contextual.length > 0) list = contextual;
        }
        if (!q) return list;
        return list.filter(r =>
            r.title.toLowerCase().includes(q) ||
            r.category.toLowerCase().includes(q) ||
            r.summary.toLowerCase().includes(q) ||
            (r.command && r.command.toLowerCase().includes(q))
        );
    }
};

// Global Keyboard Shortcuts (? / F1) for Help Drawer
window.addEventListener("keydown", (e) => {
    const tag = (e.target && e.target.tagName) ? e.target.tagName.toLowerCase() : "";
    if (tag === "input" || tag === "textarea" || tag === "select" || (e.target && e.target.isContentEditable)) {
        return;
    }
    if (e.key === "?" || e.key === "F1") {
        e.preventDefault();
        if (typeof window.openHelpDrawer === "function") {
            window.openHelpDrawer();
        }
    }
});
