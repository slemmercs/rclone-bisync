import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { ReviewSession, ActionDecision } from '../types/models'
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
    if (!session.value) return 0
    return session.value.deletions.filter((d) => !d.resolved).length
  })

  const pendingConflictsCount = computed<number>(() => {
    if (!session.value) return 0
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
      const res = await fetch('/api/review')
      if (res.status === 404 || res.status === 204) {
        session.value = null
        return
      }
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data = await res.json()
      session.value = data
      if (data && data.status === 'REVIEWING') {
        const fsmStore = useFSMStore()
        fsmStore.transitionState(fsmStore.activeProfileId, 'REVIEW_REQUIRED', 'active conflict review session detected')
      }
    } catch (e: any) {
      error.value = e.message || 'Error fetching review session'
    } finally {
      loading.value = false
    }
  }

  async function stageDecision(type: 'deletion' | 'conflict', id: string, decision: ActionDecision) {
    if (!session.value) return
    try {
      const res = await fetch('/api/review/stage', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          session_id: session.value.id,
          candidate_type: type,
          candidate_id: id,
          decision,
        }),
      })
      if (!res.ok) throw new Error(`Failed to stage decision: HTTP ${res.status}`)

      // Update local state optimistically
      if (type === 'deletion') {
        const item = session.value.deletions.find((d) => d.id === id)
        if (item) {
          item.decision = decision
          item.resolved = true
        }
      } else {
        const item = session.value.conflicts.find((c) => c.id === id)
        if (item) {
          item.decision = decision
          item.resolved = true
        }
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
      const res = await fetch('/api/review/apply', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          session_id: session.value.id,
          confirmation_token: confirmationToken,
        }),
      })
      const data = await res.json()
      if (!res.ok) {
        fsmStore.transitionState(fsmStore.activeProfileId, 'ABORTED', data.error || 'resolution apply failed')
        throw new Error(data.error || 'Failed to apply review decisions')
      }
      fsmStore.transitionState(fsmStore.activeProfileId, 'COMPLETED', 'review resolutions applied successfully')
      session.value = null
      return data
    } catch (e: any) {
      error.value = e.message
      throw e
    } finally {
      applying.value = false
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
  }
})
