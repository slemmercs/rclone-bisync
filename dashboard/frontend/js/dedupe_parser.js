/**
 * Rclone Bisync Dashboard — Deduplication Log Parser
 * Parses rclone dedupe CLI output into structured collision and deletion summaries.
 */

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
            if (!itemMap[p]) { itemMap[p] = { path: p, found: count, deleted: 0, isDir: false }; totalCollisions++; }
            else { itemMap[p].found = count; }
            continue;
        }
        m = line.match(/(?:NOTICE:\s+)?(.*?):\s+Deleted\s+(\d+)\s+extra\s+copies/i);
        if (m) {
            const p = m[1].replace(/^\d{4}\/\d{2}\/\d{2}\s+\d{2}:\d{2}:\d{2}\s+(?:NOTICE:\s+)?/, "").trim();
            const count = parseInt(m[2], 10) || 1;
            if (!itemMap[p]) { itemMap[p] = { path: p, found: count + 1, deleted: count, isDir: false }; totalCollisions++; }
            else { itemMap[p].deleted = count; }
            totalDeleted += count;
            continue;
        }
        m = line.match(/(.*?):\s+(\d+)\s+duplicates\s+of\s+this\s+directory/i);
        if (m) {
            const p = m[1].replace(/^\d{4}\/\d{2}\/\d{2}\s+\d{2}:\d{2}:\d{2}\s+(?:NOTICE:\s+)?/, "").trim();
            const count = parseInt(m[2], 10) || 2;
            if (!itemMap[p]) { itemMap[p] = { path: p, found: count, deleted: 0, isDir: true }; totalCollisions++; }
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

if (typeof window !== "undefined") {
    window.parseDedupeOutput = parseDedupeOutput;
}
