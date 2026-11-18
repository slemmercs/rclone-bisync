<template>
  <div
    class="p-4 flex flex-col relative h-full min-h-[540px]"
    @dragover="onDragOver"
    @dragleave="onDragLeave"
    @drop="onDrop"
  >
    <!-- Drag Over Active Overlay -->
    <div
      v-if="isDragOver"
      class="absolute inset-0 z-30 flex flex-col items-center justify-center pointer-events-none border-2 border-dashed rounded-2xl backdrop-blur-sm p-4 text-center select-none"
      :class="isLocal ? 'bg-indigo-950/90 border-indigo-400' : 'bg-emerald-950/90 border-emerald-400'"
    >
      <template v-if="explorerStore.draggedPayload && explorerStore.draggedPayload.sourcePane !== pane">
        <i
          class="pi pi-arrows-alt text-3xl animate-bounce mb-2 text-amber-400"
        ></i>
        <span class="text-sm font-bold text-white uppercase tracking-wider">
          Drop to transfer "{{ explorerStore.draggedPayload.item.Name }}" to {{ isLocal ? 'Local Storage' : remote }}
        </span>
        <span class="text-xs text-amber-300 font-mono mt-1">
          from {{ explorerStore.draggedPayload.sourcePane === 'left' ? 'Left Pane' : 'Right Pane' }} ({{ explorerStore.draggedPayload.sourceRemote }})
        </span>
      </template>
      <template v-else>
        <i
          class="pi pi-upload text-3xl animate-bounce mb-2"
          :class="isLocal ? 'text-indigo-400' : 'text-emerald-400'"
        ></i>
        <span class="text-sm font-bold text-white uppercase tracking-wider">
          Drop to upload to {{ isLocal ? 'Local Storage' : remote }}
        </span>
      </template>
    </div>

    <!-- Hidden File Upload Input -->
    <input
      type="file"
      ref="fileInputRef"
      class="hidden"
      multiple
      @change="handleFileInputChange"
    />

    <!-- Pane Header -->
    <div class="flex items-center justify-between gap-2 mb-2 pb-2 border-b border-carbon-700">
      <div class="flex items-center gap-2">
        <i :class="isLocal ? 'pi pi-desktop text-indigo-400' : 'pi pi-cloud text-emerald-400'"></i>
        <template v-if="isLocal">
          <span class="text-xs font-bold text-white uppercase tracking-wide">Local Storage</span>
        </template>
        <template v-else>
          <!-- Scoping: Locked to profile -->
          <div
            v-if="isRemoteLocked && activeProfile?.remote"
            class="flex items-center gap-1.5 px-2 py-0.5 rounded bg-carbon-800 border border-carbon-700 text-xs font-bold text-white uppercase"
            :title="'Locked to active profile: ' + activeProfile.name"
          >
            <i class="pi pi-lock text-[10px] text-amber-400"></i>
            <span>{{ remote }}</span>
          </div>
          <!-- Scoping: Unlocked dropdown -->
          <select
            v-else
            :value="remote"
            @change="onRemoteSelectChange($event)"
            class="bg-carbon-950 border border-carbon-700 rounded px-2 py-0.5 text-xs text-white focus:outline-none focus:border-electric-blue font-bold uppercase cursor-pointer"
            title="Explorer Remote Scoping Unlocked - Switch to any configured remote"
          >
            <option
              v-for="r in remotes"
              :key="r.name"
              :value="r.name.replace(/:$/, '')"
            >
              {{ r.name.replace(/:$/, '') }}{{ activeProfile?.remote && r.name.replace(/:$/, '') === activeProfile.remote.replace(/:$/, '') ? ' (Active)' : '' }}
            </option>
          </select>
        </template>

        <!-- Canary Guard Pill -->
        <span
          class="text-[10px] font-mono px-2 py-0.5 rounded flex items-center gap-1"
          :class="canary ? 'bg-emerald-950 text-emerald-400 border border-emerald-700' : 'bg-amber-950 text-amber-400 border border-amber-700'"
          :title="canary ? 'Canary Active: RCLONE_TEST verified' : 'Canary Missing: Click Touch Canary to provision'"
        >
          <i :class="canary ? 'pi pi-check' : 'pi pi-exclamation-triangle'" class="text-[9px]"></i>
          {{ canary ? 'Canary OK' : 'No Canary' }}
        </span>
        <button
          v-if="!canary"
          @click="$emit('touchCanary')"
          class="text-[10px] px-1.5 py-0.5 rounded bg-amber-900/60 hover:bg-amber-800 text-amber-200 border border-amber-700 cursor-pointer"
          title="Create RCLONE_TEST canary file"
        >
          Touch Canary
        </button>
      </div>

      <!-- Pane Action Icons -->
      <div class="flex items-center gap-1">
        <button
          @click="triggerUploadPicker"
          :disabled="uploading"
          class="p-1.5 rounded-lg bg-carbon-800 text-carbon-300 hover:text-white hover:bg-carbon-750 transition-colors cursor-pointer disabled:opacity-50"
          title="Upload file to this directory"
        >
          <i :class="uploading ? 'pi pi-spin pi-spinner' : 'pi pi-upload'" class="text-xs"></i>
        </button>
        <button
          @click="$emit('mkdir')"
          class="p-1.5 rounded-lg bg-carbon-800 text-carbon-300 hover:text-white hover:bg-carbon-750 transition-colors cursor-pointer"
          title="Create new folder"
        >
          <i class="pi pi-folder-plus text-xs"></i>
        </button>
        <button
          @click="$emit('upDir')"
          class="p-1.5 rounded-lg bg-carbon-800 text-carbon-300 hover:text-white hover:bg-carbon-750 transition-colors cursor-pointer"
          title="Navigate up"
        >
          <i class="pi pi-arrow-up text-xs"></i>
        </button>
        <button
          @click="$emit('refresh')"
          class="p-1.5 rounded-lg bg-carbon-800 text-carbon-300 hover:text-white hover:bg-carbon-750 transition-colors cursor-pointer"
          title="Refresh directory"
        >
          <i :class="loading ? 'pi pi-spin pi-spinner' : 'pi pi-refresh'" class="text-xs"></i>
        </button>
      </div>
    </div>

    <!-- Breadcrumb Location Bar -->
    <ExplorerBreadcrumbs
      :displayPath="displayPath"
      :breadcrumbs="breadcrumbs"
      :itemCount="items.length"
      :copied="copied"
      @navigate="$emit('navigate', $event.path)"
      @copy="handleCopyPath"
    />

    <!-- Items Table & Sort/Filter Header -->
    <ExplorerTable
      :items="items"
      :loading="loading"
      :selectedItem="selectedItem"
      :isLocal="isLocal"
      :remote="remote"
      :currentPath="path"
      :pane="pane"
      @itemClick="$emit('itemClick', $event)"
      @itemDblClick="$emit('itemDblClick', $event)"
      @contextmenu="(e, item) => $emit('contextmenu', e, item)"
      @toggleCompare="$emit('toggleCompare', $event)"
      @nameClick="$emit('nameClick', $event)"
      @openOnRemote="$emit('openOnRemote', $event)"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import ExplorerBreadcrumbs from './ExplorerBreadcrumbs.vue'
import ExplorerTable from './ExplorerTable.vue'
import { useExplorerStore } from '../../stores/explorer'
import type { BrowseItem, RemoteEntry, Profile } from '../../types/models'
import type { BreadcrumbItem, ExplorerDragPayload } from '../../types/explorer'

const props = defineProps<{
  pane: 'left' | 'right'
  isLocal: boolean
  remote: string
  path: string
  items: BrowseItem[]
  loading: boolean
  canary: boolean
  selectedItem: BrowseItem | null
  uploading: boolean
  isRemoteLocked: boolean
  remotes: RemoteEntry[] | Array<Record<string, any>>
  activeProfile: Profile | null
}>()

const emit = defineEmits<{
  (e: 'update:remote', remote: string): void
  (e: 'navigate', path: string): void
  (e: 'refresh'): void
  (e: 'touchCanary'): void
  (e: 'mkdir'): void
  (e: 'upDir'): void
  (e: 'uploadFiles', files: File[]): void
  (e: 'itemClick', item: BrowseItem): void
  (e: 'itemDblClick', item: BrowseItem): void
  (e: 'contextmenu', event: MouseEvent, item: BrowseItem): void
  (e: 'toggleCompare', item: BrowseItem): void
  (e: 'nameClick', item: BrowseItem): void
  (e: 'openOnRemote', item: BrowseItem): void
  (e: 'crossPaneDrop', payload: ExplorerDragPayload): void
}>()

const explorerStore = useExplorerStore()
const fileInputRef = ref<HTMLInputElement | null>(null)
const isDragOver = ref(false)
const copied = ref(false)

const displayPath = computed(() => {
  if (props.isLocal) {
    const root = props.activeProfile?.local_path || 'local'
    return props.path ? `${root}/${props.path}` : root
  }
  return props.path ? `${props.remote}:${props.path}` : `${props.remote}:/`
})

const breadcrumbs = computed<BreadcrumbItem[]>(() => {
  const rootLabel = props.isLocal ? '~' : `${props.remote}:`
  const allCrumbs: BreadcrumbItem[] = [{ name: rootLabel, path: '' }]
  if (!props.path) return allCrumbs

  const parts = props.path.split('/').filter(Boolean)
  let cumulative = ''
  for (const part of parts) {
    cumulative = cumulative ? `${cumulative}/${part}` : part
    allCrumbs.push({ name: part, path: cumulative })
  }

  if (allCrumbs.length <= 4) {
    return allCrumbs
  }

  // Middle collapse deep paths: root / … / parent / current
  const rootCrumb = allCrumbs[0]
  const parentCrumb = allCrumbs[allCrumbs.length - 2]
  const currentCrumb = allCrumbs[allCrumbs.length - 1]
  const middleCrumbs = allCrumbs.slice(1, allCrumbs.length - 2)
  const hiddenPath = middleCrumbs.map((c) => c.name).join('/')

  const ellipsisCrumb: BreadcrumbItem = {
    name: '…',
    path: middleCrumbs[middleCrumbs.length - 1].path,
    title: `Hidden folders: /${hiddenPath}`,
    isEllipsis: true,
  }

  return [rootCrumb, ellipsisCrumb, parentCrumb, currentCrumb]
})

function triggerUploadPicker() {
  if (fileInputRef.value) {
    fileInputRef.value.click()
  }
}

function handleFileInputChange(e: Event) {
  const target = e.target as HTMLInputElement
  const files = Array.from(target.files || [])
  if (files.length > 0) {
    emit('uploadFiles', files)
  }
  target.value = ''
}

function onDragOver(e: DragEvent) {
  const isFiles = e.dataTransfer?.types?.includes('Files')
  const isCrossPane = !!explorerStore.draggedPayload && explorerStore.draggedPayload.sourcePane !== props.pane
  if (isFiles || isCrossPane) {
    e.preventDefault()
    if (e.dataTransfer) {
      e.dataTransfer.dropEffect = 'copy'
    }
    isDragOver.value = true
  }
}

function onDragLeave() {
  isDragOver.value = false
}

function onDrop(e: DragEvent) {
  isDragOver.value = false
  e.preventDefault()

  // 1. External files upload drop
  const files = Array.from(e.dataTransfer?.files || [])
  if (files.length > 0) {
    emit('uploadFiles', files)
    return
  }

  // 2. Cross-pane internal drag-and-drop
  let payload = explorerStore.draggedPayload
  if (!payload && e.dataTransfer) {
    try {
      const raw = e.dataTransfer.getData('application/json')
      if (raw) payload = JSON.parse(raw)
    } catch (_) {}
  }

  if (payload && payload.sourcePane !== props.pane) {
    emit('crossPaneDrop', payload)
    explorerStore.setDraggedPayload(null)
  }
}

function onRemoteSelectChange(e: Event) {
  const target = e.target as HTMLSelectElement
  emit('update:remote', target.value)
}

function handleCopyPath() {
  navigator.clipboard.writeText(displayPath.value).then(() => {
    copied.value = true
    setTimeout(() => {
      copied.value = false
    }, 1500)
  }).catch(() => {})
}
</script>
