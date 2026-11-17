/**
 * API Contract Models mirroring internal/models/ and internal/profiles/.
 */

export interface Profile {
  id: string
  name: string
  description?: string
  enabled: boolean
  archived: boolean
  remote: string
  remote_path?: string
  local_path: string
  schedule: string
  review_mode: string
  notification_pref?: Record<string, any>
  safety?: {
    max_delete_percent?: number
    canary_protection?: boolean
  }
  flags?: Record<string, any>
  exclude_patterns?: string[]
  tags?: string[]
  version: number
  created_at: string
  updated_at: string
  state?: {
    last_run_time?: string
    last_run_status?: string
    last_run_duration?: number
    last_run_id?: string
    last_error?: string
    is_syncing?: boolean
  }
}

export interface RemotesResponse {
  remotes: Array<Record<string, any>>
  count: number
  active_remote?: any
  summary: Record<string, any>
  error?: string
}

export interface TransferStats {
  bytes: number
  total_bytes: number
  speed: number
  speed_str: string
  eta?: number
  percentage: number
  transferring: Array<Record<string, any>>
}

export interface TelemetryPayload {
  daemon_health: {
    status?: string
    response_time_ms?: number
    last_check?: number
    error?: string
  } | null
  notification_state: Record<string, any>
  metrics: Array<{
    timestamp: number
    datetime: string
    run_id: string
    job_id?: number
    origin?: string
    success: boolean
    duration_seconds: number
    files_synced?: number
    error_type?: string
  }>
  bisync_logs: string[]
  daemon_logs: string[]
  config: Record<string, any>
  is_syncing: boolean
  execution_status: string
  transfer_stats: TransferStats | null
  has_review: boolean
  review_pending: boolean
  mcp_status?: {
    running: boolean
    tools_count?: number
    clients_count?: number
  }
}

export interface BrowseItem {
  Path?: string
  Name: string
  Size: number
  Mtime?: string
  ModTime?: string
  MimeType?: string
  IsDir: boolean
  IsShortcut?: boolean
}

export interface BrowseResponse {
  remote: string
  path: string
  items: BrowseItem[]
  has_canary: boolean
  error?: string
}

export type ReviewSessionStatus =
  | 'CREATED'
  | 'REVIEWING'
  | 'APPLYING'
  | 'READY'
  | 'RUNNING_BISYNC'
  | 'COMPLETE'
  | 'FAILED'
  | 'ROLLBACK'
  | 'DISMISSED'

export type ActionDecision =
  | 'keep-local'
  | 'keep-remote'
  | 'keep-both'
  | 'accept-delete'
  | 'restore-item'
  | 'skip'

export interface DeletionCandidate {
  id: string
  path: string
  name: string
  is_dir: boolean
  source: 'local' | 'remote'
  target_action: string
  size?: number
  mod_time?: string
  condition: string
  decision?: ActionDecision
  resolved: boolean
}

export interface ConflictCandidate {
  id: string
  path: string
  name: string
  local_size: number
  local_mod_time: string
  remote_size: number
  remote_mod_time: string
  decision?: ActionDecision
  resolved: boolean
}

export interface ReviewSession {
  id: string
  status: ReviewSessionStatus
  condition: string
  message: string
  deletions: DeletionCandidate[]
  conflicts: ConflictCandidate[]
  created_at: string
  updated_at: string
}

export interface AuditLogRecord {
  time: string
  level: string
  type: string
  msg: string
  object?: string
  seq_id: number
  profile_id: string
  action: string
  actor: string
  hash: string
  prev_hash: string
  details?: Record<string, any>
}

export interface DaemonLogsResponse {
  run_id?: string
  job_id?: number
  count: number
  logs: string[]
  correlated: boolean
  error?: string
}

export interface PaginatedLogsResponse {
  page: number
  limit: number
  total: number
  total_pages: number
  logs: string[]
}

export interface DiagnosticCheck {
  id: string
  category: string
  name: string
  status: 'pass' | 'warning' | 'fail'
  message: string
  detail?: string
  action_title?: string
  action_url?: string
}

export interface DiagnosticSystemInfo {
  version: string
  go_version: string
  os: string
  arch: string
  pid: number
  uptime_sec: number
  base_dir: string
  log_file_path: string
  is_systemd: boolean
}

export interface DiagnosticsResponse {
  status: 'ok' | 'warning' | 'error'
  summary: Record<string, number>
  checks: DiagnosticCheck[]
  system_info: DiagnosticSystemInfo
  timestamp: string
}

