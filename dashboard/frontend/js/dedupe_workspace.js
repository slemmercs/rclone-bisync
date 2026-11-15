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

    getDedupeCacheKey(remote, path) {
        const cleanRemote = (remote || "").replace(/:$/, "").trim();
        const cleanPath = (path || "").replace(/^\/+|\/+$/g, "");
        return `dedupe_scan_${cleanRemote}_${cleanPath}`;
    },

    getDedupeCache(remote, path) {
        const key = this.getDedupeCacheKey(remote, path);
        try {
            const raw = sessionStorage.getItem(key);
            if (raw) return JSON.parse(raw);
        } catch (_) {}
        return null;
    },

    setDedupeCache(remote, path, data) {
        const key = this.getDedupeCacheKey(remote, path);
        try {
            sessionStorage.setItem(key, JSON.stringify(data));
        } catch (_) {}
    },

    clearDedupeCache(remote, path) {
        const key = this.getDedupeCacheKey(remote, path);
        try {
            sessionStorage.removeItem(key);
        } catch (_) {}
    },

    clearDedupeCacheAndReset() {
        if (!this.dedupeModal) return;
        this.clearDedupeCache(this.dedupeModal.remote, this.dedupeModal.path);
        this.dedupeModal.parsedOutput = null;
        this.dedupeModal.output = "";
        this.dedupeModal.selectedCollision = null;
        this.dedupeModal.runId = null;
        if (typeof this.showToast === "function") {
            this.showToast("Deduplication scan cache cleared", "info");
        }
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

        // Preserve active session if re-opening same target with existing results
        if (this.dedupeModal && this.dedupeModal.remote === clean && this.dedupeModal.path === p && this.dedupeModal.parsedOutput) {
            this.dedupeModal.show = true;
            return;
        }

        const cached = this.getDedupeCache(clean, p);

        this.dedupeModal = {
            show: true,
            remote: clean,
            path: p,
            mode: cached?.mode || "oldest",
            byHash: cached ? Boolean(cached.byHash) : isOnedrive,
            isOnedrive: isOnedrive,
            dedupeBoth: cached ? Boolean(cached.dedupeBoth) : hasPairedProfile,
            folderBrowserOpen: false,
            folders: [],
            loadingFolders: false,
            executing: false,
            runId: cached?.runId || null,
            output: cached?.output || "",
            parsedOutput: cached?.parsedOutput || null,
            showRawOutput: false,
            filterQuery: "",
            selectedCollision: cached?.parsedOutput?.items?.[0] || null,
            activeTab: "candidates",
            explorerItems: [],
            explorerLoading: false,
            explorerError: null,
            explorerPath: p
        };
    },

    closeDedupeModal() {
        this.dedupeModal.show = false;
        if (this.dedupeModal?.parsedOutput) {
            this.setDedupeCache(this.dedupeModal.remote, this.dedupeModal.path, {
                output: this.dedupeModal.output,
                parsedOutput: this.dedupeModal.parsedOutput,
                runId: this.dedupeModal.runId,
                mode: this.dedupeModal.mode,
                byHash: this.dedupeModal.byHash,
                dedupeBoth: this.dedupeModal.dedupeBoth,
                timestamp: Date.now()
            });
        }
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
                if (this.dedupeModal.parsedOutput) {
                    this.setDedupeCache(this.dedupeModal.remote, this.dedupeModal.path, {
                        output: this.dedupeModal.output,
                        parsedOutput: this.dedupeModal.parsedOutput,
                        runId: this.dedupeModal.runId,
                        mode: this.dedupeModal.mode,
                        byHash: this.dedupeModal.byHash,
                        dedupeBoth: this.dedupeModal.dedupeBoth,
                        timestamp: Date.now()
                    });
                }
                if (this.dedupeModal.parsedOutput?.items && this.dedupeModal.parsedOutput.items.length > 0) {
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