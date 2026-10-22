/**
 * Storage Remotes & OAuth Connection Manager Module for Bisync Hub Dashboard
 * Handles remote listing, storage quota percentages, provider styling,
 * OAuth lifecycle monitoring, reconnect flows, Google Drive migration wizard,
 * credential validation, and audit event inspection.
 */

window.RemotesModule = {
    async fetchRemotes(showLoading = true) {
        if (showLoading) this.remotesData.loading = true;
        this.remotesData.error = null;
        try {
            const [remotesRes, authRes] = await Promise.allSettled([
                fetch("/api/remotes"),
                fetch("/api/remotes/auth")
            ]);

            if (remotesRes.status === "fulfilled" && remotesRes.value.ok) {
                const data = await remotesRes.value.json();
                this.remotesData.remotes = data.remotes || [];
                this.remotesData.summary = data.summary || {};
                this.remotesData.count = data.count || 0;
                this.remotesData.active_remote = data.active_remote || null;
                this.remotesData.last_checked = new Date().toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit", hour12: false });
            } else if (remotesRes.status === "fulfilled") {
                throw new Error("HTTP " + remotesRes.value.status);
            }

            if (authRes.status === "fulfilled" && authRes.value.ok) {
                const authData = await authRes.value.json();
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

    async reconnectRemote(name) {
        this.remotesData.reconnecting = { ...(this.remotesData.reconnecting || {}), [name]: true };
        this.reconnectModal = {
            show: true,
            remote: name,
            loading: true,
            authUrl: null,
            msg: "Initiating desktop OAuth reconnect flow...",
            error: null
        };

        try {
            const res = await fetch(`/api/remotes/${encodeURIComponent(name)}/reconnect`, {
                method: "POST",
                headers: { "X-Dashboard-Request": "1" }
            });
            const data = await res.json();
            if (data.status === "success") {
                this.reconnectModal.msg = data.msg || `Re-authenticated "${name}" successfully.`;
                this.reconnectModal.authUrl = data.auth_url || null;
                this.showToast(`Re-authenticated ${name} successfully!`, "success");
                await this.fetchRemotes(false);
                if (this.auditModal && this.auditModal.show) {
                    await this.viewAuthAudit();
                }
            } else {
                this.reconnectModal.error = data.error || data.msg || "Re-authentication failed.";
                this.reconnectModal.authUrl = data.auth_url || null;
                this.showToast(`Reconnect failed: ${this.reconnectModal.error}`, "error");
            }
        } catch (err) {
            this.reconnectModal.error = "Network error: " + err.message;
            this.showToast("Reconnect network error: " + err.message, "error");
        } finally {
            this.remotesData.reconnecting = { ...(this.remotesData.reconnecting || {}), [name]: false };
            this.reconnectModal.loading = false;
        }
    },

    async validateRemote(name) {
        // Immutable updates ensure Alpine.js reactivity triggers DOM re-render immediately
        this.remotesData.validating = { ...(this.remotesData.validating || {}), [name]: true };
        this.remotesData.validated = { ...(this.remotesData.validated || {}), [name]: false };
        this.showToast(`Testing connection to "${name}"...`, "info");
        try {
            const res = await fetch(`/api/remotes/${encodeURIComponent(name)}/validate`, {
                method: "POST",
                headers: { "X-Dashboard-Request": "1" }
            });
            const data = await res.json();
            if (data.valid) {
                this.showToast(`Remote "${name}" is connected and healthy!`, "success");
                this.remotesData.validated = { ...(this.remotesData.validated || {}), [name]: true };
                setTimeout(() => {
                    this.remotesData.validated = { ...(this.remotesData.validated || {}), [name]: false };
                }, 3000);
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

    openMigrationWizard(remote) {
        this.migrationModal = {
            show: true,
            remote: remote.name,
            type: remote.type,
            typeLabel: remote.type_label || remote.name,
            clientId: (remote.auth && remote.auth.client_id) ? remote.auth.client_id : "",
            clientSecret: "",
            showSecret: false,
            loading: false,
            error: null,
            success: null
        };
    },

    closeMigrationWizard() {
        this.migrationModal.show = false;
        this.migrationModal.error = null;
        this.migrationModal.success = null;
    },

    async saveMigrationCredentials() {
        if (!this.migrationModal.clientId || !this.migrationModal.clientId.trim()) {
            this.migrationModal.error = "Client ID is required.";
            return;
        }

        this.migrationModal.loading = true;
        this.migrationModal.error = null;
        this.migrationModal.success = null;

        try {
            const res = await fetch(`/api/remotes/${encodeURIComponent(this.migrationModal.remote)}/credentials`, {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                    "X-Dashboard-Request": "1"
                },
                body: JSON.stringify({
                    client_id: this.migrationModal.clientId.trim(),
                    client_secret: this.migrationModal.clientSecret ? this.migrationModal.clientSecret.trim() : ""
                })
            });
            const data = await res.json();
            if (data.status === "success") {
                this.migrationModal.success = "OAuth credentials successfully saved!";
                this.showToast("Credentials saved successfully!", "success");
                await this.fetchRemotes(false);
                setTimeout(() => {
                    this.closeMigrationWizard();
                }, 1200);
            } else {
                this.migrationModal.error = data.error || "Failed to update OAuth credentials.";
            }
        } catch (err) {
            this.migrationModal.error = "Network error: " + err.message;
        } finally {
            this.migrationModal.loading = false;
        }
    },

    async cancelReconnect(name) {
        try {
            await fetch(`/api/remotes/${encodeURIComponent(name)}/reconnect/cancel`, {
                method: "POST",
                headers: { "X-Dashboard-Request": "1" }
            });
        } catch (e) {
            console.error(e);
        } finally {
            this.remotesData.reconnecting = { ...(this.remotesData.reconnecting || {}), [name]: false };
            if (this.reconnectModal.remote === name) {
                this.reconnectModal.show = false;
                this.reconnectModal.loading = false;
            }
            this.showToast(`Re-authentication cancelled for "${name}".`, "info");
            await this.fetchRemotes(false);
            if (this.auditModal && this.auditModal.show) {
                await this.viewAuthAudit();
            }
        }
    },

    async viewAuthAudit() {
        this.auditModal = { show: true, loading: true, events: [] };
        try {
            const res = await fetch("/api/remotes/auth/audit?limit=50");
            if (res.ok) {
                const data = await res.json();
                this.auditModal.events = data.events || [];
            }
        } catch (err) {
            console.error("Failed to fetch auth audit:", err);
        } finally {
            this.auditModal.loading = false;
        }
    },

    async stopOAuth() {
        try {
            await fetch("/api/remotes/auth/oauth-stop", {
                method: "POST",
                headers: { "X-Dashboard-Request": "1" }
            });
            this.showToast("Stopped OAuth authentication processes.", "info");
            if (this.reconnectModal.show) {
                this.reconnectModal.loading = false;
                this.reconnectModal.error = "Process stopped by user.";
            }
        } catch (err) {
            this.showToast("Error stopping OAuth: " + err.message, "error");
        }
    },
};
