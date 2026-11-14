/**
 * Rclone Bisync Dashboard — Log Clustering & Run Consolidation Engine
 * Handles message normalization, repetitive folding, and run-level event consolidation.
 */

function normalizeLogMessage(msg) {
    if (!msg) return "";
    return msg.replace(/\[cid=[a-zA-Z0-9_-]+\]/g, "")
        .replace(/\b\d+(\.\d+)?\s*(s|ms|m|h|B|KB|MB|GB|TB|PB)\b/gi, "")
        .replace(/\b\d{4}-\d{2}-\d{2}[T\s]\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:?\d{2})?\b/g, "")
        .replace(/\b\d+\s+(transferred|checked|deleted|matching|errors)\b/gi, "")
        .replace(/\s+/g, " ").trim();
}

function consolidateRunEntries(entries, profiles = []) {
    if (!Array.isArray(entries) || entries.length === 0) return [];

    const profileMap = {};
    if (Array.isArray(profiles)) {
        for (const p of profiles) {
            if (p && p.id) profileMap[p.id] = p.name || p.id;
        }
    }

    const runGroups = new Map();
    const result = [];

    for (let i = 0; i < entries.length; i++) {
        const item = entries[i];
        const runId = item.run_id;
        if (!runId) {
            result.push(item);
            continue;
        }

        if (!runGroups.has(runId)) {
            const group = { id: "run-" + runId, run_id: runId, sub_entries: [item] };
            runGroups.set(runId, group);
            result.push(group);
        } else {
            runGroups.get(runId).sub_entries.push(item);
        }
    }

    for (let i = 0; i < result.length; i++) {
        const item = result[i];
        if (!item.sub_entries) continue;

        const subs = item.sub_entries;
        const auditEntry = subs.find(e => e.type === "audit" || e.action === "SYNC_EXECUTED" || e.action === "DEDUPE_EXECUTED");
        const terminalEntry = subs.find(e => e.status || e.duration_sec != null);
        const errorEntry = subs.find(e => (e.level && e.level.trim().toUpperCase() === "ERROR") || e.status === "error" || e.error_type);
        const newest = subs[0];

        const runId = item.run_id;
        const jobId = subs.find(e => e.job_id)?.job_id || null;
        const profileId = subs.find(e => e.profile_id)?.profile_id || null;
        let profileName = subs.find(e => e.profile_name)?.profile_name || null;
        if (!profileName && profileId && profileMap[profileId]) {
            profileName = profileMap[profileId];
        }

        const seqId = auditEntry?.seq_id || subs.find(e => e.seq_id)?.seq_id || null;
        const actor = auditEntry?.actor || subs.find(e => e.actor)?.actor || null;
        const hash = auditEntry?.hash || subs.find(e => e.hash)?.hash || null;
        const prevHash = auditEntry?.prev_hash || subs.find(e => e.prev_hash)?.prev_hash || null;
        const origin = auditEntry?.origin || subs.find(e => e.origin)?.origin || null;
        const originBadge = auditEntry?.origin_badge || subs.find(e => e.origin_badge)?.origin_badge || (typeof getOriginBadge === "function" ? getOriginBadge(origin) : null);
        const durationSec = terminalEntry?.duration_sec ?? auditEntry?.duration_sec ?? subs.find(e => e.duration_sec != null)?.duration_sec ?? null;
        const filesSynced = terminalEntry?.files_synced ?? auditEntry?.files_synced ?? subs.find(e => e.files_synced != null)?.files_synced ?? null;
        const errorType = errorEntry?.error_type || (errorEntry && errorEntry.status === "error" ? "SYNC_ERROR" : null);
        const isError = !!errorEntry;

        let action = null;
        if (runId.startsWith("dedupe-")) action = isError ? "DEDUPE_FAILED" : "DEDUPE_EXECUTED";
        else if (isError) action = subs.find(e => e.action?.includes("ABORT"))?.action || "SYNC_FAILED";
        else action = auditEntry?.action || subs.find(e => e.action)?.action || "SYNC_EXECUTED";

        let status = isError ? "error" : (terminalEntry?.status || auditEntry?.status || "success");
        let msg = "";
        if (runId.startsWith("dedupe-")) {
            msg = isError ? `Deduplication failed${errorType ? ': ' + errorType : ''}` : "Deduplication complete";
        } else if (isError) {
            msg = `Sync failed${errorType ? ' (' + errorType : ''}${errorEntry?.msg && !errorEntry.msg.startsWith("Sync failed") ? ': ' + errorEntry.msg : ''}${errorType ? ')' : ''}`;
        } else if (filesSynced !== null && durationSec !== null) {
            msg = `Sync completed (${filesSynced} ${filesSynced === 1 ? 'file' : 'files'} in ${durationSec.toFixed(1)}s)`;
        } else if (durationSec !== null) {
            msg = `Sync completed in ${durationSec.toFixed(1)}s`;
        } else {
            msg = terminalEntry?.msg || auditEntry?.msg || newest.msg || "Sync run executed";
        }

        result[i] = {
            id: item.id,
            time: newest.time,
            level: isError ? "ERROR  " : (auditEntry ? "AUDIT  " : "INFO   "),
            levelClass: isError ? "text-rose-400 font-bold" : (auditEntry ? "text-purple-400 font-bold" : "text-emerald-400"),
            msg: msg,
            type: auditEntry ? "audit" : "sync",
            is_http_access: false,
            is_read_only_access: false,
            seq_id: seqId,
            profile_id: profileId,
            profile_name: profileName,
            action: action,
            actor: actor,
            hash: hash,
            prev_hash: prevHash,
            object: subs.find(e => e.object)?.object || null,
            objectType: subs.find(e => e.objectType)?.objectType || null,
            source: subs.find(e => e.source)?.source || null,
            details: auditEntry?.details || terminalEntry?.details || newest.details || null,
            origin: origin,
            origin_badge: originBadge,
            run_id: runId,
            job_id: jobId,
            status: status,
            error_type: errorType,
            duration_sec: durationSec,
            files_synced: filesSynced,
            start_time: newest.start_time || subs[subs.length - 1].start_time || null,
            end_time: newest.end_time || null,
            isJson: true,
            consolidated: true,
            sub_entries: subs,
            raw: newest.raw
        };
    }

    return result;
}

function clusterLogEntries(entries, foldEnabled, expandedMap) {
    if (!Array.isArray(entries) || entries.length === 0) return [];

    const clusters = [];
    let currentCluster = null;

    for (let i = 0; i < entries.length; i++) {
        const item = entries[i];

        if (item.consolidated) {
            if (currentCluster) {
                currentCluster.isCluster = currentCluster.count > 1;
                clusters.push(currentCluster);
                currentCluster = null;
            }
            const clusterId = "run-" + (item.run_id || ("cluster-" + i));
            clusters.push({
                id: clusterId,
                key: clusterId,
                entry: item,
                count: (item.sub_entries && item.sub_entries.length) || 1,
                isCluster: item.sub_entries && item.sub_entries.length > 1,
                isConsolidatedRun: true,
                expanded: !!(expandedMap && expandedMap[clusterId]),
                entries: item.sub_entries || [item]
            });
            continue;
        }

        if (!foldEnabled) {
            if (currentCluster) {
                currentCluster.isCluster = currentCluster.count > 1;
                clusters.push(currentCluster);
                currentCluster = null;
            }
            clusters.push({
                id: "entry-" + i,
                key: "entry-" + i,
                entry: item,
                count: 1,
                isCluster: false,
                isConsolidatedRun: false,
                expanded: false,
                entries: [item]
            });
            continue;
        }

        const norm = normalizeLogMessage(item.msg || "");
        const prof = item.profile_id || item.profile_name || "";
        const obj = item.object || "", act = item.action || "";
        const key = `${item.level || ""}::${item.type || ""}::${prof}::${obj}::${act}::${norm}`;

        if (currentCluster && currentCluster.key === key && norm.length > 5) {
            currentCluster.count++;
            currentCluster.entries.push(item);
        } else {
            if (currentCluster) {
                currentCluster.isCluster = currentCluster.count > 1;
                clusters.push(currentCluster);
            }
            const clusterId = "cluster-" + clusters.length + "-" + i;
            currentCluster = {
                id: clusterId,
                key: key,
                entry: item,
                count: 1,
                isCluster: false,
                isConsolidatedRun: false,
                expanded: !!(expandedMap && expandedMap[clusterId]),
                entries: [item]
            };
        }
    }
    if (currentCluster) {
        currentCluster.isCluster = currentCluster.count > 1;
        clusters.push(currentCluster);
    }
    return clusters;
}

if (typeof window !== "undefined") {
    window.normalizeLogMessage = normalizeLogMessage;
    window.consolidateRunEntries = consolidateRunEntries;
    window.clusterLogEntries = clusterLogEntries;
    window.LogsClusteringModule = {
        normalizeLogMessage,
        consolidateRunEntries,
        clusterLogEntries
    };
}
