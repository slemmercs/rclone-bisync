/**
 * Fleet View & Multi-Workload Grid Module for Bisync Hub Dashboard
 * Provides aggregate health metrics, filtering, search, sorting, and tile status formatting.
 */

window.FleetModule = {
    fleetFilter: "all", // 'all' | 'healthy' | 'review' | 'failing' | 'syncing'
    fleetSearch: "",
    fleetSort: "status", // 'status' | 'name' | 'last_run'

    get fleetSummary() {
        const list = (this.profilesData && this.profilesData.list) || [];
        const summary = {
            total: list.length,
            healthy: 0,
            review: 0,
            failing: 0,
            syncing: 0
        };

        for (const p of list) {
            const isSync = typeof this.isProfileSyncing === "function" && this.isProfileSyncing(p);
            const status = (p.state && p.state.status) || "";
            const errCount = (p.state && p.state.error_count) || 0;
            const reviewReq = (p.state && p.state.review_required) || false;

            if (isSync || status === "running" || status === "syncing") {
                summary.syncing++;
            } else if (reviewReq || status === "review_required") {
                summary.review++;
            } else if (status === "failed" || status === "error" || errCount > 0) {
                summary.failing++;
            } else {
                summary.healthy++;
            }
        }
        return summary;
    },

    get filteredProfiles() {
        let list = [...((this.profilesData && this.profilesData.list) || [])];

        // Search Filter
        const query = (this.fleetSearch || "").trim().toLowerCase();
        if (query) {
            list = list.filter(p => {
                const name = (p.name || "").toLowerCase();
                const id = (p.id || "").toLowerCase();
                const remote = (p.remote || "").toLowerCase();
                const local = (p.local_path || "").toLowerCase();
                return name.includes(query) || id.includes(query) || remote.includes(query) || local.includes(query);
            });
        }

        // Status Filter
        if (this.fleetFilter && this.fleetFilter !== "all") {
            list = list.filter(p => {
                const isSync = typeof this.isProfileSyncing === "function" && this.isProfileSyncing(p);
                const status = (p.state && p.state.status) || "";
                const errCount = (p.state && p.state.error_count) || 0;
                const reviewReq = (p.state && p.state.review_required) || false;

                if (this.fleetFilter === "syncing") {
                    return isSync || status === "running" || status === "syncing";
                }
                if (this.fleetFilter === "review") {
                    return reviewReq || status === "review_required";
                }
                if (this.fleetFilter === "failing") {
                    return status === "failed" || status === "error" || errCount > 0;
                }
                if (this.fleetFilter === "healthy") {
                    return !isSync && !reviewReq && status !== "failed" && status !== "error" && errCount === 0;
                }
                return true;
            });
        }

        // Sort
        list.sort((a, b) => {
            if (this.fleetSort === "name") {
                return (a.name || "").localeCompare(b.name || "");
            }
            if (this.fleetSort === "last_run") {
                const aTime = (a.state && a.state.last_run_timestamp) || 0;
                const bTime = (b.state && b.state.last_run_timestamp) || 0;
                return bTime - aTime;
            }
            // Default sort: "status" (failing first, then review, then syncing, then healthy, then idle)
            const score = p => {
                const isSync = typeof this.isProfileSyncing === "function" && this.isProfileSyncing(p);
                const status = (p.state && p.state.status) || "";
                const errCount = (p.state && p.state.error_count) || 0;
                const reviewReq = (p.state && p.state.review_required) || false;

                if (status === "failed" || errCount > 0) return 1;
                if (reviewReq || status === "review_required") return 2;
                if (isSync || status === "running") return 3;
                if (status === "healthy") return 4;
                return 5;
            };
            return score(a) - score(b);
        });

        return list;
    },

    getProfileStatusText(p) {
        if (!p) return "UNKNOWN";
        const isSync = typeof this.isProfileSyncing === "function" && this.isProfileSyncing(p);
        const status = (p.state && p.state.status) || "";
        const errCount = (p.state && p.state.error_count) || 0;
        const reviewReq = (p.state && p.state.review_required) || false;

        if (isSync || status === "running" || status === "syncing") return "SYNCING";
        if (reviewReq || status === "review_required") return "REVIEW REQUIRED";
        if (status === "failed" || status === "error" || errCount > 0) return "FAILED";
        if (status === "healthy") return "HEALTHY";
        if (status === "idle") return "IDLE";
        return status ? status.toUpperCase() : "IDLE";
    },

    getProfileStatusBadgeClass(p) {
        const text = this.getProfileStatusText(p);
        switch (text) {
            case "SYNCING":
                return "bg-sky-500/10 text-sky-400 border-sky-500/20";
            case "REVIEW REQUIRED":
                return "bg-amber-500/10 text-amber-400 border-amber-500/20";
            case "FAILED":
                return "bg-rose-500/10 text-rose-400 border-rose-500/20";
            case "HEALTHY":
                return "bg-emerald-500/10 text-emerald-400 border-emerald-500/20";
            default:
                return "bg-slate-800 text-slate-400 border-slate-700/60";
        }
    },

    getProfileStatusDotClass(p) {
        const text = this.getProfileStatusText(p);
        switch (text) {
            case "SYNCING":
                return "bg-sky-400 animate-pulse";
            case "REVIEW REQUIRED":
                return "bg-amber-400";
            case "FAILED":
                return "bg-rose-400";
            case "HEALTHY":
                return "bg-emerald-400";
            default:
                return "bg-slate-500";
        }
    },

    formatProfileLastRun(p) {
        const ts = p.state && p.state.last_run_timestamp;
        if (!ts || ts === 0) return "Never";
        const sec = Math.floor(Date.now() / 1000 - ts);
        if (sec < 60) return "Just now";
        if (sec < 3600) return `${Math.floor(sec / 60)}m ago`;
        if (sec < 86400) return `${Math.floor(sec / 3600)}h ago`;
        return `${Math.floor(sec / 86400)}d ago`;
    },

    openProfileWorkspace(profileId) {
        if (typeof this.selectProfile === "function") {
            this.selectProfile(profileId);
        }
        if (typeof this.switchTab === "function") {
            this.switchTab("telemetry");
        }
    }
};
