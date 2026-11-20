/**
 * Standardized data and string formatters for bisync cockpits.
 */

export function formatBytes(bytes?: number | null, decimals: number = 1): string {
  if (bytes === undefined || bytes === null || isNaN(bytes) || bytes < 0) return '--'
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
  const i = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1)
  if (i < 0) return '0 B'
  const val = bytes / Math.pow(1024, i)
  return `${val.toFixed(i === 0 ? 0 : decimals)} ${units[i]}`
}

export function formatSpeed(bytesPerSec?: number | null, decimals: number = 1): string {
  if (bytesPerSec === undefined || bytesPerSec === null || isNaN(bytesPerSec) || bytesPerSec <= 0) return '0 B/s'
  return `${formatBytes(bytesPerSec, decimals)}/s`
}

export function formatDuration(seconds?: number | null): string {
  if (seconds === undefined || seconds === null || isNaN(seconds) || seconds <= 0) return '0s'
  const secs = Math.floor(seconds)
  const d = Math.floor(secs / 86400)
  const h = Math.floor((secs % 86400) / 3600)
  const m = Math.floor((secs % 3600) / 60)
  const s = secs % 60
  const parts: string[] = []
  if (d > 0) parts.push(`${d}d`)
  if (h > 0) parts.push(`${h}h`)
  if (m > 0) parts.push(`${m}m`)
  if (s > 0 || parts.length === 0) parts.push(`${s}s`)
  return parts.join(' ')
}

export function formatDate(iso?: string | Date | number | null, fallback: string = 'Never'): string {
  if (!iso) return fallback
  try {
    const d = new Date(iso)
    if (isNaN(d.getTime())) return String(iso)
    return d.toLocaleString()
  } catch {
    return String(iso)
  }
}

export function formatShortDate(iso?: string | Date | number | null, fallback: string = '--'): string {
  if (!iso) return fallback
  try {
    const d = new Date(iso)
    if (isNaN(d.getTime())) return fallback
    const pad = (n: number) => n.toString().padStart(2, '0')
    const yyyy = d.getFullYear()
    const mm = pad(d.getMonth() + 1)
    const dd = pad(d.getDate())
    const hh = pad(d.getHours())
    const min = pad(d.getMinutes())
    const ss = pad(d.getSeconds())
    return `${yyyy}-${mm}-${dd} ${hh}:${min}:${ss}`
  } catch {
    return fallback
  }
}

export function formatRelativeTime(dateOrIso?: string | Date | number | null): string {
  if (!dateOrIso) return '--'
  try {
    const d = new Date(dateOrIso)
    if (isNaN(d.getTime())) return '--'
    const diffSec = Math.floor((Date.now() - d.getTime()) / 1000)
    if (diffSec < 5) return 'just now'
    if (diffSec < 60) return `${diffSec}s ago`
    const diffMin = Math.floor(diffSec / 60)
    if (diffMin < 60) return `${diffMin}m ago`
    const diffHr = Math.floor(diffMin / 60)
    if (diffHr < 24) return `${diffHr}h ago`
    const diffDays = Math.floor(diffHr / 24)
    return `${diffDays}d ago`
  } catch {
    return '--'
  }
}

export function useFormatters() {
  return {
    formatBytes,
    formatSpeed,
    formatDuration,
    formatDate,
    formatShortDate,
    formatRelativeTime,
  }
}
