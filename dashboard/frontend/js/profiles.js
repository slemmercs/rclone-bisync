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
            max_delete: 50,
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
                this.profilesData.list = data.profiles || [];
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

    createDefaultProfile() {
        const randomHex = Math.random().toString(16).substring(2, 8);
        const remotes = this.remotesData && this.remotesData.remotes;
        const defaultRemote = (remotes && remotes.length > 0) ? remotes[0].name : "";
        const defaultBase = (this.profilesData && this.profilesData.homeDir) || "~";
        return {
            id: `prof_${randomHex}`,
            name: "",
            remote: defaultRemote,
            remote_path: "",
            local_path: defaultRemote ? `${defaultBase}/${defaultRemote}` : `${defaultBase}/sync`,
            exclude_patterns: [".git/**", "*.tmp", ".DS_Store"],
            schedule: "manual",
            review_mode: "deletions",
            notification_pref: "all",
            enabled: true,
            safety: { delete_threshold_pct: 10.0, failure_threshold: 3, canary_validation: true },
            flags: {
                dry_run: null,
                resync: null,
                force: null,
                conflict_resolve: null,
                max_delete: null,
                canary_validation: null,
                compare_mode: null,
                remove_empty_dirs: null,
                create_empty_dirs: null,
                filters_file_path: null,
                custom_flags: null,
                provider_options: {}
            }
        };
    },

    closeProfilesModal() {
        this.profileModal.show = false;
        this.profileModal.editing = this.createDefaultProfile();
        this.profileModal.tab = "list";
        this.profileModal.isNew = false;
        this.profileModal.history = [];
        this.profileModal.showClonePrompt = false;
        this.cancelDeleteProfile();
    },

    startNewProfile() {
        this.profileModal.isNew = true;
        this.profileModal.error = null;
        this.profileModal.editing = this.createDefaultProfile();
        this.profileModal.excludePatternsInput = ".git/**, *.tmp, .DS_Store";
        this.profileModal.tab = "edit";
    },

    startEditProfile(prof) {
        this.profileModal.isNew = false;
        this.profileModal.error = null;
        this.profileModal.editing = JSON.parse(JSON.stringify(prof));
        if (!this.profileModal.editing.safety) {
            this.profileModal.editing.safety = {
                delete_threshold_pct: 10.0,
                failure_threshold: 3,
                canary_validation: true
            };
        }
        if (!this.profileModal.editing.flags) {
            this.profileModal.editing.flags = {
                dry_run: null,
                resync: null,
                force: null,
                conflict_resolve: null,
                max_delete: null,
                canary_validation: null,
                compare_mode: null,
                remove_empty_dirs: null,
                create_empty_dirs: null,
                filters_file_path: null,
                custom_flags: null,
                provider_options: {}
            };
        } else if (!this.profileModal.editing.flags.provider_options) {
            this.profileModal.editing.flags.provider_options = {};
        }
        const patterns = this.profileModal.editing.exclude_patterns || [];
        this.profileModal.excludePatternsInput = patterns.join(", ");
        this.profileModal.tab = "edit";
    },

    async saveProfileForm() {
        const p = this.profileModal.editing;
        if (!p) return;
        if (this.profileModal.isNew && (!p.id || !p.id.trim())) {
            p.id = "prof_" + Math.random().toString(16).substring(2, 8);
        }
        if (typeof this.profileModal.excludePatternsInput === "string") {
            p.exclude_patterns = this.profileModal.excludePatternsInput
                .split(",")
                .map(s => s.trim())
                .filter(Boolean);
        }
        this.profileModal.saving = true;
        this.profileModal.error = null;

        try {
            if (this.profileModal.isNew) {
                const res = await window.API.post("/api/profiles", p);
                if (res.status === "error") throw new Error(res.error || "Creation failed");
                if (typeof this.showToast === "function") {
                    this.showToast(`Profile "${p.name}" created successfully.`, "success");
                }
            } else {
                const res = await window.API.put(`/api/profiles/${encodeURIComponent(p.id)}`, {
                    ...p,
                    reason: "Updated via Bisync Hub dashboard"
                });
                if (res.status === "error") throw new Error(res.error || "Update failed");
                if (typeof this.showToast === "function") {
                    this.showToast(`Profile "${p.name}" saved.`, "success");
                }
            }
            await this.fetchProfiles();
            this.profileModal.tab = "list";
            if (this.profilesData.activeId === p.id && typeof this.syncExplorerWithActiveProfile === "function") {
                this.syncExplorerWithActiveProfile();
            }
        } catch (err) {
            this.profileModal.error = err.message;
            if (typeof this.showToast === "function") {
                this.showToast(`Save failed: ${err.message}`, "error");
            }
        } finally {
            this.profileModal.saving = false;
        }
    },

    isFlagOverridden(flagKey) {
        const flags = this.profileModal?.editing?.flags;
        if (!flags) return false;
        return flags[flagKey] !== null && flags[flagKey] !== undefined;
    },

    getEffectiveFlagValue(flagKey) {
        const flags = this.profileModal?.editing?.flags;
        if (flags && flags[flagKey] !== null && flags[flagKey] !== undefined) {
            return flags[flagKey];
        }
        const globals = this.profilesData?.globalDefaults;
        return globals ? globals[flagKey] : null;
    },

    resetFlagOverride(flagKey) {
        if (!this.profileModal?.editing) return;
        if (!this.profileModal.editing.flags) {
            this.profileModal.editing.flags = {};
        }
        this.profileModal.editing.flags[flagKey] = null;
    },

    setFlagOverride(flagKey, val) {
        if (!this.profileModal?.editing) return;
        if (!this.profileModal.editing.flags) {
            this.profileModal.editing.flags = {};
        }
        this.profileModal.editing.flags[flagKey] = val;
    },

    getCustomFlagsValidationError(flagsStr) {
        if (!flagsStr || typeof flagsStr !== "string") return null;
        const prohibited = ["--config", "-c", "--use-mmap", "--rc", "--rc-addr", "--rc-user", "--rc-pass", "--log-file", "--json"];
        const tokens = flagsStr.trim().split(/\s+/);
        for (const t of tokens) {
            const flag = t.split("=")[0];
            if (prohibited.includes(flag.toLowerCase())) {
                return `Flag "${flag}" is restricted and cannot be overridden`;
            }
        }
        return null;
    },

    getEditingRemoteType() {
        if (!this.profileModal?.editing?.remote) return "generic";
        const rName = this.profileModal.editing.remote.replace(/:$/, "").trim();
        const found = (this.remotesData?.remotes || []).find(r => (r.name || "").replace(/:$/, "").trim() === rName);
        if (found && found.type) {
            return found.type.toLowerCase();
        }
        const lower = rName.toLowerCase();
        if (lower.includes("drive") || lower.includes("gdrive")) return "drive";
        if (lower.includes("onedrive")) return "onedrive";
        if (lower.includes("s3")) return "s3";
        if (lower.includes("dropbox")) return "dropbox";
        return "generic";
    },

    isTier1Provider() {
        const type = this.getEditingRemoteType();
        return ["drive", "onedrive", "s3", "dropbox"].includes(type);
    },

    getProviderOption(key, defVal = "") {
        if (!this.profileModal?.editing?.flags) return defVal;
        if (!this.profileModal.editing.flags.provider_options) {
            this.profileModal.editing.flags.provider_options = {};
        }
        const val = this.profileModal.editing.flags.provider_options[key];
        return (val !== undefined && val !== null) ? val : defVal;
    },

    setProviderOption(key, val) {
        if (!this.profileModal?.editing?.flags) return;
        if (!this.profileModal.editing.flags.provider_options) {
            this.profileModal.editing.flags.provider_options = {};
        }
        if (val === "" || val === null || val === undefined) {
            delete this.profileModal.editing.flags.provider_options[key];
        } else {
            this.profileModal.editing.flags.provider_options[key] = String(val);
        }
    },

    getActiveProfileRemoteType() {
        if (!this.activeProfile?.remote) return "generic";
        const rName = this.activeProfile.remote.replace(/:$/, "").trim();
        const found = (this.remotesData?.remotes || []).find(r => (r.name || "").replace(/:$/, "").trim() === rName);
        if (found && found.type) {
            return found.type.toLowerCase();
        }
        const lower = rName.toLowerCase();
        if (lower.includes("drive") || lower.includes("gdrive")) return "drive";
        if (lower.includes("onedrive")) return "onedrive";
        if (lower.includes("s3")) return "s3";
        if (lower.includes("dropbox")) return "dropbox";
        return "generic";
    },

    openActiveProfileSettings() {
        if (this.activeProfile) {
            this.startEditProfile(this.activeProfile);
            this.openProfilesModal("edit");
        }
    },

};

