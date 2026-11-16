import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api } from '../api/client'
import type { AuditLogRecord } from '../types/models'

export type DockHeightPreset = 'collapsed' | 'docked' | 'expanded'

export const useLogsStore = defineStore('logs', () => {
  const logs = ref<string[]>([])
  const auditLogs = ref<AuditLogRecord[]>([])
  const filterLevel = ref<string>('ALL')
  const searchQuery = ref<string>('')
  const loading = ref<boolean>(false)
  const error = ref<string | null>(null)
  const dockHeightPreset = ref<DockHeightPreset>('collapsed')

  const isDockOpen = computed(() => dockHeightPreset.value !== 'collapsed')

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

  function toggleDock(preset?: DockHeightPreset) {
    if (preset) {
      dockHeightPreset.value = preset
    } else {
      dockHeightPreset.value = dockHeightPreset.value === 'collapsed' ? 'docked' : 'collapsed'
    }
  }

  async function fetchLogs(profileId?: string, level?: string) {
    loading.value = true
    error.value = null
    try {
      const params = new URLSearchParams()
      if (profileId) params.set('profile_id', profileId)
      if (level && level !== 'ALL') params.set('level', level)
      params.set('limit', '200')

      const data = await api.get<any>(`/api/logs/bisync?${params.toString()}`)
      const rawLogs = Array.isArray(data) ? data : data.logs || []
      logs.value = rawLogs.map((item: any) => {
        if (typeof item === 'string') return item
        if (typeof item === 'object') {
          const time = item.time ? `[${item.time.substring(11, 19)}] ` : ''
          const lvl = item.level ? `[${item.level.toUpperCase()}] ` : ''
          const msg = item.msg || item.message || JSON.stringify(item)
          return `${time}${lvl}${msg}`
        }
        return String(item)
      })
    } catch (e: any) {
      error.value = e.message || 'Error fetching bisync logs'
    } finally {
      loading.value = false
    }
  }

  async function fetchAuditLogs(profileId?: string) {
    try {
      const params = new URLSearchParams()
      if (profileId) params.set('profile_id', profileId)
      params.set('limit', '100')
      const data = await api.get<any>(`/api/audit/records?${params.toString()}`)
      auditLogs.value = Array.isArray(data) ? data : data.records || []
    } catch (e: any) {
      console.error('Error fetching audit logs:', e)
    }
  }

  return {
    logs,
    auditLogs,
    filterLevel,
    searchQuery,
    loading,
    error,
    dockHeightPreset,
    isDockOpen,
    filteredLogs,
    toggleDock,
    fetchLogs,
    fetchAuditLogs,
  }
})
