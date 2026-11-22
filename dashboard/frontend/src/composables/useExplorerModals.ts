import { ref, type Ref, type ComputedRef } from 'vue'
import { useExplorerStore } from '../stores/explorer'
import { useToastNotification } from './useToastNotification'
import type { BrowseItem, Profile } from '../types/models'
import { isItemPreviewable, type FilePreviewResponse, type FileCompareResponse, type ExplorerDragPayload } from '../types/explorer'

export interface UseExplorerModalsOptions {
  activeProfile: ComputedRef<Profile | null | undefined>
  refreshAll: () => Promise<void>
  leftRemote: ComputedRef<string>
  rightRemote: ComputedRef<string>
  leftPath: Ref<string>
  rightPath: Ref<string>
  getRelativeItemPath: (pane: 'left' | 'right', item: BrowseItem) => string
  setSelectedItem: (pane: 'left' | 'right', item: BrowseItem) => void
}

export function useExplorerModals(options: UseExplorerModalsOptions) {
  const explorerStore = useExplorerStore()
  const toast = useToastNotification()

  const {
    activeProfile,
    refreshAll,
    leftRemote,
    rightRemote,
    leftPath,
    rightPath,
    getRelativeItemPath,
    setSelectedItem,
  } = options

  // Modals State
  const showPreviewModal = ref(false)
  const previewData = ref<FilePreviewResponse | null>(null)
  const previewLoading = ref(false)
  const previewError = ref<string | null>(null)

  const showCompareModal = ref(false)
  const compareData = ref<FileCompareResponse | null>(null)
  const compareLoading = ref(false)
  const compareError = ref<string | null>(null)

  const showMkdirModal = ref(false)
  const mkdirTargetRemote = ref('local')
  const mkdirTargetRemotePath = ref('')
  const mkdirLoading = ref(false)
  const mkdirError = ref<string | null>(null)

  const showMoveModal = ref(false)
  const moveTargetRemote = ref('local')
  const moveSrcPath = ref('')
  const moveIsDir = ref(false)
  const moveLoading = ref(false)
  const moveError = ref<string | null>(null)

  // Cross-Target Transfer Modal State
  const showTransferModal = ref(false)
  const transferSourceRemote = ref('local')
  const transferSourcePath = ref('')
  const transferDestRemote = ref('local')
  const transferDestPath = ref('')
  const transferItemName = ref('')
  const transferIsDir = ref(false)
  const transferLoading = ref(false)
  const transferError = ref<string | null>(null)

  const showConfirmModal = ref(false)
  const confirmTitle = ref('')
  const confirmMessage = ref('')
  const confirmTarget = ref('')
  const deleteRemote = ref('local')
  const deletePath = ref('')
  const deleteIsDir = ref(false)
  const confirmLoading = ref(false)

  // Context Menu State
  const contextMenuVisible = ref(false)
  const contextMenuX = ref(0)
  const contextMenuY = ref(0)
  const contextMenuItem = ref<BrowseItem | null>(null)
  const contextMenuRemote = ref('local')

  function toggleCompare(pane: 'left' | 'right', item: BrowseItem) {
    const remote = pane === 'left' ? leftRemote.value : rightRemote.value
    const path = getRelativeItemPath(pane, item)
    explorerStore.toggleCompareItem({
      remote,
      path,
      name: item.Name,
      size: item.Size || 0,
      pane,
    })
  }

  async function openCompareModal() {
    if (explorerStore.compareSelection.length < 2) return
    showCompareModal.value = true
    compareLoading.value = true
    compareError.value = null
    compareData.value = null

    try {
      const items = explorerStore.compareSelection.map((item) => ({
        remote: item.remote,
        path: item.path,
        profile_id: activeProfile.value?.id,
      }))
      const data = await explorerStore.compareFiles(items)
      compareData.value = data
    } catch (err: any) {
      compareError.value = err.message || 'Comparison failed'
    } finally {
      compareLoading.value = false
    }
  }

  async function triggerPreview(remote: string, path: string, name: string) {
    showPreviewModal.value = true
    previewLoading.value = true
    previewError.value = null
    previewData.value = null

    try {
      const data = await explorerStore.previewFile(remote, path, activeProfile.value?.id)
      previewData.value = data
    } catch (err: any) {
      previewError.value = err.message || `Failed to preview ${name}`
    } finally {
      previewLoading.value = false
    }
  }

  async function openOnRemoteDirect(remote: string, path: string) {
    try {
      const data = await explorerStore.getPublicLink(remote, path, activeProfile.value?.id)
      if (data && data.supported && data.url) {
        window.open(data.url, '_blank')
      } else {
        alert('Cloud web links are unsupported for this storage provider.')
      }
    } catch (err: any) {
      alert(err.message || 'Failed to generate cloud link')
    }
  }

  function openMkdir(pane: 'left' | 'right') {
    mkdirTargetRemote.value = pane === 'left' ? leftRemote.value : rightRemote.value
    mkdirTargetRemotePath.value = pane === 'left' ? leftPath.value : rightPath.value
    mkdirError.value = null
    showMkdirModal.value = true
  }

  async function handleCreateDirectory(dirName: string) {
    const base = mkdirTargetRemotePath.value
    const fullPath = base ? `${base.replace(/\/+$/, '')}/${dirName}` : dirName
    mkdirLoading.value = true
    mkdirError.value = null

    try {
      await explorerStore.createDirectory(mkdirTargetRemote.value, fullPath, activeProfile.value?.id)
      showMkdirModal.value = false
      refreshAll()
    } catch (e: any) {
      mkdirError.value = e.message || 'Failed to create directory'
    } finally {
      mkdirLoading.value = false
    }
  }

  function handleContextMenu(e: MouseEvent, pane: 'left' | 'right', item: BrowseItem) {
    contextMenuX.value = e.clientX
    contextMenuY.value = e.clientY
    contextMenuItem.value = item
    contextMenuRemote.value = pane === 'left' ? leftRemote.value : rightRemote.value
    contextMenuVisible.value = true
    setSelectedItem(pane, item)
  }

  function handleContextAction(act: 'preview' | 'compare' | 'download' | 'open_on_remote' | 'move' | 'copy_path' | 'delete') {
    if (!contextMenuItem.value) return
    const item = contextMenuItem.value
    const remote = contextMenuRemote.value
    const pane = remote === leftRemote.value ? 'left' : 'right'
    const fullPath = getRelativeItemPath(pane, item)

    switch (act) {
      case 'preview':
        if (isItemPreviewable(item)) {
          triggerPreview(remote, fullPath, item.Name)
        }
        break
      case 'compare':
        toggleCompare(pane, item)
        break
      case 'download': {
        const url = explorerStore.getDownloadUrl(remote, fullPath, activeProfile.value?.id)
        const a = document.createElement('a')
        a.href = url
        a.download = item.Name
        document.body.appendChild(a)
        a.click()
        document.body.removeChild(a)
        break
      }
      case 'open_on_remote':
        openOnRemoteDirect(remote, fullPath)
        break
      case 'move':
        moveTargetRemote.value = remote
        moveSrcPath.value = fullPath
        moveIsDir.value = item.IsDir
        moveError.value = null
        showMoveModal.value = true
        break
      case 'copy_path':
        navigator.clipboard.writeText(fullPath)
        break
      case 'delete':
        confirmTitle.value = item.IsDir ? 'Confirm Directory Purge' : 'Confirm File Deletion'
        confirmMessage.value = `Are you sure you want to permanently delete "${item.Name}"?`
        confirmTarget.value = `${remote}:${fullPath}`
        deleteRemote.value = remote
        deletePath.value = fullPath
        deleteIsDir.value = item.IsDir
        showConfirmModal.value = true
        break
    }
  }

  async function handleExecuteMove(dstPath: string) {
    moveLoading.value = true
    moveError.value = null
    try {
      await explorerStore.moveItem({
        src_remote: moveTargetRemote.value,
        src_path: moveSrcPath.value,
        dst_remote: moveTargetRemote.value,
        dst_path: dstPath,
        is_dir: moveIsDir.value,
        profile_id: activeProfile.value?.id,
      })
      showMoveModal.value = false
      refreshAll()
    } catch (e: any) {
      moveError.value = e.message || 'Move failed'
    } finally {
      moveLoading.value = false
    }
  }

  async function handleExecuteDelete() {
    confirmLoading.value = true
    try {
      await explorerStore.deleteItem(deleteRemote.value, deletePath.value, deleteIsDir.value, activeProfile.value?.id)
      showConfirmModal.value = false
      refreshAll()
    } catch (e: any) {
      alert(e.message || 'Deletion failed')
    } finally {
      confirmLoading.value = false
    }
  }

  function handleCrossPaneDrop(payload: ExplorerDragPayload, targetPaneKey: 'left' | 'right') {
    const dstRemote = targetPaneKey === 'left' ? leftRemote.value : rightRemote.value
    const dstPath = targetPaneKey === 'left' ? leftPath.value : rightPath.value

    transferSourceRemote.value = payload.sourceRemote
    transferSourcePath.value = payload.sourcePath
    transferDestRemote.value = dstRemote
    transferDestPath.value = dstPath
    transferItemName.value = payload.item.Name
    transferIsDir.value = payload.item.IsDir
    transferError.value = null
    showTransferModal.value = true
  }

  async function handleExecuteTransfer(action: 'copy' | 'move', dstPath: string) {
    transferLoading.value = true
    transferError.value = null

    try {
      if (action === 'copy') {
        const res = await explorerStore.copyItem({
          src_remote: transferSourceRemote.value,
          src_path: transferSourcePath.value,
          dst_remote: transferDestRemote.value,
          dst_path: dstPath,
          is_dir: transferIsDir.value,
          profile_id: activeProfile.value?.id,
        })
        if (res && res.status === 'error') {
          throw new Error(res.error || 'Copy failed')
        }
        toast.success(`Copied "${transferItemName.value}" successfully to ${transferDestRemote.value}:${dstPath}`, 'File Transfer')
      } else {
        const res = await explorerStore.moveItem({
          src_remote: transferSourceRemote.value,
          src_path: transferSourcePath.value,
          dst_remote: transferDestRemote.value,
          dst_path: dstPath,
          is_dir: transferIsDir.value,
          profile_id: activeProfile.value?.id,
        })
        if (res && res.status === 'error') {
          throw new Error(res.error || 'Move failed')
        }
        toast.success(`Moved "${transferItemName.value}" successfully to ${transferDestRemote.value}:${dstPath}`, 'File Transfer')
      }
      showTransferModal.value = false
      refreshAll()
    } catch (err: any) {
      transferError.value = err.message || 'Transfer failed'
    } finally {
      transferLoading.value = false
    }
  }

  return {
    showPreviewModal,
    previewData,
    previewLoading,
    previewError,
    showCompareModal,
    compareData,
    compareLoading,
    compareError,
    showMkdirModal,
    mkdirTargetRemote,
    mkdirTargetRemotePath,
    mkdirLoading,
    mkdirError,
    showMoveModal,
    moveTargetRemote,
    moveSrcPath,
    moveIsDir,
    moveLoading,
    moveError,
    showTransferModal,
    transferSourceRemote,
    transferSourcePath,
    transferDestRemote,
    transferDestPath,
    transferItemName,
    transferIsDir,
    transferLoading,
    transferError,
    showConfirmModal,
    confirmTitle,
    confirmMessage,
    confirmTarget,
    deleteRemote,
    deletePath,
    deleteIsDir,
    confirmLoading,
    contextMenuVisible,
    contextMenuX,
    contextMenuY,
    contextMenuItem,
    contextMenuRemote,
    toggleCompare,
    openCompareModal,
    triggerPreview,
    openOnRemoteDirect,
    openMkdir,
    handleCreateDirectory,
    handleContextMenu,
    handleContextAction,
    handleExecuteMove,
    handleExecuteDelete,
    handleCrossPaneDrop,
    handleExecuteTransfer,
  }
}
