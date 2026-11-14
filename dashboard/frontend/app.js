/**
 * Rclone Bisync Dashboard — Main Application Orchestrator
 * Modularized Alpine.js architecture. Zero build step, offline-ready.
 */

document.addEventListener("alpine:init", () => {
    function composeDashboard(base, ...modules) {
        for (const mod of modules) {
            if (!mod) continue;
            Object.defineProperties(base, Object.getOwnPropertyDescriptors(mod));
        }
        return base;
    }

    Alpine.data("dashboard", () => composeDashboard({
        currentTab: (() => {
            try {
                const saved = localStorage.getItem("bisync_active_tab");
                if (saved && ["fleet", "telemetry", "remotes", "explorer", "audit"].includes(saved)) return saved;
            } catch (_) {}
            return "fleet";
        })(),
        telemetry: {
            daemon_health: { status: "evaluating", response_time_ms: 0 },
            notification_state: {},
            metrics: [],
            bisync_logs: [],
            daemon_logs: [],
            config: {},
            is_syncing: false,
            transfer_stats: null
        },
        currentLogLimit: 40,
        maxLogLimit: 300,
        telemetryLogFilter: "all", // 'all' | 'file_ops' | 'sync' | 'audit'
        actionLoading: null,
        configExpanded: false,

        showDeletionsModal: false,
        deletionsData: { total_count: 0, source: "", target_action: "", files: [] },
        deletionsSearch: "",

        // Correlated Daemon Log Modal State (Branch 6)
        showDaemonModal: false,
        daemonModalLoading: false,
        daemonModalRun: null,
        daemonModalLogs: [],
        daemonModalSearch: "",
        daemonModalLevelFilter: "all",
        daemonModalCopied: false,
        daemonModalCopiedJSON: false,
        daemonModalError: null,


        // Dual-Pane File Explorer State
        explorer: {
            left: { target: "local", path: "", items: [], loading: false, error: null, hasCanary: false, totalSize: 0, itemCount: 0, copied: false, search: "", sortBy: "name", sortAsc: true, foldersFirst: true },
            right: { target: "", path: "", items: [], loading: false, error: null, hasCanary: false, totalSize: 0, itemCount: 0, copied: false, search: "", sortBy: "name", sortAsc: true, foldersFirst: true },
            draggedItem: null,
            draggedSourcePane: null,
            isDragOverLeft: false,
            isDragOverRight: false,
            isSwapped: false
        },

        deleteConfirmMode: localStorage.getItem("rclone_delete_confirm_mode") || "double",
        deleteModal: {
            show: false,
            step: 1,
            paneKey: null,
            item: null,
            deleting: false
        },

        mkdirModal: {
            show: false,
            paneKey: null,
            dirName: "",
            creating: false
        },

        // File Preview Modal State
        filePreviewModal: {
            show: false,
            loading: false,
            error: null,
            remote: "",
            path: "",
            name: "",
            preview: null
        },

        // File Context Menu State
        contextMenu: { show: false, x: 0, y: 0, paneKey: null, item: null },

        // Upload Dock & Remediation State
        uploadDock: { pending_count: 0, is_sync_active: false, items: [] },
        remediationCandidate: null,

        // Interactive Sync Review State
        reviewState: {
            active: false,
            loading: false,
            error: null,
            session: null,
            executing: false,
            rollingBack: false,
            dismissing: false,
            filter: "all", // 'all' | 'deletions' | 'conflicts'
            statusFilter: "all", // 'all' | 'pending' | 'decided'
            search: "",
            showModal: false,
            lastResult: null
        },

        // Lifecycle & Navigation
        async initDashboard() {
            initPerformanceChart();
            this.initPolling();
            window.openHelpDrawer = (tab, topic) => this.openHelpDrawer(tab, topic);
            if (!this.isPollingPaused) await this.poll();
            if (typeof this.fetchProfiles === "function") await this.fetchProfiles();
            if (typeof this.fetchConfigEncryptionStatus === "function") this.fetchConfigEncryptionStatus();
            await this.fetchRemotes(false);
            this.fetchReviewSession(false);
            if (typeof this.fetchBandwidthLimit === "function") this.fetchBandwidthLimit();
            if (typeof this.fetchCoreStats === "function") this.fetchCoreStats();
            if (typeof this.fetchBisyncLogs === "function") this.fetchBisyncLogs(1);
            if (typeof this.fetchAuditRecords === "function") this.fetchAuditRecords();
            if (this.currentTab === "remotes") this.fetchRemotes(true);
            else if (this.currentTab === "explorer" && typeof this.initExplorer === "function") await this.initExplorer();
            else if (this.currentTab === "audit" && typeof this.initAudit === "function") this.initAudit();
            this.setupLogScroll("log-bisync");
        },

        async triggerAction(actionStr) {
            this.actionLoading = actionStr;
            try {
                const payload = { action: actionStr };
                if (this.profilesData && this.profilesData.activeId) {
                    payload.profile_id = this.profilesData.activeId;
                }
                const data = await window.API.post("/api/action", payload);
                if (data.requires_resync) {
                    const profId = payload.profile_id || "selected profile";
                    const executeResync = async () => {
                        await this.triggerAction("force_resync");
                    };
                    if (typeof this.askConfirmation === "function") {
                        this.askConfirmation({
                            title: "Initial Resync Required",
                            icon: "sync",
                            message: `Profile "${profId}" has never been synchronized.\n\nInitial synchronization requires --resync to establish baseline listings.\n\nProceed with --resync now?`,
                            confirmText: "Proceed with Resync",
                            confirmClass: "bg-indigo-600 hover:bg-indigo-500",
                            onConfirm: executeResync
                        });
                    } else if (confirm(`Profile "${profId}" has never been synced. Initial synchronization requires --resync to establish baseline listings.\n\nProceed with --resync now?`)) {
                        await executeResync();
                    }
                    return;
                }
                if (data.status === "error") {
                    const errDetail = data.msg || data.error || "Request failed";
                    this.showToast("Action Error: " + errDetail, "error");
                } else {
                    this.showToast(data.msg || `Action "${actionStr}" completed successfully.`, "success");
                }
                await this.poll();
                if (typeof this.fetchBisyncLogs === "function") this.fetchBisyncLogs(1, true);
            } catch (err) {
                this.showToast("Action failed: " + err.message, "error");
            } finally {
                setTimeout(() => { this.actionLoading = null; }, 1200);
            }
        },

        switchTab(tab) {
            this.currentTab = tab;
            try { localStorage.setItem("bisync_active_tab", tab); } catch (_) {}
            if (tab === "telemetry" && typeof this.fetchBisyncLogs === "function") this.fetchBisyncLogs(1);
            else if (tab === "remotes") this.fetchRemotes(true);
            else if (tab === "explorer" && typeof this.initExplorer === "function") this.initExplorer();
            else if (tab === "audit" && typeof this.initAudit === "function") this.initAudit();
        },
    },
        window.FleetModule,
        window.AppStateModule,
        window.ProfilesModule,
        window.ProfilesModalModule,
        window.ProviderOptionsModule,
        window.FlagsBuilderModule, window.ProfilesHistoryModule,
        window.RemotesModule, window.RemotesModalsModule, window.RemotesWizardModule,
        window.ExplorerModule, window.ExplorerActionsModule, window.ExplorerPreviewModule, window.ExplorerOperationsModule, window.ExplorerCompareModule,
        window.ReviewModule, window.ReviewActionsModule,
        window.LogsClusteringModule, window.LogsModule, window.LogsDaemonModule, window.LogDockModule, window.AuditModule,
        window.SettingsModule, window.BandwidthModule,
        window.McpModule, window.PollingModule,
        window.SystemHealthModule,
        window.HelpDiagnosticsModule, window.HelpModule
    ));
});
