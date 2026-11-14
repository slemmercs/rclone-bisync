/**
 * Explorer Checksum & Multi-File Comparison Module
 * Handles multi-file selection state across explorer panes and cryptographic hash comparison.
 */

window.ExplorerCompareModule = {
    compareState: {
        selectedItems: [],
        modalOpen: false,
        loading: false,
        error: null,
        response: null
    },

    getItemRelativePath(item, paneKey) {
        if (!item) return "";
        const pane = this.explorer[paneKey];
        if (!pane) return item.Path || item.Name || "";
        let path = item.Path || item.Name || "";
        if (pane.path && !path.startsWith(pane.path)) {
            path = `${pane.path.replace(/\/+$/, "")}/${item.Name}`;
        }
        return path;
    },

    isItemSelected(item, paneKey) {
        if (!item || item.IsDir || item.Name === "RCLONE_TEST") return false;
        const pane = this.explorer[paneKey];
        if (!pane) return false;
        const remote = pane.target || (paneKey === "left" ? "local" : "");
        const path = this.getItemRelativePath(item, paneKey);
        return this.compareState.selectedItems.some(i => i.remote === remote && i.path === path);
    },

    toggleSelectItem(item, paneKey) {
        if (!item || item.IsDir || item.Name === "RCLONE_TEST") return;
        const pane = this.explorer[paneKey];
        if (!pane) return;
        const remote = pane.target || (paneKey === "left" ? "local" : "");
        const path = this.getItemRelativePath(item, paneKey);
        const idx = this.compareState.selectedItems.findIndex(i => i.remote === remote && i.path === path);
        if (idx >= 0) {
            this.compareState.selectedItems.splice(idx, 1);
        } else {
            this.compareState.selectedItems.push({
                remote,
                path,
                name: item.Name,
                size: item.Size || 0,
                modTime: item.ModTime || "",
                paneKey
            });
        }
    },

    clearCompareSelection() {
        this.compareState.selectedItems = [];
    },

    async openCompareModal() {
        if (this.compareState.selectedItems.length < 2) {
            if (typeof this.showToast === "function") {
                this.showToast("Select at least 2 files to compare checksums", "warning");
            }
            return;
        }
        this.compareState.modalOpen = true;
        await this.runChecksumComparison();
    },

    async compareDirect(leftItem, rightItem) {
        if (!leftItem || !rightItem || leftItem.IsDir || rightItem.IsDir) return;
        this.compareState.selectedItems = [
            {
                remote: this.explorer.left.target || "local",
                path: this.getItemRelativePath(leftItem, "left"),
                name: leftItem.Name,
                size: leftItem.Size || 0,
                modTime: leftItem.ModTime || "",
                paneKey: "left"
            },
            {
                remote: this.explorer.right.target || "remote",
                path: this.getItemRelativePath(rightItem, "right"),
                name: rightItem.Name,
                size: rightItem.Size || 0,
                modTime: rightItem.ModTime || "",
                paneKey: "right"
            }
        ];
        this.compareState.modalOpen = true;
        await this.runChecksumComparison();
    },

    async runChecksumComparison() {
        this.compareState.loading = true;
        this.compareState.error = null;
        this.compareState.response = null;

        try {
            const profileId = this.activeProfile ? this.activeProfile.id : "";
            const payload = {
                profile_id: profileId,
                items: this.compareState.selectedItems.map(i => ({
                    remote: i.remote,
                    path: i.path
                }))
            };
            const res = await window.API.post("/api/browse/compare", payload);
            if (res && res.status === "success") {
                this.compareState.response = res;
            } else {
                this.compareState.error = (res && res.error) || "Comparison failed";
            }
        } catch (err) {
            this.compareState.error = err.message || "Failed to execute checksum comparison";
        } finally {
            this.compareState.loading = false;
        }
    },

    closeCompareModal() {
        this.compareState.modalOpen = false;
    },

    isCompareLoading() {
        return Boolean(this.compareState && this.compareState.loading);
    },

    getClusterItems(clusterIndices) {
        if (!this.compareState.response || !this.compareState.response.items) return [];
        return (clusterIndices || []).map(idx => this.compareState.response.items[idx]).filter(Boolean);
    }
};
