/**
 * Profile Modal, Configuration Form, and Flag Overrides Module
 * Modular companion to profiles.js keeping all modules strictly under 400 lines of code.
 */

window.ProfilesModalModule = {
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
        if (typeof this.cancelDeleteProfile === "function") {
            this.cancelDeleteProfile();
        }
    },

    startNewProfile() {
        this.profileModal.isNew = true;
        this.profileModal.error = null;
        this.profileModal.editing = this.createDefaultProfile();
        this.profileModal.excludePatternsInput = ".git/**, *.tmp, .DS_Store";
        this.profileModal.tab = "edit";
        this.profileModal.show = true;
        if (typeof this.fetchRemotes === "function" && (!this.remotesData || !this.remotesData.remotes || this.remotesData.remotes.length === 0)) {
            this.fetchRemotes(false);
        }
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
        this.profileModal.show = true;
        if (typeof this.fetchRemotes === "function" && (!this.remotesData || !this.remotesData.remotes || this.remotesData.remotes.length === 0)) {
            this.fetchRemotes(false);
        }
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
    }
};
