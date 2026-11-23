/**
 * API Contract Models mirroring internal/models/ and internal/profiles/.
 */

export type RemoteAuthStatus =
  | 'connected'
  | 'migration_required'
  | 'authorization_required'
  | 'configuration_required'
  | 'error'

export type ConflictResolutionMode =
  | 'newer'
  | 'older'
  | 'larger'
  | 'smaller'
  | 'path1'
  | 'path2'

export type CompareMode =
  | 'size,modtime'
  | 'size,modtime,checksum'
  | 'checksum'
  | 'modtime'

export type NotificationPreference =
  | 'all'
  | 'failures-only'
  | 'review-required'
  | 'disabled'

export type AuditActor =
  | 'USER'
  | 'DAEMON'
  | 'MCP'
  | 'SYSTEM'

export interface ProfileSafety {
  delete_threshold_pct?: number
  failure_threshold?: number
  canary_validation?: boolean
  max_delete_percent?: number
  canary_protection?: boolean
}

export interface ActionResponse {
  status: string
  msg?: string
  error?: string
  error_type?: string
}

export interface ProfileFlags {
  conflict_resolve?: ConflictResolutionMode
  max_delete?: string | number
  canary_validation?: boolean
  compare_mode?: CompareMode
  remove_empty_dirs?: boolean
  create_empty_dirs?: boolean
  filters_file_path?: string
  custom_flags?: string
  provider_options?: Record<string, string | boolean | number>
}

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
  notification_pref?: NotificationPreference
  safety?: ProfileSafety
  flags?: ProfileFlags
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

export interface ProfileVersion {
  version: number
  profile_id: string
  action?: string
  timestamp: string
  reason: string
  snapshot: Profile
}

export interface RemoteEntry {
  name: string
  fs: string
  type: string
  type_label?: string
  is_active_target?: boolean
  about?: {
    total?: number
    used?: number
    free?: number
    trashed?: number
    other?: number
    [key: string]: any
  }
  can_public_link?: boolean
  underlying_remote?: string
  underlying_path?: string
  crypt_overlay?: string
  hasher_overlay?: string
  overlays?: Array<{ name: string; type: string }>
  [key: string]: any
}

export interface RemotesResponse {
  remotes: RemoteEntry[]
  count: number
  active_remote?: string | RemoteEntry | null
  summary: Record<string, any>
  error?: string
}

export interface RemoteAuthInfo {
  name: string
  type: string
  type_label: string
  is_oauth: boolean
  status: RemoteAuthStatus
  status_message: string
  client_id?: string
  has_client_secret: boolean
  has_token: boolean
  token_expiry?: string
  is_active_target: boolean
  last_checked: string
}

export interface AuthHealthSummary {
  total_remotes: number
  oauth_remotes: number
  connected: number
  migration_required: number
  authorization_required: number
  configuration_required: number
  errors: number
}

export interface AuthStatusResponse {
  summary: AuthHealthSummary
  remotes: RemoteAuthInfo[]
}

export interface OAuthAuditEvent {
  event: string
  remote: string
  status: string
  details?: string
  timestamp: string
}

export interface AuditEventsResponse {
  events: OAuthAuditEvent[]
  count: number
}

export interface ReconnectResponse {
  status: string
  remote: string
  auth_url?: string
  msg?: string
  error?: string
}

export interface ValidateResponse {
  status: string
  remote: string
  valid: boolean
  msg?: string
  error?: string
}

export interface TransferStats {
  bytes: number
  total_bytes: number
  speed: number
  speed_str: string
  eta?: number
  percentage: number
  transferring: ActiveTransfer[]
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
  config: Partial<GlobalConfigResponse> | Record<string, any>
  is_syncing: boolean
  execution_status: string
  transfer_stats: TransferStats | null
  has_review: boolean
  review_pending: boolean
  review_profile_id?: string
  mcp_status?: MCPStatus
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
  ID?: string
  Id?: string
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

export interface RuleResult {
  rule_id: string
  confidence: number
  risk_tier: 'LOW' | 'MEDIUM' | 'HIGH' | 'CRITICAL'
  suggested_action: ActionDecision
  reason: string
}

export interface ConflictRecommendation {
  suggested_action: ActionDecision
  confidence: number
  risk?: 'LOW' | 'MEDIUM' | 'HIGH' | 'CRITICAL'
  risk_tier?: 'LOW' | 'MEDIUM' | 'HIGH' | 'CRITICAL'
  primary_reason: string
  rule_evaluations?: RuleResult[]
}

export interface DeletionCandidate {
  id: string
  profile_id?: string
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
  recommendation?: ConflictRecommendation
}

export interface ConflictCandidate {
  id: string
  profile_id?: string
  path: string
  name: string
  local_size: number
  local_mod_time: string
  remote_size: number
  remote_mod_time: string
  condition?: string
  decision?: ActionDecision
  resolved: boolean
  recommendation?: ConflictRecommendation
}

export interface BatchReviewActionRequest {
  session_id?: string
  min_confidence: number
  max_allowed_risk: 'LOW' | 'MEDIUM' | 'HIGH'
}

export interface BatchReviewActionResponse {
  status: string
  session?: ReviewSession
  staged_count: number
  error?: string
}

export interface ReviewSession {
  id: string
  profile_id?: string
  status: ReviewSessionStatus
  condition: string
  message: string
  deletions: DeletionCandidate[]
  conflicts: ConflictCandidate[]
  created_at: string
  updated_at: string
}

export interface AuditRecord {
  seq_id: number
  timestamp: string
  profile_id: string
  profile_name?: string
  action: string
  actor: AuditActor
  details?: Record<string, any>
  prev_hash: string
  hash: string
}

export interface AuditVerification {
  valid: boolean
  total_records: number
  checked_at: string | null
  head_hash: string
  error?: string | null
  checking?: boolean
}

export interface AuditRecordsResponse {
  status: string
  records: AuditRecord[]
  total: number
  page: number
  limit: number
  error?: string
}

export interface AuditResetResponse {
  status: string
  message: string
  backup?: string
  error?: string
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
  actor: AuditActor | string
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

export interface GlobalDefaults {
  max_delete: number
  canary_validation: boolean
  conflict_resolve: ConflictResolutionMode
  compare_mode: CompareMode
  remove_empty_dirs: boolean
  create_empty_dirs: boolean
  lock_explorer_remote: boolean
  custom_flags?: string
  default_local_path?: string
  default_remote_path?: string
}

export interface ConfigDaemonSettings {
  job_poll_initial_sleep_sec: number
  job_poll_max_sleep_sec: number
  job_poll_backoff_multiplier?: number
  job_timeout_seconds: number
  lock_stale_timeout_seconds: number
  rate_limit_backoff_hours: number
  rate_limit_retry_after_backoff?: boolean
  check_stale_locks?: boolean
}

export interface ConfigLoggingSettings {
  max_log_size_mb: number
  log_backup_count: number
}

export interface ConfigCleanupSettings {
  files_to_cleanup: string
}

export interface ConfigTelemetrySettings {
  enable_metrics: boolean
  metrics_max_entries: number
  metrics_max_backups: number
}

export interface ConfigFeatureSettings {
  debug_mode: boolean
  mcp_read_only: boolean
  stale_lock_detection: boolean
  rate_limit_backoff: boolean
  daemon_health_check: boolean
  metrics_collection: boolean
  smart_notifications: boolean
  webhook_alerts: boolean
}

export interface ConfigNotificationSettings {
  app_name?: string
  success_interval_min?: number
  error_repeat_hours?: number
  webhook_url: string
  webhook_timeout_sec?: number
  watchdog_ping_url: string
  watchdog_timeout_sec?: number
}

export interface GlobalConfigResponse {
  status: string
  global_defaults: GlobalDefaults
  daemon: ConfigDaemonSettings
  logging: ConfigLoggingSettings
  cleanup: ConfigCleanupSettings
  telemetry: ConfigTelemetrySettings
  notifications: ConfigNotificationSettings
  features: ConfigFeatureSettings
}

export interface UpdateConfigRequest {
  global_defaults?: Partial<GlobalDefaults>
  daemon?: Partial<ConfigDaemonSettings>
  logging?: Partial<ConfigLoggingSettings>
  cleanup?: Partial<ConfigCleanupSettings>
  telemetry?: Partial<ConfigTelemetrySettings>
  notifications?: Partial<ConfigNotificationSettings>
  features?: Partial<ConfigFeatureSettings>
}

export interface ConfigEncryptionResponse {
  is_encrypted: boolean
  has_daemon_passphrase: boolean
  warning?: string
  config_file?: string
  error?: string
}

export interface MCPStatus {
  status: 'active' | 'offline'
  transport?: 'stdio' | 'sse'
  read_only: boolean
  host?: string
  port?: number
  pid?: number
  started_at?: string
}

export interface ActiveTransfer {
  name: string
  size: number
  bytes: number
  percentage: number
  speed: number
  speed_avg?: number
  eta?: number
}

export interface CoreStatsResponse {
  bytes: number
  speed: number
  transfers: number
  checks: number
  errors: number
  fatal_errors: boolean
  transferring?: ActiveTransfer[]
}

export interface BandwidthLimitResponse {
  bytes_per_second: number
  rate_formatted: string
  status?: string
}

export interface BandwidthLimitRequest {
  rate: string
}

export interface PendingDeletionsData {
  has_deletions: boolean
  total_count: number
  source: string
  target_action: string
  message?: string
  files: string[]
  timestamp?: number
}

export interface PendingDeletionCandidate {
  path: string
  selected?: boolean
}




