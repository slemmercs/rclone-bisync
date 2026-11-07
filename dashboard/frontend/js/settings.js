/**
 * Global Fleet Settings Module for Bisync Hub Dashboard
 * Manages daemon-wide operational defaults, engine polling/timeouts,
 * and notification integrations backed by /api/config.
 */

window.SettingsModule = {
    globalSettingsModal: {
        show: false,
        tab: "defaults", // 'defaults' | 'engine' | 'notifications'
        loading: false,
        saving: false,
        error: null,
        config: {
            global_defaults: {
                max_delete: 0,
                conflict_resolve: "newer",
                compare_mode: "modtime",
                canary_validation: true,
                remove_empty_dirs: true,
                create_empty_dirs: true,
                lock_explorer_remote: true
            },
            daemon: {
                job_poll_initial_sleep_sec: 2,
                job_poll_max_sleep_sec: 10,
                job_timeout_seconds: 1800,
                lock_stale_timeout_seconds: 3600,
                rate_limit_backoff_hours: 1
            },
            notifications: {
                webhook_url: "",
                watchdog_ping_url: ""
            }
        },
        encryption: {
            loading: false,
            is_encrypted: false,
            has_daemon_passphrase: true,
            warning: "",
            config_file: "",
            new_password: "",
            old_password: "",
            confirm_password: "",
            daemon_password: "",
            saving_daemon_passphrase: false,
            error: null
        }
    },

    isGlobalSettingsSaving() {
        return Boolean(this.globalSettingsModal?.saving);
    },

    isEncryptionLoading() {
        return Boolean(this.globalSettingsModal?.encryption?.loading);
    },

    isSavingDaemonPassphrase() {
        return Boolean(this.globalSettingsModal?.encryption?.saving_daemon_passphrase);
    },

    isExplorerRemoteLocked() {
        return Boolean(this.globalSettingsModal?.config?.global_defaults?.lock_explorer_remote);
    },

    toggleExplorerRemoteLock() {
        if (!this.globalSettingsModal?.config?.global_defaults) return;
        this.globalSettingsModal.config.global_defaults.lock_explorer_remote = !this.globalSettingsModal.config.global_defaults.lock_explorer_remote;
        if (typeof this.saveGlobalSettings === "function") {
            this.saveGlobalSettings();
        }
    },

    async openGlobalSettingsModal(tab = "defaults") {
        this.globalSettingsModal.tab = tab;
        this.globalSettingsModal.error = null;
        this.globalSettingsModal.loading = true;
        this.globalSettingsModal.show = true;

        try {
            const data = await window.API.get("/api/config");
            if (data && data.status === "success") {
                if (data.global_defaults) {
                    this.globalSettingsModal.config.global_defaults = { ...data.global_defaults };
                    if (this.profilesData) {
                        this.profilesData.globalDefaults = { ...data.global_defaults };
                    }
                }
                if (data.daemon) {
                    this.globalSettingsModal.config.daemon = { ...data.daemon };
                }
                if (data.notifications) {
                    this.globalSettingsModal.config.notifications = { ...data.notifications };
                }
            }
        } catch (err) {
            this.globalSettingsModal.error = "Failed to load global configuration: " + err.message;
        } finally {
            this.globalSettingsModal.loading = false;
        }
        this.fetchConfigEncryptionStatus();
        if (typeof this.fetchBandwidthLimit === "function") {
            this.fetchBandwidthLimit();
        }
        if (typeof this.fetchCoreStats === "function") {
            this.fetchCoreStats();
        }
    },

    async fetchConfigEncryptionStatus() {
        try {
            const data = await window.API.get("/api/config/encryption");
            if (data) {
                this.globalSettingsModal.encryption.is_encrypted = Boolean(data.is_encrypted);
                this.globalSettingsModal.encryption.has_daemon_passphrase = Boolean(data.has_daemon_passphrase);
                this.globalSettingsModal.encryption.warning = data.warning || "";
                this.globalSettingsModal.encryption.config_file = data.config_file || "";
            }
        } catch (e) {
            console.warn("Failed to probe config encryption status:", e);
        }
    },

    async saveDaemonPassphrase() {
        const enc = this.globalSettingsModal.encryption;
        if (!enc.daemon_password || !enc.daemon_password.trim()) {
            if (typeof this.showToast === "function") this.showToast("Please enter the daemon passphrase.", "warning");
            return;
        }

        enc.saving_daemon_passphrase = true;
        try {
            const res = await window.API.post("/api/config/encryption/save-passphrase", {
                password: enc.daemon_password
            });
            if (res && res.status === "success") {
                if (typeof this.showToast === "function") {
                    this.showToast("Daemon passphrase verified and persisted to rc-secrets.env.", "success");
                }
                enc.daemon_password = "";
                await this.fetchConfigEncryptionStatus();
            } else {
                throw new Error(res.error || "Failed to persist passphrase");
            }
        } catch (err) {
            if (typeof this.showToast === "function") {
                this.showToast(`Verification Failed: ${err.message}`, "error");
            }
        } finally {
            enc.saving_daemon_passphrase = false;
        }
    },

    async setConfigPassword() {
        const enc = this.globalSettingsModal.encryption;
        if (!enc.new_password) {
            if (typeof this.showToast === "function") this.showToast("Please enter a new password.", "error");
            return;
        }
        if (enc.new_password !== enc.confirm_password) {
            if (typeof this.showToast === "function") this.showToast("Passwords do not match.", "error");
            return;
        }

        enc.loading = true;
        try {
            const res = await window.API.post("/api/config/encryption/set", {
                password: enc.new_password,
                old_password: enc.old_password || ""
            });
            if (res && res.status === "success") {
                if (typeof this.showToast === "function") this.showToast("Configuration password set successfully.", "success");
                enc.new_password = "";
                enc.old_password = "";
                enc.confirm_password = "";
                await this.fetchConfigEncryptionStatus();
            } else {
                throw new Error(res.error || "Failed to set password");
            }
        } catch (err) {
            if (typeof this.showToast === "function") this.showToast(`Error: ${err.message}`, "error");
        } finally {
            enc.loading = false;
        }
    },

    async removeConfigPassword() {
        const enc = this.globalSettingsModal.encryption;
        if (!enc.old_password || !enc.old_password.trim()) {
            if (typeof this.showToast === "function") {
                this.showToast("Current passphrase is required to remove encryption.", "warning");
            }
            return;
        }
        enc.loading = true;
        try {
            const res = await window.API.post("/api/config/encryption/remove", {
                password: enc.old_password || ""
            });
            if (res && res.status === "success") {
                if (typeof this.showToast === "function") this.showToast("Configuration password removed.", "success");
                enc.new_password = "";
                enc.old_password = "";
                enc.confirm_password = "";
                await this.fetchConfigEncryptionStatus();
            } else {
                throw new Error(res.error || "Failed to remove password");
            }
        } catch (err) {
            const isWarning = err.status === "warning" || (err.message && err.message.toLowerCase().includes("current passphrase is required"));
            if (typeof this.showToast === "function") {
                this.showToast(isWarning ? err.message : `Error: ${err.message}`, isWarning ? "warning" : "error");
            }
        } finally {
            enc.loading = false;
        }
    },

    closeGlobalSettingsModal() {
        this.globalSettingsModal.show = false;
        this.globalSettingsModal.error = null;
    },

    async saveGlobalSettings() {
        this.globalSettingsModal.saving = true;
        this.globalSettingsModal.error = null;

        try {
            const payload = {
                global_defaults: this.globalSettingsModal.config.global_defaults,
                daemon: this.globalSettingsModal.config.daemon,
                notifications: this.globalSettingsModal.config.notifications
            };

            const data = await window.API.put("/api/config", payload);
            if (data && data.status === "success") {
                if (data.global_defaults && this.profilesData) {
                    this.profilesData.globalDefaults = { ...data.global_defaults };
                }
                if (typeof this.showToast === "function") {
                    this.showToast("Global fleet settings updated successfully.", "success");
                }
                this.closeGlobalSettingsModal();
                if (typeof this.poll === "function") {
                    this.poll();
                }
            } else {
                throw new Error(data.error || "Update rejected by server");
            }
        } catch (err) {
            this.globalSettingsModal.error = err.message;
        } finally {
            this.globalSettingsModal.saving = false;
        }
    }
};
