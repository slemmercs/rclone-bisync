/**
 * Storage Remotes Modals & Wizards Module for Bisync Hub Dashboard
 * Handles Reconnect flow, OAuth stop/cancel, Auth Audit modal,
 * and Google Drive OAuth Migration Wizard.
 */

window.RemotesModalsModule = {
    migrationModal: {
        show: false,
        remote: "",
        type: "",
        typeLabel: "",
        clientId: "",
        clientSecret: "",
        showSecret: false,
        loading: false,
        error: null,
        success: null
    },
    reconnectModal: { show: false, remote: "", loading: false, authUrl: null, msg: null, error: null },
    auditModal: {
        show: false,
        loading: false,
        events: []
    },

    async reconnectRemote(name) {
        this.remotesData.reconnecting = { ...(this.remotesData.reconnecting || {}), [name]: true };
        this.reconnectModal = { show: true, remote: name, loading: true, authUrl: null, msg: "Initiating desktop OAuth reconnect flow...", error: null };
        try {
            const data = await window.API.post(`/api/remotes/${encodeURIComponent(name)}/reconnect`);
            if (data.status === "success") {
                this.reconnectModal.msg = data.msg || `Re-authenticated "${name}" successfully.`;
                this.reconnectModal.authUrl = data.auth_url || null;
                this.showToast(`Re-authenticated ${name} successfully!`, "success");
                await this.fetchRemotes(false);
                if (this.auditModal && this.auditModal.show) await this.viewAuthAudit();
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

    async cancelReconnect(name) {
        try {
            await window.API.post(`/api/remotes/${encodeURIComponent(name)}/reconnect/cancel`);
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
            if (this.auditModal && this.auditModal.show) await this.viewAuthAudit();
        }
    },

    async viewAuthAudit() {
        this.auditModal = { show: true, loading: true, events: [] };
        try {
            const data = await window.API.get("/api/remotes/auth/audit?limit=50");
            this.auditModal.events = data.events || [];
        } catch (err) {
            console.error("Failed to fetch auth audit:", err);
        } finally {
            this.auditModal.loading = false;
        }
    },

    async stopOAuth() {
        try {
            await window.API.post("/api/remotes/auth/oauth-stop");
            this.showToast("Stopped OAuth authentication processes.", "info");
            if (this.reconnectModal.show) {
                this.reconnectModal.loading = false;
                this.reconnectModal.error = "Process stopped by user.";
            }
        } catch (err) {
            this.showToast("Error stopping OAuth: " + err.message, "error");
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
            const data = await window.API.post(`/api/remotes/${encodeURIComponent(this.migrationModal.remote)}/credentials`, {
                client_id: this.migrationModal.clientId.trim(),
                client_secret: this.migrationModal.clientSecret ? this.migrationModal.clientSecret.trim() : ""
            });
            if (data.status === "success") {
                this.migrationModal.success = "OAuth credentials successfully saved!";
                this.showToast("Credentials saved successfully!", "success");
                await this.fetchRemotes(false);
                setTimeout(() => { this.closeMigrationWizard(); }, 1200);
            } else {
                this.migrationModal.error = data.error || "Failed to update OAuth credentials.";
            }
        } catch (err) {
            this.migrationModal.error = "Network error: " + err.message;
        } finally {
            this.migrationModal.loading = false;
        }
    },

    isMigrationLoading() { return Boolean(this.migrationModal?.loading); },
    cannotSaveMigration() { return Boolean(this.migrationModal?.loading || !this.migrationModal?.clientId?.trim()); }
};
