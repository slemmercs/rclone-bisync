/**
 * Catalog of standard flags for Rclone Bisync with metadata and constraints.
 */

export interface FlagCatalogItem {
  flag: string
  label: string
  desc: string
  isBool: boolean
  placeholder?: string
  backends: string[]
}

export const RCLONE_FLAGS_CATALOG: FlagCatalogItem[] = [
  { flag: '--track-renames', label: '--track-renames', desc: 'Track file renames and moves between runs', isBool: true, backends: ['all'] },
  { flag: '--track-renames-strategy', label: '--track-renames-strategy', desc: 'Rename tracking strategy (hash, modtime, or leaf)', isBool: false, placeholder: 'modtime,leaf', backends: ['all'] },
  { flag: '--fast-list', label: '--fast-list', desc: 'Use recursive list (faster scanning, uses more RAM)', isBool: true, backends: ['all'] },
  { flag: '--checksum', label: '--checksum', desc: 'Skip based on checksum comparison instead of modtime', isBool: true, backends: ['all'] },
  { flag: '--ignore-checksum', label: '--ignore-checksum', desc: 'Skip checksum comparison even if available', isBool: true, backends: ['all'] },
  { flag: '--size-only', label: '--size-only', desc: 'Skip based on size only (not modtime or checksum)', isBool: true, backends: ['all'] },
  { flag: '--resilient', label: '--resilient', desc: 'Allow bisync to continue on non-fatal errors', isBool: true, backends: ['all'] },
  { flag: '--recover', label: '--recover', desc: 'Automatically recover from state inconsistencies', isBool: true, backends: ['all'] },
  { flag: '--checkers', label: '--checkers', desc: 'Number of checkers to run in parallel', isBool: false, placeholder: '8', backends: ['all'] },
  { flag: '--transfers', label: '--transfers', desc: 'Number of file transfers to run in parallel', isBool: false, placeholder: '4', backends: ['all'] },
  { flag: '--bwlimit', label: '--bwlimit', desc: 'Bandwidth limit (e.g. 10M, 500k, 10M:off)', isBool: false, placeholder: '10M', backends: ['all'] },
  { flag: '--tpslimit', label: '--tpslimit', desc: 'Limit HTTP transactions per second', isBool: false, placeholder: '10', backends: ['all'] },
  { flag: '--tpslimit-burst', label: '--tpslimit-burst', desc: 'Max HTTP transactions burst', isBool: false, placeholder: '20', backends: ['all'] },
  { flag: '--buffer-size', label: '--buffer-size', desc: 'In-memory buffer size per transfer (e.g. 16M, 64M)', isBool: false, placeholder: '16M', backends: ['all'] },
  { flag: '--retries', label: '--retries', desc: 'Number of retries on network failure', isBool: false, placeholder: '3', backends: ['all'] },
  { flag: '--retries-sleep', label: '--retries-sleep', desc: 'Sleep interval between retries', isBool: false, placeholder: '500ms', backends: ['all'] },
  { flag: '--low-level-retries', label: '--low-level-retries', desc: 'Low-level retry count', isBool: false, placeholder: '10', backends: ['all'] },
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
