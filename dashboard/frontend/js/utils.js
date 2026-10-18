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
            object: parsed.object || null,
            objectType: parsed.objectType || null,
            source: parsed.source || null,
            details: structuredDetails,
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
