/**
 * Dual-Pane File Explorer Module for Rclone Bisync Dashboard
 * Handles directory browsing, canary management, multi-key sorting,
 * bidirectional file transfers, drag-and-drop, and 2-step deletion modal.
 */

window.ExplorerModule = {
    initExplorer() {
        if (!this.remotesData.remotes || this.remotesData.remotes.length === 0) {
            this.fetchRemotes(false);
        }

        const p = this.activeProfile;
        if (p) {
            const remote = (p.remote || "").replace(/:$/, "");
            const remotePath = (p.remote_path || "").replace(/^\/+/, "");
            if (!this.explorer.isSwapped) {
                this.explorer.left.target = "local";
                if (remote && !this.explorer.right.target) this.explorer.right.target = remote;
                if (remotePath && !this.explorer.right.path) this.explorer.right.path = remotePath;
            } else {
                if (remote && !this.explorer.left.target) this.explorer.left.target = remote;
                if (remotePath && !this.explorer.left.path) this.explorer.left.path = remotePath;
                this.explorer.right.target = "local";
            }
        } else if (this.remotesData.remotes && this.remotesData.remotes.length > 0) {
            const active = this.remotesData.remotes.find(r => r.is_active_target);
            if (!this.explorer.isSwapped && !this.explorer.right.target) {
                this.explorer.right.target = active ? active.name : this.remotesData.remotes[0].name;
            }
        }

        if (this.explorer.left.items.length === 0) {
            this.fetchExplorerDirectory("left", this.explorer.left.target, this.explorer.left.path);
        }
        if (this.explorer.right.items.length === 0) {
            this.fetchExplorerDirectory("right", this.explorer.right.target, this.explorer.right.path);
        }
    },

    syncExplorerWithActiveProfile() {
        const p = this.activeProfile;
        if (!p) return;
        const remote = (p.remote || "").replace(/:$/, "");
        const remotePath = (p.remote_path || "").replace(/^\/+/, "");
        if (!this.explorer.isSwapped) {
            this.explorer.left.target = "local";
            this.explorer.left.path = "";
            if (remote) this.explorer.right.target = remote;
            this.explorer.right.path = remotePath;
        } else {
            if (remote) this.explorer.left.target = remote;
            this.explorer.left.path = remotePath;
            this.explorer.right.target = "local";
            this.explorer.right.path = "";
        }
        this.fetchExplorerDirectory("left", this.explorer.left.target, this.explorer.left.path);
        this.fetchExplorerDirectory("right", this.explorer.right.target, this.explorer.right.path);
    },

    swapExplorerPanes() {
        this.explorer.isSwapped = !this.explorer.isSwapped;
        [this.explorer.left.target, this.explorer.right.target] = [this.explorer.right.target, this.explorer.left.target];
        [this.explorer.left.path, this.explorer.right.path] = [this.explorer.right.path, this.explorer.left.path];
        [this.explorer.left.items, this.explorer.right.items] = [this.explorer.right.items, this.explorer.left.items];
        [this.explorer.left.hasCanary, this.explorer.right.hasCanary] = [this.explorer.right.hasCanary, this.explorer.left.hasCanary];
        [this.explorer.left.totalSize, this.explorer.right.totalSize] = [this.explorer.right.totalSize, this.explorer.left.totalSize];
        [this.explorer.left.itemCount, this.explorer.right.itemCount] = [this.explorer.right.itemCount, this.explorer.left.itemCount];

        if (typeof this.showToast === "function") {
            this.showToast(this.explorer.isSwapped ? "Explorer swapped: Remote ↔ Local" : "Explorer restored: Local ↔ Remote", "info");
        }
    },

    isItemExcluded(item) {
        if (!item || !item.Name) return false;
        if (item.Name === "RCLONE_TEST") return false;
        const p = this.activeProfile;
        const rawPatterns = (p && p.exclude_patterns && p.exclude_patterns.length > 0)
            ? p.exclude_patterns
            : [".git/**", "*.tmp", ".DS_Store", "Thumbs.db", "desktop.ini"];

        const name = item.Name;
        for (let pat of rawPatterns) {
            if (!pat) continue;
            pat = pat.trim();
            const cleanPat = pat.replace(/\/+\*{1,2}$/, "").replace(/^\/+/, "");
            if (cleanPat && cleanPat === name) {
                return true;
            }
            if (pat.startsWith("*.") && name.endsWith(pat.slice(1))) {
                return true;
            }
            if (pat.endsWith("*") && name.startsWith(pat.slice(0, -1))) {
                return true;
            }
            if (pat.includes("*")) {
                const regexStr = "^" + pat.replace(/[-/\\^$*+?.()|[\]{}]/g, "\\$&").replace(/\\\*/g, ".*") + "$";
                try {
                    if (new RegExp(regexStr, "i").test(name)) return true;
                } catch (_) {}
            }
        }
        return false;
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
                const data = await window.API.post("/api/browse", { remote: target, path: path });
                if (data.error) {
                    pane.error = data.error;
                    pane.items = [];
                    pane.hasCanary = false;
                } else {
                    pane.path = data.path || "";
                    if (!path) {
                        pane.hasCanary = Boolean(data.has_canary);
                    }
                    pane.items = (data.items || []).map(item => ({ ...item, _copying: false }));
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
            const rootLabel = pane.target === "local" ? "~" : `${pane.target}:`;
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

        getPaneTargetLabel(paneKey) {
            const pane = this.explorer[paneKey];
            if (!pane) return "";
            return pane.target === "local" ? "Local" : "Remote";
        },

        getPaneDisplayPath(paneKey) {
            const pane = this.explorer[paneKey];
            if (!pane) return "";
            if (pane.target === "local") {
                const p = this.activeProfile;
                return (p && p.local_path) || this.telemetry.config?.LOCAL_PATH || "~/gdrive";
            }
            return `${pane.target}:${pane.path || ""}`;
        },

        async copyPanePath(paneKey) {
            const pane = this.explorer[paneKey];
            if (!pane) return;
            let fullPath = "";
            if (pane.target === "local") {
                const p = this.activeProfile;
                const base = (p && p.local_path) || this.telemetry.config?.LOCAL_PATH || this.telemetry.config?.LOCALPATH || "~/gdrive";
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
                const data = await window.API.post("/api/browse/canary", { remote: pane.target, path: pane.path || "" });
                if (data.status === "success") {
                    pane.hasCanary = true;
                    this.showToast("Canary file RCLONE_TEST provisioned successfully", "success");
                    this.fetchExplorerDirectory(paneKey, pane.target, pane.path);
                } else {
                    this.showToast(data.error || "Failed to provision canary", "error");
                }
            } catch (err) {
                this.showToast("Failed to provision canary: " + err.message, "error");
            }
        },

        isExplorerLoading() {
            return Boolean(this.explorer?.left?.loading || this.explorer?.right?.loading);
        },

        isItemCopying(item) {
            return Boolean(item && item._copying);
        },

        isCanaryItem(item) {
            return Boolean(item && item.Name === 'RCLONE_TEST');
        },

        isDeleteModalDeleting() {
            return Boolean(this.deleteModal?.deleting);
        }
};
