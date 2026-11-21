<template>
  <div class="flex flex-col flex-1 min-h-0">
    <!-- Column Sort & Filter Header -->
    <div class="px-3 py-1.5 bg-carbon-950/90 border-b border-carbon-800 text-[10px] uppercase font-bold text-slate-500 flex items-center justify-between gap-2 select-none tracking-wider shrink-0 font-sans rounded-t-lg">
      <div class="flex items-center gap-2 flex-1 min-w-0">
        <div class="relative w-28 sm:w-36 shrink-0">
          <input
            v-model="searchQuery"
            type="text"
            placeholder="Filter..."
            class="w-full bg-carbon-900 border border-carbon-800 rounded px-2 py-0.5 text-[10px] text-white placeholder-slate-500 focus:outline-none focus:border-electric-500 transition-colors font-sans normal-case"
          />
          <button
            v-if="searchQuery"
            @click="searchQuery = ''"
            class="absolute inset-y-0 right-0 pr-1.5 flex items-center text-slate-400 hover:text-white text-[10px] cursor-pointer"
          >✕</button>
        </div>
        <span class="text-carbon-700">|</span>
        <button
          type="button"
          @click="toggleSort('name')"
          class="flex items-center gap-1 hover:text-slate-200 transition-colors cursor-pointer"
          title="Sort alphabetically by Name"
        >
          <span>Name</span>
          <span v-if="sortBy === 'name'" class="text-electric-400">{{ sortAsc ? '▲' : '▼' }}</span>
        </button>
        <span class="text-carbon-700">|</span>
        <button
          type="button"
          @click="toggleSort('type')"
          class="flex items-center gap-1 hover:text-slate-200 transition-colors cursor-pointer"
          title="Sort by File Type"
        >
          <span>Type</span>
          <span v-if="sortBy === 'type'" class="text-electric-400">{{ sortAsc ? '▲' : '▼' }}</span>
        </button>
      </div>
      <div class="flex items-center gap-2.5 shrink-0">
        <button
          type="button"
          @click="toggleSort('mtime')"
          class="w-24 text-left hidden sm:flex items-center gap-1 hover:text-slate-200 transition-colors cursor-pointer"
          title="Sort by Last Modified Date"
        >
          <span>Modified</span>
          <span v-if="sortBy === 'mtime'" class="text-electric-400">{{ sortAsc ? '▲' : '▼' }}</span>
        </button>
        <button
          type="button"
          @click="toggleSort('size')"
          class="w-16 text-right flex items-center justify-end gap-1 hover:text-slate-200 transition-colors cursor-pointer"
          title="Sort by File Size"
        >
          <span>Size</span>
          <span v-if="sortBy === 'size'" class="text-electric-400">{{ sortAsc ? '▲' : '▼' }}</span>
        </button>
        <div class="w-4 shrink-0"></div>
      </div>
    </div>

    <!-- Items List -->
    <div class="flex-1 overflow-y-auto max-h-[460px] divide-y divide-carbon-800/40 text-xs font-mono">
      <div v-if="loading" class="py-16 text-center text-carbon-400">
        <i class="pi pi-spin pi-spinner text-xl text-electric-400 mb-2"></i>
        <div class="text-[11px]">Reading directory entries...</div>
      </div>
      <div v-else-if="sortedItems.length === 0" class="py-16 text-center text-carbon-500">
        {{ searchQuery ? 'No matching items found.' : 'Directory is empty.' }}
      </div>
      <div
        v-else
        v-for="item in sortedItems"
        :key="item.Name"
        :data-filename="item.Name"
        :draggable="item.Name !== 'RCLONE_TEST'"
        @dragstart="handleDragStart(item, $event)"
        @dragend="handleDragEnd"
        @click="$emit('itemClick', item)"
        @dblclick="$emit('itemDblClick', item)"
        @contextmenu.prevent="$emit('contextmenu', $event, item)"
        class="flex items-center justify-between px-3 py-2 cursor-pointer select-none transition-colors rounded-lg group"
        :class="[
          selectedItem?.Name === item.Name ? 'bg-carbon-800 text-electric-400 font-semibold' : 'hover:bg-carbon-800/60 text-carbon-200',
          explorerStore.highlightName === item.Name ? 'ring-2 ring-electric-400 bg-electric-950/50 text-electric-300 font-bold animate-pulse' : '',
          item.Name !== 'RCLONE_TEST' ? 'cursor-grab active:cursor-grabbing' : '',
          draggingItemName === item.Name ? 'opacity-40 border border-dashed border-electric-400' : ''
        ]"
      >
        <div class="flex items-center gap-2.5 truncate flex-1 min-w-0">
          <input
            v-if="!item.IsDir && item.Name !== 'RCLONE_TEST'"
            type="checkbox"
            draggable="false"
            :checked="isItemInCompare(item)"
            @mousedown.stop
            @click.stop="$emit('toggleCompare', item)"
            class="rounded bg-carbon-900 border-carbon-700 text-electric-500 cursor-pointer shrink-0"
            title="Select for checksum comparison"
          />
          <span v-else class="w-3.5 shrink-0"></span>

          <i
            class="text-xs shrink-0"
            :class="item.IsDir ? 'pi pi-folder text-amber-400' : 'pi pi-file text-carbon-400'"
          ></i>
          <span
            class="truncate"
            :class="isItemPreviewable(item) ? 'hover:text-electric-300 hover:underline' : ''"
            @click.stop="$emit('nameClick', item)"
          >{{ item.Name }}</span>
        </div>

        <div class="flex items-center gap-2.5 text-[11px] text-carbon-400 shrink-0">
          <span v-if="item.ModTime" class="hidden sm:inline text-carbon-500 font-sans text-[10px] w-24 truncate">
            {{ formatShortDate(item.ModTime) }}
          </span>
          <span v-if="!item.IsDir" class="w-16 text-right">{{ formatBytes(item.Size) }}</span>
          <span v-else class="w-16 text-right text-carbon-500 text-[10px]">DIR</span>

          <button
            v-if="!isLocal && !item.IsDir"
            draggable="false"
            @mousedown.stop
            @click.stop="$emit('openOnRemote', item)"
            class="opacity-0 group-hover:opacity-100 hover:text-electric-400 p-0.5 transition-opacity cursor-pointer"
            title="Open on remote in new tab"
          >
            <i class="pi pi-external-link text-xs"></i>
          </button>
          <span v-else class="w-4 shrink-0"></span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick } from 'vue'
import { useExplorerStore } from '../../stores/explorer'
import { formatBytes } from '../../composables/useFormatters'
import { isItemPreviewable, type ExplorerSortField, type ExplorerDragPayload } from '../../types/explorer'
import type { BrowseItem } from '../../types/models'

const props = defineProps<{
  items: BrowseItem[]
  loading: boolean
  selectedItem: BrowseItem | null
  isLocal: boolean
  remote: string
  currentPath: string
  pane: 'left' | 'right'
}>()

defineEmits<{
  (e: 'itemClick', item: BrowseItem): void
  (e: 'itemDblClick', item: BrowseItem): void
  (e: 'contextmenu', event: MouseEvent, item: BrowseItem): void
  (e: 'toggleCompare', item: BrowseItem): void
  (e: 'nameClick', item: BrowseItem): void
  (e: 'openOnRemote', item: BrowseItem): void
}>()

const explorerStore = useExplorerStore()

const searchQuery = ref('')
const sortBy = ref<ExplorerSortField>('name')
const sortAsc = ref(true)
const draggingItemName = ref<string | null>(null)

function handleDragStart(item: BrowseItem, event: DragEvent) {
  if (item.Name === 'RCLONE_TEST') {
    event.preventDefault()
    return
  }
  draggingItemName.value = item.Name
  const relativePath = getItemRelativePath(item)
  const payload: ExplorerDragPayload = {
    item,
    sourcePane: props.pane,
    sourceRemote: props.remote,
    sourcePath: relativePath,
  }
  explorerStore.setDraggedPayload(payload)

  if (event.dataTransfer) {
    event.dataTransfer.effectAllowed = 'copyMove'
    try {
      event.dataTransfer.setData('application/json', JSON.stringify(payload))
      event.dataTransfer.setData('text/plain', relativePath)
    } catch (_) {}
  }
}

function handleDragEnd() {
  draggingItemName.value = null
  explorerStore.setDraggedPayload(null)
}

function toggleSort(field: ExplorerSortField) {
  if (sortBy.value === field) {
    sortAsc.value = !sortAsc.value
  } else {
    sortBy.value = field
    sortAsc.value = (field === 'name' || field === 'type')
  }
}

function getItemRelativePath(item: BrowseItem): string {
  return props.currentPath ? `${props.currentPath.replace(/\/+$/, '')}/${item.Name}` : item.Name
}

function isItemInCompare(item: BrowseItem): boolean {
  const path = getItemRelativePath(item)
  return explorerStore.isItemInCompare(props.remote, path)
}

function formatShortDate(dateStr?: string): string {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  if (isNaN(d.getTime())) return ''
  return d.toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}

const sortedItems = computed(() => {
  const query = searchQuery.value.toLowerCase().trim()
  let filtered = props.items
  if (query) {
    filtered = filtered.filter(item => (item.Name || '').toLowerCase().includes(query))
  }

  return [...filtered].sort((a, b) => {
    // Folders always sorted first
    if (a.IsDir !== b.IsDir) {
      return a.IsDir ? -1 : 1
    }

    let res = 0
    if (sortBy.value === 'size') {
      const sizeA = a.IsDir ? -1 : (a.Size ?? 0)
      const sizeB = b.IsDir ? -1 : (b.Size ?? 0)
      res = sizeA - sizeB
    } else if (sortBy.value === 'mtime') {
      const timeA = a.ModTime ? new Date(a.ModTime).getTime() : 0
      const timeB = b.ModTime ? new Date(b.ModTime).getTime() : 0
      res = timeA - timeB
    } else if (sortBy.value === 'type') {
      const extA = a.IsDir ? '' : (a.Name.split('.').pop() || '').toLowerCase()
      const extB = b.IsDir ? '' : (b.Name.split('.').pop() || '').toLowerCase()
      res = extA.localeCompare(extB)
      if (res === 0) {
        res = (a.Name || '').localeCompare(b.Name || '', undefined, { numeric: true, sensitivity: 'base' })
      }
    } else {
      res = (a.Name || '').localeCompare(b.Name || '', undefined, { numeric: true, sensitivity: 'base' })
    }

    return sortAsc.value ? res : -res
  })
})

watch(
  [() => props.items, () => explorerStore.highlightName],
  async () => {
    const target = explorerStore.highlightName
    if (!target) return
    await nextTick()
    try {
      const targetEl = document.querySelector(`[data-filename="${CSS.escape(target)}"]`)
      if (targetEl) {
        targetEl.scrollIntoView({ block: 'nearest', behavior: 'smooth' })
      }
    } catch (_) {
      // Fallback if selector escape fails on unusual characters
      const allEls = document.querySelectorAll('[data-filename]')
      for (let i = 0; i < allEls.length; i++) {
        if (allEls[i].getAttribute('data-filename') === target) {
          allEls[i].scrollIntoView({ block: 'nearest', behavior: 'smooth' })
          break
        }
      }
    }
  },
  { deep: true }
)
</script>
