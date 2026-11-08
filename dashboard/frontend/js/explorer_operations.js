/**
 * Dual-Pane File Explorer Domain Operations Module for Bisync Hub Dashboard
 * Handles Move/Rename, Purge Directory, Public Links, and Upload Dock interactions.
 */

window.ExplorerOperationsModule = {
    moveModal: {
        show: false,
        paneKey: null,
        item: null,
        newName: "",
        moving: false
    },

    mkdirModal: {
        show: false,
        paneKey: null,
        dirName: "",
        creating: false
    },

    uploadDockModal: {
        show: false,
        loading: false
    },

    promptCreateFolder(paneKey) {
        this.mkdirModal = {
            show: true,
            paneKey: paneKey || "left",
            dirName: "",
            creating: false
        };
        if (this.$nextTick) {
            this.$nextTick(() => {
                const el = document.getElementById("mkdir-input");
                if (el) el.focus();
            });
        }
    },

    cancelCreateFolder() {
        this.mkdirModal = { show: false, paneKey: null, dirName: "", creating: false };
    },

    isMkdirDisabled() {
        return Boolean(this.mkdirModal.creating || !this.mkdirModal.dirName || this.mkdirModal.dirName.trim() === "");
    },

    async executeCreateFolder() {
        const { paneKey, dirName } = this.mkdirModal;
        if (!paneKey || !dirName || dirName.trim() === "") return;
        const trimmed = dirName.trim();
        if (trimmed === "RCLONE_TEST") {
            if (typeof this.showToast === "function") {
                this.showToast("Cannot create directory named canary test file (RCLONE_TEST)", "error");
            }
            return;
        }
        if (trimmed.includes("..") || trimmed.includes("/") || trimmed.includes("\\")) {
            if (typeof this.showToast === "function") {
                this.showToast("Invalid folder name: slashes and path traversal forbidden", "error");
            }
            return;
        }

        this.mkdirModal.creating = true;
        const remote = this.explorer[paneKey].target;
        const currentPanePath = this.explorer[paneKey].path || "";
        const fullDirPath = currentPanePath ? `${currentPanePath}/${trimmed}`.replace(/^\/+/, "") : trimmed;
        const profileId = this.profilesData?.activeId || (this.activeProfile ? this.activeProfile.id : "");

        try {
            const data = await window.API.post("/api/operations/mkdir", {
                remote,
                path: fullDirPath,
                profile_id: profileId
            });

            if (data.status === "success") {
                if (typeof this.showToast === "function") {
                    this.showToast(`Folder "${trimmed}" created successfully`, "success");
                }
                this.cancelCreateFolder();
                this.fetchExplorerDirectory(paneKey, remote, currentPanePath);
            } else {
                if (typeof this.showToast === "function") {
                    this.showToast(`Create folder failed: ${data.error || 'Unknown error'}`, "error");
                }
            }
        } catch (err) {
            if (typeof this.showToast === "function") {
                this.showToast(`Create folder failed: ${err.message}`, "error");
            }
        } finally {
            this.mkdirModal.creating = false;
        }
    },

    promptMoveItem(paneKey, item) {
        if (!item) return;
        if (item.Name === "RCLONE_TEST") {
            if (typeof this.showToast === "function") {
                this.showToast("Canary file (RCLONE_TEST) cannot be moved or renamed.", "warning");
            }
            return;
        }
        this.moveModal = {
            show: true,
            paneKey,
            item,
            newName: item.Name,
            moving: false
        };
    },

    cancelMove() {
        this.moveModal = { show: false, paneKey: null, item: null, newName: "", moving: false };
    },

    isMoveDisabled() {
        return Boolean(this.moveModal.moving || !this.moveModal.newName || this.moveModal.newName.trim() === "");
    },

    async executeMoveItem() {
        const { paneKey, item, newName } = this.moveModal;
        if (!item || !paneKey || !newName || newName.trim() === "") return;
        const trimmed = newName.trim();
        if (trimmed === item.Name) {
            this.cancelMove();
            return;
        }
        if (trimmed === "RCLONE_TEST") {
            if (typeof this.showToast === "function") {
                this.showToast("Cannot rename to canary file name (RCLONE_TEST)", "error");
            }
            return;
        }

        this.moveModal.moving = true;
        const remote = this.explorer[paneKey].target;
        const currentPanePath = this.explorer[paneKey].path || "";

        let srcPath = item.Path || item.Name;
        if (currentPanePath && !srcPath.startsWith(currentPanePath)) {
            srcPath = `${currentPanePath}/${srcPath}`.replace(/^\/+/, "");
        }

        let dstPath = currentPanePath ? `${currentPanePath}/${trimmed}`.replace(/^\/+/, "") : trimmed;
        const profileId = this.profilesData?.activeId || (this.activeProfile ? this.activeProfile.id : "");

        try {
            const data = await window.API.post("/api/operations/move", {
                remote,
                src_remote: remote,
                dst_remote: remote,
                src_path: srcPath,
                dst_path: dstPath,
                is_dir: !!item.IsDir,
                profile_id: profileId
            });

            if (data.status === "success") {
                if (typeof this.showToast === "function") {
                    this.showToast(`Moved to "${trimmed}" successfully`, "success");
                }
                this.cancelMove();
                this.fetchExplorerDirectory(paneKey, remote, currentPanePath);
            } else {
                if (typeof this.showToast === "function") {
                    this.showToast(`Move failed: ${data.error || 'Unknown error'}`, "error");
                }
            }
        } catch (err) {
            if (typeof this.showToast === "function") {
                this.showToast(`Move failed: ${err.message}`, "error");
            }
        } finally {
            this.moveModal.moving = false;
        }
    },

    promptPurgeDirectory(paneKey, item) {
        if (!item || !item.IsDir) return;
        const remote = this.explorer[paneKey].target;
        const currentPanePath = this.explorer[paneKey].path || "";
        let path = item.Path || item.Name;
        if (currentPanePath && !path.startsWith(currentPanePath)) {
            path = `${currentPanePath}/${path}`.replace(/^\/+/, "");
        }

        const executePurge = async () => {
            const profileId = this.profilesData?.activeId || (this.activeProfile ? this.activeProfile.id : "");
            try {
                const data = await window.API.post("/api/operations/purge", {
                    remote,
                    path,
                    profile_id: profileId
                });
                if (data.status === "success") {
                    if (typeof this.showToast === "function") {
                        this.showToast(`Directory "${item.Name}" purged successfully`, "success");
                    }
                    this.fetchExplorerDirectory(paneKey, remote, currentPanePath);
                } else {
                    if (typeof this.showToast === "function") {
                        this.showToast(`Purge failed: ${data.error || 'Unknown error'}`, "error");
                    }
                }
            } catch (err) {
                if (typeof this.showToast === "function") {
                    this.showToast(`Purge failed: ${err.message}`, "error");
                }
            }
        };

        if (typeof this.askConfirmation === "function") {
            this.askConfirmation({
                title: "Confirm Directory Purge",
                message: `PERMANENT DELETION: Purge directory "${item.Name}" and all contents? This cannot be undone.\nCanary safeguards (RCLONE_TEST) will block deletion if present.`,
                confirmText: "Purge Directory",
                confirmClass: "bg-rose-600 hover:bg-rose-500 shadow-rose-950/40",
                icon: "warning",
                onConfirm: executePurge
            });
        } else if (confirm(`PERMANENT DELETION: Purge directory "${item.Name}" on ${remote}?`)) {
            executePurge();
        }
    },

    async openOnRemote(paneKey, item) {
        if (!item) return;
        const remote = this.explorer[paneKey].target;
        if (remote === "local") {
            if (typeof this.showToast === "function") {
                this.showToast("Local filesystem does not support cloud web links", "info");
            }
            return;
        }
        const currentPanePath = this.explorer[paneKey].path || "";
        let path = item.Path || item.Name;
        if (currentPanePath && !path.startsWith(currentPanePath)) {
            path = `${currentPanePath}/${path}`.replace(/^\/+/, "");
        }
        const profileId = this.profilesData?.activeId || (this.activeProfile ? this.activeProfile.id : "");

        try {
            const data = await window.API.post("/api/operations/publiclink", {
                remote,
                path,
                profile_id: profileId
            });

            if (data && data.supported && data.url) {
                window.open(data.url, "_blank");
            } else if (data && !data.supported) {
                if (typeof this.showToast === "function") {
                    this.showToast("Web view link unsupported for this storage provider", "info");
                }
            } else {
                if (typeof this.showToast === "function") {
                    this.showToast(data.error || "Failed to retrieve public link", "error");
                }
            }
        } catch (err) {
            if (typeof this.showToast === "function") {
                this.showToast("Failed to fetch public link: " + err.message, "error");
            }
        }
    },

    async fetchUploadDockStatus() {
        try {
            const profileId = this.profilesData?.activeId || (this.activeProfile ? this.activeProfile.id : "");
            const res = await window.API.get(`/api/explorer/dock?profile_id=${encodeURIComponent(profileId)}`);
            if (res && typeof res.pending_count === "number") {
                this.uploadDock = res;
            }
        } catch (_) {}
    },

    async clearCompletedUploads() {
        try {
            const profileId = this.profilesData?.activeId || (this.activeProfile ? this.activeProfile.id : "");
            await window.API.delete(`/api/explorer/dock?profile_id=${encodeURIComponent(profileId)}`);
            await this.fetchUploadDockStatus();
            if (typeof this.showToast === "function") {
                this.showToast("Cleared completed uploads from dock", "info");
            }
        } catch (err) {
            if (typeof this.showToast === "function") {
                this.showToast("Failed to clear dock: " + err.message, "error");
            }
        }
    },

    openUploadDockModal() {
        this.uploadDockModal.show = true;
        this.fetchUploadDockStatus();
    },

    closeUploadDockModal() {
        this.uploadDockModal.show = false;
    },

    clearRemediationCandidate() {
        this.remediationCandidate = null;
        if (typeof this.showToast === "function") {
            this.showToast("Remediation cockpit cleared", "info");
        }
    }
};
