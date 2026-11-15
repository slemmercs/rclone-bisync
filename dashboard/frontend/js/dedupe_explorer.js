/**
 * Rclone Bisync Dashboard — Deduplication Workspace Explorer Module
 * Manages directory explorer browsing, candidate item matching, composite keying,
 * breadcrumbs navigation, and candidate previews within the dedupe workspace.
 */

window.DedupeExplorerModule = {
    cannotNavigateDedupeExplorerParent() {
        return !this.dedupeModal?.explorerPath;
    },

    switchDedupeTab(tabName) {
        this.dedupeModal.activeTab = tabName;
        if (tabName === "explorer" && (!this.dedupeModal.explorerItems || this.dedupeModal.explorerItems.length === 0)) {
            this.loadDedupeExplorer();
        }
    },

    hasContentDivergence(collision) {
        if (!collision || !collision.candidates || collision.candidates.length < 2) return false;
        const cands = collision.candidates;
        const firstSize = cands[0].size;
        for (let i = 1; i < cands.length; i++) {
            if (cands[i].size !== undefined && firstSize !== undefined && cands[i].size !== firstSize) {
                return true;
            }
            if (cands[i].hash && cands[0].hash && cands[i].hash !== cands[0].hash) {
                return true;
            }
        }
        return false;
    },

    getDedupeBreadcrumbs() {
        const raw = (this.dedupeModal?.explorerPath || "").trim().replace(/^\/+|\/+$/g, "");
        const rootLabel = this.dedupeModal?.remote ? `${this.dedupeModal.remote}:` : "~";
        const crumbs = [{ name: rootLabel, path: "" }];
        if (!raw) return crumbs;
        const parts = raw.split("/").filter(Boolean);
        let accum = "";
        for (const part of parts) {
            accum = accum ? `${accum}/${part}` : part;
            crumbs.push({ name: part, path: accum });
        }
        return crumbs;
    },

    async loadDedupeExplorer(folderPath) {
        if (folderPath === undefined || folderPath === null) {
            folderPath = this.dedupeModal.explorerPath !== undefined ? this.dedupeModal.explorerPath : (this.dedupeModal.path || "");
        }
        this.dedupeModal.explorerPath = folderPath;
        this.dedupeModal.explorerLoading = true;
        this.dedupeModal.explorerError = null;
        try {
            const profileId = this.activeProfile ? this.activeProfile.id : "";
            const cleanPath = (folderPath || "").trim().replace(/^\/+|\/+$/g, "");
            const res = await window.API.post("/api/browse", {
                remote: this.dedupeModal.remote,
                path: cleanPath,
                profile_id: profileId
            });
            if (res && res.error) {
                this.dedupeModal.explorerError = res.error;
                this.dedupeModal.explorerItems = [];
            } else if (res && res.items) {
                this.dedupeModal.explorerItems = res.items;
                this.dedupeModal.explorerError = null;
            } else {
                this.dedupeModal.explorerItems = [];
                this.dedupeModal.explorerError = null;
            }
        } catch (err) {
            console.error("Failed to load dedupe explorer directory:", err);
            this.dedupeModal.explorerError = err.message || "Failed to load directory";
            this.dedupeModal.explorerItems = [];
        } finally {
            this.dedupeModal.explorerLoading = false;
        }
    },

    navigateDedupeExplorerParent() {
        let current = (this.dedupeModal.explorerPath || "").trim().replace(/^\/+|\/+$/g, "");
        if (!current) {
            this.dedupeModal.explorerPath = "";
            return;
        }
        const parts = current.split("/").filter(Boolean);
        parts.pop();
        const parentPath = parts.join("/");
        this.dedupeModal.explorerPath = parentPath;
        this.loadDedupeExplorer(parentPath);
    },

    selectDedupeExplorerItem(item) {
        if (!item) return;
        if (item.IsDir) {
            let current = (this.dedupeModal.explorerPath || "").replace(/^\/+|\/+$/g, "");
            const newPath = current ? `${current}/${item.Name}` : item.Name;
            this.loadDedupeExplorer(newPath);
        } else {
            const match = this.getCollisionForItem(item);
            if (match) {
                this.dedupeModal.selectedCollision = match;
                this.dedupeModal.activeTab = "candidates";
            }
        }
    },

    getDedupeExplorerRowKey(item, idx) {
        if (!item) return `dedupe_exp_${idx}`;
        const id = item.ID || item.Id || "";
        const name = item.Name || item.Path || "";
        const mtime = item.ModTime || item.Mtime || "";
        const size = item.Size !== undefined ? item.Size : "";
        return `dedupe_exp_${id}_${name}_${size}_${mtime}_${idx}`;
    },

    isDedupeExplorerItemDuplicate(item) {
        if (!item || item.IsDir) return false;
        const items = this.dedupeModal?.explorerItems || [];
        let count = 0;
        for (const it of items) {
            if (!it.IsDir && it.Name === item.Name) {
                count++;
                if (count > 1) return true;
            }
        }
        const collisions = this.dedupeModal?.parsedOutput?.items || [];
        return collisions.some(c => c.name === item.Name || c.path === item.Path);
    },

    getCollisionForItem(item) {
        if (!item || item.IsDir) return null;
        const collisions = this.dedupeModal?.parsedOutput?.items || [];
        return collisions.find(c => c.name === item.Name || c.path === item.Path) || null;
    },

    getCandidateForExplorerItem(item) {
        const col = this.getCollisionForItem(item);
        if (!col || !col.candidates || col.candidates.length === 0) return null;
        const itemId = item.ID || item.Id || "";
        if (itemId) {
            const foundById = col.candidates.find(c => c.id && c.id === itemId);
            if (foundById) return foundById;
        }
        if (item.Size !== undefined && item.ModTime) {
            const foundByMeta = col.candidates.find(c => c.size === item.Size && c.modtime === item.ModTime);
            if (foundByMeta) return foundByMeta;
        }
        return col.candidates[0] || null;
    },

    previewDedupeCandidate(collision, cand) {
        if (!collision || !cand) return;
        const remote = this.dedupeModal.remote;
        let filePath = collision.path || collision.name || "";
        const dir = this.dedupeModal.path || "";
        if (dir && !filePath.startsWith(dir)) {
            filePath = `${dir.replace(/\/+$/, '')}/${filePath.replace(/^\/+/, '')}`;
        }
        const profileId = this.activeProfile ? this.activeProfile.id : "";
        const candName = collision.name || filePath.split("/").pop() || "file";

        this.filePreviewModal = {
            show: true,
            loading: true,
            error: null,
            remote,
            path: filePath,
            name: candName,
            preview: null
        };

        const query = new URLSearchParams({
            remote,
            path: filePath,
            profile_id: profileId
        });
        if (cand.id) {
            query.set("id", cand.id);
        }

        fetch(`/api/explorer/preview?${query.toString()}`)
            .then(res => {
                if (!res.ok) {
                    return res.json().catch(() => ({})).then(errData => {
                        throw new Error(errData.error || `HTTP ${res.status}`);
                    });
                }
                return res.json();
            })
            .then(data => {
                this.filePreviewModal.preview = data;
            })
            .catch(err => {
                this.filePreviewModal.error = err.message || "Failed to load preview";
            })
            .finally(() => {
                this.filePreviewModal.loading = false;
            });
    },

    previewDedupeExplorerItem(item) {
        if (!item || item.IsDir) return;
        const remote = this.dedupeModal.remote;
        let filePath = item.Path || item.Name || "";
        const dir = this.dedupeModal.explorerPath || this.dedupeModal.path || "";
        if (dir && !filePath.startsWith(dir)) {
            filePath = `${dir.replace(/\/+$/, '')}/${item.Name}`;
        }
        const profileId = this.activeProfile ? this.activeProfile.id : "";

        this.filePreviewModal = {
            show: true,
            loading: true,
            error: null,
            remote,
            path: filePath,
            name: item.Name,
            preview: null
        };

        const query = new URLSearchParams({
            remote,
            path: filePath,
            profile_id: profileId
        });
        if (item.ID || item.Id) {
            query.set("id", item.ID || item.Id);
        }

        fetch(`/api/explorer/preview?${query.toString()}`)
            .then(res => {
                if (!res.ok) {
                    return res.json().catch(() => ({})).then(errData => {
                        throw new Error(errData.error || `HTTP ${res.status}`);
                    });
                }
                return res.json();
            })
            .then(data => {
                this.filePreviewModal.preview = data;
            })
            .catch(err => {
                this.filePreviewModal.error = err.message || "Failed to load preview";
            })
            .finally(() => {
                this.filePreviewModal.loading = false;
            });
    }
};
