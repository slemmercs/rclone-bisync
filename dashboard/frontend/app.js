/**
 * Rclone Bisync Dashboard — Alpine.js Declarative Application
 * Zero build step, offline-ready, unified JSON log viewer.
 */

// Helper to escape HTML characters for security
function escapeHtml(str) {
    if (str === null || str === undefined) return "";
    return String(str)
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#39;");
}

// Module-level unproxied Chart instance to prevent Alpine Proxy recursion
let performanceChart = null;

// Rich styling helper for config parameters (restores original parameter cards design)
function formatConfigValue(key, value) {
    if (Array.isArray(value)) {
        let argsHtml = '<div class="space-y-1 mt-1.5">';
        for (let i = 0; i < value.length; i++) {
            let currentItem = String(value[i]);
            if (currentItem.startsWith('-') && i + 1 < value.length && !String(value[i + 1]).startsWith('-')) {
                let nextValue = String(value[i + 1]);
                argsHtml += `
                    <div class="bg-slate-950 px-2 py-0.5 rounded text-[10px] text-slate-300 font-mono border border-slate-800/40 flex justify-between gap-x-2 truncate" title="${escapeHtml(currentItem)} ${escapeHtml(nextValue)}">
                        <span class="text-slate-400 truncate">${escapeHtml(currentItem)}</span>
                        <span class="text-amber-400 font-medium shrink-0">${escapeHtml(nextValue)}</span>
                    </div>`;
                i++;
            } else {
                let isFlag = currentItem.startsWith('-');
                let flagColor = isFlag ? 'text-slate-400' : 'text-amber-400 font-medium';
                argsHtml += `
                    <div class="bg-slate-950 px-2 py-0.5 rounded text-[10px] ${flagColor} font-mono border border-slate-800/40 truncate text-left" title="${escapeHtml(currentItem)}">
                        ${escapeHtml(currentItem)}
                    </div>`;
            }
        }
        argsHtml += '</div>';
        return argsHtml;
    } else if (typeof value === 'object' && value !== null) {
        return `<div class="space-y-1 mt-1.5 font-mono text-[10px]">` +
            Object.entries(value).map(([subKey, subVal]) => {
                const stateBadge = subVal
                    ? `<span class="text-emerald-400 font-semibold text-[9px] bg-emerald-950/40 px-1 py-0.2 rounded border border-emerald-900/30">True</span>`
                    : `<span class="text-rose-400 font-semibold text-[9px] bg-rose-950/40 px-1 py-0.2 rounded border border-rose-900/30">False</span>`;
                return `<div class="flex justify-between items-center border-b border-slate-900 pb-1 last:border-0"><span class="text-slate-500 truncate mr-2" title="${escapeHtml(subKey)}">${escapeHtml(subKey)}:</span>${stateBadge}</div>`;
            }).join('') + `</div>`;
    } else if (typeof value === 'boolean') {
        return value
            ? `<span class="text-emerald-400 font-semibold bg-emerald-950/30 px-1.5 py-0.5 rounded border border-emerald-900/40 inline-block mt-1">True</span>`
            : `<span class="text-rose-400 font-semibold bg-rose-950/30 px-1.5 py-0.5 rounded border border-rose-900/40 inline-block mt-1">False</span>`;
    } else {
        const strVal = String(value);
        return `<span class="text-slate-300 block mt-1 select-all truncate" title="${escapeHtml(strVal)}">${escapeHtml(strVal)}</span>`;
    }
}

// Unified parser for Rclone and Bisync JSON log lines with legacy fallback
function parseJsonLogLine(rawLine) {
    if (typeof rawLine !== "string") return null;
    rawLine = rawLine.trim();
    if (!rawLine) return null;

    try {
        const parsed = JSON.parse(rawLine);
        const level = (parsed.level || "info").toLowerCase();
        let levelClass = "text-slate-400";
        if (level === "error") levelClass = "text-rose-400 font-bold";
        else if (level === "warning" || level === "warn") levelClass = "text-amber-400 font-medium";
        else if (level === "notice") levelClass = "text-sky-400";
        else if (level === "info") levelClass = "text-emerald-400";

        let timeStr = "N/A";
        if (parsed.time) {
            const dateObj = new Date(parsed.time);
            if (!isNaN(dateObj)) {
                timeStr = dateObj.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit", hour12: false });
            }
        }

        return {
            time: timeStr,
            level: level.toUpperCase().padEnd(7),
            levelClass: levelClass,
            msg: (parsed.msg || "").trim(),
            object: parsed.object || null,
            origin: parsed.origin || null,
            run_id: parsed.run_id || null,
            isJson: true
        };
    } catch (e) {
        // Regex fallback for legacy format: "YYYY-MM-DD HH:MM:SS [LEVEL] message"
        const match = rawLine.match(/^(\d{4}-\d{2}-\d{2}\s+(\d{2}:\d{2}:\d{2}))\s+\[([A-Z]+)\]\s+(.*)$/);
        if (match) {
            const timeStr = match[2];
            const level = match[3].toLowerCase();
            let levelClass = "text-slate-400";
            if (level === "error") levelClass = "text-rose-400 font-bold";
            else if (level === "warning" || level === "warn") levelClass = "text-amber-400 font-medium";
            else if (level === "notice") levelClass = "text-sky-400";
            else if (level === "info") levelClass = "text-emerald-400";

            return {
                time: timeStr,
                level: level.toUpperCase().padEnd(7),
                levelClass: levelClass,
                msg: match[4].trim(),
                object: null,
                origin: null,
                run_id: null,
                isJson: false
            };
        }

        let colorClass = "text-slate-400";
        if (rawLine.includes("[ERROR]")) colorClass = "text-rose-400 font-semibold";
        else if (rawLine.includes("[WARNING]")) colorClass = "text-amber-400";
        else if (rawLine.includes("SUCCESS")) colorClass = "text-emerald-400 font-medium";

        return {
            time: "--:--:--",
            level: "RAW    ",
            levelClass: colorClass,
            msg: rawLine,
            object: null,
            origin: null,
            run_id: null,
            isJson: false
        };
    }
}

// Helper to format bytes into human-readable strings
function formatBytes(bytes, decimals = 1) {
    if (bytes === null || bytes === undefined || isNaN(bytes) || bytes < 0) return "--";
    if (bytes === 0) return "0 B";
    const k = 1024;
    const dm = decimals < 0 ? 0 : decimals;
    const sizes = ["B", "KB", "MB", "GB", "TB", "PB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    const idx = Math.min(i, sizes.length - 1);
    return parseFloat((bytes / Math.pow(k, idx)).toFixed(dm)) + " " + sizes[idx];
}

document.addEventListener("alpine:init", () => {
    Alpine.data("dashboard", () => ({
        currentTab: "telemetry", // 'telemetry' | 'remotes' | 'explorer'
        remotesData: {
            remotes: [],
            summary: {},
            count: 0,
            loading: false,
            error: null,
            last_checked: null
        },
        telemetry: {
            daemon_health: { status: "evaluating", response_time_ms: 0 },
            notification_state: {},
            metrics: [],
            bisync_logs: [],
            daemon_logs: [],
            config: {},
            is_syncing: false,
            transfer_stats: null
        },
        currentLogLimit: 40,
        maxLogLimit: 300,
        actionLoading: null,
        configExpanded: false,

        showDeletionsModal: false,
        deletionsData: { total_count: 0, source: "", target_action: "", files: [] },
        deletionsSearch: "",

        get parsedBisyncLogs() {
            return (this.telemetry.bisync_logs || []).map(parseJsonLogLine).filter(Boolean);
        },

        get parsedDaemonLogs() {
            return (this.telemetry.daemon_logs || []).map(parseJsonLogLine).filter(Boolean);
        },

        get filteredDeletions() {
            const files = this.deletionsData.files || [];
            const q = (this.deletionsSearch || "").toLowerCase().trim();
            if (!q) return files;
            return files.filter(f => f.toLowerCase().includes(q));
        },

        get isTooManyDeletesActive() {
            const notif = this.telemetry.notification_state || {};
            const lastErr = notif.last_error_type;
            const lastErrTs = notif.last_error_timestamp || 0;
            const lastSuccessTs = notif.last_success_timestamp || 0;
            const metrics = this.telemetry.metrics || [];
            const lastMetric = metrics.length > 0 ? metrics[metrics.length - 1] : null;

            return (
                (lastMetric && !lastMetric.success && lastMetric.error_type === "TOO_MANY_DELETES") ||
                (lastErr === "TOO_MANY_DELETES" && lastErrTs > lastSuccessTs)
            );
        },

        get daemonStatusText() {
            const status = this.telemetry.daemon_health?.status;
            return status === "healthy" ? "ONLINE" : "OFFLINE";
        },

        get daemonStatusClass() {
            const status = this.telemetry.daemon_health?.status;
            return status === "healthy" ? "text-emerald-400 tracking-wide" : "text-rose-500 tracking-wide";
        },

        get daemonLatencyText() {
            const lat = this.telemetry.daemon_health?.response_time_ms;
            return typeof lat === "number" ? lat.toFixed(2) : "--";
        },

        get executionStatusText() {
            if (this.telemetry.is_syncing) return "SYNC IN PROGRESS";
            const notif = this.telemetry.notification_state || {};
            const successTs = notif.last_success_timestamp || 0;
            const errorTs = notif.last_error_timestamp || 0;
            if (errorTs > successTs && notif.last_error_type) return `FAULT (${notif.last_error_type})`;
            if (successTs > 0) return "OPERATIONAL";
            return "UNKNOWN";
        },

        get executionStatusClass() {
            if (this.telemetry.is_syncing) return "text-amber-400 font-bold tracking-wide";
            const notif = this.telemetry.notification_state || {};
            const successTs = notif.last_success_timestamp || 0;
            const errorTs = notif.last_error_timestamp || 0;
            if (errorTs > successTs && notif.last_error_type) return "text-rose-400 text-lg";
            if (successTs > 0) return "text-emerald-400";
            return "text-slate-400";
        },

        get lastUpdatedText() {
            const notif = this.telemetry.notification_state || {};
            const ts = Math.max(notif.last_success_timestamp || 0, notif.last_error_timestamp || 0);
            if (!ts) return "--";
            return new Date(ts * 1000).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit", hour12: false });
        },

        get transferStats() {
            const stats = this.telemetry.transfer_stats || {};
            const currentFile = (stats.transferring && stats.transferring.length > 0)
                ? stats.transferring[0].name
                : (this.telemetry.is_syncing ? "Scanning differences & checking access..." : "Idle");
            let etaStr = "--";
            if (typeof stats.eta === "number") {
                const mins = Math.floor(stats.eta / 60);
                const secs = stats.eta % 60;
                etaStr = `${mins}m ${secs}s`;
            }
            return {
                speed_str: stats.speed_str || "0 B/s",
                eta_str: etaStr,
                percentage: typeof stats.percentage === "number" ? stats.percentage : 0,
                current_file: currentFile
            };
        },

        get filteredConfigEntries() {
            const cfg = this.telemetry.config || {};
            const exclude = new Set(["LOCAL_PATH", "REMOTE_PATH", "BASE_DIR"]);
            return Object.entries(cfg).filter(([k]) => !exclude.has(k));
        },

        formatConfig(key, val) {
            return formatConfigValue(key, val);
        },

        async initDashboard() {
            this.initChart();
            await this.poll();
            this.fetchRemotes(false);
            setInterval(() => this.poll(), 3000);
            this.setupLogScroll("log-bisync");
            this.setupLogScroll("log-daemon");
        },

        initChart() {
            const canvas = document.getElementById("livePerformanceChart");
            if (!canvas) return;
            const ctx = canvas.getContext("2d");
            performanceChart = new Chart(ctx, {
                type: "line",
                data: {
                    labels: [],
                    datasets: [{
                        label: "Duration (s)",
                        data: [],
                        borderColor: "#6366f1",
                        backgroundColor: "rgba(99, 102, 241, 0.03)",
                        borderWidth: 2,
                        pointBackgroundColor: [],
                        pointBorderColor: [],
                        pointRadius: 4,
                        tension: 0.15,
                        fill: true
                    }]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    plugins: { legend: { display: false } },
                    scales: {
                        y: { grid: { color: "#1e293b" }, ticks: { color: "#475569", font: { family: "monospace" } } },
                        x: { grid: { display: false }, ticks: { color: "#475569", font: { family: "monospace" } } }
                    }
                }
            });
        },

        updateChartData(metrics) {
            if (!performanceChart || !metrics || metrics.length === 0) return;
            performanceChart.data.labels = metrics.map(m => new Date(m.datetime).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }));
            performanceChart.data.datasets[0].data = metrics.map(m => m.duration_seconds);
            const pointColors = metrics.map(m => m.success ? "#10b981" : "#f43f5e");
            performanceChart.data.datasets[0].pointBackgroundColor = pointColors;
            performanceChart.data.datasets[0].pointBorderColor = pointColors;
            performanceChart.update("none");
        },

        async poll() {
            try {
                const res = await fetch("/api/telemetry?limit=" + this.currentLogLimit);
                if (!res.ok) throw new Error("HTTP error: " + res.status);
                const data = await res.json();
                this.telemetry = data;
                this.updateChartData(data.metrics || []);
            } catch (err) {
                console.error("Telemetry poll failed:", err);
            }
        },

        async triggerAction(actionStr) {
            this.actionLoading = actionStr;
            try {
                const res = await fetch("/api/action", {
                    method: "POST",
                    headers: {
                        "Content-Type": "application/json",
                        "X-Dashboard-Request": "1"
                    },
                    body: JSON.stringify({ action: actionStr })
                });
                const data = await res.json();
                if (!res.ok || data.status === "error") {
                    alert("Action Error: " + (data.msg || "Request failed"));
                }
                await this.poll();
            } catch (err) {
                alert("Network error: " + err.message);
            } finally {
                setTimeout(() => { this.actionLoading = null; }, 1200);
            }
        },

        async openDeletionsModal() {
            this.deletionsSearch = "";
            this.showDeletionsModal = true;
            this.deletionsData = { total_count: 0, source: "Scanning...", target_action: "Reading daemon logs...", files: [] };
            try {
                const res = await fetch("/api/deletions/pending");
                const data = await res.json();
                this.deletionsData = data;
            } catch (err) {
                this.deletionsData = { total_count: 0, source: "Error", target_action: err.message, files: [] };
            }
        },

        confirmResync() {
            if (confirm("⚠ FORCE RESYNC ⚠\n\nAre you sure you want to rebuild baseline sync states?\nThis will take significantly longer.")) {
                this.triggerAction("force_resync");
            }
        },

        setupLogScroll(containerId) {
            const container = document.getElementById(containerId);
            if (!container) return;
            container.addEventListener("scroll", () => {
                if (container.scrollTop <= 40 && this.currentLogLimit < this.maxLogLimit) {
                    const oldHeight = container.scrollHeight;
                    this.currentLogLimit += 60;
                    this.poll().then(() => {
                        this.$nextTick(() => {
                            container.scrollTop = container.scrollHeight - oldHeight;
                        });
                    });
                }
            });
        },

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
        },

        switchTab(tab) {
            this.currentTab = tab;
            if (tab === "remotes") {
                this.fetchRemotes(true);
            } else if (tab === "explorer") {
                if (typeof this.initExplorer === "function") {
                    this.initExplorer();
                }
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

    }));
});
