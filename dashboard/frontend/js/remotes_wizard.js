/**
 * Cloud Storage Remote Creation Wizard & Sync Profile Bridge Module
 * Multi-step guided provisioning for cloud remotes with category filtering,
 * OAuth detection, and automatic sync profile generation.
 */

window.RemotesWizardModule = {
    createRemoteModal: {
        show: false,
        step: 1, // 1: provider select, 2: config & profile, 3: review & finish
        loading: false,
        saving: false,
        error: null,
        search: "",
        category: "popular", // 'popular' | 'all' | 'cloud' | 'object' | 'protocol' | 'virtual'
        providers: [],
        selectedProvider: null,
        authMode: "oauth", // 'oauth' | 'custom'
        remoteName: "",
        nameError: "",
        parameters: {},
        createProfile: true,
        profile: {
            id: "",
            name: "",
            local_path: "",
            remote_path: ""
        },
        result: null,
        oauthPending: false,
        authUrl: ""
    },

    async openCreateRemoteWizard(defaultProvider = null) {
        this.createRemoteModal.show = true;
        this.createRemoteModal.step = 1;
        this.createRemoteModal.loading = false;
        this.createRemoteModal.saving = false;
        this.createRemoteModal.error = null;
        this.createRemoteModal.search = "";
        this.createRemoteModal.category = "popular";
        this.createRemoteModal.authMode = "oauth";
        this.createRemoteModal.remoteName = "";
        this.createRemoteModal.nameError = "";
        this.createRemoteModal.parameters = {};
        this.createRemoteModal.createProfile = true;
        this.createRemoteModal.profile = { id: "", name: "", local_path: "", remote_path: "" };
        this.createRemoteModal.result = null;
        this.createRemoteModal.oauthPending = false;
        this.createRemoteModal.authUrl = "";

        await this.fetchWizardProviders();
        if (defaultProvider && this.createRemoteModal.providers.length > 0) {
            const found = this.createRemoteModal.providers.find(p => p.name === defaultProvider || p.prefix === defaultProvider);
            if (found) this.selectWizardProvider(found);
        }
    },

    closeCreateRemoteWizard() {
        if (this.createRemoteModal.saving) {
            this.cancelProvisioning();
        }
        this.createRemoteModal.show = false;
    },

    async cancelProvisioning() {
        if (this._provisionAbortController) {
            this._provisionAbortController.abort();
            this._provisionAbortController = null;
        }
        this.createRemoteModal.saving = false;
        try {
            if (window.API && typeof window.API.post === "function") {
                await window.API.post("/api/remotes/auth/oauth-stop");
            }
        } catch (_) {}
        if (typeof this.showToast === "function") {
            this.showToast("Remote provisioning cancelled.", "info");
        }
    },

    isWizardSaving() {
        return Boolean(this.createRemoteModal && this.createRemoteModal.saving);
    },

    isWizardNextDisabled() {
        return Boolean(!this.createRemoteModal || this.createRemoteModal.nameError || !this.createRemoteModal.remoteName);
    },

    isWizardStep2Disabled() {
        return Boolean(!this.createRemoteModal || !this.createRemoteModal.selectedProvider);
    },

    isWizardStep3Disabled() {
        return Boolean(this.isWizardStep2Disabled() || this.isWizardNextDisabled());
    },

    async fetchWizardProviders(force = false) {
        if (this.createRemoteModal.providers.length > 0 && !force) return;
        this.createRemoteModal.loading = true;
        try {
            const data = await window.API.get("/api/remotes/providers");
            this.createRemoteModal.providers = data.providers || [];
        } catch (err) {
            this.createRemoteModal.error = "Failed to load providers: " + err.message;
        } finally {
            this.createRemoteModal.loading = false;
        }
    },

    get tier1Providers() {
        const tier1Prefixes = ["drive", "onedrive", "s3", "dropbox", "b2"];
        const list = this.createRemoteModal.providers || [];
        const result = [];
        for (const pref of tier1Prefixes) {
            const found = list.find(p => p.prefix === pref);
            if (found) {
                result.push(found);
            }
        }
        return result;
    },

    isOAuthProvider(p) {
        if (!p) return false;
        if (p.supports_oauth) return true;
        const oauthTypes = ["drive", "onedrive", "dropbox", "box", "pcloud", "hubic", "yandex", "mailru", "google photos", "zoho"];
        const name = (p.name || "").toLowerCase();
        const prefix = (p.prefix || "").toLowerCase();
        return oauthTypes.includes(name) || oauthTypes.includes(prefix);
    },

    setAuthMode(mode) {
        this.createRemoteModal.authMode = mode;
        if (mode === "oauth" && this.createRemoteModal.parameters) {
            delete this.createRemoteModal.parameters.client_id;
            delete this.createRemoteModal.parameters.client_secret;
        }
    },

    goToWizardStep(targetStep) {
        if (targetStep === 1) {
            this.createRemoteModal.step = 1;
            return;
        }
        if (targetStep === 2) {
            if (this.createRemoteModal.selectedProvider) {
                this.createRemoteModal.step = 2;
            }
            return;
        }
        if (targetStep === 3) {
            if (this.createRemoteModal.selectedProvider && !this.isWizardNextDisabled()) {
                this.nextWizardStep();
            }
        }
    },

    getFilteredWizardProviders() {
        const cat = this.createRemoteModal.category;
        const q = (this.createRemoteModal.search || "").trim().toLowerCase();
        let list = this.createRemoteModal.providers || [];
        if (cat === "popular") {
            list = list.filter(p => p.popular);
        } else if (cat && cat !== "all") {
            list = list.filter(p => p.category === cat);
        }
        if (q) {
            list = list.filter(p => 
                (p.name || "").toLowerCase().includes(q) ||
                (p.description || "").toLowerCase().includes(q) ||
                (p.prefix || "").toLowerCase().includes(q)
            );
        }
        return list;
    },

    selectWizardProvider(provider) {
        if (!provider) return;
        this.createRemoteModal.selectedProvider = provider;
        this.createRemoteModal.authMode = "oauth";
        this.createRemoteModal.parameters = {};
        if (!this.createRemoteModal.remoteName) {
            const baseName = provider.prefix || provider.name || "remote";
            this.createRemoteModal.remoteName = this.generateUniqueRemoteName(baseName);
        }
        this.onWizardRemoteNameChange();
        this.createRemoteModal.step = 2;
    },

    generateUniqueRemoteName(prefix) {
        const existing = new Set((this.remotesData?.remotes || []).map(r => (r.name || "").replace(/:$/, "").toLowerCase()));
        let candidate = prefix;
        let counter = 1;
        while (existing.has(candidate.toLowerCase())) {
            counter++;
            candidate = `${prefix}_${counter}`;
        }
        return candidate;
    },

    onWizardRemoteNameChange() {
        const raw = this.createRemoteModal.remoteName || "";
        const clean = raw.trim().replace(/:+$/, "");
        if (!clean) {
            this.createRemoteModal.nameError = "Remote name is required";
            return;
        }
        if (!/^[a-zA-Z0-9_\-\.]{1,64}$/.test(clean)) {
            this.createRemoteModal.nameError = "Only alphanumeric, underscores, hyphens, and dots allowed (max 64 chars)";
            return;
        }
        const existing = (this.remotesData?.remotes || []).some(r => (r.name || "").replace(/:$/, "").toLowerCase() === clean.toLowerCase());
        if (existing) {
            this.createRemoteModal.nameError = `A remote named "${clean}" already exists`;
            return;
        }
        this.createRemoteModal.nameError = "";

        const baseId = clean.toLowerCase().replace(/[^a-z0-9_-]/g, "-");
        this.createRemoteModal.profile.id = `${baseId}-sync`;
        this.createRemoteModal.profile.name = `${clean} Sync`;
        this.createRemoteModal.profile.local_path = `~/Sync/${clean}`;
    },

    nextWizardStep() {
        if (this.createRemoteModal.step === 2) {
            this.onWizardRemoteNameChange();
            if (this.createRemoteModal.nameError) return;
            this.createRemoteModal.step = 3;
        }
    },

    prevWizardStep() {
        if (this.createRemoteModal.step > 1) {
            this.createRemoteModal.step--;
        }
    },

    async submitCreateRemote() {
        this.createRemoteModal.saving = true;
        this.createRemoteModal.error = null;
        this._provisionAbortController = new AbortController();
        try {
            const name = this.createRemoteModal.remoteName.trim().replace(/:+$/, "");
            const payload = {
                name: name,
                type: this.createRemoteModal.selectedProvider.prefix || this.createRemoteModal.selectedProvider.name,
                parameters: this.createRemoteModal.parameters || {}
            };

            if (this.createRemoteModal.createProfile) {
                const p = this.createRemoteModal.profile;
                payload.create_profile = true;
                payload.profile_name = (p.name || "").trim() || `${name} Sync`;
                payload.local_path = (p.local_path || "").trim() || `~/Sync/${name}`;
                payload.remote_subpath = (p.remote_path || "").trim();
            }

            const res = await window.API.post("/api/remotes/create", payload, {
                signal: this._provisionAbortController.signal
            });
            if (res.status === "error") {
                throw new Error(res.error || "Failed to create remote");
            }

            this.createRemoteModal.result = res;
            this.createRemoteModal.oauthPending = Boolean(res.oauth_pending || res.needs_auth);
            this.createRemoteModal.authUrl = res.auth_url || "";

            this.showToast(`Remote "${name}" created successfully!`, "success");
            await this.fetchRemotes(false);
            if (typeof this.fetchProfiles === "function") await this.fetchProfiles();

            if (this.createRemoteModal.oauthPending && typeof this.reconnectRemote === "function") {
                this.reconnectRemote(name);
            }
        } catch (err) {
            if (err.name === "AbortError" || (this._provisionAbortController && this._provisionAbortController.signal.aborted)) {
                return;
            }
            this.createRemoteModal.error = err.message;
        } finally {
            this._provisionAbortController = null;
            this.createRemoteModal.saving = false;
        }
    },

    finishWizardAndExplore() {
        const rName = this.createRemoteModal.remoteName;
        this.closeCreateRemoteWizard();
        if (rName && typeof this.openRemoteInExplorer === "function") {
            this.openRemoteInExplorer(rName);
        }
    },

    finishWizardAndProfile() {
        const profId = this.createRemoteModal.profile?.id;
        this.closeCreateRemoteWizard();
        if (profId && typeof this.selectProfile === "function") {
            this.selectProfile(profId);
            this.switchTab("fleet");
        }
    },

    async confirmDeleteRemote(remoteName) {
        if (!remoteName) return;
        const name = remoteName.replace(/:$/, "");
        if (typeof this.askConfirmation !== "function") return;
        this.askConfirmation({
            title: `Delete Remote "${name}"?`,
            icon: "trash",
            message: `Are you sure you want to delete remote "${name}"?\n\nThis will remove the remote definition from Rclone configuration.\nAny profiles bound to this remote will be prevented from syncing.`,
            confirmText: "Delete Remote",
            confirmClass: "bg-rose-600 hover:bg-rose-500 text-white",
            onConfirm: async () => {
                try {
                    const res = await window.API.delete(`/api/remotes/${encodeURIComponent(name)}`);
                    if (res.status === "error") {
                        this.showToast(`Failed to delete remote: ${res.error || res.message}`, "error");
                        return;
                    }
                    this.showToast(`Remote "${name}" deleted successfully.`, "success");
                    await this.fetchRemotes(true);
                } catch (err) {
                    this.showToast(`Delete error: ${err.message}`, "error");
                }
            }
        });
    }
};

