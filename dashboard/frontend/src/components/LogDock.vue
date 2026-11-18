<template>
  <div
    class="fixed bottom-0 left-0 right-0 z-40 bg-carbon-900 border-t border-carbon-700 shadow-2xl transition-all duration-200 flex flex-col font-mono"
    :class="{
      'h-10': logsStore.dockHeightPreset === 'collapsed',
      'h-60': logsStore.dockHeightPreset === 'docked',
      'h-96': logsStore.dockHeightPreset === 'expanded',
    }"
  >
    <!-- Dock Header Toolbar -->
    <div class="px-4 py-2 bg-carbon-950 border-b border-carbon-800 flex items-center justify-between gap-3 text-xs select-none shrink-0">
      <div class="flex items-center gap-2 min-w-0">
        <button
          @click="logsStore.toggleDock('docked')"
          class="flex items-center gap-1.5 font-bold text-slate-200 hover:text-white transition-colors cursor-pointer"
          title="Toggle Dock"
        >
          <i class="pi pi-terminal text-electric-400"></i>
          <span class="truncate">Live Sync Log Dock</span>
        </button>

        <!-- Count Badge -->
        <span class="px-1.5 py-0.2 rounded text-[10px] font-mono bg-carbon-800 text-carbon-400 border border-carbon-700/60">
          {{ filteredEntries.length }} entries
        </span>

        <!-- Pagination Controls -->
        <div v-if="logsStore.isDockOpen" class="flex items-center gap-1 text-[10px] font-mono bg-carbon-900 border border-carbon-800 rounded px-1.5 py-0.5">
          <button
            type="button"
            @click="prevPage"
            :disabled="page <= 1"
            class="text-slate-400 hover:text-white disabled:opacity-30 disabled:cursor-not-allowed cursor-pointer"
            title="Previous page"
          >&lt;</button>
          <span class="text-slate-400">Page {{ page }} of {{ totalPages }}</span>
          <button
            type="button"
            @click="nextPage"
            :disabled="page >= totalPages"
            class="text-slate-400 hover:text-white disabled:opacity-30 disabled:cursor-not-allowed cursor-pointer"
            title="Next page"
          >&gt;</button>
        </div>

        <!-- Keyboard Hints -->
        <span v-if="logsStore.isDockOpen" class="hidden md:inline-flex items-center gap-1 text-[10px] text-slate-500 font-mono">
          <kbd class="px-1 py-0.2 bg-carbon-800 text-slate-400 rounded border border-carbon-700 text-[9px]">N</kbd> /
          <kbd class="px-1 py-0.2 bg-carbon-800 text-slate-400 rounded border border-carbon-700 text-[9px]">P</kbd> step
        </span>
      </div>


      <div class="flex items-center gap-2 shrink-0">
        <!-- Category Filter Pills -->
        <div v-if="logsStore.isDockOpen" class="flex items-center bg-carbon-950 p-0.5 rounded-lg border border-carbon-800 text-[10px] font-mono">
          <button
            type="button"
            @click="activeFilter = 'all'"
            :class="activeFilter === 'all' ? 'bg-electric-600 text-white font-semibold' : 'text-slate-400 hover:text-slate-200'"
            class="px-2 py-0.5 rounded transition-all cursor-pointer"
          >All</button>
          <button
            type="button"
            @click="activeFilter = 'file_ops'"
            :class="activeFilter === 'file_ops' ? 'bg-amber-600 text-white font-semibold' : 'text-slate-400 hover:text-slate-200'"
            class="px-2 py-0.5 rounded transition-all cursor-pointer"
            title="Transfers, deletes, uploads, conflicts"
          >File Ops</button>
          <button
            type="button"
            @click="activeFilter = 'sync'"
            :class="activeFilter === 'sync' ? 'bg-indigo-600 text-white font-semibold' : 'text-slate-400 hover:text-slate-200'"
            class="px-2 py-0.5 rounded transition-all cursor-pointer"
          >Sync</button>
          <button
            type="button"
            @click="activeFilter = 'audit'"
            :class="activeFilter === 'audit' ? 'bg-purple-600 text-white font-semibold' : 'text-slate-400 hover:text-slate-200'"
            class="px-2 py-0.5 rounded transition-all cursor-pointer"
          >Audit</button>
        </div>

        <!-- Level Filter Pills -->
        <div v-if="logsStore.isDockOpen" class="flex items-center gap-1 bg-carbon-900 p-0.5 rounded border border-carbon-800 text-[10px]">
          <button
            v-for="lvl in ['ALL', 'ERROR', 'NOTICE']"
            :key="lvl"
            @click="levelFilter = lvl"
            class="px-1.5 py-0.5 rounded transition-colors"
            :class="levelFilter === lvl ? 'bg-carbon-800 text-electric-400 font-semibold' : 'text-carbon-500 hover:text-carbon-200'"
          >{{ lvl }}</button>
        </div>

        <!-- Daemon Stream Modal Button -->
        <button
          v-if="logsStore.isDockOpen"
          @click="logsStore.openDaemonModal(null)"
          class="px-2 py-0.5 rounded bg-electric-500/10 hover:bg-electric-500/20 text-electric-400 border border-electric-500/20 text-[10px] transition-colors flex items-center gap-1 cursor-pointer"
          title="Open Correlated Daemon Log Modal"
        >
          <i class="pi pi-external-link text-[9px]"></i>
          <span>Daemon Stream</span>
        </button>

        <!-- Auto-scroll Toggle -->
        <label v-if="logsStore.isDockOpen" class="flex items-center gap-1 cursor-pointer text-slate-400 hover:text-slate-200 text-[10px]">
          <input type="checkbox" v-model="autoScroll" class="rounded bg-carbon-800 border-carbon-700 text-electric-500 text-xs" />
          <span>Auto</span>
        </label>

        <!-- Height Presets -->
        <button
          v-if="logsStore.dockHeightPreset !== 'expanded'"
          @click="logsStore.dockHeightPreset = 'expanded'"
          class="p-1 text-slate-400 hover:text-white cursor-pointer"
          title="Expand"
        >
          <i class="pi pi-window-maximize text-xs"></i>
        </button>
        <button
          v-if="logsStore.dockHeightPreset !== 'docked'"
          @click="logsStore.dockHeightPreset = 'docked'"
          class="p-1 text-slate-400 hover:text-white cursor-pointer"
          title="Dock"
        >
          <i class="pi pi-window-minimize text-xs"></i>
        </button>
        <button
          @click="logsStore.toggleDock('collapsed')"
          class="p-1 text-slate-400 hover:text-white cursor-pointer"
          :title="logsStore.isDockOpen ? 'Collapse' : 'Open'"
        >
          <i :class="logsStore.isDockOpen ? 'pi pi-chevron-down' : 'pi pi-chevron-up'" class="text-xs"></i>
        </button>
      </div>
    </div>

    <!-- Structured Log Dock Body -->
    <div
      v-if="logsStore.isDockOpen"
      ref="logContainerRef"
      class="flex-1 overflow-y-auto divide-y divide-carbon-800/40 text-[11px] leading-relaxed bg-carbon-950 font-mono select-text"
    >
      <div v-if="paginatedEntries.length === 0" class="p-6 text-center text-slate-500 text-xs">
        No log entries match the active dock filter.
      </div>
      <div
        v-for="(log, idx) in paginatedEntries"
        :key="idx"
        @click="handleRowClick(log, idx)"
        class="px-3 py-1.5 hover:bg-carbon-800/50 flex items-start justify-between gap-3 transition-colors cursor-pointer group"
        :class="selectedIdx === idx ? 'bg-indigo-950/40 border-l-2 border-indigo-500' : ''"
      >
        <div class="flex items-start gap-2 min-w-0 flex-1">
          <!-- Time -->
          <span class="text-slate-500 shrink-0 text-[10px] tabular-nums">[{{ log.time || '--:--:--' }}]</span>

          <!-- Level Badge -->
          <span
            class="px-1.5 py-0.2 rounded text-[9px] font-bold uppercase shrink-0"
            :class="log.levelClass || 'text-slate-400 bg-carbon-800'"
          >
            {{ (log.level || 'INFO').trim() }}
          </span>

          <!-- Action Badge -->
          <span
            v-if="log.action || log.type === 'sync_file'"
            class="px-1.5 py-0.2 rounded text-[9px] font-bold uppercase shrink-0 font-mono border"
            :class="getActionBadgeClass(log)"
          >
            {{ log.action || 'SYNC' }}
          </span>

          <!-- Profile Badge -->
          <span
            v-if="log.profile_name || log.profile_id"
            class="text-[9px] px-1 py-0.2 rounded bg-carbon-800 text-slate-400 border border-carbon-700/60 shrink-0 truncate max-w-[8rem]"
          >
            {{ log.profile_name || log.profile_id }}
          </span>

          <!-- Run ID Button -->
          <button
            v-if="log.run_id"
            type="button"
            @click.stop="logsStore.openDaemonModal(log)"
            class="text-[9px] px-1.5 py-0.2 rounded bg-carbon-800 hover:bg-indigo-950 text-electric-400 hover:text-white border border-carbon-700 hover:border-indigo-600 shrink-0 transition-colors cursor-pointer"
            :title="'Open correlated daemon logs for run ' + log.run_id"
          >
            run:{{ log.run_id.length > 8 ? log.run_id.substring(0, 8) : log.run_id }}
          </button>

          <!-- Clickable File Path Chip -->
          <button
            v-if="getEntryPath(log)"
            type="button"
            @click.stop="handleJump(getEntryPath(log))"
            class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] bg-electric-950/60 text-electric-300 border border-electric-800/60 hover:bg-electric-900 hover:border-electric-500 transition-colors shrink-0 max-w-[14rem] truncate font-mono cursor-pointer"
            :title="'Jump to: ' + getEntryPath(log)"
          >
            <i class="pi pi-external-link text-[9px] text-electric-400 shrink-0"></i>
            <span class="truncate">{{ getEntryPath(log) }}</span>
          </button>

          <!-- Message Text -->
          <span class="text-slate-300 break-words min-w-0 flex-1 select-text">{{ log.msg }}</span>

          <!-- Quick Copy Action Button -->
          <button
            type="button"
            @click.stop="handleCopyLog(log)"
            class="opacity-0 group-hover:opacity-100 p-1 hover:bg-carbon-800 text-slate-400 hover:text-white rounded text-[10px] transition-all cursor-pointer shrink-0"
            title="Copy log entry to clipboard"
          >
            <i class="pi pi-copy"></i>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue'
import { useLogsStore } from '../stores/logs'
import { useExplorerStore } from '../stores/explorer'
import { useAppStore } from '../stores/app'
import { useClipboard } from '../composables/useClipboard'
import { useToastNotification } from '../composables/useToastNotification'
import type { LogEntry } from '../types/logs'
import { getEntryPath } from '../utils/logClustering'

const logsStore = useLogsStore()
const explorerStore = useExplorerStore()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()
const toast = useToastNotification()

async function handleCopyLog(log: LogEntry) {
  const text = log.raw || `[${log.time || '--:--:--'}] ${log.level || 'INFO'} ${log.msg}`
  const ok = await copyToClipboard(text)
  if (ok) {
    toast.info('Copied log entry to clipboard', 'Copied', 2000)
  }
}

const autoScroll = ref<boolean>(true)
const logContainerRef = ref<HTMLDivElement | null>(null)

const activeFilter = ref<'all' | 'file_ops' | 'sync' | 'audit'>('all')
const levelFilter = ref<string>('ALL')
const page = ref<number>(1)
const pageSize = 50
const selectedIdx = ref<number>(-1)

const filteredEntries = computed<LogEntry[]>(() => {
  let list = logsStore.dockEntries || []

  if (activeFilter.value === 'file_ops') {
    list = list.filter(e => e.type === 'sync_file' || e.type === 'sync_overflow' || !!getEntryPath(e) || e.action === 'DELETED' || e.action === 'COPIED')
  } else if (activeFilter.value === 'sync') {
    list = list.filter(e => e.type === 'sync' || e.action?.includes('SYNC'))
  } else if (activeFilter.value === 'audit') {
    list = list.filter(e => e.type === 'audit' || !!e.seq_id)
  }

  if (levelFilter.value !== 'ALL') {
    list = list.filter(e => (e.level || '').toUpperCase().includes(levelFilter.value))
  }

  return list
})

const totalPages = computed(() => Math.max(1, Math.ceil(filteredEntries.value.length / pageSize)))

const paginatedEntries = computed(() => {
  const start = (page.value - 1) * pageSize
  return filteredEntries.value.slice(start, start + pageSize)
})

function prevPage() {
  if (page.value > 1) {
    page.value--
    selectedIdx.value = -1
  }
}

function nextPage() {
  if (page.value < totalPages.value) {
    page.value++
    selectedIdx.value = -1
  }
}

function setCategoryFilter(filter: 'all' | 'file_ops' | 'sync' | 'audit') {
  activeFilter.value = filter
  page.value = 1
  selectedIdx.value = -1
}

function getActionBadgeClass(log: LogEntry): string {
  const act = (log.action || '').toUpperCase()
  if (act.includes('DELETE') || act.includes('PURGE')) return 'bg-rose-500/20 text-rose-300 border-rose-500/40'
  if (act.includes('RENAME') || act.includes('MOVE')) return 'bg-purple-500/20 text-purple-300 border-purple-500/40'
  if (act.includes('CREATE') || act.includes('MKDIR') || act.includes('RESTORE')) return 'bg-emerald-500/20 text-emerald-300 border-emerald-500/40'
  if (log.type === 'sync_overflow') return 'bg-amber-500/20 text-amber-300 border-amber-500/40'
  return 'bg-emerald-500/20 text-emerald-300 border-emerald-500/40'
}

function handleJump(path: string | null) {
  if (!path) return
  appStore.switchTab('explorer')
  explorerStore.jumpToPath(path)
}

function handleRowClick(log: LogEntry, idx: number) {
  selectedIdx.value = idx
  const path = getEntryPath(log)
  if (path) {
    handleJump(path)
  }
}

function logDockStep(direction: number) {
  const currentItems = paginatedEntries.value
  if (!currentItems || currentItems.length === 0) return

  let newIdx = selectedIdx.value + direction
  if (newIdx < 0) {
    if (page.value > 1) {
      prevPage()
      selectedIdx.value = paginatedEntries.value.length - 1
      const item = paginatedEntries.value[selectedIdx.value]
      if (item && getEntryPath(item)) handleJump(getEntryPath(item))
      return
    }
    newIdx = 0
  } else if (newIdx >= currentItems.length) {
    if (page.value < totalPages.value) {
      nextPage()
      selectedIdx.value = 0
      const item = paginatedEntries.value[0]
      if (item && getEntryPath(item)) handleJump(getEntryPath(item))
      return
    }
    newIdx = currentItems.length - 1
  }

  selectedIdx.value = newIdx
  const item = currentItems[newIdx]
  if (item && getEntryPath(item)) {
    handleJump(getEntryPath(item))
  }
}

function handleKeyDown(e: KeyboardEvent) {
  if (!logsStore.isDockOpen) return
  const tag = (e.target as HTMLElement)?.tagName?.toLowerCase() || ''
  if (tag === 'input' || tag === 'textarea' || tag === 'select' || (e.target as HTMLElement)?.isContentEditable) return

  if (e.key === 'n' || e.key === 'N') {
    e.preventDefault()
    logDockStep(1)
  } else if (e.key === 'p' || e.key === 'P') {
    e.preventDefault()
    logDockStep(-1)
  }
}

function scrollToBottom() {
  if (!autoScroll.value || !logContainerRef.value) return
  nextTick(() => {
    if (logContainerRef.value) {
      logContainerRef.value.scrollTop = logContainerRef.value.scrollHeight
    }
  })
}

watch(
  () => logsStore.dockEntries.length,
  () => {
    scrollToBottom()
  }
)

onMounted(() => {
  logsStore.fetchLogs()
  window.addEventListener('keydown', handleKeyDown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeyDown)
})
</script>
