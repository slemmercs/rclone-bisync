export interface LogEntry {
  id?: string
  time?: string
  timestamp?: string
  level?: string
  levelClass?: string
  msg?: string
  type?: string
  is_http_access?: boolean
  is_read_only_access?: boolean
  seq_id?: number
  profile_id?: string
  profile_name?: string
  action?: string
  actor?: string
  hash?: string
  prev_hash?: string
  object?: string
  objectType?: string
  source?: string
  details?: Record<string, any> | null
  origin?: string
  origin_badge?: string
  run_id?: string
  job_id?: string | number
  status?: string
  error_type?: string
  duration_sec?: number
  files_synced?: number
  start_time?: string
  end_time?: string
  isJson?: boolean
  consolidated?: boolean
  sub_entries?: LogEntry[]
  raw?: any
}

export interface LogCluster {
  id: string
  key: string
  entry: LogEntry
  count: number
  isCluster: boolean
  isConsolidatedRun: boolean
  expanded: boolean
  entries: LogEntry[]
}

export interface DaemonLogParams {

  run_id?: string
  job_id?: string | number
  start_time?: string
  end_time?: string
  limit?: number
}


export interface LogSeverityCounts {
  total: number
  errors: number
  warnings: number
  notices: number
  info: number
}

export type LogCategoryFilter = 'all' | 'file_ops' | 'sync' | 'audit'
