<template>
  <div class="flex-1 flex flex-col min-h-0 overflow-hidden">
    <!-- Collision Search & Filter Controls Toolbar -->
    <div class="p-3 bg-carbon-surface/80 border-b border-carbon-border flex flex-col sm:flex-row sm:items-center justify-between gap-2 shrink-0">
      <!-- Search Input -->
      <div class="relative flex-1 max-w-sm">
        <i class="pi pi-search absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-500 text-xs"></i>
        <input
          v-model="collisionSearch"
          type="text"
          placeholder="Filter by filename, directory, or ID..."
          class="w-full pl-7 pr-7 py-1 bg-carbon-canvas border border-carbon-border rounded-lg text-xs font-mono text-slate-200 placeholder-slate-500 focus:outline-none focus:border-amber-500 transition-colors"
        />
        <button
          v-if="collisionSearch"
          @click="collisionSearch = ''"
          class="absolute right-2 top-1/2 -translate-y-1/2 text-slate-400 hover:text-white cursor-pointer"
          title="Clear search"
        >
          <i class="pi pi-times text-[10px]"></i>
        </button>
      </div>

      <!-- Segment Filters & Actions -->
      <div class="flex items-center gap-1.5 flex-wrap font-mono text-xs">
        <button
          v-for="pill in collisionFilterOptions"
          :key="pill.value"
          @click="collisionFilter = pill.value"
          class="px-2 py-0.5 rounded text-[10px] font-semibold border transition-all cursor-pointer flex items-center gap-1 select-none"
          :class="collisionFilter === pill.value
            ? 'bg-amber-950 text-amber-300 border-amber-600/80 shadow-sm'
            : 'bg-carbon-canvas text-slate-400 border-carbon-border hover:bg-carbon-elevated hover:text-slate-200'"
        >
          <span>{{ pill.label }}</span>
          <span class="text-[9px] opacity-80 font-mono">({{ pill.count }})</span>
        </button>

        <div class="h-4 w-px bg-carbon-border mx-1 hidden sm:block"></div>

        <button type="button" @click="collapseAll" class="px-2 py-0.5 rounded bg-carbon-canvas border border-carbon-border text-slate-400 hover:text-white text-[10px] cursor-pointer" title="Collapse all collision items">Collapse All</button>
        <button type="button" @click="expandAll" class="px-2 py-0.5 rounded bg-carbon-canvas border border-carbon-border text-slate-400 hover:text-white text-[10px] cursor-pointer" title="Expand all collision items">Expand All</button>
      </div>
    </div>

    <!-- Active Filter Summary & Reset Bar -->
    <div
      v-if="collisionSearch || collisionFilter !== 'all'"
      class="px-3 py-1.5 bg-amber-500/10 border-b border-amber-500/20 text-amber-300 text-[11px] font-mono flex items-center justify-between gap-2 shrink-0"
    >
      <div class="flex items-center gap-2 truncate">
        <i class="pi pi-filter text-[10px]"></i>
        <span>Showing {{ filteredItems.length }} of {{ dedupeStore.parsed.items.length }} collisions</span>
        <span v-if="collisionSearch" class="truncate text-slate-400">matching "{{ collisionSearch }}"</span>
      </div>
      <button
        @click="resetCollisionFilters"
        class="text-amber-400 hover:underline cursor-pointer shrink-0"
      >
        Reset filters
      </button>
    </div>

    <!-- Collision List -->
    <div class="flex-1 overflow-y-auto p-4 space-y-3">
      <!-- Zero State -->
      <div v-if="dedupeStore.parsed.items.length === 0" class="text-center py-12 text-slate-500 text-xs">
        <i class="pi pi-check-circle text-2xl text-emerald-500/40 mb-2 block"></i>
        <span>No duplicate collisions found. Run a scan to discover duplicates.</span>
      </div>

      <!-- No Filter Match Zero State -->
      <div
        v-else-if="filteredItems.length === 0"
        class="text-center py-12 text-slate-500 text-xs border border-dashed border-carbon-border rounded-xl"
      >
        <i class="pi pi-search text-2xl text-slate-600 mb-2 block"></i>
        <span>No duplicate collisions matching current filter.</span>
        <div class="mt-2">
          <button
            @click="resetCollisionFilters"
            class="px-3 py-1 bg-carbon-elevated hover:bg-slate-700 text-amber-400 rounded-lg text-xs font-mono transition-colors cursor-pointer"
          >
            Clear Search & Filters
          </button>
        </div>
      </div>

      <!-- Collision Item Card -->
      <div
        v-for="item in filteredItems"
        :key="item.path"
        class="p-3.5 bg-carbon-surface border border-carbon-border rounded-xl space-y-2"
      >
        <div class="flex items-start justify-between gap-3">
          <div class="min-w-0 flex items-center gap-2">
            <button
              type="button"
              @click="toggleCollapse(item.path)"
              class="p-1 rounded text-slate-400 hover:text-white hover:bg-carbon-elevated transition-colors cursor-pointer shrink-0"
              :title="collapsedItems[item.path] ? 'Expand duplicate copies' : 'Collapse duplicate copies'"
            >
              <i :class="collapsedItems[item.path] ? 'pi pi-chevron-right' : 'pi pi-chevron-down'" class="text-xs"></i>
            </button>
            <div class="min-w-0">
              <div class="flex items-center gap-2">
                <span class="text-white font-bold text-xs truncate">{{ item.name }}</span>
                <span class="text-[9px] px-1.5 py-0.2 rounded bg-amber-950 text-amber-300 border border-amber-800/60 uppercase font-bold">
                  {{ item.found }} Copies
                </span>
                <span v-if="item.isDir" class="text-[9px] px-1.5 py-0.2 rounded bg-indigo-950 text-indigo-300 border border-indigo-800">DIR</span>
              </div>
              <div class="text-[10px] text-slate-400 truncate mt-0.5">{{ item.path }}</div>
            </div>
          </div>

          <div class="flex items-center gap-1.5 shrink-0">
            <button
              type="button"
              @click="inspectInExplorer(item)"
              class="px-2 py-0.5 rounded bg-carbon-elevated hover:bg-slate-700 text-slate-300 hover:text-white text-[10px] font-mono border border-carbon-border flex items-center gap-1 cursor-pointer transition-colors"
              title="Inspect directory in in-modal Dedupe Explorer"
            >
              <i class="pi pi-folder-open text-[10px] text-amber-400"></i>
              <span class="hidden sm:inline">Explorer</span>
            </button>
            <button
              type="button"
              @click="jumpToMainExplorer(item)"
              class="p-1 rounded bg-carbon-elevated hover:bg-slate-700 text-slate-400 hover:text-white text-[10px] font-mono border border-carbon-border flex items-center cursor-pointer transition-colors"
              title="Open in Dual-Pane File Explorer (closes dedupe modal)"
            >
              <i class="pi pi-external-link text-[10px]"></i>
            </button>

            <!-- Staged Decision Badge -->
            <span
              v-if="dedupeStore.stagedActions[item.path]"
              class="px-2 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider shrink-0"
              :class="dedupeStore.stagedActions[item.path].action === 'delete' ? 'bg-rose-950 text-rose-300 border border-rose-800' : 'bg-emerald-950 text-emerald-300 border border-emerald-800'"
            >
              {{ dedupeStore.stagedActions[item.path].action }}
            </span>
          </div>
        </div>
        <!-- Candidates Details -->
        <div v-if="!collapsedItems[item.path] && item.candidates.length > 0" class="grid grid-cols-1 sm:grid-cols-2 gap-2 pt-2 border-t border-carbon-border/40 text-[10px]">
          <div
            v-for="(cand, cIdx) in item.candidates"
            :key="cIdx"
            class="p-2 rounded bg-carbon-canvas border border-carbon-border flex items-center justify-between gap-2"
          >
            <div class="truncate min-w-0">
              <span class="text-slate-400 mr-1.5">#{{ cand.index || cIdx + 1 }}</span>
              <span v-if="cand.size !== null" class="text-slate-200 mr-2">{{ formatBytes(cand.size) }}</span>
              <span v-if="cand.modtime" class="text-slate-400">{{ cand.modtime }}</span>
              <div v-if="cand.id" class="text-[9px] text-slate-500 font-mono truncate">ID: {{ cand.id }}</div>
              <div v-if="getStagedCand(item, cand)?.new_name" class="text-[9px] text-amber-300 font-mono truncate">
                ➔ {{ getStagedCand(item, cand)?.new_name }}
              </div>
            </div>

            <div class="flex items-center gap-1 shrink-0">
              <template v-if="!getStagedCand(item, cand)">
                <button type="button" @click="dedupeStore.stageKeepOnly(item, cand)" class="px-1.5 py-0.5 rounded bg-emerald-950 hover:bg-emerald-900 text-emerald-300 border border-emerald-800/60 text-[9px] cursor-pointer" title="Keep this copy and stage delete on duplicates">Keep</button>
                <button type="button" @click="promptRename(item, cand)" class="px-1.5 py-0.5 rounded bg-amber-950 hover:bg-amber-900 text-amber-300 border border-amber-800/60 text-[9px] cursor-pointer" title="Rename this copy">Rename</button>
                <button type="button" @click="dedupeStore.stageCandidateAction(item.path, cand, 'delete')" class="px-1.5 py-0.5 rounded bg-rose-950 hover:bg-rose-900 text-rose-300 border border-rose-800/60 text-[9px] cursor-pointer" title="Delete this copy">Delete</button>
              </template>
              <template v-else>
                <span
                  class="px-1.5 py-0.5 rounded text-[9px] font-mono font-bold uppercase border"
                  :class="{
                    'bg-emerald-950/90 text-emerald-300 border-emerald-600/60': getStagedCand(item, cand)?.action === 'keep',
                    'bg-rose-950/90 text-rose-300 border-rose-600/60': getStagedCand(item, cand)?.action === 'delete',
                    'bg-amber-950/90 text-amber-300 border-amber-600/60': getStagedCand(item, cand)?.action === 'rename',
                  }"
                >
                  {{ getStagedCand(item, cand)?.action }}
                </span>
                <button type="button" @click="unstageCand(item, cand)" class="p-0.5 text-slate-400 hover:text-rose-300 cursor-pointer" title="Clear staged action">
                  <i class="pi pi-times text-[10px]"></i>
                </button>
              </template>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useDedupeStore } from '../../stores/dedupe'
import { useExplorerStore } from '../../stores/explorer'
import { useAppStore } from '../../stores/app'
import { formatBytes } from '../../composables/useFormatters'
import { useToastNotification } from '../../composables/useToastNotification'
import type { ParsedDedupeItem, ParsedCandidate } from '../../utils/dedupeParser'
import type { DedupeStageAction } from '../../types/dedupe'

const emit = defineEmits<{
  (e: 'switch-tab', tab: 'items' | 'explorer' | 'raw'): void
}>()

const dedupeStore = useDedupeStore()
const explorerStore = useExplorerStore()
const appStore = useAppStore()
const toast = useToastNotification()

// Candidate search, filter, and collapse state
const collisionSearch = ref<string>('')
type CollisionFilterType = 'all' | 'staged' | 'unresolved' | 'multi'
const collisionFilter = ref<CollisionFilterType>('all')
const collapsedItems = ref<Record<string, boolean>>({})

function toggleCollapse(path: string) {
  collapsedItems.value[path] = !collapsedItems.value[path]
}

function expandAll() {
  collapsedItems.value = {}
}

function collapseAll() {
  const map: Record<string, boolean> = {}
  for (const item of dedupeStore.parsed.items) {
    map[item.path] = true
  }
  collapsedItems.value = map
}

function resetCollisionFilters() {
  collisionSearch.value = ''
  collisionFilter.value = 'all'
}

function inspectInExplorer(item: ParsedDedupeItem) {
  const dirPath = item.isDir
    ? item.path
    : item.path.includes('/')
      ? item.path.substring(0, item.path.lastIndexOf('/'))
      : ''
  dedupeStore.loadExplorer(dirPath)
  emit('switch-tab', 'explorer')
  dedupeStore.activeTab = 'explorer'
}

function jumpToMainExplorer(item: ParsedDedupeItem) {
  explorerStore.jumpToPath(item.path)
  appStore.switchTab('explorer')
  dedupeStore.closeModal()
}

const collisionFilterOptions = computed(() => {
  const items = dedupeStore.parsed.items || []
  const all = items.length
  let staged = 0
  let unresolved = 0
  let multi = 0

  for (const item of items) {
    const hasStaged = dedupeStore.stagedActions[item.path] || item.candidates.some(cand => !!getStagedCand(item, cand))
    if (hasStaged) {
      staged++
    } else {
      unresolved++
    }
    if ((item.found || item.candidates.length) > 2) {
      multi++
    }
  }

  return [
    { label: 'All', value: 'all' as CollisionFilterType, count: all },
    { label: 'Unresolved', value: 'unresolved' as CollisionFilterType, count: unresolved },
    { label: 'Staged', value: 'staged' as CollisionFilterType, count: staged },
    { label: 'Multi-Copy (>2)', value: 'multi' as CollisionFilterType, count: multi },
  ]
})

const filteredItems = computed(() => {
  let list = dedupeStore.parsed.items || []

  if (collisionFilter.value === 'staged') {
    list = list.filter(item => {
      if (dedupeStore.stagedActions[item.path]) return true
      return item.candidates.some(cand => !!getStagedCand(item, cand))
    })
  } else if (collisionFilter.value === 'unresolved') {
    list = list.filter(item => {
      if (dedupeStore.stagedActions[item.path]) return false
      return !item.candidates.some(cand => !!getStagedCand(item, cand))
    })
  } else if (collisionFilter.value === 'multi') {
    list = list.filter(item => (item.found || item.candidates.length) > 2)
  }

  const q = collisionSearch.value.trim().toLowerCase()
  if (q) {
    list = list.filter(item => {
      if (item.name.toLowerCase().includes(q) || item.path.toLowerCase().includes(q)) return true
      return item.candidates.some(cand => 
        (cand.id && cand.id.toLowerCase().includes(q)) ||
        (cand.raw && cand.raw.toLowerCase().includes(q))
      )
    })
  }

  return list
})

function getStagedCand(collision: ParsedDedupeItem, cand: ParsedCandidate): DedupeStageAction | null {
  return dedupeStore.getStagedCandidateAction(collision.path, cand)
}

function unstageCand(collision: ParsedDedupeItem, cand: ParsedCandidate) {
  const key = dedupeStore.getCandidateKey(collision.path, cand.id, cand.index)
  dedupeStore.unstageCandidateAction(key)
}

function promptRename(collision: ParsedDedupeItem, cand: ParsedCandidate) {
  const orig = collision.name || 'file'
  const lastDot = orig.lastIndexOf('.')
  const ext = lastDot !== -1 ? orig.substring(lastDot) : ''
  const base = lastDot !== -1 ? orig.substring(0, lastDot) : orig
  const defaultNew = `${base}_dedup_${cand.index || 1}${ext}`
  const newName = window.prompt(`Enter new unique name for duplicate copy #${cand.index || 1}:`, defaultNew)
  if (newName && newName.trim() && newName.trim() !== orig) {
    dedupeStore.stageCandidateAction(collision.path, cand, 'rename', newName.trim())
    toast.info(`Staged rename to ${newName.trim()}`, 'Dedupe Rename')
  }
}
</script>

