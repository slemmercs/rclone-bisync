import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api } from '../api/client'
import type { TelemetryPayload } from '../types/models'
import { useFSMStore } from './fsm'

export const useTelemetryStore = defineStore('telemetry', () => {
  const telemetry = ref<TelemetryPayload | null>(null)
  const loading = ref<boolean>(false)
  const error = ref<string | null>(null)
  let pollTimer: ReturnType<typeof setInterval> | null = null

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

  function startPolling(intervalMs: number = 3000) {
    stopPolling()
    fetchTelemetry()
    pollTimer = setInterval(fetchTelemetry, intervalMs)
  }

  function stopPolling() {
    if (pollTimer) {
      clearInterval(pollTimer)
      pollTimer = null
    }
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
    fetchTelemetry,
    startPolling,
    stopPolling,
  }
})
