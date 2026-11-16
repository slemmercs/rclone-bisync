import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api } from '../api/client'
import type { AuditLogRecord, PaginatedLogsResponse } from '../types/models'
import type { LogEntry, LogCluster } from '../types/logs'
import { parseJsonLogLine, consolidateRunEntries, clusterLogEntries } from '../utils/logClustering'


export type DockHeightPreset = 'collapsed' | 'docked' | 'expanded'

export const useLogsStore = defineStore('logs', () => {
  const logs = ref<string[]>([])
  const auditLogs = ref<AuditLogRecord[]>([])
  const filterLevel = ref<string>('ALL')
  const searchQuery = ref<string>('')
  const loading = ref<boolean>(false)
  const error = ref<string | null>(null)
  const dockHeightPreset = ref<DockHeightPreset>('collapsed')

  // Timeline Ledger state
  const rawBisyncLogs = ref<LogEntry[]>([])
  const ledgerPage = ref<number>(1)
  const ledgerLimit = ref<number>(50)
  const ledgerTotal = ref<number>(0)
  const ledgerTotalPages = ref<number>(1)
  const ledgerCategoryFilter = ref<'all' | 'file_ops' | 'sync' | 'audit'>('all')
  const ledgerProfileFilter = ref<string>('all')
  const includeFileTransfers = ref<boolean>(false)
  const foldRepetitive = ref<boolean>(true)
  const expandedClusters = ref<Record<string, boolean>>({})

  // Correlated Daemon Logs state
  const daemonLogs = ref<LogEntry[]>([])
  const daemonLoading = ref<boolean>(false)
  const daemonError = ref<string | null>(null)
  const daemonActiveRun = ref<LogEntry | null>(null)
  const daemonModalOpen = ref<boolean>(false)


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

  async function fetchBisyncLogs(page = 1, profilesList: { id: string; name?: string }[] = []) {
    loading.value = true
    error.value = null
    ledgerPage.value = page
    try {
      const params = new URLSearchParams()
      params.set('page', String(page))
      params.set('limit', String(ledgerLimit.value))

      if (!includeFileTransfers.value && ledgerCategoryFilter.value !== 'file_ops') {
        params.set('exclude_types', 'sync_file,sync_overflow')
      }

      if (ledgerCategoryFilter.value === 'file_ops') {
        params.set('type', 'sync_file,sync_overflow,audit')
      } else if (ledgerCategoryFilter.value === 'sync') {
        params.set('type', 'sync')
      } else if (ledgerCategoryFilter.value === 'audit') {
        params.set('type', 'audit')
      }

      if (ledgerProfileFilter.value !== 'all') {
        params.set('profile_id', ledgerProfileFilter.value)
      }

      const data = await api.get<PaginatedLogsResponse>(`/api/logs/bisync?${params.toString()}`)
      const rawList = Array.isArray(data) ? data : (data.logs || [])
      const parsed = rawList.map(parseJsonLogLine).filter((l: LogEntry | null): l is LogEntry => l !== null)


      // Consolidate runs
      const consolidated = consolidateRunEntries(parsed, profilesList)
      rawBisyncLogs.value = consolidated
      ledgerTotal.value = data.total || rawList.length
      ledgerTotalPages.value = data.total_pages || Math.max(1, Math.ceil(ledgerTotal.value / ledgerLimit.value))
    } catch (e: any) {
      error.value = e.message || 'Failed to fetch timeline logs'
    } finally {
      loading.value = false
    }
  }

  const clusteredLedgerLogs = computed<LogCluster[]>(() => {
    return clusterLogEntries(rawBisyncLogs.value, foldRepetitive.value, expandedClusters.value)
  })

  function toggleCluster(clusterId: string) {
    expandedClusters.value[clusterId] = !expandedClusters.value[clusterId]
  }

  const areAllClustersExpanded = computed(() => {
    const expandable = clusteredLedgerLogs.value.filter(c => c.isCluster || c.isConsolidatedRun)
    if (expandable.length === 0) return false
    return expandable.every(c => !!expandedClusters.value[c.id])
  })

  function toggleAllClusters(expand?: boolean) {
    const target = expand !== undefined ? expand : !areAllClustersExpanded.value
    const next: Record<string, boolean> = {}
    for (const c of clusteredLedgerLogs.value) {
      if (c.isCluster || c.isConsolidatedRun) {
        next[c.id] = target
      }
    }
    expandedClusters.value = next
  }

  async function fetchDaemonLogs(run: LogEntry | null = null) {
    daemonLoading.value = true
    daemonError.value = null
    daemonActiveRun.value = run
    try {
      const params = new URLSearchParams()
      params.set('limit', '500')
      if (run) {
        if (run.run_id) params.set('run_id', run.run_id)
        if (run.job_id) params.set('job_id', String(run.job_id))
        if (run.start_time) params.set('start_time', run.start_time)
        if (run.end_time) params.set('end_time', run.end_time)
      }
      const data = await api.get<any>(`/api/logs/daemon?${params.toString()}`)
      const raw = Array.isArray(data) ? data : (data.logs || [])
      daemonLogs.value = raw.map(parseJsonLogLine).filter((l: LogEntry | null): l is LogEntry => l !== null)

    } catch (e: any) {
      daemonError.value = e.message || 'Failed to fetch daemon logs'
    } finally {
      daemonLoading.value = false
    }
  }

  function openDaemonModal(run: LogEntry | null = null) {
    daemonActiveRun.value = run
    daemonModalOpen.value = true
    fetchDaemonLogs(run)
  }

  function closeDaemonModal() {
    daemonModalOpen.value = false
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
    // Timeline ledger
    rawBisyncLogs,
    ledgerPage,
    ledgerLimit,
    ledgerTotal,
    ledgerTotalPages,
    ledgerCategoryFilter,
    ledgerProfileFilter,
    includeFileTransfers,
    foldRepetitive,
    expandedClusters,
    clusteredLedgerLogs,
    areAllClustersExpanded,
    fetchBisyncLogs,
    toggleCluster,
    toggleAllClusters,
    // Correlated daemon logs
    daemonLogs,
    daemonLoading,
    daemonError,
    daemonActiveRun,
    daemonModalOpen,
    fetchDaemonLogs,
    openDaemonModal,
    closeDaemonModal,
  }
})

