/**
 * Dual-Pane File Explorer Module for Rclone Bisync Dashboard
 * Handles directory browsing, canary management, multi-key sorting,
 * bidirectional file transfers, drag-and-drop, and 2-step deletion modal.
 */

window.ExplorerModule = {
    initExplorer() {
            if (this.remotesData.remotes.length === 0) {
                this.fetchRemotes(false);
            }
            if (this.explorer.left.items.length === 0) {
                this.fetchExplorerDirectory("left", this.explorer.left.target, this.explorer.left.path);
            }
            if (this.explorer.right.items.length === 0) {
                if (this.remotesData.remotes && this.remotesData.remotes.length > 0) {
                    const active = this.remotesData.remotes.find(r => r.is_active_target);
                    if (active) {
                        this.explorer.right.target = active.name;
                    } else {
                        this.explorer.right.target = this.remotesData.remotes[0].name;
                    }
                }
                this.fetchExplorerDirectory("right", this.explorer.right.target, this.explorer.right.path);
            }
        },

        openRemoteInExplorer(remoteName) {
            this.currentTab = "explorer";
            this.explorer.right.target = remoteName;
            this.explorer.right.path = "";
            this.fetchExplorerDirectory("right", remoteName, "");
            if (this.explorer.left.items.length === 0) {
                this.fetchExplorerDirectory("left", this.explorer.left.target, this.explorer.left.path);
            }
        },

        async fetchExplorerDirectory(paneKey, target, path = "") {
            const pane = this.explorer[paneKey];
            if (!pane) return;
            pane.loading = true;
            pane.error = null;
            try {
                const res = await fetch("/api/browse", {
                    method: "POST",
                    headers: {
                        "Content-Type": "application/json",
                        "X-Dashboard-Request": "1"
                    },
                    body: JSON.stringify({ remote: target, path: path })
                });
                if (!res.ok) {
                    throw new Error(`HTTP ${res.status}: ${res.statusText}`);
                }
                const data = await res.json();
                if (data.error) {
                    pane.error = data.error;
                    pane.items = [];
                    pane.hasCanary = false;
                } else {
                    pane.path = data.path || "";
                    if (!path) {
                        pane.hasCanary = Boolean(data.has_canary);
                    }
                    pane.items = data.items || [];
                    pane.itemCount = pane.items.length;
                    pane.totalSize = pane.items.reduce((acc, item) => (item.Size > 0 ? acc + item.Size : acc), 0);
                }
            } catch (err) {
                pane.error = err.message || "Failed to load directory";
                pane.items = [];
            } finally {
                pane.loading = false;
            }
        },

        navigateFolder(paneKey, folderName) {
            const pane = this.explorer[paneKey];
            if (!pane) return;
            let newPath = "";
            if (folderName === "..") {
                const parts = pane.path.split("/").filter(Boolean);
                parts.pop();
                newPath = parts.join("/");
            } else {
                newPath = pane.path ? `${pane.path}/${folderName}` : folderName;
            }
            this.fetchExplorerDirectory(paneKey, pane.target, newPath);
        },

        changePaneTarget(paneKey, newTarget) {
            const pane = this.explorer[paneKey];
            if (!pane) return;
            pane.target = newTarget;
            pane.path = "";
            this.fetchExplorerDirectory(paneKey, newTarget, "");
        },

        refreshExplorer() {
            this.fetchExplorerDirectory("left", this.explorer.left.target, this.explorer.left.path);
            this.fetchExplorerDirectory("right", this.explorer.right.target, this.explorer.right.path);
        },

        getBreadcrumbs(paneKey) {
            const pane = this.explorer[paneKey];
            if (!pane) return [];
            const rootLabel = paneKey === "left" ? "~" : `${pane.target}:`;
            const crumbs = [{ name: rootLabel, path: "" }];
            if (!pane.path) return crumbs;

            const parts = pane.path.split("/").filter(Boolean);
            let cumulative = "";
            for (const part of parts) {
                cumulative = cumulative ? `${cumulative}/${part}` : part;
                crumbs.push({ name: part, path: cumulative });
            }
            return crumbs;
        },

        async copyPanePath(paneKey) {
            const pane = this.explorer[paneKey];
            if (!pane) return;
            let fullPath = "";
            if (paneKey === "left") {
                const base = this.telemetry.config?.LOCAL_PATH || this.telemetry.config?.LOCALPATH || "~/gdrive";
                fullPath = pane.path ? `${base.replace(/\/+$/, "")}/${pane.path}` : base;
            } else {
                fullPath = pane.path ? `${pane.target}:${pane.path}` : `${pane.target}:`;
            }
            try {
                await navigator.clipboard.writeText(fullPath);
                pane.copied = true;
                setTimeout(() => {
                    pane.copied = false;
                }, 1800);
            } catch (err) {
                console.warn("Clipboard access failed:", err);
            }
        },

        formatModTime(isoStr) {
            return formatModTime(isoStr);
        },

        toggleSort(paneKey, field) {
            const pane = this.explorer[paneKey];
            if (!pane) return;
            if (pane.sortBy === field) {
                pane.sortAsc = !pane.sortAsc;
            } else {
                pane.sortBy = field;
                pane.sortAsc = (field === "name" || field === "type");
            }
        },

        getFilteredItems(paneKey) {
            const pane = this.explorer[paneKey];
            if (!pane) return [];
            const q = (pane.search || "").toLowerCase().trim();
            let items = pane.items || [];
            if (q) {
                items = items.filter(item => (item.Name || "").toLowerCase().includes(q));
            }

            const sortBy = pane.sortBy || "name";
            const asc = pane.sortAsc !== false;
            const foldersFirst = pane.foldersFirst !== false;

            return [...items].sort((a, b) => {
                if (foldersFirst && a.IsDir !== b.IsDir) {
                    return a.IsDir ? -1 : 1;
                }

                let res = 0;
                if (sortBy === "size") {
                    const sizeA = a.IsDir ? -1 : (a.Size || 0);
                    const sizeB = b.IsDir ? -1 : (b.Size || 0);
                    res = sizeA - sizeB;
                } else if (sortBy === "mtime") {
                    const timeA = a.ModTime ? new Date(a.ModTime).getTime() : 0;
                    const timeB = b.ModTime ? new Date(b.ModTime).getTime() : 0;
                    res = timeA - timeB;
                } else if (sortBy === "type") {
                    const extA = a.IsDir ? "" : (a.Name.split(".").pop() || "").toLowerCase();
                    const extB = b.IsDir ? "" : (b.Name.split(".").pop() || "").toLowerCase();
                    res = extA.localeCompare(extB);
                    if (res === 0) {
                        res = (a.Name || "").localeCompare(b.Name || "", undefined, { numeric: true, sensitivity: "base" });
                    }
                } else {
                    res = (a.Name || "").localeCompare(b.Name || "", undefined, { numeric: true, sensitivity: "base" });
                }

                return asc ? res : -res;
            });
        },


        async provisionCanary(paneKey) {
            const pane = this.explorer[paneKey];
            if (!pane) return;
            try {
                const res = await fetch("/api/browse/canary", {
                    method: "POST",
                    headers: {
                        "Content-Type": "application/json",
                        "X-Dashboard-Request": "1"
                    },
                    body: JSON.stringify({ remote: pane.target, path: "" })
                });
                const data = await res.json();
                if (data.status === "success") {
                    pane.hasCanary = true;
                    this.fetchExplorerDirectory(paneKey, pane.target, pane.path);
                } else {
                    alert(data.error || "Failed to provision canary");
                }
            } catch (err) {
                alert("Network error: " + err.message);
            }
        },

        showToast(message, type = "info") {
            if (this.toast.timeout) clearTimeout(this.toast.timeout);
            this.toast.message = message;
            this.toast.type = type;
            this.toast.show = true;
            this.toast.timeout = setTimeout(() => {
                this.toast.show = false;
            }, 3000);
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
                const res = await fetch("/api/browse/copy", {
                    method: "POST",
                    headers: {
                        "Content-Type": "application/json",
                        "X-Dashboard-Request": "1"
                    },
                    body: JSON.stringify({
                        src_remote,
                        src_path,
                        dst_remote,
                        dst_path,
                        is_dir: item.IsDir
                    })
                });
                const data = await res.json();
                if (data.status === "success") {
                    this.showToast(`Transferred "${item.Name}" successfully!`, "success");
                    this.fetchExplorerDirectory(targetPaneKey, dst_remote, dst_path);
                } else {
                    this.showToast(`Transfer failed: ${data.error || 'Unknown error'}`, "error");
                }
            } catch (err) {
                this.showToast(`Network error: ${err.message}`, "error");
            } finally {
                item._copying = false;
            }
        },

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
                const res = await fetch("/api/browse/delete", {
                    method: "POST",
                    headers: {
                        "Content-Type": "application/json",
                        "X-Dashboard-Request": "1"
                    },
                    body: JSON.stringify({
                        remote,
                        path,
                        is_dir
                    })
                });
                const data = await res.json();
                if (data.status === "success") {
                    this.showToast(`Deleted "${item.Name}" successfully`, "success");
                    this.cancelDelete();
                    this.fetchExplorerDirectory(paneKey, remote, this.explorer[paneKey].path);
                } else {
                    this.showToast(`Delete failed: ${data.error || 'Unknown error'}`, "error");
                }
            } catch (err) {
                this.showToast(`Network error: ${err.message}`, "error");
            } finally {
                this.deleteModal.deleting = false;
            }
        },


        async syncCurrentDirectory(sourcePaneKey) {
            const targetPaneKey = sourcePaneKey === "left" ? "right" : "left";
            const src_remote = this.explorer[sourcePaneKey].target;
            const src_path = this.explorer[sourcePaneKey].path;
            const dst_remote = this.explorer[targetPaneKey].target;
            const dst_path = this.explorer[targetPaneKey].path;

            const dirName = sourcePaneKey === "left" ? `Local (${src_remote}:${src_path || '/'}) ➔ Remote (${dst_remote}:${dst_path || '/'})` : `Remote (${src_remote}:${src_path || '/'}) ➔ Local (${dst_remote}:${dst_path || '/'})`;
            if (!confirm(`Are you sure you want to copy directory contents:\n${dirName}?`)) {
                return;
            }

            this.showToast("Initiating directory synchronization...", "info");
            try {
                const res = await fetch("/api/browse/copy", {
                    method: "POST",
                    headers: {
                        "Content-Type": "application/json",
                        "X-Dashboard-Request": "1"
                    },
                    body: JSON.stringify({
                        src_remote,
                        src_path,
                        dst_remote,
                        dst_path,
                        is_dir: true
                    })
                });
                const data = await res.json();
                if (data.status === "success") {
                    this.showToast("Directory synchronization complete!", "success");
                    this.fetchExplorerDirectory(targetPaneKey, dst_remote, dst_path);
                } else {
                    this.showToast(`Sync failed: ${data.error || 'Unknown error'}`, "error");
                }
            } catch (err) {
                this.showToast(`Network error: ${err.message}`, "error");
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
};
