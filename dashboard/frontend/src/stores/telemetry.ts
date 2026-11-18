import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api } from '../api/client'
import type { ActionResponse, PendingDeletionsData, TelemetryPayload } from '../types/models'
import { useFSMStore } from './fsm'

export const useTelemetryStore = defineStore('telemetry', () => {
  const telemetry = ref<TelemetryPayload | null>(null)
  const loading = ref<boolean>(false)
  const error = ref<string | null>(null)
  let pollTimer: ReturnType<typeof setInterval> | null = null

  // Pending Deletions State
  const pendingDeletions = ref<PendingDeletionsData | null>(null)
  const showDeletionsModal = ref<boolean>(false)
  const deletionsLoading = ref<boolean>(false)
  const deletionsError = ref<string | null>(null)
  const deletionsSearch = ref<string>('')

  const isSyncing = computed<boolean>(() => !!telemetry.value?.is_syncing)
  const daemonHealthy = computed<boolean>(() => {
    const status = telemetry.value?.daemon_health?.status
    return status === 'healthy' || status === 'ok'
  })
  const transferStats = computed(() => telemetry.value?.transfer_stats || null)
  const metrics = computed(() => telemetry.value?.metrics || [])
  const hasReview = computed<boolean>(() => !!telemetry.value?.has_review || !!telemetry.value?.review_pending)
  const latencyMs = computed<number | null>(() => {
    return telemetry.value?.daemon_health?.response_time_ms ?? null
  })

  const hasPendingDeletions = computed<boolean>(() => {
    if (!pendingDeletions.value) return false
    return (
      (pendingDeletions.value.has_deletions && pendingDeletions.value.total_count > 0) ||
      (pendingDeletions.value.files && pendingDeletions.value.files.length > 0)
    )
  })

  const filteredPendingDeletions = computed<string[]>(() => {
    const files = pendingDeletions.value?.files || []
    const q = deletionsSearch.value.toLowerCase().trim()
    if (!q) return files
    return files.filter(f => f.toLowerCase().includes(q))
  })

  async function fetchTelemetry() {
    try {
      const data = await api.get<TelemetryPayload>('/api/telemetry?limit=20')
      telemetry.value = data

      // Sync with FSM store
      const fsmStore = useFSMStore()
      if (data.is_syncing && fsmStore.currentState !== 'RUNNING') {
        fsmStore.transitionState(fsmStore.activeProfileId, 'RUNNING', 'telemetry reports active sync')
      } else if (!data.is_syncing && fsmStore.currentState === 'RUNNING') {
        fsmStore.transitionState(fsmStore.activeProfileId, 'COMPLETED', 'telemetry reports sync finished')
      }
      if (data.has_review && fsmStore.currentState !== 'REVIEW_REQUIRED') {
        fsmStore.transitionState(fsmStore.activeProfileId, 'REVIEW_REQUIRED', 'safety circuit breaker active')
      }
    } catch (e: any) {
      error.value = e.message || 'Error fetching telemetry'
    }
  }

  async function fetchPendingDeletions(): Promise<PendingDeletionsData | null> {
    deletionsLoading.value = true
    deletionsError.value = null
    try {
      const data = await api.get<PendingDeletionsData>('/api/deletions/pending')
      pendingDeletions.value = data
      return data
    } catch (e: any) {
      deletionsError.value = e.message || 'Failed to fetch pending deletions'
      pendingDeletions.value = {
        has_deletions: false,
        total_count: 0,
        source: 'Error',
        target_action: e.message || 'Error',
        files: [],
      }
      return null
    } finally {
      deletionsLoading.value = false
    }
  }

  function openDeletionsModal() {
    deletionsSearch.value = ''
    showDeletionsModal.value = true
    fetchPendingDeletions()
  }

  function closeDeletionsModal() {
    showDeletionsModal.value = false
  }

  async function authorizeForceSync(profileId?: string): Promise<ActionResponse> {
    const payload: { action: string; profile_id?: string } = { action: 'force_sync' }
    if (profileId) payload.profile_id = profileId
    const res = await api.post<ActionResponse>('/api/action', payload)
    await fetchPendingDeletions()
    await fetchTelemetry()
    return res
  }

  function startPolling(intervalMs: number = 3000) {
    stopPolling()
    fetchTelemetry()
    fetchPendingDeletions()
    pollTimer = setInterval(() => {
      fetchTelemetry()
      if (showDeletionsModal.value) {
        fetchPendingDeletions()
      }
    }, intervalMs)
  }

  function stopPolling() {
    if (pollTimer) {
      clearInterval(pollTimer)
      pollTimer = null
    }
  }

  // Backwards compatibility bridge for tests and legacy callers
  if (typeof window !== 'undefined') {
    const win = window as any
    win.openDeletionsModal = openDeletionsModal
    win.fetchPendingDeletions = fetchPendingDeletions
    Object.defineProperty(win, 'showDeletionsModal', {
      get: () => showDeletionsModal.value,
      set: (val: boolean) => { showDeletionsModal.value = val },
      configurable: true,
    })
    Object.defineProperty(win, 'deletionsData', {
      get: () => pendingDeletions.value || { total_count: 0, files: [] },
      configurable: true,
    })
  }

  return {
    telemetry,
    loading,
    error,
    isSyncing,
    latencyMs,
    daemonHealthy,
    transferStats,
    metrics,
    hasReview,
    pendingDeletions,
    showDeletionsModal,
    deletionsLoading,
    deletionsError,
    deletionsSearch,
    hasPendingDeletions,
    filteredPendingDeletions,
    filteredDeletions: filteredPendingDeletions,
    fetchTelemetry,
    fetchPendingDeletions,
    openDeletionsModal,
    closeDeletionsModal,
    authorizeForceSync,
    startPolling,
    stopPolling,
  }
})

