import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { BrowseItem, BrowseResponse } from '../types/models'

export const useExplorerStore = defineStore('explorer', () => {
  const currentRemote = ref<string>('local')
  const currentPath = ref<string>('')
  const items = ref<BrowseItem[]>([])
  const hasCanary = ref<boolean>(false)
  const loading = ref<boolean>(false)
  const error = ref<string | null>(null)
  const selectedItem = ref<BrowseItem | null>(null)

  async function browse(remote: string, path: string = '', profileId?: string) {
    loading.value = true
    error.value = null
    currentRemote.value = remote
    currentPath.value = path
    selectedItem.value = null

    try {
      const res = await fetch('/api/browse', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          remote,
          path,
          profile_id: profileId,
        }),
      })
      if (!res.ok) {
        throw new Error(`Failed to browse path: HTTP ${res.status}`)
      }
      const data: BrowseResponse = await res.json()
      items.value = Array.isArray(data.items) ? (data.items as BrowseItem[]) : []
      hasCanary.value = !!data.has_canary
    } catch (e: any) {
      error.value = e.message || 'Error browsing filesystem'
      items.value = []
    } finally {
      loading.value = false
    }
  }

  function selectItem(item: BrowseItem | null) {
    selectedItem.value = item
  }

  return {
    currentRemote,
    currentPath,
    items,
    hasCanary,
    loading,
    error,
    selectedItem,
    browse,
    selectItem,
  }
})
