import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api } from '../api/client'
import { helpPolicies, helpRunbooks, type HelpPolicy, type HelpRunbook } from '../data/help-policies'
import { helpFlags, type HelpFlag } from '../data/help-flags'
import type { DiagnosticsResponse, DiagnosticCheck } from '../types/models'

export type HelpTab = 'diagnostics' | 'safety' | 'manual' | 'runbooks'

export const useHelpStore = defineStore('help', () => {
  const isOpen = ref<boolean>(false)
  const activeTab = ref<HelpTab>('diagnostics')
  const searchQuery = ref<string>('')
  const contextId = ref<string>('global')
  const contextTitle = ref<string>('Fleet Overview')
  const contextOnly = ref<boolean>(false)

  const diagnostics = ref<DiagnosticsResponse | null>(null)
  const loading = ref<boolean>(false)
  const error = ref<string | null>(null)

  function openDrawer(tab?: HelpTab, searchTopic: string = '', activeContext?: { id: string; title: string }) {
    if (activeContext) {
      contextId.value = activeContext.id
      contextTitle.value = activeContext.title
      contextOnly.value = activeContext.id !== 'global'
    }
    if (tab) {
      activeTab.value = tab
    }
    searchQuery.value = searchTopic
    isOpen.value = true

    if (activeTab.value === 'diagnostics' && !diagnostics.value && !loading.value) {
      fetchDiagnostics()
    }
  }

  function closeDrawer() {
    isOpen.value = false
  }

  function setTab(tab: HelpTab) {
    activeTab.value = tab
    if (tab === 'diagnostics' && !diagnostics.value && !loading.value) {
      fetchDiagnostics()
    }
  }

  function toggleContextOnly() {
    contextOnly.value = !contextOnly.value
  }

  async function fetchDiagnostics() {
    loading.value = true
    error.value = null
    try {
      const data = await api.get<DiagnosticsResponse>('/api/diagnostics')
      diagnostics.value = data
    } catch (e: any) {
      error.value = e.message || 'Failed to fetch diagnostics'
    } finally {
      loading.value = false
    }
  }

  const filteredPolicies = computed<HelpPolicy[]>(() => {
    let list = helpPolicies
    if (contextOnly.value && contextId.value !== 'global') {
      list = list.filter(p => p.contexts && p.contexts.includes(contextId.value))
    }
    const q = searchQuery.value.toLowerCase().trim()
    if (!q) return list
    return list.filter(p =>
      p.title.toLowerCase().includes(q) ||
      p.tag.toLowerCase().includes(q) ||
      p.description.toLowerCase().includes(q) ||
      p.action.toLowerCase().includes(q)
    )
  })

  const filteredFlags = computed<HelpFlag[]>(() => {
    let list = helpFlags
    if (contextOnly.value && contextId.value !== 'global') {
      list = list.filter(f => f.contexts && f.contexts.includes(contextId.value))
    }
    const q = searchQuery.value.toLowerCase().trim()
    if (!q) return list
    return list.filter(f =>
      f.flag.toLowerCase().includes(q) ||
      f.name.toLowerCase().includes(q) ||
      f.category.toLowerCase().includes(q) ||
      f.description.toLowerCase().includes(q) ||
      (f.caveat && f.caveat.toLowerCase().includes(q))
    )
  })

  const filteredRunbooks = computed<HelpRunbook[]>(() => {
    let list = helpRunbooks
    if (contextOnly.value && contextId.value !== 'global') {
      list = list.filter(r => r.contexts && r.contexts.includes(contextId.value))
    }
    const q = searchQuery.value.toLowerCase().trim()
    if (!q) return list
    return list.filter(r =>
      r.title.toLowerCase().includes(q) ||
      r.category.toLowerCase().includes(q) ||
      r.summary.toLowerCase().includes(q) ||
      r.steps.some(s => s.toLowerCase().includes(q))
    )
  })

  const filteredChecks = computed<DiagnosticCheck[]>(() => {
    const checks = diagnostics.value?.checks || []
    let list = checks
    if (contextOnly.value && contextId.value !== 'global') {
      const ctxId = contextId.value
      list = checks.filter(c => {
        if (ctxId === 'view_remotes' && c.category === 'remotes') return true
        if ((ctxId === 'view_explorer' || ctxId === 'modal_profiles') && (c.category === 'storage' || c.category === 'safety')) return true
        if (ctxId === 'view_telemetry' && (c.category === 'daemon' || c.category === 'system')) return true
        return false
      })
      if (list.length === 0) list = checks
    }
    const q = searchQuery.value.toLowerCase().trim()
    if (!q) return list
    return list.filter(c =>
      c.name.toLowerCase().includes(q) ||
      c.message.toLowerCase().includes(q) ||
      c.category.toLowerCase().includes(q) ||
      (c.detail && c.detail.toLowerCase().includes(q))
    )
  })

  return {
    isOpen,
    activeTab,
    searchQuery,
    contextId,
    contextTitle,
    contextOnly,
    diagnostics,
    loading,
    error,
    openDrawer,
    closeDrawer,
    setTab,
    toggleContextOnly,
    fetchDiagnostics,
    filteredPolicies,
    filteredFlags,
    filteredRunbooks,
    filteredChecks,
  }
})
