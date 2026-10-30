/**
 * Polling & Dev-Mode Network Control Module for Bisync Hub Dashboard
 * Manages periodic telemetry fetching, pause/resume state, dev-mode auto-silence,
 * and visibilitychange power management.
 */

window.PollingModule = {
    isPollingPaused: false,
    pollingTimer: null,
    pollingIntervalMs: 8000,
    idleIntervalMs: 8000,
    activeSyncIntervalMs: 2000,
    isPollingInFlight: false,
    wasSyncing: false,
    lastProfilesFetch: Date.now(),

    initPolling() {
        const params = new URLSearchParams(window.location.search);
        const devParam = params.get("dev");
        const pollParam = params.get("poll");
        const isDev = devParam === "true" || devParam === "1" || pollParam === "false" || pollParam === "0";
        const storedPause = localStorage.getItem("bisync_polling_paused");

        if (isDev) {
            this.isPollingPaused = true;
            console.info("[PollingModule] Dev mode detected via URL query: polling initialized in PAUSED mode.");
        } else if (storedPause === "true") {
            this.isPollingPaused = true;
            console.info("[PollingModule] Polling initialized in PAUSED mode via saved localStorage preference.");
        } else {
            this.isPollingPaused = false;
        }

        document.addEventListener("visibilitychange", () => {
            if (document.hidden) {
                this.stopPollingTimer();
            } else if (!this.isPollingPaused) {
                this.startPollingTimer();
                this.poll();
            }
        });

        window.addEventListener("keydown", (e) => {
            if (e.altKey && (e.key === "p" || e.key === "P")) {
                e.preventDefault();
                this.togglePolling();
            }
        });

        if (!this.isPollingPaused) {
            this.startPollingTimer();
        }
    },

    togglePolling() {
        this.isPollingPaused = !this.isPollingPaused;
        localStorage.setItem("bisync_polling_paused", String(this.isPollingPaused));

        if (this.isPollingPaused) {
            this.stopPollingTimer();
            if (typeof this.showToast === "function") {
                this.showToast("Telemetry polling paused (Alt+P to resume)", "info");
            }
        } else {
            this.startPollingTimer();
            this.poll();
            if (typeof this.showToast === "function") {
                this.showToast("Telemetry polling resumed", "info");
            }
        }
    },

    startPollingTimer() {
        this.stopPollingTimer();
        this.pollingTimer = setInterval(() => {
            if (!this.isPollingPaused && !document.hidden) {
                this.poll();
            }
        }, this.pollingIntervalMs);
    },

    stopPollingTimer() {
        if (this.pollingTimer) {
            clearInterval(this.pollingTimer);
            this.pollingTimer = null;
        }
    },

    async poll() {
        if (this.isPollingInFlight) return;
        this.isPollingInFlight = true;
        try {
            // Lightweight telemetry query with limit=0 (logs are loaded on-demand via /api/logs/bisync)
            const data = await window.API.get("/api/telemetry?limit=0");
            this.telemetry = data;

            if (typeof updatePerformanceChart === "function") {
                updatePerformanceChart(data.metrics || []);
            }

            // Only fetch review session if review is actually flagged or currently active in UI
            const hasReviewFlag = Boolean(data.has_review || data.review_pending);
            const currentlyActive = Boolean(this.reviewState && this.reviewState.active);
            if ((hasReviewFlag || currentlyActive) && typeof this.fetchReviewSession === "function") {
                this.fetchReviewSession(false);
            }

            // Adaptive polling rate: 2s when actively syncing, 8s when idle
            const isSyncing = Boolean(data.is_syncing);
            const targetInterval = isSyncing ? this.activeSyncIntervalMs : this.idleIntervalMs;
            if (this.pollingIntervalMs !== targetInterval) {
                this.pollingIntervalMs = targetInterval;
                if (!this.isPollingPaused) {
                    this.startPollingTimer();
                }
            }

            // Only refetch profiles when sync transitions from active to finished, or every 30s
            const syncJustFinished = this.wasSyncing && !isSyncing;
            const profilesStale = Date.now() - this.lastProfilesFetch > 30000;
            if ((syncJustFinished || profilesStale) && typeof this.fetchProfiles === "function") {
                this.lastProfilesFetch = Date.now();
                this.fetchProfiles();
            }

            this.wasSyncing = isSyncing;
        } catch (err) {
            console.error("Telemetry poll failed:", err);
        } finally {
            this.isPollingInFlight = false;
        }
    }
};
