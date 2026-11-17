import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api } from '../api/client'
import type { QueueSnapshotDTO, QueueSnapshotResponse, QueueActionResponse } from '../types/queue'

export const useQueueStore = defineStore('queue', () => {
  const snapshot = ref<QueueSnapshotDTO | null>(null)
  const loading = ref<boolean>(false)
  const actionLoading = ref<boolean>(false)
  const error = ref<string | null>(null)
  const isModalOpen = ref<boolean>(false)
  let pollTimer: ReturnType<typeof setInterval> | null = null

  const runningCount = computed(() => snapshot.value?.running_count ?? 0)
  const queuedCount = computed(() => snapshot.value?.queued_count ?? 0)
  const totalCount = computed(() => runningCount.value + queuedCount.value)
  const isPaused = computed(() => snapshot.value?.paused ?? false)
  const activeSignature = computed(() => snapshot.value?.active_signature ?? 'SIG_DEFAULT')
  const activeJobs = computed(() => snapshot.value?.active_jobs ?? [])
  const queuedJobs = computed(() => snapshot.value?.queued_jobs ?? [])

  async function fetchQueue() {
    try {
      const data = await api.get<QueueSnapshotResponse>('/api/queue')
      if (data && data.snapshot) {
        snapshot.value = data.snapshot
        error.value = null
      }
    } catch (e: any) {
      error.value = e.message || 'Failed to fetch queue snapshot'
    }
  }

  async function pauseQueue() {
    actionLoading.value = true
    try {
      await api.post<QueueActionResponse>('/api/queue/pause')
      await fetchQueue()
    } catch (e: any) {
      error.value = e.message || 'Failed to pause queue'
      throw e
    } finally {
      actionLoading.value = false
    }
  }

  async function resumeQueue() {
    actionLoading.value = true
    try {
      await api.post<QueueActionResponse>('/api/queue/resume')
      await fetchQueue()
    } catch (e: any) {
      error.value = e.message || 'Failed to resume queue'
      throw e
    } finally {
      actionLoading.value = false
    }
  }

  async function cancelJob(profileId: string) {
    actionLoading.value = true
    try {
      await api.delete<QueueActionResponse>(`/api/queue/${encodeURIComponent(profileId)}`)
      await fetchQueue()
    } catch (e: any) {
      error.value = e.message || `Failed to cancel job for ${profileId}`
      throw e
    } finally {
      actionLoading.value = false
    }
  }

  async function clearQueue() {
    actionLoading.value = true
    try {
      const resp = await api.delete<QueueActionResponse>('/api/queue')
      await fetchQueue()
      return resp.count ?? 0
    } catch (e: any) {
      error.value = e.message || 'Failed to clear queue'
      throw e
    } finally {
      actionLoading.value = false
    }
  }

  function openModal() {
    isModalOpen.value = true
    fetchQueue()
  }

  function closeModal() {
    isModalOpen.value = false
  }

  function startPolling(intervalMs: number = 3000) {
    stopPolling()
    fetchQueue()
    pollTimer = setInterval(fetchQueue, intervalMs)
  }

  function stopPolling() {
    if (pollTimer) {
      clearInterval(pollTimer)
      pollTimer = null
    }
  }

  return {
    snapshot,
    loading,
    actionLoading,
    error,
    isModalOpen,
    runningCount,
    queuedCount,
    totalCount,
    isPaused,
    activeSignature,
    activeJobs,
    queuedJobs,
    fetchQueue,
    pauseQueue,
    resumeQueue,
    cancelJob,
    clearQueue,
    openModal,
    closeModal,
    startPolling,
    stopPolling
  }
})
