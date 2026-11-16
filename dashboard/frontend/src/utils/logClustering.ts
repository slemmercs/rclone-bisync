import type { LogEntry, LogCluster } from '../types/logs'

// Contextual path tokenizer extracting file mutation paths from log messages
export function extractPathFromLogMsg(msg: string): string | null {
  if (!msg || typeof msg !== 'string') return null
  let s = msg.trim()
  if (!s) return null

  // Fast-reject HTTP access, URLs, and safety abort summaries
  if (/^(GET|HEAD|POST|PUT|DELETE)\s+\S+\s+→/i.test(s) || /https?:\/\//i.test(s) || /safety threshold|too many deletes/i.test(s)) {
    return null
  }

  // Strip leading timestamp and log level prefix
  s = s.replace(/^\d{4}[/-]\d{2}[/-]\d{2}[\sT]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?\s*/, '')
  s = s.replace(/^(?:INFO|NOTICE|ERROR|WARNING|WARN|DEBUG)\s*:\s*/i, '')

  let found: string | null = null
  const matchers = [
    /Duplicate ignored:\s*(.+)/i,
    /(?:-\s+)?(?:Path[12]\s+)?(?:File\s+was\s+)?Delet(?:ed|ing)(?:\s*-\s*|\s*:\s*)(.+)/i,
    /Transferr(?:ing|ed)(?:\s+file)?:\s*(.+)/i,
    /Conflict:\s*(.+?)\s+differs between/i,
    /^([a-zA-Z0-9_.-]+(?:\{[^}]+\})?:[^\s:]+):\s*Conflict/i,
    /Conflict(?:\s*:\s*|\s+identified\s*(?:for\s*)?)(.+)/i,
    /Copied(?:\s*\([^)]+\))?(?:\s+to)?:\s*(.+)/i,
    /Updated:\s*(.+)/i,
    /(?:path|file|item|object|dest|src)[:=]\s*["']?([^"',\s]+)["']?/i,
  ]

  for (const re of matchers) {
    const m = s.match(re)
    if (m && m[1]) {
      found = m[1].trim()
      break
    }
  }

  if (!found) return null

  // Strip surrounding quotes
  found = found.replace(/^["'`\s]+|["'`\s]+$/g, '')

  // Strip remote target prefixes (e.g. 'Path1 "gdrive{tCwNP}:/docs/report.pdf"' or 'remote:path')
  found = found.replace(/^(?:Path[12]\s*)?["']?[a-zA-Z0-9_.-]+(?:\{[^}]+\})?:[/\\]?/, '')
  found = found.replace(/^["'`\s]+|["'`\s]+$/g, '')

  // Normalize Windows backslashes and strip trailing punctuation/extra info
  found = found.replace(/\\/g, '/')
  found = found.replace(/[;,]$/, '').trim()
  if (!found || found === '.' || found === '/' || found.startsWith('http:') || found.startsWith('https:')) return null

  return found
}

// Canonical path resolver across explicit object, nested details, and parsed message
export function getEntryPath(entry?: LogEntry | null): string | null {
  if (!entry) return null
  if (entry.object) return entry.object
  const d = entry.details
  if (d && typeof d === 'object') {
    if (d.item_path) return d.item_path
    if (d.path && d.filename) return `${d.path}/${d.filename}`.replace(/^\/+/, '')
    if (d.path && typeof d.path === 'string') return d.path
    if (d.object && typeof d.object === 'string') return d.object
    if (d.file && typeof d.file === 'string') return d.file
  }
  if (entry.msg) return extractPathFromLogMsg(entry.msg)
  return null
}

export function getOriginBadge(origin?: string | null): string {
  if (!origin) return 'SYSTEM'
  const o = origin.toLowerCase()
  if (o.includes('mcp')) return 'MCP'
  if (o.includes('api') || o.includes('http')) return 'API'
  if (o.includes('cron') || o.includes('sched')) return 'SCHED'
  if (o.includes('cli')) return 'CLI'
  return origin.toUpperCase()
}


export function parseJsonLogLine(rawLine: any): LogEntry | null {
  if (!rawLine) return null
  let parsed: any = null
  const rawStr = typeof rawLine === 'string' ? rawLine.trim() : ''

  if (typeof rawLine === 'object' && rawLine !== null) {
    parsed = rawLine
  } else if (rawStr) {
    try {
      parsed = JSON.parse(rawStr)
    } catch {
      // not json
    }
  } else {
    return null
  }

  if (!parsed) {
    const match = rawStr.match(/^(\d{4}-\d{2}-\d{2}\s+(\d{2}:\d{2}:\d{2}))\s+\[([A-Z]+)\]\s+(.*)$/)
    if (match) {
      const timeStr = match[2]
      const level = match[3].toLowerCase()
      let levelClass = 'text-slate-400'
      if (level === 'error') levelClass = 'text-rose-400 font-bold'
      else if (level === 'warning' || level === 'warn') levelClass = 'text-amber-400 font-medium'
      else if (level === 'notice') levelClass = 'text-sky-400'
      else if (level === 'info') levelClass = 'text-emerald-400'

      const legacyMsg = match[4].trim()
      const isReadOnly = /^(GET|HEAD)\s+\S+\s+→\s+[123]\d{2}/.test(legacyMsg)

      return {
        time: timeStr,
        level: level.toUpperCase().padEnd(7),
        levelClass,
        msg: legacyMsg,
        type: isReadOnly ? 'http_access' : 'sync',
        is_read_only_access: isReadOnly,
        object: extractPathFromLogMsg(legacyMsg) || undefined,
        isJson: false,
      }
    }

    let colorClass = 'text-slate-400'
    if (rawStr.includes('[ERROR]')) colorClass = 'text-rose-400 font-semibold'
    else if (rawStr.includes('[WARNING]')) colorClass = 'text-amber-400'
    else if (rawStr.includes('SUCCESS')) colorClass = 'text-emerald-400 font-medium'

    return {
      time: '--:--:--',
      level: 'RAW    ',
      levelClass: colorClass,
      msg: rawStr,
      object: extractPathFromLogMsg(rawStr) || undefined,
      isJson: false,
    }
  }

  const rawLevel = String(parsed.level || 'info').trim().toLowerCase()
  let levelClass = 'text-slate-400'
  if (parsed.type === 'audit') levelClass = 'text-purple-400 font-bold'
  else if (rawLevel === 'error') levelClass = 'text-rose-400 font-bold'
  else if (rawLevel === 'warning' || rawLevel === 'warn') levelClass = 'text-amber-400 font-medium'
  else if (rawLevel === 'notice') levelClass = 'text-sky-400'
  else if (rawLevel === 'info') levelClass = 'text-emerald-400'

  let timeStr = 'N/A'
  const ts = parsed.time || parsed.timestamp || parsed.start_time
  if (ts) {
    const dateObj = new Date(ts)
    if (!isNaN(dateObj.getTime())) {
      timeStr = dateObj.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false })
    }
  }

  let structuredDetails: any = null
  let cleanMsg = String(parsed.msg || '').trim()
  let isHttpAccess = false
  let isReadOnlyAccess = false

  if ((cleanMsg === 'http_access' || parsed.msg === 'http_access') && parsed.method && parsed.path) {
    isHttpAccess = true
    const method = String(parsed.method).toUpperCase()
    const status = parsed.status || 200
    if ((method === 'GET' || method === 'HEAD') && status < 400) {
      isReadOnlyAccess = true
    }
    const dur = parsed.duration_ms !== undefined && parsed.duration_ms !== null
      ? ` (${Number(parsed.duration_ms).toFixed(1)}ms)`
      : ''
    cleanMsg = `${parsed.method} ${parsed.path} → ${status}${dur}`
  }

  return buildLogEntry(parsed, rawLevel, levelClass, timeStr, ts, cleanMsg, isHttpAccess, isReadOnlyAccess, rawLine)
}

function buildLogEntry(
  parsed: any,
  rawLevel: string,
  levelClass: string,
  timeStr: string,
  ts: string | undefined,
  cleanMsg: string,
  isHttpAccess: boolean,
  isReadOnlyAccess: boolean,
  rawLine: any
): LogEntry {
  let structuredDetails: any = null
  if (cleanMsg.includes('{') && cleanMsg.includes('}')) {
    const jsonMatch = cleanMsg.match(/(\{[\s\S]*\})/)
    if (jsonMatch) {
      try {
        const parsedObj = JSON.parse(jsonMatch[1])
        if (typeof parsedObj === 'object' && parsedObj !== null) {
          structuredDetails = parsedObj
          const headerPart = cleanMsg.substring(0, jsonMatch.index).trim()
          if (headerPart) cleanMsg = headerPart.replace(/:$/, '')
        }
      } catch {
        // ignore
      }
    }
  }

  let profileId = parsed.profile_id || undefined
  let profileName = parsed.profile_name || undefined
  if (!profileId && cleanMsg.startsWith('[')) {
    const profMatch = cleanMsg.match(/^\[([^\]]+)\]/)
    if (profMatch && !['INFO', 'WARN', 'ERROR', 'DEBUG', 'NOTICE'].includes(profMatch[1].toUpperCase())) {
      profileName = profMatch[1]
    }
  }

  const details = parsed.details || structuredDetails
  let runId = parsed.run_id || details?.run_id || undefined
  let jobId = parsed.job_id || details?.job_id || undefined
  let origin = parsed.origin || details?.origin || undefined
  let durationSec = parsed.duration_sec ?? details?.duration_seconds ?? details?.duration_sec ?? undefined
  let filesSynced = parsed.files_synced ?? details?.files_synced ?? undefined
  let status = parsed.status || details?.status || undefined
  let errorType = parsed.error_type || details?.error_type || undefined

  return {
    time: timeStr,
    timestamp: ts,
    level: rawLevel.toUpperCase().padEnd(7),
    levelClass,
    msg: cleanMsg,
    type: isHttpAccess ? 'http_access' : (parsed.type || 'sync'),
    is_http_access: isHttpAccess,
    is_read_only_access: isReadOnlyAccess,
    seq_id: parsed.seq_id || details?.seq_id || undefined,
    profile_id: profileId,
    profile_name: profileName,
    action: parsed.action || details?.action || undefined,
    actor: parsed.actor || details?.actor || undefined,
    hash: parsed.hash || details?.hash || undefined,
    prev_hash: parsed.prev_hash || details?.prev_hash || undefined,
    object: parsed.object || details?.item_path || details?.path || details?.object || details?.file || extractPathFromLogMsg(cleanMsg) || undefined,
    objectType: parsed.objectType || details?.objectType || undefined,
    source: parsed.source || details?.source || undefined,
    details,
    origin,
    origin_badge: getOriginBadge(origin),
    run_id: runId,
    job_id: jobId,
    status,
    error_type: errorType,
    duration_sec: durationSec,
    files_synced: filesSynced,
    start_time: parsed.start_time || details?.start_time || undefined,
    end_time: parsed.end_time || details?.end_time || undefined,
    isJson: true,
    raw: rawLine,
  }
}

export function normalizeLogMessage(msg?: string): string {
  if (!msg) return ''
  return msg
    .replace(/\[cid=[a-zA-Z0-9_-]+\]/g, '')
    .replace(/\b\d+(\.\d+)?\s*(s|ms|m|h|B|KB|MB|GB|TB|PB)\b/gi, '')
    .replace(/\b\d{4}-\d{2}-\d{2}[T\s]\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:?\d{2})?\b/g, '')
    .replace(/\b\d+\s+(transferred|checked|deleted|matching|errors)\b/gi, '')
    .replace(/\s+/g, ' ')
    .trim()
}
export function consolidateRunEntries(entries: LogEntry[], profiles: { id: string; name?: string }[] = []): LogEntry[] {

  if (!Array.isArray(entries) || entries.length === 0) return []

  const profileMap: Record<string, string> = {}
  for (const p of profiles) {
    if (p && p.id) profileMap[p.id] = p.name || p.id
  }

  const runGroups = new Map<string, { id: string; run_id: string; sub_entries: LogEntry[] }>()
  const result: any[] = []

  for (let i = 0; i < entries.length; i++) {
    const item = entries[i]
    const runId = item.run_id
    if (!runId) {
      result.push(item)
      continue
    }

    if (!runGroups.has(runId)) {
      const group = { id: 'run-' + runId, run_id: runId, sub_entries: [item] }
      runGroups.set(runId, group)
      result.push(group)
    } else {
      runGroups.get(runId)!.sub_entries.push(item)
    }
  }

  for (let i = 0; i < result.length; i++) {
    const item = result[i]
    if (!item.sub_entries) continue

    const subs: LogEntry[] = item.sub_entries
    const auditEntry = subs.find(e => e.type === 'audit' || e.action === 'SYNC_EXECUTED' || e.action === 'DEDUPE_EXECUTED' || e.action === 'DEDUPE_BATCH_EXECUTED')
    const terminalEntry = subs.find(e => e.status || e.duration_sec !== undefined)
    const errorEntry = subs.find(e => (e.level && e.level.trim().toUpperCase() === 'ERROR') || e.status === 'error' || e.error_type)
    const newest = subs[0]

    const runId = item.run_id
    const jobId = subs.find(e => e.job_id)?.job_id || undefined
    const profileId = subs.find(e => e.profile_id)?.profile_id || undefined
    let profileName = subs.find(e => e.profile_name)?.profile_name || undefined
    if (!profileName && profileId && profileMap[profileId]) {
      profileName = profileMap[profileId]
    }

    const seqId = auditEntry?.seq_id || subs.find(e => e.seq_id)?.seq_id || undefined
    const actor = auditEntry?.actor || subs.find(e => e.actor)?.actor || undefined
    const hash = auditEntry?.hash || subs.find(e => e.hash)?.hash || undefined
    const prevHash = auditEntry?.prev_hash || subs.find(e => e.prev_hash)?.prev_hash || undefined
    const origin = auditEntry?.origin || subs.find(e => e.origin)?.origin || undefined
    const originBadge = auditEntry?.origin_badge || subs.find(e => e.origin_badge)?.origin_badge || getOriginBadge(origin)
    const durationSec = terminalEntry?.duration_sec ?? auditEntry?.duration_sec ?? subs.find(e => e.duration_sec !== undefined)?.duration_sec ?? undefined
    const filesSynced = terminalEntry?.files_synced ?? auditEntry?.files_synced ?? subs.find(e => e.files_synced !== undefined)?.files_synced ?? undefined
    const errorType = errorEntry?.error_type || (errorEntry && errorEntry.status === 'error' ? 'SYNC_ERROR' : undefined)
    const isError = !errorEntry ? false : true

    let action: string | undefined = undefined
    if (runId.startsWith('dedupe-')) action = isError ? 'DEDUPE_FAILED' : 'DEDUPE_EXECUTED'
    else if (isError) action = subs.find(e => e.action?.includes('ABORT'))?.action || 'SYNC_FAILED'
    else action = auditEntry?.action || subs.find(e => e.action)?.action || 'SYNC_EXECUTED'

    const status = isError ? 'error' : (terminalEntry?.status || auditEntry?.status || 'success')
    let msg = ''
    if (runId.startsWith('dedupe-')) {
      msg = isError ? `Deduplication failed${errorType ? ': ' + errorType : ''}` : 'Deduplication complete'
    } else if (isError) {
      msg = `Sync failed${errorType ? ' (' + errorType : ''}${errorEntry?.msg && !errorEntry.msg.startsWith('Sync failed') ? ': ' + errorEntry.msg : ''}${errorType ? ')' : ''}`
    } else if (filesSynced !== undefined && durationSec !== undefined) {
      msg = `Sync completed (${filesSynced} ${filesSynced === 1 ? 'file' : 'files'} in ${durationSec.toFixed(1)}s)`
    } else if (durationSec !== undefined) {
      msg = `Sync completed in ${durationSec.toFixed(1)}s`
    } else {
      msg = terminalEntry?.msg || auditEntry?.msg || newest.msg || 'Sync run executed'
    }

    result[i] = {
      id: item.id,
      time: newest.time,
      timestamp: newest.timestamp,
      level: isError ? 'ERROR  ' : (auditEntry ? 'AUDIT  ' : 'INFO   '),
      levelClass: isError ? 'text-rose-400 font-bold' : (auditEntry ? 'text-purple-400 font-bold' : 'text-emerald-400'),
      msg,
      type: auditEntry ? 'audit' : 'sync',
      is_http_access: false,
      is_read_only_access: false,
      seq_id: seqId,
      profile_id: profileId,
      profile_name: profileName,
      action,
      actor,
      hash,
      prev_hash: prevHash,
      object: subs.find(e => e.object)?.object || undefined,
      objectType: subs.find(e => e.objectType)?.objectType || undefined,
      source: subs.find(e => e.source)?.source || undefined,
      details: auditEntry?.details || terminalEntry?.details || newest.details || null,
      origin,
      origin_badge: originBadge,
      run_id: runId,
      job_id: jobId,
      status,
      error_type: errorType,
      duration_sec: durationSec,
      files_synced: filesSynced,
      start_time: newest.start_time || subs[subs.length - 1]?.start_time || undefined,
      end_time: newest.end_time || undefined,
      isJson: true,
      consolidated: true,
      sub_entries: subs,
      raw: newest.raw,
    } as LogEntry
  }

  return result
}

export function clusterLogEntries(entries: LogEntry[], foldEnabled: boolean, expandedMap: Record<string, boolean> = {}): LogCluster[] {
  if (!Array.isArray(entries) || entries.length === 0) return []

  const clusters: LogCluster[] = []
  let currentCluster: LogCluster | null = null

  for (let i = 0; i < entries.length; i++) {
    const item = entries[i]

    if (item.consolidated) {
      if (currentCluster) {
        currentCluster.isCluster = currentCluster.count > 1
        clusters.push(currentCluster)
        currentCluster = null
      }
      const clusterId = 'run-' + (item.run_id || ('cluster-' + i))
      clusters.push({
        id: clusterId,
        key: clusterId,
        entry: item,
        count: (item.sub_entries && item.sub_entries.length) || 1,
        isCluster: !!(item.sub_entries && item.sub_entries.length > 1),
        isConsolidatedRun: true,
        expanded: !!expandedMap[clusterId],
        entries: item.sub_entries || [item],
      })
      continue
    }

    if (!foldEnabled) {
      if (currentCluster) {
        currentCluster.isCluster = currentCluster.count > 1
        clusters.push(currentCluster)
        currentCluster = null
      }
      clusters.push({
        id: 'entry-' + i,
        key: 'entry-' + i,
        entry: item,
        count: 1,
        isCluster: false,
        isConsolidatedRun: false,
        expanded: false,
        entries: [item],
      })
      continue
    }

    const norm = normalizeLogMessage(item.msg || '')
    const prof = item.profile_id || item.profile_name || ''
    const obj = item.object || ''
    const act = item.action || ''
    const key = `${item.level || ''}::${item.type || ''}::${prof}::${obj}::${act}::${norm}`

    if (currentCluster && currentCluster.key === key && norm.length > 5) {
      currentCluster.count++
      currentCluster.entries.push(item)
    } else {
      if (currentCluster) {
        currentCluster.isCluster = currentCluster.count > 1
        clusters.push(currentCluster)
      }
      const clusterId = 'cluster-' + clusters.length + '-' + i
      currentCluster = {
        id: clusterId,
        key,
        entry: item,
        count: 1,
        isCluster: false,
        isConsolidatedRun: false,
        expanded: !!expandedMap[clusterId],
        entries: [item],
      }
    }
  }

  if (currentCluster) {
    currentCluster.isCluster = currentCluster.count > 1
    clusters.push(currentCluster)
  }

  return clusters
}

// Unwraps potential multi-line wrapper logs (e.g. from Rclone RC) and parses each line
export function unwrapAndParseLogLines(rawLines: any[]): LogEntry[] {
  if (!Array.isArray(rawLines)) return []
  const results: LogEntry[] = []
  for (const line of rawLines) {
    if (!line) continue

    let lineStr = ''
    if (typeof line === 'string') {
      lineStr = line
    } else if (typeof line === 'object') {
      lineStr = JSON.stringify(line)
    }

    if (lineStr.startsWith('{') && (lineStr.includes('"msg":"{\\"') || lineStr.includes('"msg":"{\'') || lineStr.includes('\\n{\\"'))) {
      try {
        const outer = typeof line === 'object' ? line : JSON.parse(lineStr)
        if (outer && typeof outer.msg === 'string' && (outer.msg.includes('{"') || outer.msg.includes("{'"))) {
          const sublines = outer.msg.split('\n')
          let unwrappedAny = false
          for (const sub of sublines) {
            const subTrim = sub.trim()
            if (subTrim.startsWith('{') && subTrim.endsWith('}')) {
              const parsedSub = parseJsonLogLine(subTrim)
              if (parsedSub) {
                results.push(parsedSub)
                unwrappedAny = true
              }
            }
          }
          if (unwrappedAny) continue
        }
      } catch {
        // Fall through to regular line parsing
      }
    }

    const parsed = parseJsonLogLine(line)
    if (parsed) results.push(parsed)
  }
  return results
}


