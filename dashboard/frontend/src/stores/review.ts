import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api } from '../api/client'
import type { ReviewSession, ActionDecision, BatchReviewActionResponse } from '../types/models'
import { useFSMStore } from './fsm'

export const useReviewStore = defineStore('review', () => {
  const session = ref<ReviewSession | null>(null)
  const loading = ref<boolean>(false)
  const applying = ref<boolean>(false)
  const error = ref<string | null>(null)

  const hasActiveReview = computed<boolean>(() => {
    return !!session.value && session.value.status === 'REVIEWING'
  })

  const pendingDeletionsCount = computed<number>(() => {
    if (!session.value || !Array.isArray(session.value.deletions)) return 0
    return session.value.deletions.filter((d) => !d.resolved).length
  })

  const pendingConflictsCount = computed<number>(() => {
    if (!session.value || !Array.isArray(session.value.conflicts)) return 0
    return session.value.conflicts.filter((c) => !c.resolved).length
  })

  const allResolved = computed<boolean>(() => {
    if (!session.value) return true
    return pendingDeletionsCount.value === 0 && pendingConflictsCount.value === 0
  })

  async function fetchActiveReview() {
    loading.value = true
    error.value = null
    try {
      const data = await api.get<any>('/api/review/session')
      if (data && data.active && data.session) {
        session.value = data.session
        const fsmStore = useFSMStore()
        fsmStore.transitionState(fsmStore.activeProfileId, 'REVIEW_REQUIRED', 'active conflict review session detected')
      } else {
        session.value = null
      }
    } catch (e: any) {
      error.value = e.message || 'Error fetching review session'
      session.value = null
    } finally {
      loading.value = false
    }
  }

  async function stageDecision(_type: 'deletion' | 'conflict', id: string, decision: ActionDecision) {
    if (!session.value) return
    try {
      const res = await api.post<any>('/api/review/action', {
        session_id: session.value.id,
        candidate_id: id,
        decision,
      })
      if (res && res.session) {
        session.value = res.session
      }
    } catch (e: any) {
      error.value = e.message
      throw e
    }
  }

  async function applyDecisions(confirmationToken?: string) {
    if (!session.value) return
    applying.value = true
    error.value = null
    const fsmStore = useFSMStore()
    fsmStore.transitionState(fsmStore.activeProfileId, 'RESOLVING', 'applying review resolutions')

    try {
      const data = await api.post<any>('/api/review/execute', {
        session_id: session.value.id,
        confirmation_token: confirmationToken,
      })
      if (data.status === 'error') {
        fsmStore.transitionState(fsmStore.activeProfileId, 'ABORTED', data.error || 'resolution apply failed')
        throw new Error(data.error || 'Failed to apply review decisions')
      }
      fsmStore.transitionState(fsmStore.activeProfileId, 'COMPLETED', 'review resolutions applied successfully')
      session.value = null
      return data
    } catch (e: any) {
      fsmStore.transitionState(fsmStore.activeProfileId, 'ABORTED', e.message)
      throw e
    } finally {
      applying.value = false
    }
  }


  async function applyBatchRecommendations(
    minConfidence: number = 0.7,
    maxRisk: 'LOW' | 'MEDIUM' | 'HIGH' = 'MEDIUM'
  ): Promise<number> {
    if (!session.value) return 0
    loading.value = true
    error.value = null
    try {
      const res = await api.post<BatchReviewActionResponse>('/api/review/action-batch', {
        session_id: session.value.id,
        min_confidence: minConfidence,
        max_allowed_risk: maxRisk,
      })
      if (res && res.status === 'error') {
        error.value = res.error || 'Batch recommendation failed'
        throw new Error(res.error || 'Batch recommendation failed')
      }
      if (res && res.session) {
        session.value = res.session
      }
      return res?.staged_count || 0
    } catch (e: any) {
      error.value = e.message
      throw e
    } finally {
      loading.value = false
    }
  }

  async function rollbackSession(): Promise<void> {
    if (!session.value) return
    loading.value = true
    error.value = null
    try {
      await api.post('/api/review/rollback', {
        session_id: session.value.id,
      })
      session.value = null
    } catch (e: any) {
      error.value = e.message
      throw e
    } finally {
      loading.value = false
    }
  }

  return {
    session,
    loading,
    applying,
    error,
    hasActiveReview,
    pendingDeletionsCount,
    pendingConflictsCount,
    allResolved,
    fetchActiveReview,
    stageDecision,
    applyDecisions,
    applyBatchRecommendations,
    rollbackSession,
  }
})
