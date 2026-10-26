/**
 * Multi-Profile Workspace Manager Module for Bisync Hub Dashboard
 * Provides profile listing, switching, modal management, version history rollback,
 * cloning, and independent on-demand sync execution.
 */

window.ProfilesModule = {
    profilesData: {
        list: [],
        activeId: localStorage.getItem("rclone_active_profile_id") || "default",
        loading: false,
        error: null,
        syncingId: null
    },

    profileModal: {
        show: false,
        tab: "list", // 'list' | 'edit' | 'history'
        editing: {
            id: "",
            name: "",
            remote: "",
            remote_path: "",
            local_path: "",
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
        isNew: false,
        history: [],
        historyLoading: false,
        saving: false,
        error: null,
        cloneTarget: { id: "", name: "" },
        showClonePrompt: false
    },

    get activeProfile() {
        if (!this.profilesData.list || this.profilesData.list.length === 0) {
            return null;
        }
        return this.profilesData.list.find(p => p.id === this.profilesData.activeId) || this.profilesData.list[0];
    },

    isProfileSaving() {
        return !!(this.profileModal && this.profileModal.saving);
    },

    isProfileSyncing(item) {
        if (!item) return false;
        const id = typeof item === "string" ? item : item.id;
        return !!(this.profilesData && this.profilesData.syncingId === id);
    },

    isProfileEditingExisting() {
        return !(this.profileModal && this.profileModal.isNew);
    },

    hasDiscoveredRemotes() {
        return !!(this.remotesData && this.remotesData.remotes && this.remotesData.remotes.length > 0);
    },

    async fetchProfiles() {
        this.profilesData.loading = true;
        this.profilesData.error = null;
        try {
            const data = await window.API.get("/api/profiles");
            if (data && data.status === "success") {
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
        if (this.currentTab === "explorer" && typeof this.initExplorer === "function") {
            this.initExplorer();
        } else if (this.currentTab === "audit" && typeof this.fetchAuditRecords === "function") {
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

    closeProfilesModal() {
        this.profileModal.show = false;
        this.profileModal.editing = null;
        this.profileModal.isNew = false;
        this.profileModal.history = [];
        this.profileModal.showClonePrompt = false;
    },

    startNewProfile() {
        this.profileModal.isNew = true;
        this.profileModal.error = null;
        const randomHex = Math.random().toString(16).substring(2, 8);
        const defaultRemote = (this.remotesData && this.remotesData.remotes && this.remotesData.remotes.length > 0)
            ? this.remotesData.remotes[0].name
            : "";
        this.profileModal.editing = {
            id: `prof_${randomHex}`,
            name: "",
            remote: defaultRemote,
            remote_path: "",
            local_path: "",
            schedule: "manual",
            review_mode: "deletions",
            notification_pref: "all",
            enabled: true,
            safety: {
                delete_threshold_pct: 10.0,
                failure_threshold: 3,
                canary_validation: true
            }
        };
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
        this.profileModal.tab = "edit";
    },

    async saveProfileForm() {
        const p = this.profileModal.editing;
        if (!p) return;
        if (this.profileModal.isNew && (!p.id || !p.id.trim())) {
            p.id = "prof_" + Math.random().toString(16).substring(2, 8);
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
        } catch (err) {
            this.profileModal.error = err.message;
            if (typeof this.showToast === "function") {
                this.showToast(`Save failed: ${err.message}`, "error");
            }
        } finally {
            this.profileModal.saving = false;
        }
    },

    promptCloneProfile(prof) {
        this.profileModal.editing = prof;
        const randomHex = Math.random().toString(16).substring(2, 8);
        this.profileModal.cloneTarget = {
            id: `prof_${randomHex}`,
            name: `${prof.name} (Copy)`
        };
        this.profileModal.showClonePrompt = true;
    },

    async submitCloneProfile() {
        const source = this.profileModal.editing;
        const target = this.profileModal.cloneTarget;
        if (!source || !target.id) return;

        this.profileModal.saving = true;
        try {
            const res = await window.API.post(`/api/profiles/${encodeURIComponent(source.id)}/clone`, target);
            if (res.status === "error") throw new Error(res.error || "Clone failed");
            if (typeof this.showToast === "function") {
                this.showToast(`Profile cloned to "${target.name}".`, "success");
            }
            this.profileModal.showClonePrompt = false;
            await this.fetchProfiles();
        } catch (err) {
            this.profileModal.error = err.message;
            if (typeof this.showToast === "function") {
                this.showToast(`Clone failed: ${err.message}`, "error");
            }
        } finally {
            this.profileModal.saving = false;
        }
    },

    async viewProfileHistory(prof) {
        this.profileModal.editing = prof;
        this.profileModal.history = [];
        this.profileModal.historyLoading = true;
        this.profileModal.tab = "history";

        try {
            const data = await window.API.get(`/api/profiles/${encodeURIComponent(prof.id)}/history`);
            this.profileModal.history = (data && data.history) || [];
        } catch (err) {
            this.profileModal.error = err.message;
            if (typeof this.showToast === "function") {
                this.showToast(`Failed to load history: ${err.message}`, "error");
            }
        } finally {
            this.profileModal.historyLoading = false;
        }
    },

    async rollbackProfile(profId, targetVersion) {
        if (!confirm(`Rollback profile to version ${targetVersion}? Current configuration will be overwritten.`)) {
            return;
        }
        this.profileModal.saving = true;
        try {
            const res = await window.API.post(`/api/profiles/${encodeURIComponent(profId)}/rollback`, {
                target_version: targetVersion
            });
            if (res.status === "error") throw new Error(res.error || "Rollback failed");
            if (typeof this.showToast === "function") {
                this.showToast(`Profile rolled back to version ${targetVersion}.`, "success");
            }
            await this.fetchProfiles();
            const updated = this.profilesData.list.find(p => p.id === profId);
            if (updated) {
                await this.viewProfileHistory(updated);
            }
        } catch (err) {
            this.profileModal.error = err.message;
            if (typeof this.showToast === "function") {
                this.showToast(`Rollback failed: ${err.message}`, "error");
            }
        } finally {
            this.profileModal.saving = false;
        }
    },

    async triggerProfileSync(profId, force = false) {
        this.profilesData.syncingId = profId;
        try {
            const res = await window.API.post(`/api/profiles/${encodeURIComponent(profId)}/sync`, { force });
            if (res.status === "error") throw new Error(res.error || "Sync trigger failed");
            if (typeof this.showToast === "function") {
                this.showToast(`Sync triggered for profile "${profId}".`, "success");
            }
            await this.fetchProfiles();
        } catch (err) {
            if (typeof this.showToast === "function") {
                this.showToast(`Sync failed: ${err.message}`, "error");
            }
        } finally {
            this.profilesData.syncingId = null;
        }
    },

    async deleteProfile(profId) {
        if (!confirm(`Are you sure you want to permanently delete profile "${profId}"? Background sync tickers will be stopped.`)) {
            return;
        }
        try {
            const res = await window.API.delete(`/api/profiles/${encodeURIComponent(profId)}`);
            if (res && res.status === "error") throw new Error(res.error || "Delete failed");
            if (typeof this.showToast === "function") {
                this.showToast(`Profile "${profId}" deleted.`, "info");
            }
            await this.fetchProfiles();
        } catch (err) {
            if (typeof this.showToast === "function") {
                this.showToast(`Delete failed: ${err.message}`, "error");
            }
        }
    }
};

