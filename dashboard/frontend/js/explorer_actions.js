/**
 * Dual-Pane File Explorer Actions & Interaction Module for Bisync Hub Dashboard
 * Manages 2-step deletion modal workflows and drag-and-drop file transfers.
 */

window.ExplorerActionsModule = {
    setDeleteConfirmMode(mode) {
        this.deleteConfirmMode = mode;
        try {
            localStorage.setItem("rclone_delete_confirm_mode", mode);
        } catch (e) {}
    },

    promptDeleteItem(paneKey, item) {
        if (!item || item.Name === "RCLONE_TEST") {
            this.showToast("Canary file (RCLONE_TEST) cannot be deleted.", "warning");
            return;
        }
        if (this.deleteConfirmMode === "none") {
            this.executeDeleteItem(paneKey, item);
        } else if (this.deleteConfirmMode === "single") {
            this.deleteModal = {
                show: true,
                step: 2,
                paneKey,
                item,
                deleting: false
            };
        } else {
            // Default: double confirmation
            this.deleteModal = {
                show: true,
                step: 1,
                paneKey,
                item,
                deleting: false
            };
        }
    },

    advanceDeleteStep() {
        if (this.deleteModal.step === 1) {
            this.deleteModal.step = 2;
        } else {
            this.executeDeleteItem(this.deleteModal.paneKey, this.deleteModal.item);
        }
    },

    cancelDelete() {
        this.deleteModal = { show: false, item: null, paneKey: null, step: 1, deleting: false };
    },

    async executeDeleteItem(paneKey, item) {
        if (!item || !paneKey) return;
        this.deleteModal.deleting = true;
        const remote = this.explorer[paneKey].target;
        let path = item.Path || item.Name;
        const currentPanePath = this.explorer[paneKey].path;
        if (currentPanePath && !path.startsWith(currentPanePath)) {
            path = `${currentPanePath}/${path}`.replace(/^\/+/, "");
        }
        const is_dir = !!item.IsDir;

        try {
            const profileId = this.profilesData?.activeId || (this.activeProfile ? this.activeProfile.id : "");
            const data = await window.API.post("/api/browse/delete", {
                remote,
                path,
                is_dir,
                profile_id: profileId
            });
            if (data.status === "success") {
                this.showToast(`Deleted "${item.Name}" successfully`, "success");
                this.cancelDelete();
                this.fetchExplorerDirectory(paneKey, remote, this.explorer[paneKey].path);
            } else {
                this.showToast(`Delete failed: ${data.error || 'Unknown error'}`, "error");
            }
        } catch (err) {
            this.showToast(`Delete failed: ${err.message}`, "error");
        } finally {
            this.deleteModal.deleting = false;
        }
    },

    onDragStart(sourcePaneKey, item, event) {
        this.explorer.draggedItem = item;
        this.explorer.draggedSourcePane = sourcePaneKey;
        if (event.dataTransfer) {
            event.dataTransfer.effectAllowed = "copy";
            try {
                event.dataTransfer.setData("text/plain", item.Path || item.Name);
            } catch (e) {}
        }
    },

    onDragOver(targetPaneKey, event) {
        if (this.explorer.draggedSourcePane && this.explorer.draggedSourcePane !== targetPaneKey) {
            event.preventDefault();
            if (event.dataTransfer) {
                event.dataTransfer.dropEffect = "copy";
            }
            if (targetPaneKey === "left") this.explorer.isDragOverLeft = true;
            if (targetPaneKey === "right") this.explorer.isDragOverRight = true;
        }
    },

    onDragLeave(targetPaneKey, event) {
        if (targetPaneKey === "left") this.explorer.isDragOverLeft = false;
        if (targetPaneKey === "right") this.explorer.isDragOverRight = false;
    },

    onDrop(targetPaneKey, event) {
        event.preventDefault();
        this.explorer.isDragOverLeft = false;
        this.explorer.isDragOverRight = false;

        const item = this.explorer.draggedItem;
        const srcPane = this.explorer.draggedSourcePane;
        this.explorer.draggedItem = null;
        this.explorer.draggedSourcePane = null;

        if (item && srcPane && srcPane !== targetPaneKey) {
            this.transferItem(srcPane, item);
        }
    },

    async transferItem(sourcePaneKey, item) {
        item._copying = true;
        const targetPaneKey = sourcePaneKey === "left" ? "right" : "left";
        const src_remote = this.explorer[sourcePaneKey].target;
        let src_path = item.Path || item.Name;
        const currentPanePath = this.explorer[sourcePaneKey].path;
        if (currentPanePath && !src_path.startsWith(currentPanePath)) {
            src_path = `${currentPanePath}/${src_path}`.replace(/^\/+/, "");
        }
        const dst_remote = this.explorer[targetPaneKey].target;
        const dst_path = this.explorer[targetPaneKey].path;

        try {
            const profileId = this.profilesData?.activeId || (this.activeProfile ? this.activeProfile.id : "");
            const data = await window.API.post("/api/browse/copy", {
                src_remote,
                src_path,
                dst_remote,
                dst_path,
                is_dir: item.IsDir,
                profile_id: profileId
            });
            if (data.status === "success") {
                this.showToast(`Transferred "${item.Name}" successfully!`, "success");
                this.fetchExplorerDirectory(targetPaneKey, dst_remote, dst_path);
            } else {
                this.showToast(`Transfer failed: ${data.error || 'Unknown error'}`, "error");
            }
        } catch (err) {
            this.showToast(`Transfer failed: ${err.message}`, "error");
        } finally {
            item._copying = false;
        }
    },

    async syncCurrentDirectory(sourcePaneKey) {
        const targetPaneKey = sourcePaneKey === "left" ? "right" : "left";
        const src_remote = this.explorer[sourcePaneKey].target;
        const src_path = this.explorer[sourcePaneKey].path;
        const dst_remote = this.explorer[targetPaneKey].target;
        const dst_path = this.explorer[targetPaneKey].path;

        const dirName = sourcePaneKey === "left" ? `Local (${src_remote}:${src_path || '/'}) ➔ Remote (${dst_remote}:${dst_path || '/'})` : `Remote (${src_remote}:${src_path || '/'}) ➔ Local (${dst_remote}:${dst_path || '/'})`;
        const executeCopy = async () => {
            this.showToast("Initiating directory synchronization...", "info");
            try {
                const profileId = this.profilesData?.activeId || (this.activeProfile ? this.activeProfile.id : "");
                const data = await window.API.post("/api/browse/copy", {
                    src_remote,
                    src_path,
                    dst_remote,
                    dst_path,
                    is_dir: true,
                    profile_id: profileId
                });
                if (data.status === "success") {
                    this.showToast("Directory synchronization complete!", "success");
                    this.fetchExplorerDirectory(targetPaneKey, dst_remote, dst_path);
                } else {
                    this.showToast(`Sync failed: ${data.error || 'Unknown error'}`, "error");
                }
            } catch (err) {
                this.showToast(`Directory sync failed: ${err.message}`, "error");
            }
        };

        const srcLabel = sourcePaneKey === "left" ? "LOCAL" : "REMOTE";
        const dstLabel = targetPaneKey === "left" ? "LOCAL" : "REMOTE";
        const fromPath = `${src_remote}:${src_path || '/'}`;
        const toPath = `${dst_remote}:${dst_path || '/'}`;

        if (typeof this.askConfirmation === "function") {
            this.askConfirmation({
                title: "Confirm Directory Copy",
                message: `Are you sure you want to copy directory contents?\nDestination files will be updated if modified.`,
                confirmText: "Start Copy",
                confirmClass: "bg-indigo-600 hover:bg-indigo-500 shadow-indigo-950/40",
                icon: "sync",
                transfer: {
                    fromLabel: `[${srcLabel}]`,
                    fromPath: fromPath,
                    toLabel: `[${dstLabel}]`,
                    toPath: toPath
                },
                onConfirm: executeCopy
            });
        } else if (confirm(`Transfer directory contents:\nFrom: [${srcLabel}] ${fromPath}\nTo: [${dstLabel}] ${toPath}`)) {
            await executeCopy();
        }
    },

    isItemPreviewable(item) {
        if (!item || item.IsDir) return false;
        const name = (item.Name || '').toLowerCase();
        const previewExtensions = [
            '.txt', '.md', '.markdown', '.log', '.conf', '.ini', '.env', '.csv',
            '.json', '.yaml', '.yml', '.xml', '.html', '.htm', '.css', '.js', '.ts', '.go', '.sh', '.bash', '.py', '.rs', '.sql',
            '.png', '.jpg', '.jpeg', '.gif', '.webp', '.svg', '.bmp', '.ico',
            '.pdf'
        ];
        return previewExtensions.some(ext => name.endsWith(ext));
    },

    async previewFile(paneKey, item) {
        if (!item || item.IsDir) return;
        const pane = this.explorer[paneKey];
        const remote = pane.target || (paneKey === 'left' ? 'local' : '');
        let path = item.Path || item.Name;
        if (pane.path && !path.startsWith(pane.path)) {
            path = `${pane.path.replace(/\/+$/, '')}/${item.Name}`;
        }
        const profileId = this.activeProfile ? this.activeProfile.id : "";

        this.filePreviewModal = {
            show: true,
            loading: true,
            error: null,
            remote,
            path,
            name: item.Name,
            preview: null
        };

        try {
            const query = new URLSearchParams({
                remote,
                path,
                profile_id: profileId
            });
            const res = await fetch(`/api/explorer/preview?${query.toString()}`);
            if (!res.ok) {
                const errData = await res.json().catch(() => ({}));
                throw new Error(errData.error || `HTTP ${res.status}`);
            }
            const data = await res.json();
            this.filePreviewModal.preview = data;
            this.filePreviewModal.loading = false;
        } catch (err) {
            this.filePreviewModal.loading = false;
            this.filePreviewModal.error = err.message || "Failed to load preview";
        }
    },

    closePreviewModal() {
        this.filePreviewModal.show = false;
        this.filePreviewModal.preview = null;
        this.filePreviewModal.error = null;
    },

    downloadFile(paneKey, item) {
        if (!item || item.IsDir) return;
        const pane = this.explorer[paneKey];
        const remote = pane.target || (paneKey === 'left' ? 'local' : '');
        let path = item.Path || item.Name;
        if (pane.path && !path.startsWith(pane.path)) {
            path = `${pane.path.replace(/\/+$/, '')}/${item.Name}`;
        }
        const profileId = this.activeProfile ? this.activeProfile.id : "";
        const query = new URLSearchParams({
            remote,
            path,
            profile_id: profileId
        });
        window.open(`/api/explorer/download?${query.toString()}`, "_blank");
    },

    downloadPreviewFile() {
        if (!this.filePreviewModal.remote || !this.filePreviewModal.path) return;
        const profileId = this.activeProfile ? this.activeProfile.id : "";
        const query = new URLSearchParams({
            remote: this.filePreviewModal.remote,
            path: this.filePreviewModal.path,
            profile_id: profileId
        });
        window.open(`/api/explorer/download?${query.toString()}`, "_blank");
    },

    triggerUpload(paneKey) {
        const input = document.getElementById(`explorer-upload-${paneKey}`);
        if (input) {
            input.value = "";
            input.click();
        }
    },

    async handleFileUpload(paneKey, event) {
        const files = event?.target?.files;
        if (!files || files.length === 0) return;
        const file = files[0];
        const pane = this.explorer[paneKey];
        const remote = pane.target || (paneKey === 'left' ? 'local' : '');
        const path = pane.path || "";
        const profileId = this.profilesData?.activeId || (this.activeProfile ? this.activeProfile.id : "");

        const formData = new FormData();
        formData.append("file", file);
        formData.append("remote", remote);
        formData.append("path", path);
        if (profileId) {
            formData.append("profile_id", profileId);
        }

        if (typeof this.showToast === "function") {
            this.showToast(`Uploading ${file.name}...`, "info");
        }

        try {
            const res = await fetch("/api/explorer/upload", {
                method: "POST",
                headers: {
                    "X-Dashboard-Request": "1"
                },
                body: formData
            });
            const data = await res.json();
            if (data.status === "success") {
                if (typeof this.showToast === "function") {
                    this.showToast(`Uploaded "${file.name}" successfully`, "success");
                }
                this.fetchExplorerDirectory(paneKey, remote, path);
            } else {
                if (typeof this.showToast === "function") {
                    this.showToast(`Upload failed: ${data.error || 'Unknown error'}`, "error");
                }
            }
        } catch (err) {
            if (typeof this.showToast === "function") {
                this.showToast(`Upload error: ${err.message}`, "error");
            }
        } finally {
            if (event.target) event.target.value = "";
        }
    },

    openContextMenu(paneKey, item, event) {
        if (!item) return;
        if (event) {
            event.preventDefault();
            event.stopPropagation();
        }

        const clientX = event ? event.clientX : window.innerWidth / 2;
        const clientY = event ? event.clientY : window.innerHeight / 2;
        const x = Math.min(clientX, window.innerWidth - 220);
        const y = Math.min(clientY, window.innerHeight - 240);

        this.contextMenu = { show: true, x: Math.max(10, x), y: Math.max(10, y), paneKey, item };
    },

    closeContextMenu() {
        this.contextMenu = { show: false, item: null, paneKey: null, x: 0, y: 0 };
    },

    copyItemPath(item) {
        if (!item) return;
        const p = item.Path || item.Name;
        if (navigator.clipboard) {
            navigator.clipboard.writeText(p);
            if (typeof this.showToast === "function") {
                this.showToast(`Copied path: ${p}`, "success");
            }
        }
    },
};
