/**
 * Rclone Bisync Dashboard — Deduplication Batch Staging Module
 * Manages staging individual candidate decisions (delete, rename, keep),
 * reviewing staged batch actions, and executing atomic dedupe batches with
 * post-dedupe resync queuing.
 */

window.DedupeBatchModule = {
    dedupeBatch: {
        stagedActions: {},
        applying: false,
        triggerResync: true,
        drawerOpen: false
    },

    isDedupeBatchApplying() {
        return Boolean(this.dedupeBatch?.applying);
    },

    getCandidateKey(collision, cand) {
        if (!cand) return "";
        const cPath = collision?.path || collision?.name || "";
        const cId = cand.id || "";
        const idx = cand.index !== undefined ? cand.index : "";
        return `${cPath}::${cId}::${idx}`;
    },

    stageCandidateAction(collision, cand, action, newName = "") {
        if (!collision || !cand) return;
        const key = this.getCandidateKey(collision, cand);
        const actionObj = {
            id: cand.id || "",
            path: collision.path || collision.name || "",
            action: action,
            new_name: newName,
            size: cand.size !== undefined ? cand.size : 0,
            mod_time: cand.modtime || "",
            candidate: cand.index !== undefined ? cand.index : 0
        };
        this.dedupeBatch.stagedActions = {
            ...this.dedupeBatch.stagedActions,
            [key]: actionObj
        };
        if (typeof this.showToast === "function") {
            const label = action === "delete" ? "Delete" : (action === "rename" ? `Rename (${newName})` : "Keep");
            this.showToast(`Staged ${label} for Copy #${cand.index || 1}`, "info");
        }
    },

    unstageCandidateAction(key) {
        if (!this.dedupeBatch.stagedActions || !this.dedupeBatch.stagedActions[key]) return;
        const copy = { ...this.dedupeBatch.stagedActions };
        delete copy[key];
        this.dedupeBatch.stagedActions = copy;
    },

    clearStagedDedupeActions() {
        this.dedupeBatch.stagedActions = {};
    },

    getStagedDedupeAction(collision, cand) {
        const key = this.getCandidateKey(collision, cand);
        return this.dedupeBatch?.stagedActions?.[key] || null;
    },

    getStagedDedupeList() {
        return Object.values(this.dedupeBatch?.stagedActions || {});
    },

    getStagedDedupeCount() {
        return Object.keys(this.dedupeBatch?.stagedActions || {}).length;
    },

    stageKeepOnly(collision, keptCand) {
        if (!collision || !collision.candidates) return;
        const newStaged = { ...(this.dedupeBatch?.stagedActions || {}) };
        for (const cand of collision.candidates) {
            const key = this.getCandidateKey(collision, cand);
            if (cand === keptCand || (cand.id && cand.id === keptCand.id) || (cand.index === keptCand.index)) {
                newStaged[key] = {
                    id: cand.id || "",
                    path: collision.path || collision.name || "",
                    action: "keep",
                    size: cand.size || 0,
                    mod_time: cand.modtime || "",
                    candidate: cand.index || 0
                };
            } else {
                newStaged[key] = {
                    id: cand.id || "",
                    path: collision.path || collision.name || "",
                    action: "delete",
                    size: cand.size || 0,
                    mod_time: cand.modtime || "",
                    candidate: cand.index || 0
                };
            }
        }
        this.dedupeBatch.stagedActions = newStaged;
        if (typeof this.showToast === "function") {
            this.showToast(`Staged Keep Copy #${keptCand.index || 1} (marking others for deletion)`, "info");
        }
    },

    promptStageRename(collision, cand) {
        if (!collision || !cand) return;
        const currentName = collision.name || (collision.path ? collision.path.split("/").pop() : "file");
        const ext = currentName.includes(".") ? ("." + currentName.split(".").pop()) : "";
        const base = ext ? currentName.slice(0, -ext.length) : currentName;
        const suggestion = `${base}-copy${cand.index || 2}${ext}`;

        if (typeof this.askPrompt === "function") {
            this.askPrompt({
                title: `Rename Copy #${cand.index || 1}`,
                message: `Provide a new filename for the duplicate candidate:\n${collision.path || collision.name || ''}`,
                inputLabel: "New Filename",
                inputValue: suggestion,
                inputPlaceholder: "e.g. photo-copy2.jpg",
                confirmText: "Stage Rename",
                confirmClass: "bg-amber-600 hover:bg-amber-500",
                icon: "warning",
                onConfirm: (newName) => {
                    if (!newName || !newName.trim() || newName.trim() === currentName) {
                        return;
                    }
                    this.stageCandidateAction(collision, cand, "rename", newName.trim());
                }
            });
        } else {
            const newName = prompt(`Enter new filename for Copy #${cand.index || 1}:`, suggestion);
            if (!newName || !newName.trim() || newName.trim() === currentName) {
                return;
            }
            this.stageCandidateAction(collision, cand, "rename", newName.trim());
        }
    },

    async applyStagedDedupeBatch() {
        const list = this.getStagedDedupeList();
        if (!list || list.length === 0) {
            if (typeof this.showToast === "function") this.showToast("No staged actions to apply", "warning");
            return;
        }
        this.dedupeBatch.applying = true;
        try {
            const payload = {
                remote: this.dedupeModal.remote,
                path: this.dedupeModal.path || "",
                profile_id: this.activeProfile ? this.activeProfile.id : "",
                actions: list,
                trigger_resync: Boolean(this.dedupeBatch.triggerResync)
            };
            const data = await window.API.post("/api/remotes/dedupe/batch", payload);
            if (data.status === "error") {
                const errMsg = data.error || (data.errors && data.errors[0]) || 'Unknown error';
                if (typeof this.showToast === "function") {
                    this.showToast(`Batch failed: ${errMsg}`, "error");
                }
                if (this.dedupeModal) {
                    let errDetails = `\n[ERROR] Batch deduplication failed: ${errMsg}`;
                    if (data.errors && data.errors.length > 0) {
                        errDetails += "\n" + data.errors.map(e => `  - ${e}`).join("\n");
                    }
                    this.dedupeModal.output = (this.dedupeModal.output || "") + errDetails;
                }
            } else {
                const succ = data.success_count || 0;
                const fail = data.failure_count || 0;
                if (typeof this.showToast === "function") {
                    this.showToast(`Batch applied: ${succ} succeeded, ${fail} failed`, fail > 0 ? "warning" : "success");
                }
                this.clearStagedDedupeActions();
                if (this.dedupeModal) {
                    let out = (this.dedupeModal.output || "") + `\nBatch execution applied: ${succ} succeeded, ${fail} failed.`;
                    if (data.errors && data.errors.length > 0) {
                        out += "\nFailures encountered:\n" + data.errors.map(e => `  - ${e}`).join("\n");
                    }
                    this.dedupeModal.output = out;
                }
                if (data.resync_queued && typeof this.showToast === "function") {
                    this.showToast(`Profile resync queued: ${data.resync_message || ''}`, "info");
                }
                if (typeof this.loadDedupeExplorer === "function") {
                    this.loadDedupeExplorer();
                }
                if (typeof this.fetchRemotes === "function") {
                    this.fetchRemotes(true);
                }
            }
        } catch (err) {
            console.error("Batch apply failed:", err);
            if (typeof this.showToast === "function") {
                this.showToast(`Batch error: ${err.message}`, "error");
            }
        } finally {
            this.dedupeBatch.applying = false;
        }
    }
};
