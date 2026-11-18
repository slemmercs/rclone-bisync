import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api } from '../api/client'
import { useRemotesStore } from './remotes'
import type {
  DedupeMode,
  DedupeStageAction,
  StagedActionEntry,
  DedupeBatchApplyRequest,
  DedupeBatchApplyResponse,
  DedupeBatchProgress,
  DedupeRequest,
  DedupeResponse,
  ProviderCatalogItem,
  ProviderCatalogResponse,
  CreateRemotePayload,
  DedupeScanCache,
} from '../types/dedupe'
import type { BrowseItem, BrowseResponse } from '../types/models'
import { parseDedupeOutput, type ParsedDedupeOutput, type ParsedCandidate, type ParsedDedupeItem } from '../utils/dedupeParser'

export const useDedupeStore = defineStore('dedupe', () => {
  // Modal & Workspace state
  const modalOpen = ref<boolean>(false)
  const remote = ref<string>('')
  const path = ref<string>('')
  const mode = ref<DedupeMode>('list')
  const byHash = ref<boolean>(true)
  const dedupeBoth = ref<boolean>(false)
  const loading = ref<boolean>(false)
  const executing = ref<boolean>(false)
  const rawOutput = ref<string>('')
  const parsed = ref<ParsedDedupeOutput>(parseDedupeOutput(''))
  const runId = ref<string | null>(null)
  const cachedTimestamp = ref<number | null>(null)
  const error = ref<string | null>(null)
  const activeTab = ref<'items' | 'explorer' | 'raw'>('items')

  // Staged batch actions & progress
  const stagedActions = ref<Record<string, DedupeStageAction>>({})
  const triggerResync = ref<boolean>(true)
  const drawerOpen = ref<boolean>(false)
  const progress = ref<DedupeBatchProgress>({
    total: 0,
    completed: 0,
    percent: 0,
    currentAction: '',
    status: 'idle',
  })

  // Directory Explorer state
  const explorerPath = ref<string>('')
  const explorerItems = ref<BrowseItem[]>([])
  const explorerLoading = ref<boolean>(false)
  const explorerError = ref<string | null>(null)

  // Remote creation wizard state
  const wizardOpen = ref<boolean>(false)
  const providers = ref<ProviderCatalogItem[]>([])
  const providersLoading = ref<boolean>(false)
  const selectedProvider = ref<ProviderCatalogItem | null>(null)

  // Computed views
  const stagedCount = computed(() => Object.keys(stagedActions.value).length)
  const hasCollisions = computed(() => parsed.value.totalCollisions > 0)

  const stagedEntries = computed<StagedActionEntry[]>(() => {
    return Object.entries(stagedActions.value).map(([key, item]) => ({
      key,
      ...item,
    }))
  })

  const stagedCounts = computed(() => {
    const list = Object.values(stagedActions.value)
    return {
      total: list.length,
      delete: list.filter((a) => a.action === 'delete').length,
      rename: list.filter((a) => a.action === 'rename').length,
      keep: list.filter((a) => a.action === 'keep').length,
    }
  })

  const breadcrumbs = computed(() => {
    const raw = (explorerPath.value || '').trim().replace(/^\/+|\/+$/g, '')
    const rootLabel = remote.value ? `${remote.value}:` : '~'
    const crumbs = [{ name: rootLabel, path: '' }]
    if (!raw) return crumbs
    const parts = raw.split('/').filter(Boolean)
    let accum = ''
    for (const part of parts) {
      accum = accum ? `${accum}/${part}` : part
      crumbs.push({ name: part, path: accum })
    }
    return crumbs
  })

  function getCandidateKey(filePath: string, candId?: string | null, index?: number | null): string {
    const cId = candId || ''
    const idx = index !== undefined && index !== null ? index : ''
    return `${filePath}::${cId}::${idx}`
  }

  function getDedupeCacheKey(rem: string, p: string): string {
    const cleanRemote = (rem || '').replace(/:+$/, '').trim()
    const cleanPath = (p || '').replace(/^\/+|\/+$/g, '')
    return `rclone_dedupe_cache_${cleanRemote}_${cleanPath}`
  }

  function getDedupeCache(rem: string, p: string): DedupeScanCache | null {
    if (typeof window === 'undefined' || !window.sessionStorage) return null
    const key = getDedupeCacheKey(rem, p)
    try {
      const raw = window.sessionStorage.getItem(key)
      if (raw) return JSON.parse(raw) as DedupeScanCache
    } catch (_) {}
    return null
  }

  function setDedupeCache(rem: string, p: string, data: DedupeScanCache) {
    if (typeof window === 'undefined' || !window.sessionStorage) return
    const key = getDedupeCacheKey(rem, p)
    try {
      window.sessionStorage.setItem(key, JSON.stringify(data))
    } catch (_) {}
  }

  function removeDedupeCache(rem: string, p: string) {
    if (typeof window === 'undefined' || !window.sessionStorage) return
    const key = getDedupeCacheKey(rem, p)
    try {
      window.sessionStorage.removeItem(key)
    } catch (_) {}
  }

  function openModal(remoteName: string, subpath: string = '', initialMode: DedupeMode = 'list', providerType?: string) {
    const cleanRemote = remoteName.replace(/:+$/, '').trim()
    remote.value = cleanRemote
    path.value = subpath
    mode.value = initialMode

    // Provider-aware strategy differentiation
    const remotesStore = useRemotesStore()
    let pType = (providerType || '').toLowerCase()
    if (!pType && remotesStore.remotes.length > 0) {
      const match = remotesStore.remotes.find(
        (r) => (r.name || '').replace(/:+$/, '').trim().toLowerCase() === cleanRemote.toLowerCase()
      )
      if (match) {
        pType = (match.type || '').toLowerCase()
      }
    }
    if (!pType && cleanRemote === 'local') {
      pType = 'local'
    }

    const singleFilenameProviders = ['onedrive', 's3', 'b2', 'swift', 'azureblob', 'local', 'sftp', 'ftp']
    if (pType && singleFilenameProviders.includes(pType)) {
      byHash.value = true
    } else if (pType === 'drive') {
      byHash.value = false
    } else {
      byHash.value = false
    }

    // Check session cache for previous scan results
    const cached = getDedupeCache(cleanRemote, subpath)
    if (cached) {
      mode.value = cached.mode || initialMode
      byHash.value = cached.byHash !== undefined ? cached.byHash : byHash.value
      dedupeBoth.value = Boolean(cached.dedupeBoth)
      rawOutput.value = cached.rawOutput || ''
      parsed.value = cached.parsed || parseDedupeOutput(cached.rawOutput || '')
      runId.value = cached.runId || null
      cachedTimestamp.value = cached.timestamp || null
    } else {
      rawOutput.value = ''
      parsed.value = parseDedupeOutput('')
      runId.value = null
      cachedTimestamp.value = null
    }

    error.value = null
    stagedActions.value = {}
    drawerOpen.value = false
    activeTab.value = 'items'
    explorerPath.value = subpath
    explorerItems.value = []
    explorerError.value = null
    progress.value = { total: 0, completed: 0, percent: 0, currentAction: '', status: 'idle' }
    modalOpen.value = true
  }

  function closeModal() {
    if (executing.value || progress.value.status === 'running') return
    modalOpen.value = false
  }

  function toggleStagedDrawer() {
    drawerOpen.value = !drawerOpen.value
  }

  async function runDedupe(dryRun: boolean = true) {
    if (!remote.value) return
    loading.value = true
    if (!dryRun) executing.value = true
    error.value = null

    const effectiveMode = dryRun ? 'list' : mode.value
    const payload: DedupeRequest = {
      remote: remote.value,
      path: path.value || '',
      mode: effectiveMode,
      by_hash: byHash.value,
      dedupe_both: dedupeBoth.value,
    }

    try {
      const res = await api.post<DedupeResponse>('/api/remotes/dedupe', payload)
      if (res.status === 'error') {
        error.value = res.error || 'Deduplication failed'
        rawOutput.value = (rawOutput.value ? rawOutput.value + '\n' : '') + `Error: ${error.value}`
      } else {
        runId.value = res.run_id || null
        rawOutput.value = res.output || 'No duplicates found. All files are unique.'
        if (res.local_output) {
          rawOutput.value += `\n\n--- Local Endpoint Output ---\n${res.local_output}`
        }
      }
      parsed.value = parseDedupeOutput(rawOutput.value)
      if (!error.value && rawOutput.value) {
        const cacheEntry: DedupeScanCache = {
          remote: remote.value,
          path: path.value || '',
          mode: mode.value,
          byHash: byHash.value,
          dedupeBoth: dedupeBoth.value,
          rawOutput: rawOutput.value,
          parsed: parsed.value,
          runId: runId.value,
          timestamp: Date.now(),
        }
        setDedupeCache(remote.value, path.value || '', cacheEntry)
        cachedTimestamp.value = cacheEntry.timestamp
      }
    } catch (e: any) {
      error.value = e.message || 'Deduplication request failed'
      rawOutput.value = (rawOutput.value ? rawOutput.value + '\n' : '') + `Request error: ${error.value}`
      parsed.value = parseDedupeOutput(rawOutput.value)
    } finally {
      loading.value = false
      executing.value = false
    }
  }

  function stageCandidateAction(
    collisionPath: string,
    cand: ParsedCandidate,
    action: 'keep' | 'delete' | 'rename',
    newName: string = ''
  ) {
    const key = getCandidateKey(collisionPath, cand.id, cand.index)
    stagedActions.value[key] = {
      id: cand.id || undefined,
      path: collisionPath,
      action,
      new_name: newName || undefined,
      size: cand.size !== null ? cand.size : undefined,
      mod_time: cand.modtime || undefined,
      candidate: cand.index !== null ? cand.index : undefined,
    }
    if (!drawerOpen.value) {
      drawerOpen.value = true
    }
  }

  function stageKeepOnly(collision: ParsedDedupeItem, keepCand: ParsedCandidate) {
    for (const c of collision.candidates) {
      if (c.index === keepCand.index && c.id === keepCand.id) {
        stageCandidateAction(collision.path, c, 'keep')
      } else {
        stageCandidateAction(collision.path, c, 'delete')
      }
    }
  }

  function unstageCandidateAction(key: string) {
    delete stagedActions.value[key]
    if (Object.keys(stagedActions.value).length === 0) {
      drawerOpen.value = false
    }
  }

  function clearStagedActions() {
    stagedActions.value = {}
    drawerOpen.value = false
    progress.value = { total: 0, completed: 0, percent: 0, currentAction: '', status: 'idle' }
  }

  function getStagedCandidateAction(collisionPath: string, cand: ParsedCandidate): DedupeStageAction | null {
    const key = getCandidateKey(collisionPath, cand.id, cand.index)
    return stagedActions.value[key] || null
  }

  async function applyBatch(): Promise<DedupeBatchApplyResponse | null> {
    if (!remote.value || stagedCount.value === 0) return null
    loading.value = true
    error.value = null

    const actionList = Object.values(stagedActions.value)
    progress.value = {
      total: actionList.length,
      completed: 0,
      percent: 10,
      currentAction: `Applying batch of ${actionList.length} action(s)...`,
      status: 'running',
    }

    const payload: DedupeBatchApplyRequest = {
      remote: remote.value,
      path: path.value || '',
      actions: actionList,
      trigger_resync: triggerResync.value,
    }

    try {
      const res = await api.post<DedupeBatchApplyResponse>('/api/remotes/dedupe/batch', payload)
      if (res.status === 'error') {
        const errMsg = res.error || (res.errors && res.errors[0]) || 'Batch apply failed'
        error.value = errMsg
        progress.value = {
          total: actionList.length,
          completed: 0,
          percent: 100,
          currentAction: `Batch failed: ${errMsg}`,
          status: 'error',
        }
        rawOutput.value = (rawOutput.value || '') + `\n[ERROR] Batch deduplication failed: ${errMsg}`
        if (res.errors && res.errors.length > 0) {
          rawOutput.value += '\n' + res.errors.map((e) => `  - ${e}`).join('\n')
        }
        return res
      }

      const succ = res.success_count || 0
      const fail = res.failure_count || 0
      progress.value = {
        total: actionList.length,
        completed: succ,
        percent: 100,
        currentAction: `Completed: ${succ} succeeded, ${fail} failed.`,
        status: fail > 0 ? 'error' : 'complete',
      }

      rawOutput.value = (rawOutput.value || '') + `\nBatch execution applied: ${succ} succeeded, ${fail} failed.`
      if (res.errors && res.errors.length > 0) {
        rawOutput.value += '\nFailures encountered:\n' + res.errors.map((e) => `  - ${e}`).join('\n')
      }

      clearStagedActions()
      if (succ > 0) {
        removeDedupeCache(remote.value, path.value || '')
        cachedTimestamp.value = null
      }
      if (explorerItems.value.length > 0) {
        await loadExplorer(explorerPath.value)
      }
      return res
    } catch (e: any) {
      const msg = e.message || 'Batch execution failed'
      error.value = msg
      progress.value = {
        total: actionList.length,
        completed: 0,
        percent: 100,
        currentAction: `Batch error: ${msg}`,
        status: 'error',
      }
      return null
    } finally {
      loading.value = false
    }
  }

  // Directory Explorer actions
  async function loadExplorer(folderPath?: string) {
    if (folderPath !== undefined) {
      explorerPath.value = folderPath
    }
    explorerLoading.value = true
    explorerError.value = null

    try {
      const cleanPath = (explorerPath.value || '').trim().replace(/^\/+|\/+$/g, '')
      const res = await api.post<BrowseResponse>('/api/browse', {
        remote: remote.value,
        path: cleanPath,
      })
      if (res.error) {
        explorerError.value = res.error
        explorerItems.value = []
      } else {
        explorerItems.value = res.items || []
      }
    } catch (e: any) {
      explorerError.value = e.message || 'Failed to load directory'
      explorerItems.value = []
    } finally {
      explorerLoading.value = false
    }
  }

  function navigateExplorerParent() {
    let current = (explorerPath.value || '').trim().replace(/^\/+|\/+$/g, '')
    if (!current) return
    const parts = current.split('/')
    parts.pop()
    loadExplorer(parts.join('/'))
  }

  function selectExplorerItem(item: BrowseItem) {
    if (item.IsDir) {
      loadExplorer(item.Path)
    }
  }

  // Provider Catalog & Remote Creation Wizard
  async function fetchProviders(force = false) {
    if (providers.value.length > 0 && !force) return
    providersLoading.value = true
    try {
      const data = await api.get<ProviderCatalogResponse>('/api/remotes/providers')
      providers.value = data.providers || []
    } catch (e: any) {
      console.error('Failed to load provider catalog:', e)
    } finally {
      providersLoading.value = false
    }
  }

  function openWizard(defaultProvider?: string) {
    wizardOpen.value = true
    fetchProviders().then(() => {
      if (defaultProvider && providers.value.length > 0) {
        const found = providers.value.find((p) => p.name === defaultProvider || p.prefix === defaultProvider)
        if (found) selectedProvider.value = found
      }
    })
  }

  function closeWizard() {
    wizardOpen.value = false
    selectedProvider.value = null
  }

  async function createRemote(payload: CreateRemotePayload) {
    loading.value = true
    try {
      const res = await api.post<any>('/api/remotes/create', payload)
      return res
    } finally {
      loading.value = false
    }
  }

  function clearCache() {
    removeDedupeCache(remote.value, path.value || '')
    rawOutput.value = ''
    parsed.value = parseDedupeOutput('')
    runId.value = null
    cachedTimestamp.value = null
  }

  return {
    modalOpen,
    remote,
    path,
    mode,
    byHash,
    dedupeBoth,
    loading,
    executing,
    rawOutput,
    parsed,
    runId,
    cachedTimestamp,
    error,
    activeTab,
    stagedActions,
    triggerResync,
    drawerOpen,
    progress,
    stagedCount,
    hasCollisions,
    stagedEntries,
    stagedCounts,
    breadcrumbs,
    explorerPath,
    explorerItems,
    explorerLoading,
    explorerError,
    getCandidateKey,
    openModal,
    closeModal,
    toggleStagedDrawer,
    runDedupe,
    stageCandidateAction,
    stageKeepOnly,
    unstageCandidateAction,
    clearStagedActions,
    getStagedCandidateAction,
    applyBatch,
    loadExplorer,
    navigateExplorerParent,
    selectExplorerItem,
    wizardOpen,
    providers,
    providersLoading,
    selectedProvider,
    fetchProviders,
    openWizard,
    closeWizard,
    createRemote,
    clearCache,
  }
})
