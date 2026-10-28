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
                max_delete: 50,
                conflict_resolve: "newer",
                compare_mode: "modtime",
                canary_validation: true,
                remove_empty_dirs: true,
                create_empty_dirs: true
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
        }
    },

    isGlobalSettingsSaving() {
        return Boolean(this.globalSettingsModal?.saving);
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
