/**
 * Rclone Bisync Dashboard — Alpine.js Declarative Application
 * Zero build step, offline-ready, unified JSON log viewer.
 */

// Helper to escape HTML characters for security
function escapeHtml(str) {
    if (str === null || str === undefined) return "";
    return String(str).replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;").replace(/'/g, "&#39;");
}

// Resolves visual badge metadata for execution caller origins (CLI, CRON, UI, MCP)
function getOriginBadge(origin) {
    if (!origin) return null;
    const o = String(origin).trim().toLowerCase();
    if (o === "cli") return { label: "CLI", class: "bg-sky-950/80 text-sky-400 border-sky-800/50" };
    if (o === "cron" || o === "timer" || o === "scheduler") return { label: "CRON", class: "bg-amber-950/80 text-amber-400 border-amber-800/50" };
    if (o === "ui" || o === "dashboard" || o === "web") return { label: "UI", class: "bg-emerald-950/80 text-emerald-400 border-emerald-800/50" };
    if (o === "mcp") return { label: "MCP", class: "bg-violet-950/80 text-violet-400 border-violet-800/50" };
    return { label: o.toUpperCase(), class: "bg-slate-900 text-slate-300 border-slate-800" };
}

// Contextual path tokenizer extracting file mutation paths from log messages
function extractPathFromLogMsg(msg) {
    if (!msg || typeof msg !== "string") return null;
    let s = msg.trim();
    if (!s) return null;

    // Fast-reject HTTP access, URLs, and safety abort summaries
    if (/^(GET|HEAD|POST|PUT|DELETE)\s+\S+\s+→/i.test(s) || /https?:\/\//i.test(s) || /safety threshold|too many deletes/i.test(s)) {
        return null;
    }

    // Strip leading timestamp and log level prefix
    s = s.replace(/^\d{4}[/-]\d{2}[/-]\d{2}[\sT]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?\s*/, "");
    s = s.replace(/^(?:INFO|NOTICE|ERROR|WARNING|WARN|DEBUG)\s*:\s*/i, "");

    let found = null;
    const matchers = [
        /Duplicate ignored:\s*(.+)/i,
        /(?:-\s+)?(?:Path[12]\s+)?(?:File\s+was\s+)?Delet(?:ed|ing)(?:\s*-\s*|\s*:\s*)(.+)/i,
        /Transferr(?:ing|ed)(?:\s+file)?:\s*(.+)/i,
        /Conflict:\s*(.+?)\s+differs between/i,
        /^([a-zA-Z0-9_.-]+(?:\{[^}]+\})?:[^\s:]+):\s*Conflict/i,
        /Conflict(?:\s*:\s*|\s+identified\s*(?:for\s*)?)(.+)/i,
        /Copied(?:\s*\([^)]+\))?(?:\s+to)?:\s*(.+)/i,
        /Updated:\s*(.+)/i
    ];

    for (const re of matchers) {
        const m = s.match(re);
        if (m && m[1]) {
            found = m[1].trim();
            break;
        }
    }

    if (!found) return null;

    // Strip surrounding quotes
    found = found.replace(/^["'`\s]+|["'`\s]+$/g, "");

    // Strip remote target prefixes (e.g. 'Path1 "gdrive{tCwNP}:/docs/report.pdf"' or 'remote:path')
    found = found.replace(/^(?:Path[12]\s*)?["']?[a-zA-Z0-9_.-]+(?:\{[^}]+\})?:[/\\]?/, "");
    found = found.replace(/^["'`\s]+|["'`\s]+$/g, "");

    // Normalize Windows backslashes and strip trailing punctuation/extra info
    found = found.replace(/\\/g, "/");
    found = found.replace(/[;,]$/, "").trim();
    if (!found || found === "." || found === "/" || found.startsWith("http:") || found.startsWith("https:")) return null;

    return found;
}

if (typeof window !== "undefined") {
    window.extractPathFromLogMsg = extractPathFromLogMsg;
}

// Unified parser for Rclone and Bisync JSON log lines with legacy fallback
function parseJsonLogLine(rawLine) {
    if (!rawLine) return null;
    let parsed = null;
    const rawStr = typeof rawLine === "string" ? rawLine.trim() : "";
    if (typeof rawLine === "object") {
        parsed = rawLine;
    } else if (rawStr) {
        try {
            parsed = JSON.parse(rawStr);
        } catch (_) {}
    } else {
        return null;
    }

    if (!parsed) {
        const match = rawStr.match(/^(\d{4}-\d{2}-\d{2}\s+(\d{2}:\d{2}:\d{2}))\s+\[([A-Z]+)\]\s+(.*)$/);
        if (match) {
            const timeStr = match[2];
            const level = match[3].toLowerCase();
            let levelClass = "text-slate-400";
            if (level === "error") levelClass = "text-rose-400 font-bold";
            else if (level === "warning" || level === "warn") levelClass = "text-amber-400 font-medium";
            else if (level === "notice") levelClass = "text-sky-400";
            else if (level === "info") levelClass = "text-emerald-400";

            const legacyMsg = match[4].trim();
            const isReadOnly = /^(GET|HEAD)\s+\S+\s+→\s+[123]\d{2}/.test(legacyMsg);

            return {
                time: timeStr,
                level: level.toUpperCase().padEnd(7),
                levelClass: levelClass,
                msg: legacyMsg,
                type: isReadOnly ? "http_access" : "sync",
                is_read_only_access: isReadOnly,
                object: extractPathFromLogMsg(legacyMsg) || null,
                origin: null,
                origin_badge: null,
                run_id: null,
                isJson: false
            };
        }

        let colorClass = "text-slate-400";
        if (rawStr.includes("[ERROR]")) colorClass = "text-rose-400 font-semibold";
        else if (rawStr.includes("[WARNING]")) colorClass = "text-amber-400";
        else if (rawStr.includes("SUCCESS")) colorClass = "text-emerald-400 font-medium";

        return {
            time: "--:--:--",
            level: "RAW    ",
            levelClass: colorClass,
            msg: rawStr,
            object: extractPathFromLogMsg(rawStr) || null,
            origin: null,
            origin_badge: null,
            run_id: null,
            isJson: false
        };
    }

    const rawLevel = String(parsed.level || "info").trim().toLowerCase();
    let levelClass = "text-slate-400";
    if (parsed.type === "audit") levelClass = "text-purple-400 font-bold";
    else if (rawLevel === "error") levelClass = "text-rose-400 font-bold";
    else if (rawLevel === "warning" || rawLevel === "warn") levelClass = "text-amber-400 font-medium";
    else if (rawLevel === "notice") levelClass = "text-sky-400";
    else if (rawLevel === "info") levelClass = "text-emerald-400";

    let timeStr = "N/A";
    const ts = parsed.time || parsed.timestamp || parsed.start_time;
    if (ts) {
        const dateObj = new Date(ts);
        if (!isNaN(dateObj)) {
            timeStr = dateObj.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit", hour12: false });
        }
    }

        let structuredDetails = null;
        let cleanMsg = (parsed.msg || "").trim();
        let isHttpAccess = false;
        let isReadOnlyAccess = false;

        // Format HTTP access log records cleanly
        if ((cleanMsg === "http_access" || parsed.msg === "http_access") && parsed.method && parsed.path) {
            isHttpAccess = true;
            const method = String(parsed.method).toUpperCase();
            const status = parsed.status || 200;
            if ((method === "GET" || method === "HEAD") && status < 400) {
                isReadOnlyAccess = true;
            }
            const dur = (parsed.duration_ms !== undefined && parsed.duration_ms !== null)
                ? ` (${Number(parsed.duration_ms).toFixed(1)}ms)`
                : "";
            cleanMsg = `${parsed.method} ${parsed.path} → ${status}${dur}`;
        }

        // Detect if msg contains embedded JSON (e.g. comparison settings or error payloads)
        if (cleanMsg.includes("{") && cleanMsg.includes("}")) {
            const jsonMatch = cleanMsg.match(/(\{[\s\S]*\})/);
            if (jsonMatch) {
                try {
                    const parsedObj = JSON.parse(jsonMatch[1]);
                    if (typeof parsedObj === "object" && parsedObj !== null) {
                        structuredDetails = parsedObj;
                        const headerPart = cleanMsg.substring(0, jsonMatch.index).trim();
                        if (headerPart) {
                            cleanMsg = headerPart.replace(/:$/, "");
                        }
                    }
                } catch (e) {
                    // Ignore parse error, keep cleanMsg
                }
            }
        }

        let profileId = parsed.profile_id || null;
        let profileName = parsed.profile_name || null;
        if (!profileId && cleanMsg.startsWith("[")) {
            const profMatch = cleanMsg.match(/^\[([^\]]+)\]/);
            if (profMatch && !["INFO", "WARN", "ERROR", "DEBUG", "NOTICE"].includes(profMatch[1].toUpperCase())) {
                profileName = profMatch[1];
            }
        }

        const details = parsed.details || structuredDetails;
        let runId = parsed.run_id || null;
        let jobId = parsed.job_id || null;
        let origin = parsed.origin || null;
        let durationSec = parsed.duration_sec !== undefined ? parsed.duration_sec : null;
        let filesSynced = parsed.files_synced !== undefined ? parsed.files_synced : null;
        let status = parsed.status || null;
        let errorType = parsed.error_type || null;

        if (details && typeof details === "object") {
            if (!runId && details.run_id) runId = details.run_id;
            if (!jobId && details.job_id) jobId = details.job_id;
            if (!origin && details.origin) origin = details.origin;
            if (durationSec === null && details.duration_seconds !== undefined) durationSec = details.duration_seconds;
            if (durationSec === null && details.duration_sec !== undefined) durationSec = details.duration_sec;
            if (filesSynced === null && details.files_synced !== undefined) filesSynced = details.files_synced;
            if (!status && details.status) status = details.status;
            if (!errorType && details.error_type) errorType = details.error_type;
        }

        return {
            time: timeStr,
            level: rawLevel.toUpperCase().padEnd(7),
            levelClass: levelClass,
            msg: cleanMsg,
            type: isHttpAccess ? "http_access" : (parsed.type || "sync"),
            is_http_access: isHttpAccess,
            is_read_only_access: isReadOnlyAccess,
            seq_id: parsed.seq_id || (details && details.seq_id) || null,
            profile_id: profileId,
            profile_name: profileName,
            action: parsed.action || (details && details.action) || null,
            actor: parsed.actor || (details && details.actor) || null,
            hash: parsed.hash || (details && details.hash) || null,
            prev_hash: parsed.prev_hash || (details && details.prev_hash) || null,
            object: parsed.object || (details && (details.item_path || (details.path && details.filename ? `${details.path}/${details.filename}`.replace(/^\/+/, '') : null) || details.path || details.object || details.file)) || extractPathFromLogMsg(cleanMsg) || null,
            objectType: parsed.objectType || (details && details.objectType) || null,
            source: parsed.source || (details && details.source) || null,
            details: details,
            origin: origin,
            origin_badge: getOriginBadge(origin),
            run_id: runId,
            job_id: jobId,
            status: status,
            error_type: errorType,
            duration_sec: durationSec,
            files_synced: filesSynced,
            start_time: parsed.start_time || (details && details.start_time) || null,
            end_time: parsed.end_time || (details && details.end_time) || null,
            isJson: true,
            raw: rawLine
        };
}

// Unwraps potential wrapper logs and parses each line into structured format
function unwrapAndParseLogLines(rawLines) {
    if (!Array.isArray(rawLines)) return [];
    const results = [];
    for (const line of rawLines) {
        if (!line) continue;
        if (typeof line === "string" && line.startsWith("{") && (line.includes(`"msg":"{\\"`) || line.includes(`"msg":"{\'`))) {
            try {
                const outer = JSON.parse(line);
                if (typeof outer.msg === "string" && (outer.msg.includes('{"') || outer.msg.includes("{'"))) {
                    const sublines = outer.msg.split("\n");
                    let unwrappedAny = false;
                    for (const sub of sublines) {
                        const subTrim = sub.trim();
                        if (subTrim.startsWith("{") && subTrim.endsWith("}")) {
                            const parsedSub = parseJsonLogLine(subTrim);
                            if (parsedSub) { results.push(parsedSub); unwrappedAny = true; }
                        }
                    }
                    if (unwrappedAny) continue;
                }
            } catch (e) { /* Fall through to regular line parsing */ }
        }
        const parsed = parseJsonLogLine(line);
        if (parsed) results.push(parsed);
    }
    return results;
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

function formatModTime(isoStr) {
    if (!isoStr) return "--";
    try {
        const d = new Date(isoStr);
        if (isNaN(d.getTime())) return "--";
        const datePart = d.toLocaleDateString([], { month: "short", day: "numeric", year: "2-digit" });
        const timePart = d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", hour12: false });
        return `${datePart} ${timePart}`;
    } catch (e) {
        return "--";
    }
}

// Canonical 7 Approved Status Vocabulary: Healthy, Warning, Critical, Running, Idle, Blocked, Review Required
const APPROVED_STATUS_LABELS = {
    HEALTHY: "Healthy",
    WARNING: "Warning",
    CRITICAL: "Critical",
    RUNNING: "Running",
    IDLE: "Idle",
    BLOCKED: "Blocked",
    REVIEW_REQUIRED: "Review Required"
};

function normalizeStatus(rawStatus) {
    if (!rawStatus) return APPROVED_STATUS_LABELS.IDLE;
    const s = String(rawStatus).toUpperCase().trim();
    if (["HEALTHY", "OPERATIONAL", "ONLINE", "OK", "SUCCESS"].includes(s)) return APPROVED_STATUS_LABELS.HEALTHY;
    if (["WARNING", "ATTENTION REQUIRED", "NEEDS ATTENTION", "DEGRADED"].includes(s)) return APPROVED_STATUS_LABELS.WARNING;
    if (["CRITICAL", "FAULT", "OFFLINE", "ERROR", "DANGER", "FAILED"].includes(s)) return APPROVED_STATUS_LABELS.CRITICAL;
    if (["RUNNING", "SYNCING", "SYNC IN PROGRESS", "IN PROGRESS"].includes(s)) return APPROVED_STATUS_LABELS.RUNNING;
    if (["BLOCKED", "STALE_LOCK", "LOCK_WAIT", "BACKOFF"].includes(s)) return APPROVED_STATUS_LABELS.BLOCKED;
    if (["REVIEW REQUIRED", "NEEDS REVIEW", "MANUAL APPROVAL", "REVIEW", "CONFLICT", "PENDING APPROVAL"].includes(s)) return APPROVED_STATUS_LABELS.REVIEW_REQUIRED;
    if (["IDLE", "STANDBY", "READY"].includes(s)) return APPROVED_STATUS_LABELS.IDLE;
    return rawStatus;
}

function getStatusBadgeClass(rawStatus) {
    const canonical = normalizeStatus(rawStatus);
    switch (canonical) {
        case APPROVED_STATUS_LABELS.HEALTHY: return "ds-badge-healthy";
        case APPROVED_STATUS_LABELS.WARNING: return "ds-badge-warning";
        case APPROVED_STATUS_LABELS.CRITICAL: return "ds-badge-critical";
        case APPROVED_STATUS_LABELS.RUNNING: return "ds-badge-running";
        case APPROVED_STATUS_LABELS.BLOCKED: return "ds-badge-blocked";
        case APPROVED_STATUS_LABELS.REVIEW_REQUIRED: return "ds-badge-review";
        case APPROVED_STATUS_LABELS.IDLE:
        default: return "ds-badge-idle";
    }
}

