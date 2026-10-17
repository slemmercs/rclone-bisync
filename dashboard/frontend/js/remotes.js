/**
 * Storage Remotes & Quota Module for Rclone Bisync Dashboard
 * Handles remote listing, storage quota percentages, and provider styling.
 */

window.RemotesModule = {
    async fetchRemotes(showLoading = true) {
            if (showLoading) this.remotesData.loading = true;
            this.remotesData.error = null;
            try {
                const res = await fetch("/api/remotes");
                if (!res.ok) throw new Error("HTTP " + res.status);
                const data = await res.json();
                this.remotesData.remotes = data.remotes || [];
                this.remotesData.summary = data.summary || {};
                this.remotesData.count = data.count || 0;
                this.remotesData.active_remote = data.active_remote || null;
                this.remotesData.last_checked = new Date().toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit", hour12: false });
            } catch (err) {
                this.remotesData.error = err.message;
            } finally {
                if (showLoading) this.remotesData.loading = false;
            }
        },,

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
};
