/**
 * Rclone Bisync Dashboard — Deduplication Log Parser
 * Parses rclone dedupe CLI output into structured collision and deletion summaries.
 */

function parseDedupeCandidateLine(line) {
    if (!line || typeof line !== "string") return null;
    const clean = line.replace(/^\d{4}\/\d{2}\/\d{2}\s+\d{2}:\d{2}:\d{2}\s+(?:NOTICE:\s+)?/, "").trim();

    // Check if line indicates a candidate entry: e.g. "1: ...", "[1] ...", or has size/modtime/id details
    const candidateMatch = clean.match(/^(?:(?:\[(\d+)\]|(\d+)[:\)])\s*)?(.*?)$/);
    if (!candidateMatch) return null;

    const index = parseInt(candidateMatch[1] || candidateMatch[2] || "0", 10) || null;
    const details = candidateMatch[3] || clean;

    let size = null;
    const sizeMatch = details.match(/(\d+)\s*(?:bytes|byte|B)\b/i);
    if (sizeMatch) {
        size = parseInt(sizeMatch[1], 10);
    }

    let modtime = null;
    const timeMatch = details.match(/\b(\d{4}[-/]\d{2}[-/]\d{2}(?:[ T]\d{2}:\d{2}:\d{2}(?:\.\d+)?)?(?:Z|[+-]\d{2}:?\d{2})?)\b/);
    if (timeMatch) {
        modtime = timeMatch[1];
    }

    let id = null;
    const idMatch = details.match(/\bID\s*[:=]?\s*["']?([^"',\s]+)["']?/i);
    if (idMatch) {
        id = idMatch[1];
    }

    // Only consider it a candidate if it has an index or at least one of (size, modtime, id)
    if (index !== null || size !== null || modtime !== null || id !== null) {
        return {
            index,
            raw: clean,
            size,
            modtime,
            id
        };
    }
    return null;
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
    let currentItem = null;

    const getOrCreateItem = (p, isDir) => {
        if (!itemMap[p]) {
            const lastSlash = p.lastIndexOf("/");
            const name = lastSlash !== -1 ? p.substring(lastSlash + 1) : p;
            const dir = lastSlash !== -1 ? p.substring(0, lastSlash) : "";
            itemMap[p] = {
                path: p,
                name: name,
                dir: dir,
                found: isDir ? 2 : 2,
                deleted: 0,
                isDir: isDir,
                candidates: []
            };
            totalCollisions++;
        }
        return itemMap[p];
    };

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
            const item = getOrCreateItem(p, false);
            item.found = count;
            currentItem = item;
            continue;
        }

        m = line.match(/(?:NOTICE:\s+)?(.*?):\s+Deleted\s+(\d+)\s+extra\s+copies/i);
        if (m) {
            const p = m[1].replace(/^\d{4}\/\d{2}\/\d{2}\s+\d{2}:\d{2}:\d{2}\s+(?:NOTICE:\s+)?/, "").trim();
            const count = parseInt(m[2], 10) || 1;
            const item = getOrCreateItem(p, false);
            item.found = Math.max(item.found, count + 1);
            item.deleted = count;
            totalDeleted += count;
            currentItem = item;
            continue;
        }

        m = line.match(/(.*?):\s+(\d+)\s+duplicates\s+of\s+this\s+directory/i);
        if (m) {
            const p = m[1].replace(/^\d{4}\/\d{2}\/\d{2}\s+\d{2}:\d{2}:\d{2}\s+(?:NOTICE:\s+)?/, "").trim();
            const count = parseInt(m[2], 10) || 2;
            const item = getOrCreateItem(p, true);
            item.found = count;
            currentItem = item;
            continue;
        }

        // Try parsing line as candidate detail associated with currentItem
        if (currentItem) {
            const candidate = parseDedupeCandidateLine(line);
            if (candidate) {
                currentItem.candidates.push(candidate);
                continue;
            }
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

if (typeof module !== "undefined" && module.exports) {
    module.exports = { parseDedupeOutput, parseDedupeCandidateLine };
}

if (typeof window !== "undefined") {
    window.parseDedupeOutput = parseDedupeOutput;
    window.parseDedupeCandidateLine = parseDedupeCandidateLine;
}
