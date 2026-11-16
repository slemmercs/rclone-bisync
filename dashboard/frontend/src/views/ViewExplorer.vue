<template>
  <div class="space-y-4">
    <!-- Header Toolbar -->
    <div class="flex flex-wrap items-center justify-between gap-3 bg-carbon-surface p-4 rounded-2xl border border-carbon-border">
      <div>
        <h2 class="text-base font-semibold text-white tracking-tight flex items-center gap-2">
          <span>Dual-Pane Storage Explorer</span>
          <span v-if="activeProfile" class="text-xs font-mono px-2 py-0.5 rounded bg-carbon-elevated text-electric-blue border border-carbon-border">
            {{ activeProfile.name }}
          </span>
        </h2>
        <p class="text-xs text-slate-400">Direct side-by-side inspection, file previewing, and cross-backend comparison with path confinement.</p>
      </div>

      <div class="flex items-center gap-2">
        <button
          @click="openCompareSelected"
          :disabled="!localSelected || !remoteSelected"
          class="px-3 py-1.5 bg-carbon-elevated hover:bg-slate-800 text-electric-blue border border-carbon-border rounded-xl text-xs font-semibold transition-colors flex items-center gap-1.5 cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed"
          title="Compare selected local file against selected remote file"
        >
          <i class="pi pi-sliders-v text-xs"></i>
          <span>Compare Selected</span>
        </button>
      </div>
    </div>

    <!-- Dual Explorer Splitter -->
    <div class="bg-carbon-surface border border-carbon-border rounded-2xl overflow-hidden shadow-sm min-h-[560px]">
      <Splitter class="min-h-[560px] bg-carbon-surface text-slate-100 border-none">
        <!-- Local Storage Panel -->
        <SplitterPanel :size="50" :minSize="30" class="p-4 flex flex-col border-r border-carbon-border">
          <div class="flex items-center justify-between gap-2 mb-2 pb-2 border-b border-carbon-border">
            <div class="flex items-center gap-2">
              <i class="pi pi-desktop text-slate-400"></i>
              <span class="text-xs font-bold text-white uppercase tracking-wide">Local Storage</span>
              <span
                class="text-[10px] font-mono px-2 py-0.5 rounded flex items-center gap-1"
                :class="localCanary ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/30' : 'bg-carbon-elevated text-slate-500'"
              >
                <i :class="localCanary ? 'pi pi-check' : 'pi pi-info-circle'" class="text-[9px]"></i>
                {{ localCanary ? 'Canary OK' : 'No Canary' }}
              </span>
            </div>
            <div class="flex items-center gap-1">
              <button @click="openMkdir('local')" class="p-1.5 rounded-lg bg-carbon-elevated text-slate-300 hover:text-white hover:bg-slate-800 transition-colors" title="New Folder">
                <i class="pi pi-folder-plus text-xs"></i>
              </button>
              <button @click="upLocal" class="p-1.5 rounded-lg bg-carbon-elevated text-slate-300 hover:text-white hover:bg-slate-800 transition-colors" title="Up Directory">
                <i class="pi pi-arrow-up text-xs"></i>
              </button>
              <button @click="refreshLocal" class="p-1.5 rounded-lg bg-carbon-elevated text-slate-300 hover:text-white hover:bg-slate-800 transition-colors" title="Refresh">
                <i :class="localLoading ? 'pi pi-spin pi-spinner' : 'pi pi-refresh'" class="text-xs"></i>
              </button>
            </div>
          </div>

          <!-- Path Breadcrumb -->
          <div class="mb-2 bg-carbon-canvas px-3 py-1.5 rounded-xl border border-carbon-border text-xs font-mono text-indigo-300 truncate">
            {{ localPath || '/' }}
          </div>

          <!-- Local Items List -->
          <div class="flex-1 overflow-y-auto max-h-[460px] divide-y divide-carbon-border/40 text-xs font-mono">
            <div v-if="localLoading" class="py-12 text-center text-slate-400">
              <i class="pi pi-spin pi-spinner text-xl text-electric-blue"></i>
            </div>
            <div v-else-if="localItems.length === 0" class="py-12 text-center text-slate-500">
              Directory is empty
            </div>
            <div
              v-else
              v-for="item in localItems"
              :key="item.Name"
              @click="handleItemClick('local', item)"
              @contextmenu.prevent="handleContextMenu($event, 'local', item)"
              class="flex items-center justify-between px-3 py-2 cursor-pointer select-none transition-colors rounded-lg group"
              :class="localSelected?.Name === item.Name ? 'bg-carbon-elevated text-electric-blue font-bold' : 'hover:bg-carbon-elevated/60 text-slate-200'"
            >
              <div class="flex items-center gap-2 truncate">
                <i
                  class="text-xs"
                  :class="item.IsDir ? 'pi pi-folder text-amber-400' : 'pi pi-file text-slate-400'"
                ></i>
                <span class="truncate">{{ item.Name }}</span>
              </div>
              <div class="flex items-center gap-2 text-[11px] text-slate-400 shrink-0">
                <span v-if="!item.IsDir">{{ formatBytes(item.Size) }}</span>
                <span v-else class="text-slate-500">DIR</span>
              </div>
            </div>
          </div>
        </SplitterPanel>

        <!-- Remote Cloud Storage Panel -->
        <SplitterPanel :size="50" :minSize="30" class="p-4 flex flex-col">
          <div class="flex items-center justify-between gap-2 mb-2 pb-2 border-b border-carbon-border">
            <div class="flex items-center gap-2">
              <i class="pi pi-cloud text-slate-400"></i>
              <span class="text-xs font-bold text-white uppercase tracking-wide">Remote Cloud</span>
              <span class="text-[10px] font-mono px-2 py-0.5 rounded uppercase font-semibold bg-carbon-elevated text-electric-blue border border-carbon-border">
                {{ remoteName }}
              </span>
              <span
                class="text-[10px] font-mono px-2 py-0.5 rounded flex items-center gap-1"
                :class="remoteCanary ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/30' : 'bg-carbon-elevated text-slate-500'"
              >
                <i :class="remoteCanary ? 'pi pi-check' : 'pi pi-info-circle'" class="text-[9px]"></i>
                {{ remoteCanary ? 'Canary OK' : 'No Canary' }}
              </span>
            </div>
            <div class="flex items-center gap-1">
              <button @click="openMkdir(remoteName)" class="p-1.5 rounded-lg bg-carbon-elevated text-slate-300 hover:text-white hover:bg-slate-800 transition-colors" title="New Folder">
                <i class="pi pi-folder-plus text-xs"></i>
              </button>
              <button @click="upRemote" class="p-1.5 rounded-lg bg-carbon-elevated text-slate-300 hover:text-white hover:bg-slate-800 transition-colors" title="Up Directory">
                <i class="pi pi-arrow-up text-xs"></i>
              </button>
              <button @click="refreshRemote" class="p-1.5 rounded-lg bg-carbon-elevated text-slate-300 hover:text-white hover:bg-slate-800 transition-colors" title="Refresh">
                <i :class="remoteLoading ? 'pi pi-spin pi-spinner' : 'pi pi-refresh'" class="text-xs"></i>
              </button>
            </div>
          </div>

          <!-- Path Breadcrumb -->
          <div class="mb-2 bg-carbon-canvas px-3 py-1.5 rounded-xl border border-carbon-border text-xs font-mono text-emerald-400 truncate">
            {{ remoteName }}:{{ remotePath || '/' }}
          </div>

          <!-- Remote Items List -->
          <div class="flex-1 overflow-y-auto max-h-[460px] divide-y divide-carbon-border/40 text-xs font-mono">
            <div v-if="remoteLoading" class="py-12 text-center text-slate-400">
              <i class="pi pi-spin pi-spinner text-xl text-electric-blue"></i>
            </div>
            <div v-else-if="remoteItems.length === 0" class="py-12 text-center text-slate-500">
              Directory is empty
            </div>
            <div
              v-else
              v-for="item in remoteItems"
              :key="item.Name"
              @click="handleItemClick(remoteName, item)"
              @contextmenu.prevent="handleContextMenu($event, remoteName, item)"
              class="flex items-center justify-between px-3 py-2 cursor-pointer select-none transition-colors rounded-lg group"
              :class="remoteSelected?.Name === item.Name ? 'bg-carbon-elevated text-electric-blue font-bold' : 'hover:bg-carbon-elevated/60 text-slate-200'"
            >
              <div class="flex items-center gap-2 truncate">
                <i
                  class="text-xs"
                  :class="item.IsDir ? 'pi pi-folder text-amber-400' : 'pi pi-file text-slate-400'"
                ></i>
                <span class="truncate">{{ item.Name }}</span>
              </div>
              <div class="flex items-center gap-2 text-[11px] text-slate-400 shrink-0">
                <span v-if="!item.IsDir">{{ formatBytes(item.Size) }}</span>
                <span v-else class="text-slate-500">DIR</span>
              </div>
            </div>
          </div>
        </SplitterPanel>
      </Splitter>
    </div>

    <!-- Modals -->
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
      :currentPath="mkdirTargetRemote === 'local' ? localPath : remotePath"
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
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import Splitter from 'primevue/splitter'
import SplitterPanel from 'primevue/splitterpanel'
import { useProfilesStore } from '../stores/profiles'
import { useExplorerStore } from '../stores/explorer'
import type { BrowseItem } from '../types/models'
import type { FilePreviewResponse, FileCompareResponse } from '../types/explorer'
import ModalFilePreview from '../components/ModalFilePreview.vue'
import ModalCompare from '../components/ModalCompare.vue'
import ModalExplorerMkdir from '../components/ModalExplorerMkdir.vue'
import ModalExplorerMove from '../components/ModalExplorerMove.vue'
import ModalConfirm from '../components/ModalConfirm.vue'
import ExplorerContextMenu from '../components/ExplorerContextMenu.vue'

const profilesStore = useProfilesStore()
const explorerStore = useExplorerStore()

const localPath = ref<string>('')
const localItems = ref<BrowseItem[]>([])
const localLoading = ref<boolean>(false)
const localCanary = ref<boolean>(false)
const localSelected = ref<BrowseItem | null>(null)

const remoteName = ref<string>('local')
const remotePath = ref<string>('')
const remoteItems = ref<BrowseItem[]>([])
const remoteLoading = ref<boolean>(false)
const remoteCanary = ref<boolean>(false)
const remoteSelected = ref<BrowseItem | null>(null)

const activeProfile = computed(() => profilesStore.activeProfile)

// Preview State
const showPreviewModal = ref(false)
const previewData = ref<FilePreviewResponse | null>(null)
const previewLoading = ref(false)
const previewError = ref<string | null>(null)

// Compare State
const showCompareModal = ref(false)
const compareData = ref<FileCompareResponse | null>(null)
const compareLoading = ref(false)
const compareError = ref<string | null>(null)

// Mkdir State
const showMkdirModal = ref(false)
const mkdirTargetRemote = ref('local')
const mkdirLoading = ref(false)
const mkdirError = ref<string | null>(null)

// Move State
const showMoveModal = ref(false)
const moveTargetRemote = ref('local')
const moveSrcPath = ref('')
const moveIsDir = ref(false)
const moveLoading = ref(false)
const moveError = ref<string | null>(null)

// Confirm Delete State
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

watch(
  () => activeProfile.value,
  (profile) => {
    if (profile) {
      localPath.value = profile.local_path || ''
      remoteName.value = profile.remote || 'local'
      remotePath.value = profile.remote_path || ''
      refreshLocal()
      refreshRemote()
    }
  },
  { immediate: true }
)

async function refreshLocal() {
  localLoading.value = true
  localSelected.value = null
  try {
    await explorerStore.browse('local', localPath.value, activeProfile.value?.id)
    localItems.value = explorerStore.items
    localCanary.value = explorerStore.hasCanary
  } catch (e) {
    localItems.value = []
  } finally {
    localLoading.value = false
  }
}

async function refreshRemote() {
  remoteLoading.value = true
  remoteSelected.value = null
  try {
    await explorerStore.browse(remoteName.value, remotePath.value, activeProfile.value?.id)
    remoteItems.value = explorerStore.items
    remoteCanary.value = explorerStore.hasCanary
  } catch (e) {
    remoteItems.value = []
  } finally {
    remoteLoading.value = false
  }
}

function handleItemClick(remote: string, item: BrowseItem) {
  if (remote === 'local') {
    if (item.IsDir) {
      const next = localPath.value.replace(/\/+$/, '') + '/' + item.Name
      localPath.value = next
      refreshLocal()
    } else {
      localSelected.value = item
      triggerPreview('local', item)
    }
  } else {
    if (item.IsDir) {
      const next = remotePath.value ? remotePath.value.replace(/\/+$/, '') + '/' + item.Name : item.Name
      remotePath.value = next
      refreshRemote()
    } else {
      remoteSelected.value = item
      triggerPreview(remote, item)
    }
  }
}

function handleContextMenu(e: MouseEvent, remote: string, item: BrowseItem) {
  contextMenuX.value = e.clientX
  contextMenuY.value = e.clientY
  contextMenuItem.value = item
  contextMenuRemote.value = remote
  contextMenuVisible.value = true

  if (remote === 'local') {
    localSelected.value = item
  } else {
    remoteSelected.value = item
  }
}

async function triggerPreview(remote: string, item: BrowseItem) {
  const fullPath = remote === 'local'
    ? (localPath.value ? localPath.value.replace(/\/+$/, '') + '/' + item.Name : item.Name)
    : (remotePath.value ? remotePath.value.replace(/\/+$/, '') + '/' + item.Name : item.Name)

  previewLoading.value = true
  previewError.value = null
  previewData.value = null
  showPreviewModal.value = true

  try {
    previewData.value = await explorerStore.previewFile(remote, fullPath, activeProfile.value?.id)
  } catch (e: any) {
    previewError.value = e.message || 'Failed to load preview'
  } finally {
    previewLoading.value = false
  }
}

function openCompareSelected() {
  if (!localSelected.value || !remoteSelected.value) return

  const path1 = localPath.value ? localPath.value.replace(/\/+$/, '') + '/' + localSelected.value.Name : localSelected.value.Name
  const path2 = remotePath.value ? remotePath.value.replace(/\/+$/, '') + '/' + remoteSelected.value.Name : remoteSelected.value.Name

  compareLoading.value = true
  compareError.value = null
  compareData.value = null
  showCompareModal.value = true

  explorerStore
    .compareFiles([
      { remote: 'local', path: path1, profile_id: activeProfile.value?.id },
      { remote: remoteName.value, path: path2, profile_id: activeProfile.value?.id },
    ])
    .then((res) => {
      compareData.value = res
    })
    .catch((e: any) => {
      compareError.value = e.message || 'Comparison failed'
    })
    .finally(() => {
      compareLoading.value = false
    })
}

function openMkdir(remote: string) {
  mkdirTargetRemote.value = remote
  mkdirError.value = null
  showMkdirModal.value = true
}

async function handleCreateDirectory(dirName: string) {
  const base = mkdirTargetRemote.value === 'local' ? localPath.value : remotePath.value
  const fullPath = base ? base.replace(/\/+$/, '') + '/' + dirName : dirName
  mkdirLoading.value = true
  mkdirError.value = null

  try {
    await explorerStore.createDirectory(mkdirTargetRemote.value, fullPath, activeProfile.value?.id)
    showMkdirModal.value = false
    if (mkdirTargetRemote.value === 'local') {
      refreshLocal()
    } else {
      refreshRemote()
    }
  } catch (e: any) {
    mkdirError.value = e.message || 'Failed to create directory'
  } finally {
    mkdirLoading.value = false
  }
}

function handleContextAction(act: 'preview' | 'compare' | 'download' | 'move' | 'copy_path' | 'delete') {
  if (!contextMenuItem.value) return
  const item = contextMenuItem.value
  const remote = contextMenuRemote.value
  const base = remote === 'local' ? localPath.value : remotePath.value
  const fullPath = base ? base.replace(/\/+$/, '') + '/' + item.Name : item.Name

  switch (act) {
    case 'preview':
      triggerPreview(remote, item)
      break
    case 'compare':
      if (remote === 'local') {
        localSelected.value = item
      } else {
        remoteSelected.value = item
      }
      openCompareSelected()
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
    if (moveTargetRemote.value === 'local') {
      refreshLocal()
    } else {
      refreshRemote()
    }
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
    if (deleteRemote.value === 'local') {
      refreshLocal()
    } else {
      refreshRemote()
    }
  } catch (e: any) {
    alert(e.message || 'Deletion failed')
  } finally {
    confirmLoading.value = false
  }
}

function upLocal() {
  const parts = localPath.value.replace(/\/+$/, '').split('/')
  if (parts.length > 1) {
    parts.pop()
    localPath.value = parts.join('/') || '/'
    refreshLocal()
  }
}

function upRemote() {
  const parts = remotePath.value.replace(/\/+$/, '').split('/')
  if (parts.length > 1) {
    parts.pop()
    remotePath.value = parts.join('/')
    refreshRemote()
  } else {
    remotePath.value = ''
    refreshRemote()
  }
}

function formatBytes(bytes?: number): string {
  if (bytes === undefined || bytes === null || bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`
}
</script>
