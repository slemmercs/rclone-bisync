export interface QueueJobDTO {
  profile_id: string
  profile_name: string
  origin: string
  dry_run: boolean
  resync: boolean
  force: boolean
  signature: string
  enqueued_at: string
  started_at?: string | null
  wait_seconds: number
}

export interface QueueSnapshotDTO {
  paused: boolean
  active_signature: string
  max_concurrency: number
  running_count: number
  queued_count: number
  active_jobs: QueueJobDTO[]
  queued_jobs: QueueJobDTO[]
}

export interface QueueSnapshotResponse {
  status: string
  snapshot: QueueSnapshotDTO
  error?: string
}

export interface QueueActionResponse {
  status: string
  message?: string
  count?: number
  error?: string
}
