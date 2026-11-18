/**
 * Catalog of standard flags for Rclone Bisync with metadata and constraints.
 */

export type FlagScope = 'engine' | 'profile'

export interface FlagCatalogItem {
  flag: string
  label: string
  desc: string
  isBool: boolean
  scope: FlagScope
  isProcessGlobal?: boolean
  placeholder?: string
  backends: string[]
}

export const RCLONE_FLAGS_CATALOG: FlagCatalogItem[] = [
  // Profile-scoped Bisync operational flags
  { flag: '--track-renames', label: '--track-renames', desc: 'Track file renames and moves between runs', isBool: true, scope: 'profile', backends: ['all'] },
  { flag: '--track-renames-strategy', label: '--track-renames-strategy', desc: 'Rename tracking strategy (hash, modtime, or leaf)', isBool: false, scope: 'profile', placeholder: 'modtime,leaf', backends: ['all'] },
  { flag: '--fast-list', label: '--fast-list', desc: 'Use recursive list (faster scanning, uses more RAM)', isBool: true, scope: 'profile', backends: ['all'] },
  { flag: '--checksum', label: '--checksum', desc: 'Skip based on checksum comparison instead of modtime', isBool: true, scope: 'profile', backends: ['all'] },
  { flag: '--ignore-checksum', label: '--ignore-checksum', desc: 'Skip checksum comparison even if available', isBool: true, scope: 'profile', backends: ['all'] },
  { flag: '--size-only', label: '--size-only', desc: 'Skip based on size only (not modtime or checksum)', isBool: true, scope: 'profile', backends: ['all'] },
  { flag: '--resilient', label: '--resilient', desc: 'Allow bisync to continue on non-fatal errors', isBool: true, scope: 'profile', backends: ['all'] },
  { flag: '--recover', label: '--recover', desc: 'Automatically recover from state inconsistencies', isBool: true, scope: 'profile', backends: ['all'] },
  { flag: '--remove-empty-dirs', label: '--remove-empty-dirs', desc: 'Purge empty directories on path1 and path2 after sync', isBool: true, scope: 'profile', backends: ['all'] },
  { flag: '--conflict-resolve', label: '--conflict-resolve', desc: 'Conflict resolution strategy (newer, older, larger, smaller, path1, path2)', isBool: false, scope: 'profile', placeholder: 'newer', backends: ['all'] },
  { flag: '--conflict-loser', label: '--conflict-loser', desc: 'Action for conflicting file on the losing side (num, pathname, delete)', isBool: false, scope: 'profile', placeholder: 'num', backends: ['all'] },

  // Daemon/Engine-scoped concurrency, output and network tuning
  { flag: '--use-json-log', label: '--use-json-log', desc: 'Output logs in structured JSON format', isBool: true, scope: 'engine', backends: ['all'] },
  { flag: '--color', label: '--color', desc: 'Terminal color output control (NEVER, ALWAYS, AUTO)', isBool: false, scope: 'engine', placeholder: 'NEVER', backends: ['all'] },
  { flag: '--check-access', label: '--check-access', desc: 'Verify root directory accessibility before syncing', isBool: true, scope: 'engine', backends: ['all'] },
  { flag: '--check-first', label: '--check-first', desc: 'Run all checks before starting transfers', isBool: true, scope: 'engine', backends: ['all'] },
  { flag: '--checkers', label: '--checkers', desc: 'Number of checkers to run in parallel (Process-Global in Rclone RC)', isBool: false, scope: 'engine', isProcessGlobal: true, placeholder: '8', backends: ['all'] },
  { flag: '--transfers', label: '--transfers', desc: 'Number of file transfers to run in parallel', isBool: false, scope: 'engine', placeholder: '4', backends: ['all'] },
  { flag: '--bwlimit', label: '--bwlimit', desc: 'Bandwidth limit (e.g. 10M, 500k, 10M:off)', isBool: false, scope: 'engine', placeholder: '10M', backends: ['all'] },
  { flag: '--tpslimit', label: '--tpslimit', desc: 'Limit HTTP transactions per second (Process-Global in Rclone RC)', isBool: false, scope: 'engine', isProcessGlobal: true, placeholder: '10', backends: ['all'] },
  { flag: '--tpslimit-burst', label: '--tpslimit-burst', desc: 'Max HTTP transactions burst (Process-Global in Rclone RC)', isBool: false, scope: 'engine', isProcessGlobal: true, placeholder: '20', backends: ['all'] },
  { flag: '--buffer-size', label: '--buffer-size', desc: 'In-memory buffer size per transfer (e.g. 16M, 64M)', isBool: false, scope: 'engine', placeholder: '16M', backends: ['all'] },
  { flag: '--retries', label: '--retries', desc: 'Number of retries on network failure', isBool: false, scope: 'engine', placeholder: '3', backends: ['all'] },
  { flag: '--retries-sleep', label: '--retries-sleep', desc: 'Sleep interval between retries', isBool: false, scope: 'engine', placeholder: '500ms', backends: ['all'] },
  { flag: '--low-level-retries', label: '--low-level-retries', desc: 'Low-level retry count', isBool: false, scope: 'engine', placeholder: '10', backends: ['all'] },
]

export const PROHIBITED_CUSTOM_FLAGS = [
  '--resync',
  '--dry-run',
  '--force',
  '--config',
  '--log-file',
  '--rc',
  '--rc-addr',
]

export function validateCustomFlagsString(raw: string): string | null {
  if (!raw || !raw.trim()) return null
  const tokens = raw.trim().split(/\s+/)
  for (const t of tokens) {
    const flagName = t.split('=')[0].toLowerCase()
    if (PROHIBITED_CUSTOM_FLAGS.includes(flagName)) {
      return `Flag '${flagName}' is managed by engine core and cannot be specified in custom flags.`
    }
  }
  return null
}
