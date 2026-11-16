import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '../api/client'
import type { BrowseItem, BrowseResponse } from '../types/models'
import type {
  FilePreviewResponse,
  CompareItemRequest,
  FileCompareResponse,
  MoveRequest,
  MoveResponse,
  MkdirRequest,
  MkdirResponse,
  DeleteRequest,
  DeleteResponse,
} from '../types/explorer'

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
      const data = await api.post<BrowseResponse>('/api/browse', {
        remote,
        path,
        profile_id: profileId,
      })
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

  async function previewFile(remote: string, path: string, profileId?: string): Promise<FilePreviewResponse> {
    const params = new URLSearchParams({
      remote,
      path,
      ...(profileId ? { profile_id: profileId } : {}),
    })
    return await api.get<FilePreviewResponse>(`/api/explorer/preview?${params.toString()}`)
  }

  async function compareFiles(compareItems: CompareItemRequest[]): Promise<FileCompareResponse> {
    return await api.post<FileCompareResponse>('/api/browse/compare', {
      items: compareItems,
    })
  }

  async function createDirectory(remote: string, path: string, profileId?: string): Promise<MkdirResponse> {
    const req: MkdirRequest = { remote, path, profile_id: profileId }
    return await api.post<MkdirResponse>('/api/operations/mkdir', req)
  }

  async function moveItem(req: MoveRequest): Promise<MoveResponse> {
    return await api.post<MoveResponse>('/api/operations/move', req)
  }

  async function deleteItem(remote: string, path: string, isDir: boolean, profileId?: string): Promise<DeleteResponse> {
    const req: DeleteRequest = { remote, path, is_dir: isDir, profile_id: profileId }
    return await api.post<DeleteResponse>('/api/browse/delete', req)
  }

  function getDownloadUrl(remote: string, path: string, profileId?: string, inline = false): string {
    const params = new URLSearchParams({
      remote,
      path,
      ...(profileId ? { profile_id: profileId } : {}),
      ...(inline ? { inline: 'true' } : {}),
    })
    return `/api/explorer/download?${params.toString()}`
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
    previewFile,
    compareFiles,
    createDirectory,
    moveItem,
    deleteItem,
    getDownloadUrl,
  }
})
