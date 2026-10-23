/**
 * Interactive Sync Review & Resolution Engine Module for Bisync Hub Dashboard
 * Modularized Alpine.js architecture. Zero build step, offline-ready.
 * Implements "Review. Decide. Sync." workflow with pre-mutation backups and rollback.
 */

window.ReviewModule = {
    async fetchReviewSession(showLoading = false) {
        if (showLoading) this.reviewState.loading = true;
        try {
            const res = await fetch("/api/review/session");
            if (!res.ok) throw new Error("HTTP " + res.status);
            const data = await res.json();
            this.reviewState.active = Boolean(data.active && data.session);
            this.reviewState.session = data.session || null;
            this.reviewState.error = data.error || null;
        } catch (err) {
            console.error("Failed to fetch review session:", err);
            this.reviewState.error = err.message;
        } finally {
            if (showLoading) this.reviewState.loading = false;
        }
    },

    async openReviewModal() {
        this.reviewState.showModal = true;
        this.reviewState.search = "";
        this.reviewState.filter = "all";
        this.reviewState.statusFilter = "all";
        this.reviewState.lastResult = null;
        await this.fetchReviewSession(true);
    },

    closeReviewModal() {
        this.reviewState.showModal = false;
    },

    async stageReviewDecision(candidateId, decision) {
        if (!this.reviewState.session) return;
        const candidate = this.reviewCandidates.find(c => c.id === candidateId);
        if (candidate) {
            candidate.decision = decision;
        }

        try {
            const res = await fetch("/api/review/action", {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                    "X-Dashboard-Request": "1"
                },
                body: JSON.stringify({
                    session_id: this.reviewState.session.id,
                    candidate_id: candidateId,
                    decision: decision
                })
            });
            const data = await res.json();
            if (!res.ok || data.status === "error") {
                this.showToast("Failed to stage decision: " + (data.error || "Unknown error"), "error");
                await this.fetchReviewSession(false);
            } else if (data.session) {
                this.reviewState.session = data.session;
            }
        } catch (err) {
            this.showToast("Network error staging decision: " + err.message, "error");
            await this.fetchReviewSession(false);
        }
    },

    async batchStageReviewDecisions(decision, filterType = null) {
        if (!this.reviewState.session) return;
        let candidates = this.reviewCandidates;
        if (filterType) {
            candidates = candidates.filter(c => c.type === filterType);
        }

        const candidateIds = candidates.map(c => c.id);
        if (candidateIds.length === 0) return;

        candidates.forEach(c => { c.decision = decision; });

        try {
            const res = await fetch("/api/review/action", {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                    "X-Dashboard-Request": "1"
                },
                body: JSON.stringify({
                    session_id: this.reviewState.session.id,
                    candidate_ids: candidateIds,
                    decision: decision
                })
            });
            const data = await res.json();
            if (!res.ok || data.status === "error") {
                this.showToast("Batch action failed: " + (data.error || "Unknown error"), "error");
                await this.fetchReviewSession(false);
            } else if (data.session) {
                this.reviewState.session = data.session;
                this.showToast(`Batch staged "${decision}" on ${candidateIds.length} candidate(s).`, "success");
            }
        } catch (err) {
            this.showToast("Network error in batch action: " + err.message, "error");
            await this.fetchReviewSession(false);
        }
    },

    async applyBatchRecommendations(minConfidence = 0.70, maxRisk = "MEDIUM") {
        if (!this.reviewState.session) return;
        this.reviewState.loading = true;
        try {
            const res = await fetch("/api/review/action-batch", {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                    "X-Dashboard-Request": "1"
                },
                body: JSON.stringify({
                    session_id: this.reviewState.session.id,
                    min_confidence: minConfidence,
                    max_allowed_risk: maxRisk
                })
            });
            const data = await res.json();
            if (!res.ok || data.status === "error") {
                this.showToast("Batch recommendation failed: " + (data.error || "Unknown error"), "error");
            } else {
                if (data.session) {
                    this.reviewState.session = data.session;
                }
                this.showToast(`Auto-staged recommendations for ${data.staged_count || 0} conflict(s).`, "success");
            }
        } catch (err) {
            this.showToast("Network error applying batch recommendations: " + err.message, "error");
        } finally {
            this.reviewState.loading = false;
        }
    },


    async executeReviewSession() {
        if (!this.reviewState.session) return;
        if (this.hasPendingReviewDecisions) {
            const proceed = confirm(`There are still ${this.reviewStats.pending} pending candidate(s). They will be skipped. Do you wish to execute resolutions for decided items?`);
            if (!proceed) return;
        }

        this.reviewState.executing = true;
        this.reviewState.lastResult = null;

        try {
            const res = await fetch("/api/review/execute", {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                    "X-Dashboard-Request": "1"
                },
                body: JSON.stringify({
                    session_id: this.reviewState.session.id
                })
            });
            const data = await res.json();
            this.reviewState.lastResult = data;
            if (!res.ok || data.status === "error") {
                this.showToast("Resolution execution failed: " + (data.error || "Unknown error"), "error");
            } else {
                this.showToast(`Applied ${data.applied} resolution(s) with pre-mutation backups.`, "success");
            }
            await this.fetchReviewSession(false);
            if (this.refreshExplorer) this.refreshExplorer();
            if (this.poll) await this.poll();
        } catch (err) {
            this.showToast("Network error executing review: " + err.message, "error");
        } finally {
            this.reviewState.executing = false;
        }
    },

    async rollbackReviewSession() {
        if (!this.reviewState.session) return;
        const proceed = confirm("Are you sure you want to rollback all applied resolutions from backups? This will restore original files.");
        if (!proceed) return;

        this.reviewState.rollingBack = true;
        try {
            const res = await fetch("/api/review/rollback", {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                    "X-Dashboard-Request": "1"
                },
                body: JSON.stringify({
                    session_id: this.reviewState.session.id
                })
            });
            const data = await res.json();
            if (!res.ok || data.status === "error") {
                this.showToast("Rollback failed: " + (data.error || "Unknown error"), "error");
            } else {
                this.showToast(`Successfully restored ${data.restored} item(s) from backups.`, "success");
            }
            await this.fetchReviewSession(false);
            if (this.refreshExplorer) this.refreshExplorer();
            if (this.poll) await this.poll();
        } catch (err) {
            this.showToast("Network error during rollback: " + err.message, "error");
        } finally {
            this.reviewState.rollingBack = false;
        }
    },

    async dismissReviewSession() {
        if (!this.reviewState.session) return;
        const proceed = confirm("Dismiss this review session? No mutations will be executed, but the safety barrier will be released.");
        if (!proceed) return;

        this.reviewState.dismissing = true;
        try {
            const res = await fetch("/api/review/dismiss", {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                    "X-Dashboard-Request": "1"
                },
                body: JSON.stringify({
                    session_id: this.reviewState.session.id
                })
            });
            const data = await res.json();
            if (!res.ok || data.status === "error") {
                this.showToast("Failed to dismiss session: " + (data.error || "Unknown error"), "error");
            } else {
                this.showToast("Review session dismissed.", "info");
                this.reviewState.showModal = false;
                this.reviewState.active = false;
                this.reviewState.session = null;
            }
            if (this.poll) await this.poll();
        } catch (err) {
            this.showToast("Network error dismissing review: " + err.message, "error");
        } finally {
            this.reviewState.dismissing = false;
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
        if (!badge) {
            return item && item.Name === "RCLONE_TEST" ? "bg-amber-500/5 border-l-2 border-amber-500" : "";
        }
        if (badge.label.includes("DELETE") || badge.label.includes("DELETION")) {
            return "bg-rose-500/10 border-l-2 border-rose-500";
        }
        if (badge.label.includes("CONFLICT")) {
            return "bg-amber-500/10 border-l-2 border-amber-500";
        }
        return "bg-emerald-500/10 border-l-2 border-emerald-500";
    }
};
