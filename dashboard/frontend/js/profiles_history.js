/**
 * Profile History, Cloning, and Deletion Actions for Bisync Hub Dashboard
 * Modular companion to profiles.js keeping all modules strictly under 400 lines of code.
 */

window.ProfilesHistoryModule = {
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
        const executeRollback = async () => {
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
        };

        if (typeof this.askConfirmation === "function") {
            this.askConfirmation({
                title: "Confirm Profile Rollback",
                message: `Rollback profile to version ${targetVersion}?\nCurrent configuration will be overwritten with archived version settings.`,
                confirmText: "Rollback",
                confirmClass: "bg-amber-600 hover:bg-amber-500 shadow-amber-950/40",
                icon: "warning",
                onConfirm: executeRollback
            });
        } else if (confirm(`Rollback profile to version ${targetVersion}? Current configuration will be overwritten.`)) {
            await executeRollback();
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

    promptDeleteProfile(prof) {
        let target = prof;
        if (typeof prof === "string") {
            target = (this.profilesData.list || []).find(p => p.id === prof) || { id: prof, name: prof };
        }
        this.profileModal.deleteConfirm = {
            show: true,
            profile: target,
            deleting: false,
            error: null
        };
    },

    cancelDeleteProfile() {
        this.profileModal.deleteConfirm = {
            show: false,
            profile: null,
            deleting: false,
            error: null
        };
    },

    async executeDeleteProfile() {
        const prof = this.profileModal.deleteConfirm.profile;
        if (!prof || !prof.id) return;
        this.profileModal.deleteConfirm.deleting = true;
        this.profileModal.deleteConfirm.error = null;
        try {
            const res = await window.API.delete(`/api/profiles/${encodeURIComponent(prof.id)}`);
            if (res && res.status === "error") throw new Error(res.error || "Delete failed");
            if (typeof this.showToast === "function") {
                this.showToast(`Profile "${prof.name || prof.id}" deleted.`, "info");
            }
            this.profileModal.deleteConfirm.show = false;
            this.profileModal.deleteConfirm.profile = null;
            await this.fetchProfiles();
            if (this.profilesData.activeId === prof.id && this.profilesData.list.length > 0) {
                this.selectProfile(this.profilesData.list[0].id);
            }
        } catch (err) {
            this.profileModal.deleteConfirm.error = err.message;
            if (typeof this.showToast === "function") {
                this.showToast(`Delete failed: ${err.message}`, "error");
            }
        } finally {
            this.profileModal.deleteConfirm.deleting = false;
        }
    },

    deleteProfile(profId) {
        this.promptDeleteProfile(profId);
    }
};
