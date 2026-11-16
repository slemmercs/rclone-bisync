import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { FSMStatusPayload, SyncFSMState, StateChangeEvent } from '../types/fsm'
import { isFSMActive, canFSMTrigger } from '../types/fsm'

export const useFSMStore = defineStore('fsm', () => {
  const statuses = ref<Record<string, FSMStatusPayload>>({})
  const activeProfileId = ref<string>('default')
  const history = ref<StateChangeEvent[]>([])

  const currentStatus = computed<FSMStatusPayload>(() => {
    return (
      statuses.value[activeProfileId.value] || {
        profile_id: activeProfileId.value,
        current_state: 'IDLE',
        is_active: false,
        can_trigger_sync: true,
        pending_review: false,
      }
    )
  })

  const currentState = computed<SyncFSMState>(() => currentStatus.value.current_state)
  const isActive = computed<boolean>(() => isFSMActive(currentState.value))
  const canTrigger = computed<boolean>(() => canFSMTrigger(currentState.value))
  const pendingReview = computed<boolean>(() => currentState.value === 'REVIEW_REQUIRED')

  function setStatus(payload: FSMStatusPayload) {
    statuses.value[payload.profile_id] = payload
    if (payload.last_transition) {
      history.value.unshift(payload.last_transition)
      if (history.value.length > 100) {
        history.value.pop()
      }
    }
  }

  function setActiveProfile(profileId: string) {
    activeProfileId.value = profileId
  }

  function transitionState(profileId: string, newState: SyncFSMState, reason?: string, errorType?: string) {
    const existing = statuses.value[profileId]
    const from = existing ? existing.current_state : 'IDLE'
    const event: StateChangeEvent = {
      profile_id: profileId,
      from,
      to: newState,
      reason,
      error_type: errorType,
      timestamp: new Date().toISOString(),
    }
    statuses.value[profileId] = {
      profile_id: profileId,
      current_state: newState,
      last_transition: event,
      is_active: isFSMActive(newState),
      can_trigger_sync: canFSMTrigger(newState),
      pending_review: newState === 'REVIEW_REQUIRED',
    }
    history.value.unshift(event)
    if (history.value.length > 100) {
      history.value.pop()
    }
  }

  return {
    statuses,
    activeProfileId,
    history,
    currentStatus,
    currentState,
    isActive,
    canTrigger,
    pendingReview,
    setStatus,
    setActiveProfile,
    transitionState,
  }
})
