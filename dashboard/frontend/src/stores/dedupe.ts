import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api } from '../api/client'
import type {
  DedupeMode,
  DedupeBatchAction,
  DedupeBatchRequest,
  DedupeRequest,
  DedupeResponse,
  ProviderCatalogItem,
  ProviderCatalogResponse,
  CreateRemotePayload,
} from '../types/dedupe'
import { parseDedupeOutput, type ParsedDedupeOutput } from '../utils/dedupeParser'

export const useDedupeStore = defineStore('dedupe', () => {
  // Modal state
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
  const error = ref<string | null>(null)

  // Staged batch actions
  const stagedActions = ref<Record<string, DedupeBatchAction>>({})
  const triggerResync = ref<boolean>(false)
  const drawerOpen = ref<boolean>(false)

  // Remote creation wizard state
  const wizardOpen = ref<boolean>(false)
  const providers = ref<ProviderCatalogItem[]>([])
  const providersLoading = ref<boolean>(false)
  const selectedProvider = ref<ProviderCatalogItem | null>(null)

  const stagedCount = computed(() => Object.keys(stagedActions.value).length)
  const hasCollisions = computed(() => parsed.value.totalCollisions > 0)

  function openModal(remoteName: string, subpath: string = '', initialMode: DedupeMode = 'list') {
    remote.value = remoteName.replace(/:+$/, '')
    path.value = subpath
    mode.value = initialMode
    rawOutput.value = ''
    parsed.value = parseDedupeOutput('')
    runId.value = null
    error.value = null
    stagedActions.value = {}
    drawerOpen.value = false
    modalOpen.value = true
  }

  function closeModal() {
    if (executing.value) return
    modalOpen.value = false
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
    } catch (e: any) {
      error.value = e.message || 'Deduplication request failed'
      rawOutput.value = (rawOutput.value ? rawOutput.value + '\n' : '') + `Request error: ${error.value}`
      parsed.value = parseDedupeOutput(rawOutput.value)
    } finally {
      loading.value = false
      executing.value = false
    }
  }

  function stageAction(filePath: string, action: 'keep' | 'delete' | 'rename', newPath?: string) {
    stagedActions.value[filePath] = { action, path: filePath, new_path: newPath }
    if (!drawerOpen.value) drawerOpen.value = true
  }

  function unstageAction(filePath: string) {
    delete stagedActions.value[filePath]
  }

  function clearStagedActions() {
    stagedActions.value = {}
    drawerOpen.value = false
  }

  async function applyBatch() {
    if (!remote.value || stagedCount.value === 0) return
    loading.value = true
    error.value = null

    const payload: DedupeBatchRequest = {
      remote: remote.value,
      actions: Object.values(stagedActions.value),
      trigger_resync: triggerResync.value,
    }

    try {
      const res = await api.post<any>('/api/remotes/dedupe/batch', payload)
      if (res.status === 'error') {
        error.value = res.error || (res.errors && res.errors[0]) || 'Batch apply failed'
      } else {
        clearStagedActions()
        await runDedupe(true)
      }
    } catch (e: any) {
      error.value = e.message || 'Batch execution failed'
    } finally {
      loading.value = false
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
        const found = providers.value.find(p => p.name === defaultProvider || p.prefix === defaultProvider)
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
    error,
    stagedActions,
    triggerResync,
    drawerOpen,
    stagedCount,
    hasCollisions,
    openModal,
    closeModal,
    runDedupe,
    stageAction,
    unstageAction,
    clearStagedActions,
    applyBatch,
    wizardOpen,
    providers,
    providersLoading,
    selectedProvider,
    fetchProviders,
    openWizard,
    closeWizard,
    createRemote,
  }
})
