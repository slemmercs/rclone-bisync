/**
 * Rclone Bisync Dashboard — Deduplication Workspace Module
 * Manages deduplication configuration, execution, collision ledger,
 * candidate inspection, and folder explorer browsing.
 */

window.DedupeWorkspaceModule = {
    dedupeModal: {
        show: false,
        remote: "",
        path: "",
        mode: "oldest",
        byHash: false,
        isOnedrive: false,
        dedupeBoth: false,
        folderBrowserOpen: false,
        folders: [],
        loadingFolders: false,
        executing: false,
        runId: null,
        output: "",
        parsedOutput: null,
        showRawOutput: false,
        filterQuery: "",
        selectedCollision: null,
        activeTab: "candidates", // 'candidates' | 'explorer' | 'logs'
        explorerItems: [],
        explorerLoading: false,
        explorerError: null,
        explorerPath: ""
    },

    isDedupeExecuting() {
        return Boolean(this.dedupeModal?.executing);
    },

    isDedupeFoldersLoading() {
        return Boolean(this.dedupeModal?.loadingFolders);
    },

    cannotNavigateDedupeExplorerParent() {
        return !this.dedupeModal?.explorerPath;
    },

    openDedupeModal(remoteName, initialPath) {
        const clean = (remoteName || "").replace(/:$/, "").trim();
        const p = (initialPath !== undefined && initialPath !== null)
            ? String(initialPath)
            : ((this.explorer?.right?.target === clean) ? (this.explorer?.right?.path || "") : "");
        const rObj = (this.remotesData?.remotes || []).find(r => (r.name || "").replace(/:$/, "").trim() === clean);
        const rType = (rObj?.type || "").toLowerCase();
        const isOnedrive = clean.toLowerCase().includes("onedrive") || rType === "onedrive";
        const hasPairedProfile = Boolean(this.activeProfile && (
            (this.activeProfile.remote || "").replace(/:$/, "").trim() === clean ||
            clean === "local"
        ));
        this.dedupeModal = {
            show: true,
            remote: clean,
            path: p,
            mode: "oldest",
            byHash: isOnedrive,
            isOnedrive: isOnedrive,
            dedupeBoth: hasPairedProfile,
            folderBrowserOpen: false,
            folders: [],
            loadingFolders: false,
            executing: false,
            runId: null,
            output: "",
            parsedOutput: null,
            showRawOutput: false,
            filterQuery: "",
            selectedCollision: null,
            activeTab: "candidates",
            explorerItems: [],
            explorerLoading: false,
            explorerError: null,
            explorerPath: p
        };
    },

    closeDedupeModal() {
        this.dedupeModal.show = false;
    },

    async toggleDedupeFolderBrowser() {
        this.dedupeModal.folderBrowserOpen = !this.dedupeModal.folderBrowserOpen;
        if (this.dedupeModal.folderBrowserOpen && (!this.dedupeModal.folders || this.dedupeModal.folders.length === 0)) {
            await this.loadDedupeFolders(this.dedupeModal.path || "");
        }
    },

    async loadDedupeFolders(folderPath) {
        this.dedupeModal.loadingFolders = true;
        try {
            const profileId = this.activeProfile ? this.activeProfile.id : "";
            const cleanPath = (folderPath || "").replace(/^\/+|\/+$/g, "");
            const res = await window.API.post("/api/browse", {
                remote: this.dedupeModal.remote,
                path: cleanPath,
                profile_id: profileId
            });
            if (res && res.items) {
                this.dedupeModal.folders = res.items.filter(item => item.IsDir);
            } else {
                this.dedupeModal.folders = [];
            }
        } catch (err) {
            console.error("Failed to load dedupe folders:", err);
            this.dedupeModal.folders = [];
        } finally {
            this.dedupeModal.loadingFolders = false;
        }
    },

    selectDedupeFolder(folderName) {
        let current = (this.dedupeModal.path || "").trim().replace(/^\/+|\/+$/g, "");
        if (!folderName) {
            this.dedupeModal.path = "";
        } else if (!current) {
            this.dedupeModal.path = folderName;
        } else {
            this.dedupeModal.path = `${current}/${folderName}`;
        }
        this.loadDedupeFolders(this.dedupeModal.path);
    },

    navigateDedupeParent() {
        let current = (this.dedupeModal.path || "").trim().replace(/^\/+|\/+$/g, "");
        if (!current) return;
        const parts = current.split("/");
        parts.pop();
        this.dedupeModal.path = parts.join("/");
        this.loadDedupeFolders(this.dedupeModal.path);
    },

    jumpToExplorerFile(filePath, remote, candidateId) {
        if (!filePath) return;
        this.closeDedupeModal();
        if (typeof this.switchTab === "function") this.switchTab("explorer");
        const clean = (remote || "").replace(/:$/, "").trim();
        const parts = filePath.replace(/^\/+/, "").split("/");
        const fileName = parts.pop();
        const dirPath = parts.join("/");

        let paneKey = "right";
        if (clean === "local" || (this.explorer?.left?.target || "").replace(/:$/, "").trim() === clean) {
            paneKey = "left";
        }
        if (this.explorer && this.explorer[paneKey]) {
            this.explorer[paneKey].path = dirPath;
            if (clean && clean !== "local") {
                this.explorer[paneKey].target = clean;
            }
            this.explorer.highlightPath = filePath;
            this.explorer.highlightName = fileName;
            this.explorer.highlightId = candidateId || null;
            setTimeout(() => {
                if (this.explorer && this.explorer.highlightName === fileName) {
                    this.explorer.highlightName = null;
                    this.explorer.highlightId = null;
                }
            }, 6000);

            if (typeof this.fetchExplorerDirectory === "function") {
                this.fetchExplorerDirectory(paneKey, this.explorer[paneKey].target, dirPath);
            } else if (typeof this.fetchDirectory === "function") {
                this.fetchDirectory(paneKey);
            }
            if (typeof this.scrollToHighlightedRow === "function") {
                const notify = () => this.scrollToHighlightedRow(fileName, candidateId);
                if (typeof this.$nextTick === "function") this.$nextTick(notify);
                else setTimeout(notify, 150);
            }
        }
        if (typeof this.showToast === "function") {
            this.showToast(`Navigated to ${fileName || 'folder'} in ${paneKey === "left" ? "Left" : "Right"} Pane`, "info");
        }
    },

    viewDedupeLogs() {
        const runId = this.dedupeModal?.runId;
        if (!runId) return;
        const meta = { run_id: runId, remote: this.dedupeModal.remote, path: this.dedupeModal.path, type: "dedupe" };
        this.closeDedupeModal();
        if (typeof this.openDaemonModal === "function") this.openDaemonModal(meta);
    },

    selectCollision(item) {
        this.dedupeModal.selectedCollision = item;
        if (item) {
            const dir = item.dir || this.dedupeModal.path || "";
            this.dedupeModal.explorerPath = dir;
            this.loadDedupeExplorer(dir);
        }
    },

    filteredCollisions() {
        const items = this.dedupeModal.parsedOutput?.items || [];
        const q = (this.dedupeModal.filterQuery || "").trim().toLowerCase();
        if (!q) return items;
        return items.filter(it => (it.name || "").toLowerCase().includes(q) || (it.path || "").toLowerCase().includes(q) || (it.dir || "").toLowerCase().includes(q));
    },

    getConflictSummary() {
        const parsed = this.dedupeModal.parsedOutput;
        if (!parsed) return { totalCollisions: 0, totalDeleted: 0, totalCandidates: 0 };
        let candidatesCount = 0;
        for (const item of (parsed.items || [])) {
            candidatesCount += (item.found || (item.candidates ? item.candidates.length : 0));
        }
        return {
            totalCollisions: parsed.totalCollisions || (parsed.items ? parsed.items.length : 0),
            totalDeleted: parsed.totalDeleted || 0,
            totalCandidates: candidatesCount
        };
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
    },

    async runDedupe(isDryRun = false) {
        if (this.dedupeModal.executing) return;
        const mode = isDryRun ? "list" : this.dedupeModal.mode;
        const byHash = Boolean(this.dedupeModal.byHash);
        const dedupeBoth = Boolean(this.dedupeModal.dedupeBoth);
        const desc = byHash ? "checksum hash (--by-hash)" : "same-folder duplicate names";
        const scopeDesc = dedupeBoth ? " (paired passes: remote + local)" : "";
        this.dedupeModal.executing = true;
        this.dedupeModal.runId = null;
        this.dedupeModal.parsedOutput = null;
        this.dedupeModal.selectedCollision = null;
        this.dedupeModal.output = (isDryRun ? "Scanning duplicates on " : "Executing deduplication on ") + `${this.dedupeModal.remote}:${this.dedupeModal.path || '/'}${scopeDesc} via ${desc} (mode=${mode})...\n`;

        try {
            const payload = {
                remote: this.dedupeModal.remote,
                path: this.dedupeModal.path,
                mode,
                by_hash: byHash,
                dedupe_both: dedupeBoth,
                profile_id: this.activeProfile ? this.activeProfile.id : ""
            };
            const data = await window.API.post("/api/remotes/dedupe", payload);
            if (data.status === "error") {
                this.dedupeModal.output += `\nError: ${data.error || 'Deduplication failed'}`;
                this.dedupeModal.parsedOutput = parseDedupeOutput(this.dedupeModal.output);
                this.showToast(`Dedupe failed: ${data.error || 'Unknown error'}`, "error");
            } else {
                this.dedupeModal.runId = data.run_id || null;
                this.dedupeModal.output = data.output || "No duplicates found. All files are unique.";
                this.dedupeModal.parsedOutput = parseDedupeOutput(this.dedupeModal.output);
                if (this.dedupeModal.parsedOutput.items && this.dedupeModal.parsedOutput.items.length > 0) {
                    this.selectCollision(this.dedupeModal.parsedOutput.items[0]);
                }
                this.showToast(isDryRun ? "Duplicate scan complete (preview only)" : "Deduplication complete!", "success");
                if (!isDryRun && typeof this.fetchRemotes === "function") this.fetchRemotes(true);
            }
        } catch (err) {
            this.dedupeModal.output += `\nRequest failed: ${err.message}`;
            this.dedupeModal.parsedOutput = parseDedupeOutput(this.dedupeModal.output);
            this.showToast(`Request failed: ${err.message}`, "error");
        } finally {
            this.dedupeModal.executing = false;
        }
    }
};