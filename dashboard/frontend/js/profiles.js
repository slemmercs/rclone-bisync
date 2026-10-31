/**
 * Multi-Profile Workspace Manager Module for Bisync Hub Dashboard
 * Provides profile listing, switching, modal management, version history rollback,
 * cloning, and independent on-demand sync execution.
 */

window.ProfilesModule = {
    profilesData: {
        list: [],
        activeId: localStorage.getItem("rclone_active_profile_id") || "default",
        homeDir: "",
        globalDefaults: {
            max_delete: 0,
            conflict_resolve: "path1",
            compare_mode: "size,modtime",
            dry_run: false,
            resync: false,
            force: false,
            canary_validation: true,
            remove_empty_dirs: false,
            create_empty_dirs: true,
            filters_file_path: "",
            custom_flags: ""
        },
        loading: false,
        error: null,
        syncingId: null
    },
    profileSwitcherOpen: false,
    toggleProfileSwitcher() { this.profileSwitcherOpen = !this.profileSwitcherOpen; },
    closeProfileSwitcher() { this.profileSwitcherOpen = false; },

    profileModal: {
        show: false,
        tab: "list", // 'list' | 'edit' | 'history'
        editing: {
            id: "",
            name: "",
            remote: "",
            remote_path: "",
            local_path: "",
            exclude_patterns: [],
            schedule: "manual",
            review_mode: "deletions",
            notification_pref: "all",
            enabled: true,
            safety: {
                delete_threshold_pct: 10.0,
                failure_threshold: 3,
                canary_validation: true
            }
        },
        excludePatternsInput: ".git/**, *.tmp, .DS_Store",
        isNew: false,
        history: [],
        historyLoading: false,
        saving: false,
        error: null,
        cloneTarget: { id: "", name: "" },
        showClonePrompt: false,
        deleteConfirm: {
            show: false,
            profile: null,
            deleting: false,
            error: null
        }
    },

    get activeProfile() {
        if (!this.profilesData.list || this.profilesData.list.length === 0) {
            return null;
        }
        return this.profilesData.list.find(p => p.id === this.profilesData.activeId) || this.profilesData.list[0];
    },

    isProfileSaving() { return !!(this.profileModal && this.profileModal.saving); },
    isProfileDeleting() { return !!(this.profileModal && this.profileModal.deleteConfirm && this.profileModal.deleteConfirm.deleting); },
    isProfileSyncing(item) {
        if (!item) return false;
        return !!(this.profilesData && this.profilesData.syncingId === (typeof item === "string" ? item : item.id));
    },
    isProfileEditingExisting() { return !(this.profileModal && this.profileModal.isNew); },
    hasDiscoveredRemotes() { return !!(this.remotesData && this.remotesData.remotes && this.remotesData.remotes.length > 0); },

    async fetchProfiles() {
        this.profilesData.loading = true;
        this.profilesData.error = null;
        try {
            const data = await window.API.get("/api/profiles");
            if (data && data.status === "success") {
                if (data.home_dir) {
                    this.profilesData.homeDir = data.home_dir;
                }
                if (data.global_defaults) {
                    this.profilesData.globalDefaults = data.global_defaults;
                }
                const rawList = data.profiles || [];
                rawList.sort((a, b) => {
                    const tA = new Date(a.created_at || 0).getTime();
                    const tB = new Date(b.created_at || 0).getTime();
                    if (tA !== tB) return tA - tB;
                    return (a.name || "").localeCompare(b.name || "");
                });
                this.profilesData.list = rawList;
                if (this.profilesData.list.length > 0) {
                    const exists = this.profilesData.list.some(p => p.id === this.profilesData.activeId);
                    if (!exists) {
                        this.profilesData.activeId = this.profilesData.list[0].id;
                        localStorage.setItem("rclone_active_profile_id", this.profilesData.activeId);
                    }
                }
            }
        } catch (err) {
            this.profilesData.error = err.message;
            console.error("Failed to fetch profiles:", err);
        } finally {
            this.profilesData.loading = false;
        }
    },

    selectProfile(id) {
        if (this.profilesData.activeId === id) return;
        this.profilesData.activeId = id;
        localStorage.setItem("rclone_active_profile_id", id);
        if (typeof this.showToast === "function") {
            const p = this.profilesData.list.find(item => item.id === id);
            this.showToast(`Active profile switched to "${p ? p.name : id}"`, "info");
        }
        if (typeof this.syncExplorerWithActiveProfile === "function") {
            this.syncExplorerWithActiveProfile();
        } else if (this.currentTab === "explorer" && typeof this.initExplorer === "function") {
            this.initExplorer();
        }
        if (this.currentTab === "audit" && typeof this.fetchAuditRecords === "function") {
            this.auditData.filterProfile = id;
            this.fetchAuditRecords();
        }
    },

    openProfilesModal(tab = "list") {
        this.profileModal.tab = tab;
        this.profileModal.error = null;
        this.profileModal.show = true;
        this.fetchProfiles();
        if (typeof this.fetchRemotes === "function" && (!this.remotesData || !this.remotesData.remotes || this.remotesData.remotes.length === 0)) {
            this.fetchRemotes(false);
        }
    },

    // Modal form, configuration lifecycle, and flag overrides extracted to profiles_modal.js
    // History, cloning, and deletion actions extracted to profiles_history.js
};


