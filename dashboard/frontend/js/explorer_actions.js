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
        this.deleteModal.show = false;
        this.deleteModal.item = null;
        this.deleteModal.paneKey = null;
        this.deleteModal.step = 1;
        this.deleteModal.deleting = false;
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
            const data = await window.API.post("/api/browse/delete", {
                remote,
                path,
                is_dir
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
    }
};
