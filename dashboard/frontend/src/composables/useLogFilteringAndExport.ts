import { ref, computed, type Ref, type ComputedRef } from 'vue'
import type { LogEntry, LogSeverityCounts } from '../types/logs'
import { useClipboard } from './useClipboard'

export function computeLogSeverityCounts(
  logs: Ref<LogEntry[]> | ComputedRef<LogEntry[]> | (() => LogEntry[])
): ComputedRef<LogSeverityCounts> {
  return computed<LogSeverityCounts>(() => {
    const list = typeof logs === 'function' ? logs() : logs.value || []
    let errors = 0
    let warnings = 0
    let notices = 0
    let info = 0
    for (const l of list) {
      const lvl = (l.level || '').toLowerCase().trim()
      if (lvl === 'error') errors++
      else if (lvl === 'warning' || lvl === 'warn') warnings++
      else if (lvl === 'notice') notices++
      else if (lvl === 'info') info++
    }
    return {
      total: list.length,
      errors,
      warnings,
      notices,
      info,
    }
  })
}

export function formatLogEntryRaw(log: LogEntry): string {
  if (log.raw) {
    return typeof log.raw === 'string' ? log.raw : JSON.stringify(log.raw)
  }
  return `[${log.time || '--:--:--'}] ${log.level || 'INFO'} ${log.msg || ''}`
}

export function useLogClipboardExport() {
  const { copyToClipboard } = useClipboard()
  const copiedRaw = ref<boolean>(false)
  const copiedJSON = ref<boolean>(false)
  let rawTimer: ReturnType<typeof setTimeout> | null = null
  let jsonTimer: ReturnType<typeof setTimeout> | null = null

  async function copyLogs(logs: LogEntry[], asJSON: boolean): Promise<boolean> {
    let text = ''
    if (asJSON) {
      text = JSON.stringify(logs, null, 2)
    } else {
      text = logs.map(l => `${l.time ? `[${l.time}] ` : ''}[${(l.level || 'INFO').trim()}] ${l.msg || ''}`).join('\n')
    }

    const ok = await copyToClipboard(text)
    if (ok) {
      if (asJSON) {
        copiedJSON.value = true
        if (jsonTimer) clearTimeout(jsonTimer)
        jsonTimer = setTimeout(() => {
          copiedJSON.value = false
          jsonTimer = null
        }, 2000)
      } else {
        copiedRaw.value = true
        if (rawTimer) clearTimeout(rawTimer)
        rawTimer = setTimeout(() => {
          copiedRaw.value = false
          rawTimer = null
        }, 2000)
      }
    }
    return ok
  }

  async function copySingleLog(log: LogEntry): Promise<boolean> {
    const text = formatLogEntryRaw(log)
    return await copyToClipboard(text)
  }

  return {
    copiedRaw,
    copiedJSON,
    copyLogs,
    copySingleLog,
  }
}

export function getAuditActionBadgeClass(action?: string, type?: string): string {
  if (type === 'sync_overflow') return 'bg-amber-500/20 text-amber-300 border-amber-500/40'
  if (!action) return 'bg-slate-800 text-slate-300 border-slate-700'
  const act = action.toUpperCase()
  if (act.includes('DELETE') || act.includes('PURGE')) return 'bg-rose-950/80 text-rose-300 border-rose-800/60'
  if (act.includes('RENAME') || act.includes('MOVE')) return 'bg-purple-950/80 text-purple-300 border-purple-800/60'
  if (act.includes('CREATE') || act.includes('MKDIR') || act.includes('RESTORE')) return 'bg-emerald-950/80 text-emerald-300 border-emerald-800/60'
  if (act.includes('SYNC') || act.includes('TRANSFER')) return 'bg-indigo-950/80 text-indigo-300 border-indigo-800/60'
  if (act.includes('DEDUPE')) return 'bg-amber-950/80 text-amber-300 border-amber-800/60'
  return 'bg-purple-950/80 text-purple-300 border-purple-800/60'
}

export function getActionBadgeClass(log: LogEntry): string {
  return getAuditActionBadgeClass(log.action, log.type)
}
