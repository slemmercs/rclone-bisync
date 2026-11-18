<template>
  <div class="space-y-4">
    <!-- Top Explorer Toolbar -->
    <ExplorerToolbar
      :isSwapped="isSwapped"
      :loading="leftLoading || rightLoading"
      :activeProfileId="activeProfile?.id"
      @swapPanes="swapPanes"
      @refreshAll="refreshAll"
      @openCompare="openCompareModal"
    />

    <!-- Dual Explorer Splitter -->
    <div class="bg-carbon-900 border border-carbon-700 rounded-2xl overflow-hidden shadow-sm min-h-[580px]">
      <Splitter class="min-h-[580px] bg-carbon-900 text-carbon-100 border-none">
        <!-- Left Pane -->
        <SplitterPanel :size="50" :minSize="30" class="border-r border-carbon-700 relative">
          <ExplorerPane
            pane="left"
            :isLocal="leftIsLocal"
            :remote="leftRemote"
            :path="leftPath"
            :items="leftItems"
            :loading="leftLoading"
            :canary="leftCanary"
            :selectedItem="leftSelected"
            :uploading="leftUploading"
            :isRemoteLocked="isRemoteLocked"
            :remotes="remotesStore.remotes"
            :activeProfile="activeProfile"
            @update:remote="handleRemoteUpdate"
            @navigate="p => handleNavigate('left', p)"
            @refresh="() => refreshPane('left')"
            @touchCanary="() => touchCanary('left')"
            @mkdir="() => openMkdir('left')"
            @upDir="() => upDirectory('left')"
            @uploadFiles="files => handleUploadFiles('left', files)"
            @itemClick="item => handleItemClick('left', item)"
            @itemDblClick="item => handleItemDblClick('left', item)"
            @contextmenu="(e, item) => handleContextMenu(e, 'left', item)"
            @toggleCompare="item => toggleCompare('left', item)"
            @nameClick="item => handleNameClick('left', item)"
            @openOnRemote="item => openOnRemoteDirect(leftRemote, getRelativeItemPath('left', item))"
          />
        </SplitterPanel>

        <!-- Right Pane -->
        <SplitterPanel :size="50" :minSize="30" class="relative">
          <ExplorerPane
            pane="right"
            :isLocal="rightIsLocal"
            :remote="rightRemote"
            :path="rightPath"
            :items="rightItems"
            :loading="rightLoading"
            :canary="rightCanary"
            :selectedItem="rightSelected"
            :uploading="rightUploading"
            :isRemoteLocked="isRemoteLocked"
            :remotes="remotesStore.remotes"
            :activeProfile="activeProfile"
            @update:remote="handleRemoteUpdate"
            @navigate="p => handleNavigate('right', p)"
            @refresh="() => refreshPane('right')"
            @touchCanary="() => touchCanary('right')"
            @mkdir="() => openMkdir('right')"
            @upDir="() => upDirectory('right')"
            @uploadFiles="files => handleUploadFiles('right', files)"
            @itemClick="item => handleItemClick('right', item)"
            @itemDblClick="item => handleItemDblClick('right', item)"
            @contextmenu="(e, item) => handleContextMenu(e, 'right', item)"
            @toggleCompare="item => toggleCompare('right', item)"
            @nameClick="item => handleNameClick('right', item)"
            @openOnRemote="item => openOnRemoteDirect(rightRemote, getRelativeItemPath('right', item))"
          />
        </SplitterPanel>
      </Splitter>
    </div>

    <!-- Modals & Context Menus -->
    <ModalFilePreview
      v-model:visible="showPreviewModal"
      :preview="previewData"
      :loading="previewLoading"
      :error="previewError"
      :profileId="activeProfile?.id"
    />

    <ModalCompare
      v-model:visible="showCompareModal"
      :comparison="compareData"
      :loading="compareLoading"
      :error="compareError"
    />

    <ModalExplorerMkdir
      v-model:visible="showMkdirModal"
      :remote="mkdirTargetRemote"
      :currentPath="mkdirTargetRemotePath"
      :loading="mkdirLoading"
      :error="mkdirError"
      @create="handleCreateDirectory"
    />

    <ModalExplorerMove
      v-model:visible="showMoveModal"
      :remote="moveTargetRemote"
      :srcPath="moveSrcPath"
      :isDir="moveIsDir"
      :loading="moveLoading"
      :error="moveError"
      @move="handleExecuteMove"
    />

    <ModalConfirm
      v-model:visible="showConfirmModal"
      :title="confirmTitle"
      :message="confirmMessage"
      :target="confirmTarget"
      :loading="confirmLoading"
      @confirm="handleExecuteDelete"
    />

    <ExplorerContextMenu
      :visible="contextMenuVisible"
      :x="contextMenuX"
      :y="contextMenuY"
      :item="contextMenuItem"
      :remote="contextMenuRemote"
      @close="contextMenuVisible = false"
      @action="handleContextAction"
    />

    <ModalUploadDock />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import Splitter from 'primevue/splitter'
import SplitterPanel from 'primevue/splitterpanel'
import { useProfilesStore } from '../stores/profiles'
import { useExplorerStore } from '../stores/explorer'
import { useSettingsStore } from '../stores/settings'
import { useRemotesStore } from '../stores/remotes'
import type { BrowseItem } from '../types/models'
import { isItemPreviewable, type FilePreviewResponse, type FileCompareResponse } from '../types/explorer'
import ExplorerToolbar from '../components/explorer/ExplorerToolbar.vue'
import ExplorerPane from '../components/explorer/ExplorerPane.vue'
import ModalFilePreview from '../components/ModalFilePreview.vue'
import ModalCompare from '../components/ModalCompare.vue'
import ModalExplorerMkdir from '../components/ModalExplorerMkdir.vue'
import ModalExplorerMove from '../components/ModalExplorerMove.vue'
import ModalConfirm from '../components/ModalConfirm.vue'
import ExplorerContextMenu from '../components/ExplorerContextMenu.vue'
import ModalUploadDock from '../components/ModalUploadDock.vue'

const profilesStore = useProfilesStore()
const explorerStore = useExplorerStore()
const settingsStore = useSettingsStore()
const remotesStore = useRemotesStore()

const activeProfile = computed(() => profilesStore.activeProfile)
const isRemoteLocked = computed(() => !!settingsStore.globalDefaults.lock_explorer_remote)

// Pane Swap & Navigation State
const isSwapped = ref(false)
const leftPath = ref('')
const leftItems = ref<BrowseItem[]>([])
const leftLoading = ref(false)
const leftCanary = ref(false)
const leftSelected = ref<BrowseItem | null>(null)
const leftUploading = ref(false)

const rightRemoteName = ref('')
const rightPath = ref('')
const rightItems = ref<BrowseItem[]>([])
const rightLoading = ref(false)
const rightCanary = ref(false)
const rightSelected = ref<BrowseItem | null>(null)
const rightUploading = ref(false)

const leftIsLocal = computed(() => !isSwapped.value)
const rightIsLocal = computed(() => isSwapped.value)
const leftRemote = computed(() => (isSwapped.value ? rightRemoteName.value : 'local'))
const rightRemote = computed(() => (isSwapped.value ? 'local' : rightRemoteName.value))

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

onMounted(() => {
  remotesStore.fetchRemotes()
  settingsStore.fetchConfig()
})

watch(
  () => activeProfile.value,
  (profile) => {
    if (profile) {
      leftPath.value = ''
      rightRemoteName.value = (profile.remote || 'local').replace(/:$/, '')
      rightPath.value = (profile.remote_path || '').replace(/^\/+/, '')
      refreshAll()
      explorerStore.fetchUploadDockStatus(profile.id)
    }
  },
  { immediate: true }
)

watch(
  () => explorerStore.targetJumpPath,
  async (target) => {
    if (!target) return
    leftPath.value = target.dir
    rightPath.value = target.dir
    await Promise.allSettled([refreshPane('left'), refreshPane('right')])
  }
)

function swapPanes() {
  isSwapped.value = !isSwapped.value
  const tempPath = leftPath.value
  leftPath.value = rightPath.value
  rightPath.value = tempPath

  const tempItems = leftItems.value
  leftItems.value = rightItems.value
  rightItems.value = tempItems

  const tempCanary = leftCanary.value
  leftCanary.value = rightCanary.value
  rightCanary.value = tempCanary
}

async function refreshPane(pane: 'left' | 'right') {
  const remote = pane === 'left' ? leftRemote.value : rightRemote.value
  const path = pane === 'left' ? leftPath.value : rightPath.value

  if (pane === 'left') {
    leftLoading.value = true
    leftSelected.value = null
  } else {
    rightLoading.value = true
    rightSelected.value = null
  }

  try {
    await explorerStore.browse(remote, path, activeProfile.value?.id)
    if (pane === 'left') {
      leftItems.value = explorerStore.items
      if (!path) leftCanary.value = explorerStore.hasCanary
    } else {
      rightItems.value = explorerStore.items
      if (!path) rightCanary.value = explorerStore.hasCanary
    }
  } catch {
    if (pane === 'left') leftItems.value = []
    else rightItems.value = []
  } finally {
    if (pane === 'left') leftLoading.value = false
    else rightLoading.value = false
  }
}

async function refreshAll() {
  await Promise.all([refreshPane('left'), refreshPane('right')])
}

function handleRemoteUpdate(newRemote: string) {
  rightRemoteName.value = newRemote.replace(/:$/, '')
  rightPath.value = ''
  const remotePane = isSwapped.value ? 'left' : 'right'
  refreshPane(remotePane)
}

function handleNavigate(pane: 'left' | 'right', path: string) {
  if (pane === 'left') {
    leftPath.value = path
    refreshPane('left')
  } else {
    rightPath.value = path
    refreshPane('right')
  }
}

function getRelativeItemPath(pane: 'left' | 'right', item: BrowseItem): string {
  const currentPath = pane === 'left' ? leftPath.value : rightPath.value
  return currentPath ? `${currentPath.replace(/\/+$/, '')}/${item.Name}` : item.Name
}

function handleItemClick(pane: 'left' | 'right', item: BrowseItem) {
  if (item.IsDir) {
    handleNavigate(pane, getRelativeItemPath(pane, item))
  } else {
    if (pane === 'left') leftSelected.value = item
    else rightSelected.value = item
  }
}

function handleNameClick(pane: 'left' | 'right', item: BrowseItem) {
  if (item.IsDir) {
    handleItemClick(pane, item)
    return
  }
  if (pane === 'left') leftSelected.value = item
  else rightSelected.value = item
  if (isItemPreviewable(item)) {
    triggerPreview(pane === 'left' ? leftRemote.value : rightRemote.value, getRelativeItemPath(pane, item), item.Name)
  }
}

function handleItemDblClick(pane: 'left' | 'right', item: BrowseItem) {
  if (item.IsDir) {
    handleItemClick(pane, item)
  } else if (isItemPreviewable(item)) {
    triggerPreview(pane === 'left' ? leftRemote.value : rightRemote.value, getRelativeItemPath(pane, item), item.Name)
  }
}

function upDirectory(pane: 'left' | 'right') {
  const current = pane === 'left' ? leftPath.value : rightPath.value
  const parts = current.replace(/\/+$/, '').split('/').filter(Boolean)
  if (parts.length > 0) {
    parts.pop()
    handleNavigate(pane, parts.join('/'))
  }
}

async function touchCanary(pane: 'left' | 'right') {
  const remote = pane === 'left' ? leftRemote.value : rightRemote.value
  try {
    await explorerStore.touchCanary(remote, '', activeProfile.value?.id)
    if (pane === 'left') leftCanary.value = true
    else rightCanary.value = true
    refreshPane(pane)
  } catch (e: any) {
    alert(e.message || 'Failed to touch canary')
  }
}

async function handleUploadFiles(pane: 'left' | 'right', files: File[]) {
  const remote = pane === 'left' ? leftRemote.value : rightRemote.value
  const path = pane === 'left' ? leftPath.value : rightPath.value

  if (pane === 'left') leftUploading.value = true
  else rightUploading.value = true

  try {
    for (const file of files) {
      await explorerStore.uploadFile(file, remote, path, activeProfile.value?.id)
    }
    refreshPane(pane)
    explorerStore.fetchUploadDockStatus(activeProfile.value?.id)
  } catch (err: any) {
    alert(err.message || 'File upload failed')
  } finally {
    if (pane === 'left') leftUploading.value = false
    else rightUploading.value = false
  }
}

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

  if (pane === 'left') leftSelected.value = item
  else rightSelected.value = item
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
</script>
