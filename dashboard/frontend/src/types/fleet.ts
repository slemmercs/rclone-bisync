import type { Profile } from './models'
import type { SyncFSMState } from './fsm'

export type StatusFilterType = 'ALL' | 'ACTIVE' | 'IDLE' | 'FAILED' | 'REVIEW_REQUIRED'

export interface FleetStatusPill {
  label: string
  value: StatusFilterType
  count: number
}

export interface FleetKpiData {
  totalProfiles: number
  activeWorkloads: number
  engineState: string
  pendingReview: boolean
}

export interface FleetProfileSyncEvent {
  profile: Profile
  dryRun: boolean
}
