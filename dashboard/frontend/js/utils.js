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

// Unified parser for Rclone and Bisync JSON log lines with legacy fallback
function parseJsonLogLine(rawLine) {
    if (typeof rawLine !== "string") return null;
    rawLine = rawLine.trim();
    if (!rawLine) return null;

    try {
        const parsed = JSON.parse(rawLine);
        const level = (parsed.level || "info").toLowerCase();
        let levelClass = "text-slate-400";
        if (parsed.type === "audit") levelClass = "text-purple-400 font-bold";
        else if (level === "error") levelClass = "text-rose-400 font-bold";
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

        let structuredDetails = null;
        let cleanMsg = (parsed.msg || "").trim();
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

        return {
            time: timeStr,
            level: level.toUpperCase().padEnd(7),
            levelClass: levelClass,
            msg: cleanMsg,
            type: parsed.type || "sync",
            seq_id: parsed.seq_id || null,
            profile_id: parsed.profile_id || null,
            action: parsed.action || null,
            actor: parsed.actor || null,
            hash: parsed.hash || null,
            prev_hash: parsed.prev_hash || null,
            object: parsed.object || (parsed.details && (parsed.details.item_path || parsed.details.path || parsed.details.object || parsed.details.file)) || null,
            objectType: parsed.objectType || null,
            source: parsed.source || null,
            details: parsed.details || structuredDetails,
            origin: parsed.origin || null,
            run_id: parsed.run_id || null,
            job_id: parsed.job_id || null,
            status: parsed.status || null,
            error_type: parsed.error_type || null,
            duration_sec: parsed.duration_sec !== undefined ? parsed.duration_sec : null,
            files_synced: parsed.files_synced !== undefined ? parsed.files_synced : null,
            start_time: parsed.start_time || null,
            end_time: parsed.end_time || null,
            isJson: true,
            raw: rawLine
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
                            if (parsedSub) {
                                results.push(parsedSub);
                                unwrappedAny = true;
                            }
                        }
                    }
                    if (unwrappedAny) continue;
                }
            } catch (e) {
                // Fall through to regular line parsing
            }
        }

        const parsed = parseJsonLogLine(line);
        if (parsed) {
            results.push(parsed);
        }
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

// Canonical 7 Approved Status Vocabulary:
// Healthy, Warning, Critical, Running, Idle, Blocked, Review Required
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
    if (s === "HEALTHY" || s === "OPERATIONAL" || s === "ONLINE" || s === "OK" || s === "SUCCESS") return APPROVED_STATUS_LABELS.HEALTHY;
    if (s === "WARNING" || s === "ATTENTION REQUIRED" || s === "NEEDS ATTENTION" || s === "DEGRADED") return APPROVED_STATUS_LABELS.WARNING;
    if (s === "CRITICAL" || s === "FAULT" || s === "OFFLINE" || s === "ERROR" || s === "DANGER" || s === "FAILED") return APPROVED_STATUS_LABELS.CRITICAL;
    if (s === "RUNNING" || s === "SYNCING" || s === "SYNC IN PROGRESS" || s === "IN PROGRESS") return APPROVED_STATUS_LABELS.RUNNING;
    if (s === "BLOCKED" || s === "STALE_LOCK" || s === "LOCK_WAIT" || s === "BACKOFF") return APPROVED_STATUS_LABELS.BLOCKED;
    if (s === "REVIEW REQUIRED" || s === "NEEDS REVIEW" || s === "MANUAL APPROVAL" || s === "REVIEW" || s === "CONFLICT" || s === "PENDING APPROVAL") return APPROVED_STATUS_LABELS.REVIEW_REQUIRED;
    if (s === "IDLE" || s === "STANDBY" || s === "READY") return APPROVED_STATUS_LABELS.IDLE;
    return rawStatus;
}

function parseDedupeOutput(raw) {
    if (!raw || typeof raw !== "string") {
        return { totalCollisions: 0, totalDeleted: 0, items: [], notices: [], errors: [], raw: "" };
    }
    const lines = raw.split("\n");
    const itemMap = {};
    const notices = [];
    const errors = [];
    let totalCollisions = 0;
    let totalDeleted = 0;

    for (let line of lines) {
        line = line.trim();
        if (!line) continue;

        if (line.includes("Can't have duplicate names here") || line.includes("Perhaps you wanted --by-hash")) {
            notices.push("Backend enforces unique filenames per directory. Use 'Cross-Folder Hash' mode to find identical files.");
            continue;
        }

        if (line.includes("ERROR :") || line.includes("Failed to dedupe") || line.includes("Error:")) {
            errors.push(line.replace(/^.*ERROR\s*:\s*/, "").replace(/^.*Error:\s*/, "").trim());
            continue;
        }

        let m = line.match(/(?:NOTICE:\s+)?(.*?):\s+Found\s+(\d+)\s+files\s+with\s+duplicate/i);
        if (m) {
            const p = m[1].replace(/^\d{4}\/\d{2}\/\d{2}\s+\d{2}:\d{2}:\d{2}\s+(?:NOTICE:\s+)?/, "").trim();
            const count = parseInt(m[2], 10) || 2;
            if (!itemMap[p]) {
                itemMap[p] = { path: p, found: count, deleted: 0, isDir: false };
                totalCollisions++;
            } else {
                itemMap[p].found = count;
            }
            continue;
        }

        m = line.match(/(?:NOTICE:\s+)?(.*?):\s+Deleted\s+(\d+)\s+extra\s+copies/i);
        if (m) {
            const p = m[1].replace(/^\d{4}\/\d{2}\/\d{2}\s+\d{2}:\d{2}:\d{2}\s+(?:NOTICE:\s+)?/, "").trim();
            const count = parseInt(m[2], 10) || 1;
            if (!itemMap[p]) {
                itemMap[p] = { path: p, found: count + 1, deleted: count, isDir: false };
                totalCollisions++;
            } else {
                itemMap[p].deleted = count;
            }
            totalDeleted += count;
            continue;
        }

        m = line.match(/(.*?):\s+(\d+)\s+duplicates\s+of\s+this\s+directory/i);
        if (m) {
            const p = m[1].replace(/^\d{4}\/\d{2}\/\d{2}\s+\d{2}:\d{2}:\d{2}\s+(?:NOTICE:\s+)?/, "").trim();
            const count = parseInt(m[2], 10) || 2;
            if (!itemMap[p]) {
                itemMap[p] = { path: p, found: count, deleted: 0, isDir: true };
                totalCollisions++;
            }
            continue;
        }
    }

    return {
        totalCollisions,
        totalDeleted,
        items: Object.values(itemMap),
        notices,
        errors,
        raw
    };
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

