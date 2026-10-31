/**
 * Interactive Sync Review & Resolution Engine Module for Bisync Hub Dashboard
 * Modularized Alpine.js architecture. Zero build step, offline-ready.
 * Implements "Review. Decide. Sync." workflow with pre-mutation backups and rollback.
 */

window.ReviewModule = {
    async fetchReviewSession(showLoading = false) {
        if (!showLoading && (this.reviewState.executing || this.reviewState.rollingBack)) {
            return;
        }
        if (showLoading) this.reviewState.loading = true;
        try {
            const data = await window.API.get("/api/review/session");
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

    get reviewCandidates() {
        const sess = this.reviewState?.session;
        if (!sess) return [];
        const dels = (sess.deletions || []).map(d => ({
            ...d,
            type: "deletion",
            candidateSide: d.side || "Path1",
            path: d.path
        }));
        const cnfs = (sess.conflicts || []).map(c => ({
            ...c,
            type: "conflict",
            candidateSide: "Both",
            path: c.path
        }));
        return [...dels, ...cnfs];
    },

    get filteredReviewCandidates() {
        let list = this.reviewCandidates;
        const q = (this.reviewState.search || "").toLowerCase().trim();
        const typeFilter = this.reviewState.filter;
        const statusFilter = this.reviewState.statusFilter;

        if (typeFilter === "deletions") {
            list = list.filter(c => c.type === "deletion");
        } else if (typeFilter === "conflicts") {
            list = list.filter(c => c.type === "conflict");
        }

        if (statusFilter === "pending") {
            list = list.filter(c => !c.decision || c.decision === "pending");
        } else if (statusFilter === "decided") {
            list = list.filter(c => c.decision && c.decision !== "pending");
        }

        if (q) {
            list = list.filter(c => (c.path || "").toLowerCase().includes(q) || (c.reason || "").toLowerCase().includes(q));
        }

        return list;
    },

    get reviewStats() {
        const all = this.reviewCandidates;
        const dels = all.filter(c => c.type === "deletion");
        const cnfs = all.filter(c => c.type === "conflict");
        const decided = all.filter(c => c.decision && c.decision !== "pending");
        const pending = all.filter(c => !c.decision || c.decision === "pending");

        return {
            total: all.length,
            deletions: dels.length,
            conflicts: cnfs.length,
            decided: decided.length,
            pending: pending.length
        };
    },

    get hasPendingReviewDecisions() {
        return this.reviewStats.pending > 0;
    },

    get canApplyReview() {
        const s = this.reviewState;
        if (!s.session || s.executing || s.rollingBack) return false;
        return this.reviewStats.decided > 0 || s.session.status === "STAGED";
    },

    get canRollbackReview() {
        const sess = this.reviewState.session;
        if (!sess || this.reviewState.executing || this.reviewState.rollingBack) return false;
        return sess.status === "COMPLETED" || sess.status === "APPLIED" || (sess.actions && sess.actions.length > 0);
    },

    async stageReviewDecision(candidateId, decision) {
        if (!this.reviewState.session) return;
        const candidate = this.reviewCandidates.find(c => c.id === candidateId);
        if (candidate) {
            candidate.decision = decision;
        }

        try {
            const data = await window.API.post("/api/review/action", {
                session_id: this.reviewState.session.id,
                candidate_id: candidateId,
                decision: decision
            });
            if (data.status === "error") {
                this.showToast("Failed to stage decision: " + (data.error || "Unknown error"), "error");
                await this.fetchReviewSession(false);
            } else if (data.session) {
                this.reviewState.session = data.session;
            }
        } catch (err) {
            this.showToast("Failed to stage decision: " + err.message, "error");
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
            const data = await window.API.post("/api/review/action", {
                session_id: this.reviewState.session.id,
                candidate_ids: candidateIds,
                decision: decision
            });
            if (data.status === "error") {
                this.showToast("Batch action failed: " + (data.error || "Unknown error"), "error");
                await this.fetchReviewSession(false);
            } else if (data.session) {
                this.reviewState.session = data.session;
                this.showToast(`Batch staged "${decision}" on ${candidateIds.length} candidate(s).`, "success");
            }
        } catch (err) {
            this.showToast("Batch action failed: " + err.message, "error");
            await this.fetchReviewSession(false);
        }
    },

    async applyBatchRecommendations(minConfidence = 0.70, maxRisk = "MEDIUM") {
        if (!this.reviewState.session) return;
        this.reviewState.loading = true;
        try {
            const data = await window.API.post("/api/review/action-batch", {
                session_id: this.reviewState.session.id,
                min_confidence: minConfidence,
                max_allowed_risk: maxRisk
            });
            if (data.status === "error") {
                this.showToast("Batch recommendation failed: " + (data.error || "Unknown error"), "error");
            } else {
                if (data.session) {
                    this.reviewState.session = data.session;
                }
                this.showToast(`Auto-staged recommendations for ${data.staged_count || 0} candidate(s).`, "success");
            }
        } catch (err) {
            this.showToast("Failed to apply recommendations: " + err.message, "error");
        } finally {
            this.reviewState.loading = false;
        }
    },

    // Execution, rollback, dismissal, and badges extracted to review_actions.js
};

