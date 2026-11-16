/**
 * FSM Storage Control Plane TypeScript contracts.
 * Matches internal/models/fsm.go and internal/engine/state_machine.go.
 */

export type SyncFSMState =
  | 'IDLE'
  | 'QUEUED'
  | 'RUNNING'
  | 'REVIEW_REQUIRED'
  | 'RESOLVING'
  | 'COMPLETED'
  | 'ABORTED'

export interface StateChangeEvent {
  profile_id: string
  run_id?: string
  from: SyncFSMState
  to: SyncFSMState
  reason?: string
  error_type?: string
  timestamp: string
}

export interface FSMStatusPayload {
  profile_id: string
  current_state: SyncFSMState
  last_transition?: StateChangeEvent
  is_active: boolean
  can_trigger_sync: boolean
  pending_review: boolean
}

export function isFSMActive(state: SyncFSMState): boolean {
  return state === 'QUEUED' || state === 'RUNNING' || state === 'RESOLVING'
}

export function canFSMTrigger(state: SyncFSMState): boolean {
  return state === 'IDLE' || state === 'COMPLETED' || state === 'ABORTED'
}
