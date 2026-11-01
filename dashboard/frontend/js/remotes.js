/**
 * Storage Remotes & OAuth Connection Manager Module for Bisync Hub Dashboard
 * Handles remote listing, storage quota percentages, provider styling,
 * OAuth lifecycle monitoring, reconnect flows, Google Drive migration wizard,
 * credential validation, and audit event inspection.
 */

window.RemotesModule = {
    remotesData: {
        remotes: [],
        summary: {},
        count: 0,
        showAll: false,
        loading: false,
        error: null,
        last_checked: null,
        authSummary: { total_remotes: 0, oauth_remotes: 0, connected: 0, migration_required: 0, authorization_required: 0, configuration_required: 0, errors: 0 },
        reconnecting: {},
        validating: {},
        validated: {}
    },

    async fetchRemotes(showLoading = true) {
        if (showLoading) this.remotesData.loading = true;
        this.remotesData.error = null;
        try {
            const [remotesRes, authRes] = await Promise.allSettled([
                window.API.get("/api/remotes"),
                window.API.get("/api/remotes/auth")
            ]);

            if (remotesRes.status === "fulfilled") {
                const data = remotesRes.value;
                this.remotesData.remotes = data.remotes || [];
                this.remotesData.summary = data.summary || {};
                this.remotesData.count = data.count || 0;
                this.remotesData.active_remote = data.active_remote || null;
                this.remotesData.last_checked = new Date().toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit", hour12: false });
            } else {
                throw remotesRes.reason;
            }

            if (authRes.status === "fulfilled") {
                const authData = authRes.value;
                this.remotesData.authSummary = authData.summary || {};
                const authMap = {};
                (authData.remotes || []).forEach(r => {
                    authMap[r.name] = r;
                });
                this.remotesData.remotes.forEach(r => {
                    r.auth = authMap[r.name] || null;
                });
            }
        } catch (err) {
            this.remotesData.error = err.message;
        } finally {
            if (showLoading) this.remotesData.loading = false;
        }
    },

    async checkAuthAndQuotas(targetRemote = null) {
        this.remotesData.loading = true;
        const remoteToCheck = targetRemote !== undefined ? targetRemote : (this.activeProfile && this.activeProfile.remote ? this.activeProfile.remote : null);
        const msg = remoteToCheck ? `Checking quota for ${remoteToCheck}...` : "Checking authentication health & storage quotas...";
        this.showToast(msg, "info");
        const startTime = Date.now();
        try {
            const checkUrl = remoteToCheck ? `/api/remotes/auth/check?remote=${encodeURIComponent(remoteToCheck)}` : "/api/remotes/auth/check";
            const [remotesRes, checkRes] = await Promise.allSettled([
                window.API.get("/api/remotes"),
                window.API.post(checkUrl)
            ]);

            if (remotesRes.status === "fulfilled") {
                const data = remotesRes.value;
                this.remotesData.remotes = data.remotes || [];
                this.remotesData.summary = data.summary || {};
                this.remotesData.count = data.count || 0;
                this.remotesData.active_remote = data.active_remote || null;
                this.remotesData.last_checked = new Date().toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit", hour12: false });
            }
            if (checkRes.status === "fulfilled") {
                const authData = checkRes.value;
                this.remotesData.authSummary = authData.summary || {};
                const authMap = {};
                (authData.remotes || []).forEach(r => { authMap[r.name] = r; });
                this.remotesData.remotes.forEach(r => { r.auth = authMap[r.name] || null; });
            }

            const elapsed = Date.now() - startTime;
            if (elapsed < 600) await new Promise(r => setTimeout(r, 600 - elapsed));
            this.showToast("Authentication and storage quotas verified!", "success");
            if (this.auditModal && this.auditModal.show) await this.viewAuthAudit();
        } catch (err) {
            this.showToast("Check error: " + err.message, "error");
        } finally {
            this.remotesData.loading = false;
        }
    },

    getQuotaPct(used, total) {
        if (!total || total <= 0 || !used) return 0;
        return Math.min(100, Math.max(0, (used / total) * 100));
    },

    getProviderColor(type) {
        switch ((type || "").toLowerCase()) {
            case "drive":
                return { bg: "bg-amber-950/20", border: "border-amber-800/40", text: "text-amber-400", badge: "bg-amber-500/10 text-amber-300 border-amber-500/20", bar: "bg-amber-500" };
            case "onedrive":
                return { bg: "bg-sky-950/20", border: "border-sky-800/40", text: "text-sky-400", badge: "bg-sky-500/10 text-sky-300 border-sky-500/20", bar: "bg-sky-500" };
            case "s3":
                return { bg: "bg-orange-950/20", border: "border-orange-800/40", text: "text-orange-400", badge: "bg-orange-500/10 text-orange-300 border-orange-500/20", bar: "bg-orange-500" };
            case "dropbox":
                return { bg: "bg-blue-950/20", border: "border-blue-800/40", text: "text-blue-400", badge: "bg-blue-500/10 text-blue-300 border-blue-500/20", bar: "bg-blue-500" };
            default:
                return { bg: "bg-indigo-950/20", border: "border-indigo-800/40", text: "text-indigo-400", badge: "bg-indigo-500/10 text-indigo-300 border-indigo-500/20", bar: "bg-indigo-500" };
        }
    },

    getAuthBadge(status) {
        switch ((status || "").toLowerCase()) {
            case "connected":
                return { text: "Connected", class: "bg-emerald-500/10 text-emerald-400 border-emerald-500/30", textClass: "text-emerald-400", dot: "bg-emerald-400" };
            case "migration_required":
                return { text: "Migration Recommended", class: "bg-amber-500/10 text-amber-300 border-amber-500/30", textClass: "text-amber-400", dot: "bg-amber-400" };
            case "authorization_required":
                return { text: "Auth Required", class: "bg-rose-500/10 text-rose-300 border-rose-500/30", textClass: "text-rose-400", dot: "bg-rose-400" };
            case "configuration_required":
                return { text: "Config Required", class: "bg-amber-500/10 text-amber-300 border-amber-500/30", textClass: "text-amber-400", dot: "bg-amber-400" };
            case "error":
                return { text: "Auth Error", class: "bg-red-500/10 text-red-300 border-red-500/30", textClass: "text-rose-400", dot: "bg-red-400" };
            default:
                return { text: "Standard", class: "bg-slate-800 text-slate-400 border-slate-700/60", textClass: "text-slate-400", dot: "bg-slate-500" };
        }
    },

    async validateRemote(name) {
        if (!name) return;
        if (this.remotesData.validating && this.remotesData.validating[name]) return;
        // Immutable updates ensure Alpine.js reactivity triggers DOM re-render immediately
        this.remotesData.validating = { ...(this.remotesData.validating || {}), [name]: true };
        this.remotesData.validated = { ...(this.remotesData.validated || {}), [name]: false };
        this.showToast(`Testing connection to "${name}"...`, "info");
        const startTime = Date.now();
        try {
            const data = await window.API.post(`/api/remotes/${encodeURIComponent(name)}/validate`);

            // Enforce minimum visual duration (600ms) so user clearly perceives the check
            const elapsed = Date.now() - startTime;
            if (elapsed < 600) {
                await new Promise(r => setTimeout(r, 600 - elapsed));
            }

            if (data.valid) {
                this.showToast(`Remote "${name}" is connected and healthy!`, "success");
                this.remotesData.validated = { ...(this.remotesData.validated || {}), [name]: true };
                setTimeout(() => {
                    this.remotesData.validated = { ...(this.remotesData.validated || {}), [name]: false };
                }, 4000);
            } else {
                this.showToast(`Validation issue on "${name}": ${data.msg || data.error || 'Connection failed'}`, "error");
            }
            await this.fetchRemotes(false);
            if (this.auditModal && this.auditModal.show) {
                await this.viewAuthAudit();
            }
        } catch (err) {
            this.showToast(`Validation error: ${err.message}`, "error");
        } finally {
            this.remotesData.validating = { ...(this.remotesData.validating || {}), [name]: false };
        }
    },

    isRemotesLoading() { return Boolean(this.remotesData?.loading); },
    isRemoteValidating(remote) { const k = typeof remote === "string" ? remote : remote?.name; return Boolean(k && this.remotesData?.validating?.[k]); },

    isRemoteActiveTarget(remote) {
        if (!remote) return false;
        const rName = (remote.name || "").replace(/:$/, "").trim();
        if (this.activeProfile && this.activeProfile.remote) return rName === this.activeProfile.remote.replace(/:$/, "").trim();
        return !!remote.is_active_target;
    },

    viewSyncForRemote(remoteName) {
        if (this.profilesData && this.profilesData.list) {
            const clean = (remoteName || "").replace(/:$/, "").trim();
            const prof = this.profilesData.list.find(p => (p.remote || "").replace(/:$/, "").trim() === clean);
            if (prof && typeof this.selectProfile === "function") this.selectProfile(prof.id);
        }
        this.switchTab("telemetry");
    },

    getDisplayedRemotes() {
        if (!this.remotesData) return [];
        if (this.remotesData.showAll || !this.activeProfile || !this.activeProfile.remote) {
            return this.remotesData.remotes || [];
        }
        const target = this.activeProfile.remote.replace(/:$/, "").trim();
        const matched = (this.remotesData.remotes || []).filter(r => (r.name || "").replace(/:$/, "").trim() === target);
        return matched.length > 0 ? matched : (this.remotesData.remotes || []);
    }
};
