/**
 * Review Execution, Rollback, Dismissal, and Badging Module
 * Modular companion to review.js keeping all modules strictly under 400 lines of code.
 */

window.ReviewActionsModule = {
    async executeReviewSession() {
        if (!this.reviewState.session) return;
        const doExecute = async () => {
            this.reviewState.executing = true;
            this.reviewState.lastResult = null;

            try {
                const data = await window.API.post("/api/review/execute", {
                    session_id: this.reviewState.session.id
                });
                this.reviewState.lastResult = data;
                if (data.status === "error") {
                    this.showToast("Resolution execution failed: " + (data.error || "Unknown error"), "error");
                    await this.fetchReviewSession(false);
                } else {
                    const appliedCount = data.applied || 0;
                    this.showToast(`Applied ${appliedCount} resolution(s) with pre-mutation backups. Starting sync to finalize...`, "success");
                    this.closeReviewModal();
                    this.reviewState.active = false;
                    this.reviewState.session = null;

                    if (this.refreshExplorer) this.refreshExplorer();
                    if (this.triggerAction) {
                        await this.triggerAction("force_sync");
                    } else if (this.poll) {
                        await this.poll();
                    }
                }
            } catch (err) {
                this.showToast("Execution failed: " + err.message, "error");
            } finally {
                this.reviewState.executing = false;
            }
        };

        if (this.hasPendingReviewDecisions) {
            if (typeof this.askConfirmation === "function") {
                this.askConfirmation({
                    title: "Pending Decisions",
                    message: `There are still ${this.reviewStats.pending} pending candidate(s). They will be skipped.\n\nDo you wish to execute resolutions for decided items?`,
                    confirmText: "Execute Decided",
                    confirmClass: "bg-indigo-600 hover:bg-indigo-500 shadow-indigo-950/40",
                    icon: "warning",
                    onConfirm: doExecute
                });
                return;
            } else if (!confirm(`There are still ${this.reviewStats.pending} pending candidate(s). They will be skipped. Do you wish to execute resolutions for decided items?`)) {
                return;
            }
        }
        await doExecute();
    },

    async rollbackReviewSession() {
        if (!this.reviewState.session) return;
        const doRollback = async () => {
            this.reviewState.rollingBack = true;
            try {
                const data = await window.API.post("/api/review/rollback", {
                    session_id: this.reviewState.session.id
                });
                if (data.status === "error") {
                    this.showToast("Rollback failed: " + (data.error || "Unknown error"), "error");
                } else {
                    this.showToast(`Successfully restored ${data.restored} item(s) from backups.`, "success");
                }
                await this.fetchReviewSession(false);
                if (this.refreshExplorer) this.refreshExplorer();
                if (this.poll) await this.poll();
            } catch (err) {
                this.showToast("Rollback failed: " + err.message, "error");
            } finally {
                this.reviewState.rollingBack = false;
            }
        };

        if (typeof this.askConfirmation === "function") {
            this.askConfirmation({
                title: "Rollback Review Session",
                message: "Are you sure you want to rollback all applied resolutions from backups?\nThis will restore original files from safety snapshots.",
                confirmText: "Rollback Files",
                confirmClass: "bg-rose-600 hover:bg-rose-500 shadow-rose-950/40",
                icon: "danger",
                onConfirm: doRollback
            });
        } else if (confirm("Are you sure you want to rollback all applied resolutions from backups? This will restore original files.")) {
            await doRollback();
        }
    },

    async dismissReviewSession() {
        if (!this.reviewState.session) return;
        const doDismiss = async () => {
            this.reviewState.dismissing = true;
            try {
                const data = await window.API.post("/api/review/dismiss", {
                    session_id: this.reviewState.session.id
                });
                if (data.status === "error") {
                    this.showToast("Failed to dismiss session: " + (data.error || "Unknown error"), "error");
                } else {
                    this.showToast("Review session dismissed.", "info");
                    this.reviewState.showModal = false;
                    this.reviewState.active = false;
                    this.reviewState.session = null;
                }
                if (this.poll) await this.poll();
            } catch (err) {
                this.showToast("Failed to dismiss: " + err.message, "error");
            } finally {
                this.reviewState.dismissing = false;
            }
        };

        if (typeof this.askConfirmation === "function") {
            this.askConfirmation({
                title: "Dismiss Review Session",
                message: "Dismiss this review session?\nNo file mutations will be executed, and the safety sync barrier will be released.",
                confirmText: "Dismiss Session",
                confirmClass: "bg-slate-700 hover:bg-slate-600",
                icon: "warning",
                onConfirm: doDismiss
            });
        } else if (confirm("Dismiss this review session? No mutations will be executed, but the safety barrier will be released.")) {
            await doDismiss();
        }
    },

    getItemReviewBadge(paneKey, item) {
        if (!this.reviewState?.active || !this.reviewState?.session || !item) return null;
        const pane = this.explorer ? this.explorer[paneKey] : null;
        const fullRelPath = (pane && pane.path) ? `${pane.path}/${item.Name}` : item.Name;

        const candidate = this.reviewCandidates.find(c => c.path === fullRelPath || c.path === item.Name);
        if (!candidate) return null;

        if (candidate.type === "deletion") {
            if (candidate.decision && candidate.decision !== "pending") {
                return {
                    label: candidate.decision === "accept-delete" ? "DELETE AUTHORIZED" : "PRESERVE",
                    class: candidate.decision === "accept-delete" ? "bg-rose-950/80 text-rose-300 border-rose-700/60" : "bg-emerald-950/80 text-emerald-300 border-emerald-700/60",
                    title: `Deletion decision: ${candidate.decision}`
                };
            }
            return {
                label: "DELETION QUEUED",
                class: "bg-rose-950/70 text-rose-300 border-rose-800/80 animate-pulse",
                title: "Queued deletion candidate awaiting review decision"
            };
        } else if (candidate.type === "conflict") {
            if (candidate.decision && candidate.decision !== "pending") {
                return {
                    label: candidate.decision.toUpperCase(),
                    class: "bg-indigo-950/80 text-indigo-300 border-indigo-700/60",
                    title: `Conflict decision: ${candidate.decision}`
                };
            }
            return {
                label: "CONFLICT",
                class: "bg-amber-950/70 text-amber-300 border-amber-800/80 animate-pulse",
                title: "File modified on both sides. Awaiting review decision"
            };
        }

        return null;
    },

    getItemReviewRowClass(paneKey, item) {
        const badge = this.getItemReviewBadge(paneKey, item);
        if (badge) {
            if (badge.label.includes("DELETE") || badge.label.includes("DELETION")) {
                return "bg-rose-500/10 border-l-2 border-rose-500";
            }
            if (badge.label.includes("CONFLICT")) {
                return "bg-amber-500/10 border-l-2 border-amber-500";
            }
            return "bg-emerald-500/10 border-l-2 border-emerald-500";
        }
        if (item && item.Name === "RCLONE_TEST") {
            return "bg-amber-500/5 border-l-2 border-amber-500";
        }
        if (typeof this.isItemExcluded === "function" && this.isItemExcluded(item)) {
            return "opacity-50 text-slate-500";
        }
        return "";
    },

    isReviewDismissing() {
        return Boolean(this.reviewState?.dismissing);
    },

    isReviewRollingBack() {
        return Boolean(this.reviewState?.rollingBack);
    },

    cannotApplyReview() {
        return Boolean(!this.canApplyReview || this.reviewState?.executing);
    }
};
