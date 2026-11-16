import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { AuditLogRecord } from '../types/models'

export const useLogsStore = defineStore('logs', () => {
  const logs = ref<string[]>([])
  const auditLogs = ref<AuditLogRecord[]>([])
  const filterLevel = ref<string>('ALL')
  const searchQuery = ref<string>('')
  const loading = ref<boolean>(false)
  const error = ref<string | null>(null)

  const filteredLogs = computed<string[]>(() => {
    return logs.value.filter((line) => {
      if (filterLevel.value !== 'ALL') {
        const upper = line.toUpperCase()
        if (!upper.includes(filterLevel.value)) {
          return false
        }
      }
      if (searchQuery.value) {
        const query = searchQuery.value.toLowerCase()
        if (!line.toLowerCase().includes(query)) {
          return false
        }
      }
      return true
    })
  })

  async function fetchLogs(profileId?: string, level?: string) {
    loading.value = true
    error.value = null
    try {
      const params = new URLSearchParams()
      if (profileId) params.set('profile_id', profileId)
      if (level && level !== 'ALL') params.set('level', level)
      params.set('limit', '200')

      const res = await fetch(`/api/logs?${params.toString()}`)
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data = await res.json()
      logs.value = Array.isArray(data) ? data : data.logs || []
    } catch (e: any) {
      error.value = e.message || 'Error fetching logs'
    } finally {
      loading.value = false
    }
  }

  async function fetchAuditLogs(profileId?: string) {
    loading.value = true
    error.value = null
    try {
      const params = new URLSearchParams()
      if (profileId) params.set('profile_id', profileId)
      params.set('limit', '100')

      const res = await fetch(`/api/audit/logs?${params.toString()}`)
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data = await res.json()
      auditLogs.value = Array.isArray(data) ? data : data.records || []
    } catch (e: any) {
      error.value = e.message || 'Error fetching audit logs'
    } finally {
      loading.value = false
    }
  }

  return {
    logs,
    auditLogs,
    filterLevel,
    searchQuery,
    loading,
    error,
    filteredLogs,
    fetchLogs,
    fetchAuditLogs,
  }
})
