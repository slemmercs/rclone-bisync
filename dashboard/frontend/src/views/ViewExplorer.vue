<template>
  <div class="space-y-4">
    <!-- Header Toolbar -->
    <div class="flex flex-wrap items-center justify-between gap-3 bg-carbon-900 p-3.5 rounded-2xl border border-carbon-700 shadow-md">
      <div class="flex items-center gap-3">
        <div class="h-8 w-8 rounded-lg bg-carbon-800 border border-carbon-700 flex items-center justify-center text-electric-400">
          <i class="pi pi-folder text-sm"></i>
        </div>
        <div>
          <h2 class="text-sm font-bold text-white tracking-tight flex items-center gap-2">
            <span>Dual-Pane Directory Explorer</span>
            <span class="text-[10px] font-mono px-2 py-0.5 rounded bg-carbon-800 text-carbon-400 border border-carbon-700">
              {{ isSwapped ? 'Remote ⇄ Local' : 'Local ⇄ Remote' }}
            </span>
          </h2>
          <p class="text-[11px] text-carbon-400">Direct side-by-side inspection, multi-file checksum comparisons, and cross-backend operations.</p>
        </div>
      </div>

      <!-- Global Explorer Action Buttons -->
      <div class="flex flex-wrap items-center gap-2">
        <!-- Compare Selection Pill Button -->
        <div v-if="explorerStore.compareSelection.length > 0" class="flex items-center gap-1.5 bg-carbon-950 p-1 rounded-xl border border-carbon-700">
          <button
            @click="openCompareModal"
            class="px-2.5 py-1 rounded-lg text-xs font-semibold transition-all flex items-center gap-1.5 cursor-pointer shadow-sm"
            :class="explorerStore.compareSelection.length >= 2
              ? 'bg-indigo-600 hover:bg-indigo-500 text-white animate-pulse'
              : 'bg-carbon-800 hover:bg-carbon-700 text-carbon-300'"
            :title="explorerStore.compareSelection.length >= 2 ? 'Run cryptographic checksum comparison' : 'Select at least 2 files to compare'"
          >
            <i class="pi pi-sliders-v text-xs"></i>
            <span>Compare ({{ explorerStore.compareSelection.length }})</span>
          </button>
          <button
            @click="explorerStore.clearCompareSelection()"
            class="px-2 py-1 bg-carbon-800 hover:bg-carbon-700 text-carbon-400 hover:text-white rounded-lg text-xs transition-colors cursor-pointer"
            title="Clear compare selection"
          >
            ✕
          </button>
        </div>

        <!-- Swap Panes Button -->
        <button
          @click="swapPanes"
          class="px-3 py-1.5 bg-carbon-800 hover:bg-carbon-750 text-carbon-200 border border-carbon-700 rounded-xl text-xs font-medium transition-colors flex items-center gap-1.5 cursor-pointer"
          title="Swap Left and Right Panes"
        >
          <i class="pi pi-arrows-h text-xs text-electric-400"></i>
          <span>Swap Panes</span>
        </button>

        <!-- Refresh Explorer Button -->
        <button
          @click="refreshAll"
          class="px-3 py-1.5 bg-carbon-800 hover:bg-carbon-750 text-carbon-200 border border-carbon-700 rounded-xl text-xs font-medium transition-colors flex items-center gap-1.5 cursor-pointer"
          title="Refresh both storage panes"
        >
          <i :class="(leftLoading || rightLoading) ? 'pi pi-spin pi-spinner' : 'pi pi-refresh'" class="text-xs"></i>
          <span>Refresh</span>
        </button>

        <!-- Toggle Log Dock Button -->
        <button
          @click="logsStore.toggleDock()"
          class="px-3 py-1.5 bg-carbon-800 hover:bg-carbon-750 text-carbon-200 border border-carbon-700 rounded-xl text-xs font-medium transition-colors flex items-center gap-1.5 cursor-pointer"
          :class="logsStore.isDockOpen ? 'text-electric-400 border-electric-500' : ''"
          title="Toggle Streaming Log Dock"
        >
          <i class="pi pi-terminal text-xs"></i>
          <span>Log Dock</span>
        </button>
      </div>
    </div>

    <!-- Hidden File Upload Inputs -->
    <input type="file" ref="leftFileInput" class="hidden" @change="handleFileUpload('left', $event)" />
    <input type="file" ref="rightFileInput" class="hidden" @change="handleFileUpload('right', $event)" />

    <!-- Dual Explorer Splitter -->
    <div class="bg-carbon-900 border border-carbon-700 rounded-2xl overflow-hidden shadow-sm min-h-[580px]">
      <Splitter class="min-h-[580px] bg-carbon-900 text-carbon-100 border-none">
        <!-- Left Pane -->
        <SplitterPanel :size="50" :minSize="30" class="p-4 flex flex-col border-r border-carbon-700">
          <!-- Pane Header -->
          <div class="flex items-center justify-between gap-2 mb-2 pb-2 border-b border-carbon-700">
            <div class="flex items-center gap-2">
              <i :class="leftIsLocal ? 'pi pi-desktop text-indigo-400' : 'pi pi-cloud text-emerald-400'"></i>
              <span class="text-xs font-bold text-white uppercase tracking-wide">
                {{ leftIsLocal ? 'Local Storage' : leftRemote }}
              </span>
              <!-- Canary Guard Pill -->
              <span
                class="text-[10px] font-mono px-2 py-0.5 rounded flex items-center gap-1"
                :class="leftCanary ? 'bg-emerald-950 text-emerald-400 border border-emerald-700' : 'bg-amber-950 text-amber-400 border border-amber-700'"
                :title="leftCanary ? 'Canary Active: RCLONE_TEST verified' : 'Canary Missing: Click Touch Canary to provision'"
              >
                <i :class="leftCanary ? 'pi pi-check' : 'pi pi-exclamation-triangle'" class="text-[9px]"></i>
                {{ leftCanary ? 'Canary OK' : 'No Canary' }}
              </span>
              <button
                v-if="!leftCanary"
                @click="touchCanary('left')"
                class="text-[10px] px-1.5 py-0.5 rounded bg-amber-900/60 hover:bg-amber-800 text-amber-200 border border-amber-700 cursor-pointer"
                title="Create RCLONE_TEST canary file"
              >
                Touch Canary
              </button>
            </div>

            <!-- Pane Action Icons -->
            <div class="flex items-center gap-1">
              <button
                @click="triggerUploadPicker('left')"
                :disabled="leftUploading"
                class="p-1.5 rounded-lg bg-carbon-800 text-carbon-300 hover:text-white hover:bg-carbon-750 transition-colors cursor-pointer"
                title="Upload file to this directory"
              >
                <i :class="leftUploading ? 'pi pi-spin pi-spinner' : 'pi pi-upload'" class="text-xs"></i>
              </button>
              <button
                @click="openMkdir('left')"
                class="p-1.5 rounded-lg bg-carbon-800 text-carbon-300 hover:text-white hover:bg-carbon-750 transition-colors cursor-pointer"
                title="Create new folder"
              >
                <i class="pi pi-folder-plus text-xs"></i>
              </button>
              <button
                @click="upDirectory('left')"
                class="p-1.5 rounded-lg bg-carbon-800 text-carbon-300 hover:text-white hover:bg-carbon-750 transition-colors cursor-pointer"
                title="Navigate up"
              >
                <i class="pi pi-arrow-up text-xs"></i>
              </button>
              <button
                @click="refreshPane('left')"
                class="p-1.5 rounded-lg bg-carbon-800 text-carbon-300 hover:text-white hover:bg-carbon-750 transition-colors cursor-pointer"
                title="Refresh directory"
              >
                <i :class="leftLoading ? 'pi pi-spin pi-spinner' : 'pi pi-refresh'" class="text-xs"></i>
              </button>
            </div>
          </div>

          <!-- Path Breadcrumbs -->
          <div class="mb-2 bg-carbon-950 px-3 py-1.5 rounded-xl border border-carbon-800 text-xs font-mono text-indigo-300 truncate flex items-center justify-between">
            <span class="truncate">{{ getDisplayPath('left') }}</span>
            <span class="text-[10px] text-carbon-500 shrink-0 font-sans ml-2">{{ leftItems.length }} item(s)</span>
          </div>

          <!-- Left Items List -->
          <div class="flex-1 overflow-y-auto max-h-[460px] divide-y divide-carbon-800/40 text-xs font-mono">
            <div v-if="leftLoading" class="py-16 text-center text-carbon-400">
              <i class="pi pi-spin pi-spinner text-xl text-electric-400 mb-2"></i>
              <div class="text-[11px]">Reading directory entries...</div>
            </div>
            <div v-else-if="leftItems.length === 0" class="py-16 text-center text-carbon-500">
              Directory is empty.
            </div>
            <div
              v-else
              v-for="item in leftItems"
              :key="item.Name"
              @click="handleItemClick('left', item)"
              @contextmenu.prevent="handleContextMenu($event, 'left', item)"
              class="flex items-center justify-between px-3 py-2 cursor-pointer select-none transition-colors rounded-lg group"
              :class="isRowSelected('left', item) ? 'bg-carbon-800 text-electric-400 font-semibold' : 'hover:bg-carbon-800/60 text-carbon-200'"
            >
              <div class="flex items-center gap-2.5 truncate flex-1 min-w-0">
                <!-- Checkbox for multi-file comparison -->
                <input
                  v-if="!item.IsDir && item.Name !== 'RCLONE_TEST'"
                  type="checkbox"
                  :checked="isItemInCompare('left', item)"
                  @click.stop="toggleCompare('left', item)"
                  class="rounded bg-carbon-900 border-carbon-700 text-electric-500 cursor-pointer shrink-0"
                  title="Select for checksum comparison"
                />
                <span v-else class="w-3.5 shrink-0"></span>

                <i
                  class="text-xs shrink-0"
                  :class="item.IsDir ? 'pi pi-folder text-amber-400' : 'pi pi-file text-carbon-400'"
                ></i>
                <span class="truncate">{{ item.Name }}</span>
              </div>

              <div class="flex items-center gap-2 text-[11px] text-carbon-400 shrink-0">
                <span v-if="!item.IsDir">{{ formatBytes(item.Size) }}</span>
                <span v-else class="text-carbon-500 text-[10px]">DIR</span>

                <!-- Quick Open on Remote Icon for Cloud Remotes -->
                <button
                  v-if="!leftIsLocal && !item.IsDir"
                  @click.stop="openOnRemoteDirect(leftRemote, getRelativeItemPath('left', item))"
                  class="opacity-0 group-hover:opacity-100 hover:text-electric-400 p-0.5 transition-opacity cursor-pointer"
                  title="Open on remote in new tab"
                >
                  <i class="pi pi-external-link text-xs"></i>
                </button>
              </div>
            </div>
          </div>
        </SplitterPanel>

        <!-- Right Pane -->
        <SplitterPanel :size="50" :minSize="30" class="p-4 flex flex-col">
          <!-- Pane Header -->
          <div class="flex items-center justify-between gap-2 mb-2 pb-2 border-b border-carbon-700">
            <div class="flex items-center gap-2">
              <i :class="rightIsLocal ? 'pi pi-desktop text-indigo-400' : 'pi pi-cloud text-emerald-400'"></i>
              <span class="text-xs font-bold text-white uppercase tracking-wide">
                {{ rightIsLocal ? 'Local Storage' : rightRemote }}
              </span>
              <!-- Canary Guard Pill -->
              <span
                class="text-[10px] font-mono px-2 py-0.5 rounded flex items-center gap-1"
                :class="rightCanary ? 'bg-emerald-950 text-emerald-400 border border-emerald-700' : 'bg-amber-950 text-amber-400 border border-amber-700'"
                :title="rightCanary ? 'Canary Active: RCLONE_TEST verified' : 'Canary Missing: Click Touch Canary to provision'"
              >
                <i :class="rightCanary ? 'pi pi-check' : 'pi pi-exclamation-triangle'" class="text-[9px]"></i>
                {{ rightCanary ? 'Canary OK' : 'No Canary' }}
              </span>
              <button
                v-if="!rightCanary"
                @click="touchCanary('right')"
                class="text-[10px] px-1.5 py-0.5 rounded bg-amber-900/60 hover:bg-amber-800 text-amber-200 border border-amber-700 cursor-pointer"
                title="Create RCLONE_TEST canary file"
              >
                Touch Canary
              </button>
            </div>

            <!-- Pane Action Icons -->
            <div class="flex items-center gap-1">
              <button
                @click="triggerUploadPicker('right')"
                :disabled="rightUploading"
                class="p-1.5 rounded-lg bg-carbon-800 text-carbon-300 hover:text-white hover:bg-carbon-750 transition-colors cursor-pointer"
                title="Upload file to this directory"
              >
                <i :class="rightUploading ? 'pi pi-spin pi-spinner' : 'pi pi-upload'" class="text-xs"></i>
              </button>
              <button
                @click="openMkdir('right')"
                class="p-1.5 rounded-lg bg-carbon-800 text-carbon-300 hover:text-white hover:bg-carbon-750 transition-colors cursor-pointer"
                title="Create new folder"
              >
                <i class="pi pi-folder-plus text-xs"></i>
              </button>
              <button
                @click="upDirectory('right')"
                class="p-1.5 rounded-lg bg-carbon-800 text-carbon-300 hover:text-white hover:bg-carbon-750 transition-colors cursor-pointer"
                title="Navigate up"
              >
                <i class="pi pi-arrow-up text-xs"></i>
              </button>
              <button
                @click="refreshPane('right')"
                class="p-1.5 rounded-lg bg-carbon-800 text-carbon-300 hover:text-white hover:bg-carbon-750 transition-colors cursor-pointer"
                title="Refresh directory"
              >
                <i :class="rightLoading ? 'pi pi-spin pi-spinner' : 'pi pi-refresh'" class="text-xs"></i>
              </button>
            </div>
          </div>

          <!-- Path Breadcrumbs -->
          <div class="mb-2 bg-carbon-950 px-3 py-1.5 rounded-xl border border-carbon-800 text-xs font-mono text-emerald-400 truncate flex items-center justify-between">
            <span class="truncate">{{ getDisplayPath('right') }}</span>
            <span class="text-[10px] text-carbon-500 shrink-0 font-sans ml-2">{{ rightItems.length }} item(s)</span>
          </div>

          <!-- Right Items List -->
          <div class="flex-1 overflow-y-auto max-h-[460px] divide-y divide-carbon-800/40 text-xs font-mono">
            <div v-if="rightLoading" class="py-16 text-center text-carbon-400">
              <i class="pi pi-spin pi-spinner text-xl text-electric-400 mb-2"></i>
              <div class="text-[11px]">Reading directory entries...</div>
            </div>
            <div v-else-if="rightItems.length === 0" class="py-16 text-center text-carbon-500">
              Directory is empty.
            </div>
            <div
              v-else
              v-for="item in rightItems"
              :key="item.Name"
              @click="handleItemClick('right', item)"
              @contextmenu.prevent="handleContextMenu($event, 'right', item)"
              class="flex items-center justify-between px-3 py-2 cursor-pointer select-none transition-colors rounded-lg group"
              :class="isRowSelected('right', item) ? 'bg-carbon-800 text-electric-400 font-semibold' : 'hover:bg-carbon-800/60 text-carbon-200'"
            >
              <div class="flex items-center gap-2.5 truncate flex-1 min-w-0">
                <!-- Checkbox for multi-file comparison -->
                <input
                  v-if="!item.IsDir && item.Name !== 'RCLONE_TEST'"
                  type="checkbox"
                  :checked="isItemInCompare('right', item)"
                  @click.stop="toggleCompare('right', item)"
                  class="rounded bg-carbon-900 border-carbon-700 text-electric-500 cursor-pointer shrink-0"
                  title="Select for checksum comparison"
                />
                <span v-else class="w-3.5 shrink-0"></span>

                <i
                  class="text-xs shrink-0"
                  :class="item.IsDir ? 'pi pi-folder text-amber-400' : 'pi pi-file text-carbon-400'"
                ></i>
                <span class="truncate">{{ item.Name }}</span>
              </div>

              <div class="flex items-center gap-2 text-[11px] text-carbon-400 shrink-0">
                <span v-if="!item.IsDir">{{ formatBytes(item.Size) }}</span>
                <span v-else class="text-carbon-500 text-[10px]">DIR</span>

                <!-- Quick Open on Remote Icon for Cloud Remotes -->
                <button
                  v-if="!rightIsLocal && !item.IsDir"
                  @click.stop="openOnRemoteDirect(rightRemote, getRelativeItemPath('right', item))"
                  class="opacity-0 group-hover:opacity-100 hover:text-electric-400 p-0.5 transition-opacity cursor-pointer"
                  title="Open on remote in new tab"
                >
                  <i class="pi pi-external-link text-xs"></i>
                </button>
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
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import Splitter from 'primevue/splitter'
import SplitterPanel from 'primevue/splitterpanel'
import { useProfilesStore } from '../stores/profiles'
import { useExplorerStore } from '../stores/explorer'
import { useLogsStore } from '../stores/logs'
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
const logsStore = useLogsStore()

// Pane Swap State
const isSwapped = ref(false)

// Left Pane State (defaults to Local)
const leftPath = ref('')
const leftItems = ref<BrowseItem[]>([])
const leftLoading = ref(false)
const leftCanary = ref(false)
const leftSelected = ref<BrowseItem | null>(null)
const leftUploading = ref(false)
const leftFileInput = ref<HTMLInputElement | null>(null)

// Right Pane State (defaults to Remote)
const rightRemoteName = ref('')
const rightPath = ref('')
const rightItems = ref<BrowseItem[]>([])
const rightLoading = ref(false)
const rightCanary = ref(false)
const rightSelected = ref<BrowseItem | null>(null)
const rightUploading = ref(false)
const rightFileInput = ref<HTMLInputElement | null>(null)

const activeProfile = computed(() => profilesStore.activeProfile)

const leftIsLocal = computed(() => !isSwapped.value)
const rightIsLocal = computed(() => isSwapped.value)

const leftRemote = computed(() => {
  return isSwapped.value ? rightRemoteName.value : 'local'
})

const rightRemote = computed(() => {
  return isSwapped.value ? 'local' : rightRemoteName.value
})

// Modal states
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

// Watch active profile to load directories
watch(
  () => activeProfile.value,
  (profile) => {
    if (profile) {
      leftPath.value = ''
      rightRemoteName.value = (profile.remote || 'local').replace(/:$/, '')
      rightPath.value = (profile.remote_path || '').replace(/^\/+/, '')
      refreshAll()
    }
  },
  { immediate: true }
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
  const isLocal = pane === 'left' ? leftIsLocal.value : rightIsLocal.value

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
      if (!path) {
        leftCanary.value = explorerStore.hasCanary
      }
    } else {
      rightItems.value = explorerStore.items
      if (!path) {
        rightCanary.value = explorerStore.hasCanary
      }
    }
  } catch (e) {
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

function getDisplayPath(pane: 'left' | 'right'): string {
  const isLocal = pane === 'left' ? leftIsLocal.value : rightIsLocal.value
  const relPath = pane === 'left' ? leftPath.value : rightPath.value
  const remote = pane === 'left' ? leftRemote.value : rightRemote.value

  if (isLocal) {
    const root = activeProfile.value?.local_path || 'local'
    return relPath ? `${root}/${relPath}` : root
  }
  return relPath ? `${remote}:${relPath}` : `${remote}:/`
}

function getRelativeItemPath(pane: 'left' | 'right', item: BrowseItem): string {
  const currentPath = pane === 'left' ? leftPath.value : rightPath.value
  return currentPath ? `${currentPath.replace(/\/+$/, '')}/${item.Name}` : item.Name
}

function isRowSelected(pane: 'left' | 'right', item: BrowseItem): boolean {
  const selected = pane === 'left' ? leftSelected.value : rightSelected.value
  return selected?.Name === item.Name
}

function isItemInCompare(pane: 'left' | 'right', item: BrowseItem): boolean {
  const remote = pane === 'left' ? leftRemote.value : rightRemote.value
  const path = getRelativeItemPath(pane, item)
  return explorerStore.isItemInCompare(remote, path)
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

function handleItemClick(pane: 'left' | 'right', item: BrowseItem) {
  if (item.IsDir) {
    if (pane === 'left') {
      leftPath.value = getRelativeItemPath('left', item)
      refreshPane('left')
    } else {
      rightPath.value = getRelativeItemPath('right', item)
      refreshPane('right')
    }
  } else {
    if (pane === 'left') leftSelected.value = item
    else rightSelected.value = item
    triggerPreview(pane === 'left' ? leftRemote.value : rightRemote.value, getRelativeItemPath(pane, item), item.Name)
  }
}

function upDirectory(pane: 'left' | 'right') {
  const current = pane === 'left' ? leftPath.value : rightPath.value
  const parts = current.replace(/\/+$/, '').split('/').filter(Boolean)
  if (parts.length > 0) {
    parts.pop()
    const next = parts.join('/')
    if (pane === 'left') {
      leftPath.value = next
      refreshPane('left')
    } else {
      rightPath.value = next
      refreshPane('right')
    }
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

function triggerUploadPicker(pane: 'left' | 'right') {
  if (pane === 'left' && leftFileInput.value) {
    leftFileInput.value.click()
  } else if (pane === 'right' && rightFileInput.value) {
    rightFileInput.value.click()
  }
}

async function handleFileUpload(pane: 'left' | 'right', e: Event) {
  const target = e.target as HTMLInputElement
  const file = target.files?.[0]
  if (!file) return

  const remote = pane === 'left' ? leftRemote.value : rightRemote.value
  const path = pane === 'left' ? leftPath.value : rightPath.value

  if (pane === 'left') leftUploading.value = true
  else rightUploading.value = true

  try {
    await explorerStore.uploadFile(file, remote, path, activeProfile.value?.id)
    refreshPane(pane)
  } catch (err: any) {
    alert(err.message || 'File upload failed')
  } finally {
    if (pane === 'left') leftUploading.value = false
    else rightUploading.value = false
    target.value = ''
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
    alert(err.message || 'Failed to resolve cloud web link')
  }
}

async function triggerPreview(remote: string, path: string, name: string) {
  previewLoading.value = true
  previewError.value = null
  previewData.value = null
  showPreviewModal.value = true

  try {
    previewData.value = await explorerStore.previewFile(remote, path, activeProfile.value?.id)
  } catch (e: any) {
    previewError.value = e.message || 'Failed to load preview'
  } finally {
    previewLoading.value = false
  }
}

function openCompareModal() {
  if (explorerStore.compareSelection.length < 2) return

  compareLoading.value = true
  compareError.value = null
  compareData.value = null
  showCompareModal.value = true

  const items = explorerStore.compareSelection.map((i) => ({
    remote: i.remote,
    path: i.path,
    profile_id: activeProfile.value?.id,
  }))

  explorerStore
    .compareFiles(items)
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
      triggerPreview(remote, fullPath, item.Name)
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

function formatBytes(bytes?: number): string {
  if (bytes === undefined || bytes === null || bytes <= 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`
}
</script>
