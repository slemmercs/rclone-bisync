export type DedupeMode = 'list' | 'oldest' | 'newest' | 'rename' | 'largest' | 'smallest'

export interface DedupeCandidate {
  id: string
  name: string
  path: string
  size?: number
  modTime?: string
  hash?: string
  stagedAction?: 'keep' | 'delete' | 'rename'
  newPath?: string
  isLocal?: boolean
}

export interface DuplicateGroup {
  hashOrName: string
  title: string
  files: DedupeCandidate[]
  resolved?: boolean
}

export interface DedupeStageAction {
  id?: string
  path: string
  action: 'delete' | 'rename' | 'keep'
  new_name?: string
  size?: number
  mod_time?: string
  candidate?: number
}

export interface StagedActionEntry extends DedupeStageAction {
  key: string
}

export interface DedupeBatchApplyRequest {
  remote: string
  path: string
  profile_id?: string
  actions: DedupeStageAction[]
  trigger_resync?: boolean
  confirmation_token?: string
}

export interface DedupeBatchApplyResponse {
  status: string
  run_id?: string
  success_count: number
  failure_count: number
  errors?: string[]
  resync_queued?: boolean
  resync_message?: string
  error?: string
}

export interface DedupeBatchProgress {
  total: number
  completed: number
  percent: number
  currentAction: string
  status: 'idle' | 'running' | 'complete' | 'error'
}

export interface DedupeBatchAction {
  action: 'delete' | 'rename' | 'keep'
  path: string
  new_path?: string
}

export interface DedupeBatchRequest {
  remote: string
  actions: DedupeBatchAction[]
  trigger_resync?: boolean
}

export interface DedupeRequest {
  remote: string
  path?: string
  mode?: DedupeMode
  by_hash?: boolean
  dedupe_both?: boolean
  profile_id?: string
}

export interface DedupeResponse {
  status: string
  run_id?: string
  remote: string
  path: string
  mode: string
  by_hash?: boolean
  output: string
  local_path?: string
  local_output?: string
  collisions: number
  deleted?: number
  error?: string
}

export interface ProviderOption {
  Name: string
  Help?: string
  Type?: string
  Default?: any
  Required?: boolean
  IsPassword?: boolean
  Advanced?: boolean
  Examples?: Array<{ Value: string; Help: string }>
}

export interface ProviderCatalogItem {
  name: string
  prefix?: string
  description?: string
  category: 'popular' | 'cloud' | 'object' | 'protocol' | 'virtual' | 'other'
  options?: ProviderOption[]
  oauth_supported?: boolean
  supports_oauth?: boolean
}

export interface ProviderCatalogResponse {
  providers: ProviderCatalogItem[]
  categories: Record<string, string[]>
  popular: string[]
}

export interface CreateRemotePayload {
  name: string
  type: string
  parameters: Record<string, any>
  create_profile?: boolean
  local_path?: string
  profile_name?: string
  schedule?: string
  remote_subpath?: string
}
