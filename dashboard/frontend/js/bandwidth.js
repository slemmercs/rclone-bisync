/**
 * Bandwidth & Core Stats Module for Bisync Hub Dashboard
 * Provides dynamic bandwidth throttling, preset toggles, and live transfer monitoring.
 */

window.BandwidthModule = {
    bandwidth: {
        speed: 0,
        bytes: 0,
        transfers: 0,
        checks: 0,
        errors: 0,
        fatalErrors: false,
        transferring: [],
        currentLimit: "off",
        bytesPerSecond: 0,
        rateFormatted: "off",
        popoverOpen: false,
        customInput: "",
        loading: false,
        updating: false,
        presets: ["off", "500k", "2M", "10M", "50M"]
    },

    isBandwidthUpdating() {
        return Boolean(this.bandwidth?.updating);
    },

    isBandwidthPopoverOpen() {
        return Boolean(this.bandwidth?.popoverOpen);
    },

    toggleBandwidthPopover() {
        this.bandwidth.popoverOpen = !this.bandwidth.popoverOpen;
        if (this.bandwidth.popoverOpen) {
            this.fetchBandwidthLimit();
            this.fetchCoreStats();
        }
    },

    closeBandwidthPopover() {
        this.bandwidth.popoverOpen = false;
    },

    async fetchBandwidthLimit() {
        try {
            const data = await window.API.get("/api/core/bwlimit");
            if (data) {
                this.bandwidth.bytesPerSecond = data.bytes_per_second || 0;
                this.bandwidth.rateFormatted = data.rate_formatted || (data.bytes_per_second > 0 ? this.formatSpeed(data.bytes_per_second) : "off");
                this.bandwidth.currentLimit = data.rate_formatted || "off";
            }
        } catch (err) {
            console.warn("Failed to fetch bandwidth limit:", err);
        }
    },

    async fetchCoreStats() {
        try {
            const data = await window.API.get("/api/core/stats");
            if (data) {
                this.bandwidth.speed = data.speed || 0;
                this.bandwidth.bytes = data.bytes || 0;
                this.bandwidth.transfers = data.transfers || 0;
                this.bandwidth.checks = data.checks || 0;
                this.bandwidth.errors = data.errors || 0;
                this.bandwidth.fatalErrors = Boolean(data.fatal_errors);
                this.bandwidth.transferring = Array.isArray(data.transferring) ? data.transferring : [];
            }
        } catch (_) {
            // Stats may be unavailable if daemon is restarting
        }
    },

    async setBandwidthLimit(rate) {
        if (!rate) return;
        this.bandwidth.updating = true;
        try {
            const res = await window.API.post("/api/core/bwlimit", { rate });
            if (res && (res.status === "ok" || res.status === "success" || res.rate_formatted)) {
                this.bandwidth.rateFormatted = res.rate_formatted || rate;
                this.bandwidth.currentLimit = res.rate_formatted || rate;
                this.bandwidth.bytesPerSecond = res.bytes_per_second || 0;
                if (typeof this.showToast === "function") {
                    const label = rate === "off" ? "Bandwidth throttle disabled (unlimited speed)" : `Bandwidth limit set to ${this.bandwidth.rateFormatted}`;
                    this.showToast(label, "success");
                }
            } else {
                throw new Error(res.error || "Failed to set bandwidth limit");
            }
        } catch (err) {
            if (typeof this.showToast === "function") {
                this.showToast("Failed to set bandwidth limit: " + err.message, "error");
            }
        } finally {
            this.bandwidth.updating = false;
        }
    },

    async applyCustomBandwidthLimit() {
        const rate = (this.bandwidth.customInput || "").trim();
        if (!rate) {
            if (typeof this.showToast === "function") {
                this.showToast("Please enter a valid bandwidth rate (e.g. 10M, 500k, off)", "error");
            }
            return;
        }
        await this.setBandwidthLimit(rate);
        this.bandwidth.customInput = "";
    },

    formatSpeed(bps) {
        if (!bps || bps <= 0) return "0 B/s";
        const units = ["B/s", "KB/s", "MB/s", "GB/s"];
        let val = bps;
        let idx = 0;
        while (val >= 1024 && idx < units.length - 1) {
            val /= 1024;
            idx++;
        }
        return (idx === 0 ? val : val.toFixed(1)) + " " + units[idx];
    },

    formatBytes(bytes) {
        if (!bytes || bytes <= 0) return "0 B";
        const units = ["B", "KB", "MB", "GB", "TB"];
        let val = bytes;
        let idx = 0;
        while (val >= 1024 && idx < units.length - 1) {
            val /= 1024;
            idx++;
        }
        return (idx === 0 ? val : val.toFixed(1)) + " " + units[idx];
    }
};
