<template>
  <Dialog
    :visible="logsStore.daemonModalOpen"
    @update:visible="val => !val && logsStore.closeDaemonModal()"
    modal
    :header="modalTitle"
    :style="{ width: '90vw', maxWidth: '1000px' }"
    :pt="{
      root: { class: 'bg-carbon-surface border border-carbon-border rounded-2xl shadow-2xl overflow-hidden' },
      header: { class: 'p-4 sm:p-5 border-b border-carbon-border bg-carbon-surface flex justify-between items-center text-white' },
      content: { class: 'p-0 bg-carbon-canvas' },
      closeButton: { class: 'text-slate-400 hover:text-white hover:bg-carbon-elevated rounded-lg p-1.5 transition-colors cursor-pointer' }
    }"
  >
    <template #header>
      <div class="flex items-center gap-3">
        <div class="p-2 bg-electric-blue/10 text-electric-blue rounded-xl border border-electric-blue/20 shrink-0">
          <i class="pi pi-terminal text-base"></i>
        </div>
        <div>
          <div class="flex flex-wrap items-center gap-2">
            <h3 class="text-sm font-bold text-white uppercase tracking-wider">{{ modalTitle }}</h3>
            <span v-if="activeRun?.run_id" class="text-[10px] font-mono px-2 py-0.5 rounded bg-indigo-950 text-indigo-300 border border-indigo-800/60">
              {{ activeRun.run_id }}
            </span>
            <span v-if="activeRun?.job_id" class="text-[10px] font-mono px-2 py-0.5 rounded bg-carbon-elevated text-slate-300 border border-carbon-border">
              Job: #{{ activeRun.job_id }}
            </span>
            <span v-if="runFlags.resync" class="px-2 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider bg-amber-500/15 text-amber-300 border border-amber-500/30 font-mono">
              RESYNC
            </span>
            <span v-if="runFlags.dryRun" class="px-2 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider bg-sky-500/15 text-sky-300 border border-sky-500/30 font-mono">
              DRY RUN
            </span>
            <span v-if="runFlags.force" class="px-2 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider bg-rose-500/15 text-rose-300 border border-rose-500/30 font-mono">
              FORCE
            </span>
          </div>
          <div class="text-xs text-slate-400 mt-0.5 flex items-center gap-3 font-mono">
            <span>{{ subTitleText }}</span>
            <span class="text-slate-600">|</span>
            <span>{{ filteredLogs.length }} line(s) loaded</span>
          </div>
        </div>
      </div>

      <div class="flex items-center gap-2 mr-2">
        <button
          @click="copyLogs(false)"
          class="px-2.5 py-1 bg-carbon-elevated hover:bg-slate-700 text-slate-300 rounded-lg text-xs font-mono transition-colors flex items-center gap-1.5 border border-carbon-border cursor-pointer"
        >
          <i class="pi pi-copy text-xs"></i>
          <span>{{ copiedRaw ? 'Copied!' : 'Copy Raw' }}</span>
        </button>
        <button
          @click="copyLogs(true)"
          class="px-2.5 py-1 bg-carbon-elevated hover:bg-slate-700 text-slate-300 rounded-lg text-xs font-mono transition-colors flex items-center gap-1.5 border border-carbon-border cursor-pointer"
        >
          <i class="pi pi-code text-xs"></i>
          <span>{{ copiedJSON ? 'Copied JSON!' : 'Copy JSON' }}</span>
        </button>
        <a
          href="/api/logs/daemon/raw"
          target="_blank"
          rel="noopener noreferrer"
          class="px-2.5 py-1 bg-carbon-elevated hover:bg-slate-700 text-electric-blue rounded-lg text-xs font-mono transition-colors flex items-center gap-1.5 border border-carbon-border"
        >
          <span>Raw</span>
          <i class="pi pi-external-link text-xs"></i>
        </a>
      </div>
    </template>

    <!-- Toolbar Filters -->

    <div class="px-4 py-3 bg-carbon-surface/80 border-b border-carbon-border flex flex-wrap items-center justify-between gap-3">
      <div class="flex flex-wrap items-center gap-1.5 text-xs font-mono">
        <button
          @click="levelFilter = 'all'"
          :class="levelFilter === 'all' ? 'bg-electric-blue text-white font-semibold' : 'bg-carbon-elevated text-slate-400 hover:text-white'"
          class="px-2.5 py-1 rounded transition-colors cursor-pointer"
        >
          All
        </button>
        <button
          @click="levelFilter = 'error'"
          :class="levelFilter === 'error' ? 'bg-rose-600 text-white font-semibold' : 'bg-carbon-elevated text-slate-400 hover:text-rose-400'"
          class="px-2.5 py-1 rounded transition-colors flex items-center gap-1 cursor-pointer"
        >
          <span>Errors</span>
          <span class="text-[9px] px-1 py-0.2 rounded bg-black/40 tabular-nums">{{ counts.errors }}</span>
        </button>
        <button
          @click="levelFilter = 'warning'"
          :class="levelFilter === 'warning' ? 'bg-amber-600 text-white font-semibold' : 'bg-carbon-elevated text-slate-400 hover:text-amber-400'"
          class="px-2.5 py-1 rounded transition-colors flex items-center gap-1 cursor-pointer"
        >
          <span>Warnings</span>
          <span class="text-[9px] px-1 py-0.2 rounded bg-black/40 tabular-nums">{{ counts.warnings }}</span>
        </button>
        <button
          @click="levelFilter = 'notice'"
          :class="levelFilter === 'notice' ? 'bg-sky-600 text-white font-semibold' : 'bg-carbon-elevated text-slate-400 hover:text-sky-400'"
          class="px-2.5 py-1 rounded transition-colors flex items-center gap-1 cursor-pointer"
        >
          <span>Notices</span>
          <span class="text-[9px] px-1 py-0.2 rounded bg-black/40 tabular-nums">{{ counts.notices }}</span>
        </button>
        <button
          @click="levelFilter = 'info'"
          :class="levelFilter === 'info' ? 'bg-emerald-600 text-white font-semibold' : 'bg-carbon-elevated text-slate-400 hover:text-emerald-400'"
          class="px-2.5 py-1 rounded transition-colors flex items-center gap-1 cursor-pointer"
        >
          <span>Info</span>
          <span class="text-[9px] px-1 py-0.2 rounded bg-black/40 tabular-nums">{{ counts.info }}</span>
        </button>

        <label class="flex items-center gap-1.5 text-[11px] font-mono text-slate-400 cursor-pointer select-none ml-2">
          <input type="checkbox" v-model="foldRepetitive" class="rounded bg-carbon-elevated border-carbon-border text-electric-blue focus:ring-0 w-3.5 h-3.5">
          <span>Fold</span>
        </label>
        <button
          v-if="foldRepetitive"
          @click="toggleAllClusters()"
          class="text-[10px] font-mono px-2 py-0.5 rounded bg-carbon-elevated hover:bg-slate-700 text-slate-300 border border-carbon-border transition-colors cursor-pointer"
        >
          {{ areAllClustersExpanded ? 'Collapse All' : 'Expand All' }}
        </button>
      </div>

      <div class="flex-1 max-w-xs">
        <input
          type="text"
          v-model="searchQuery"
          placeholder="Search daemon output..."
          class="w-full bg-carbon-elevated border border-carbon-border rounded-lg px-2.5 py-1 text-xs text-slate-200 placeholder-slate-500 focus:outline-none focus:border-electric-blue font-mono transition-colors"
        >
      </div>
    </div>

    <!-- Terminal Viewer Body -->
    <div class="p-4 font-mono text-xs overflow-y-auto h-[28rem] bg-carbon-canvas text-left space-y-1">
      <div v-if="logsStore.daemonLoading" class="text-electric-blue animate-pulse text-center py-10">
        <i class="pi pi-spin pi-spinner mr-2"></i>Loading correlated daemon logs...
      </div>

      <div v-if="logsStore.daemonError" class="p-3 bg-rose-500/10 border border-rose-500/20 rounded-xl text-xs text-rose-300 flex items-center gap-2">
        <i class="pi pi-exclamation-triangle text-rose-400"></i>
        <span>{{ logsStore.daemonError }}</span>
      </div>

      <div v-if="!logsStore.daemonLoading && clusteredLogs.length === 0" class="text-slate-500 text-center py-12 space-y-2">
        <div>No correlated daemon log entries recorded for this execution window.</div>
        <button
          v-if="levelFilter !== 'all' || searchQuery"
          type="button"
          @click="levelFilter = 'all'; searchQuery = ''"
          class="text-xs text-electric-blue hover:underline cursor-pointer font-mono"
        >
          Reset filter and search
        </button>
      </div>

      <template v-for="cluster in clusteredLogs" :key="cluster.id">
        <!-- Single entry -->
        <div v-if="!cluster.isCluster" class="py-1 border-b border-carbon-border/40 hover:bg-carbon-surface/60 rounded px-1 flex flex-col group transition-colors">
          <div class="flex items-start gap-2 text-left">
            <span class="text-slate-500 text-[10px] shrink-0 tabular-nums select-none">[{{ cluster.entry.time || '--:--:--' }}]</span>
            <span :class="cluster.entry.levelClass || 'text-slate-400'" class="shrink-0 text-[10px] uppercase font-bold select-none w-14">
              {{ (cluster.entry.level || 'INFO').trim() }}
            </span>
            <div class="text-slate-200 min-w-0 flex-1 break-words">
              <span class="whitespace-pre-wrap leading-relaxed">{{ cluster.entry.msg }}</span>
              <button
                v-if="getEntryPath(cluster.entry)"
                type="button"
                @click.stop="handleJump(getEntryPath(cluster.entry))"
                class="inline-flex items-center gap-1 text-electric-300 hover:text-white text-[10px] bg-electric-950/60 hover:bg-electric-900 px-1.5 py-0.5 rounded border border-electric-800/60 hover:border-electric-500 ml-1.5 font-sans cursor-pointer transition-colors"
                :title="'Jump to file in Explorer: ' + getEntryPath(cluster.entry)"
              >
                <i class="pi pi-external-link text-[9px] text-electric-400"></i>
                <span>→ {{ getEntryPath(cluster.entry) }}</span>
              </button>
              <span v-else-if="cluster.entry.object" class="text-electric-300/80 text-[10px] bg-electric-950/40 px-1.5 py-0.5 rounded border border-electric-900/40 ml-1.5 font-sans inline-block">
                → {{ cluster.entry.object }}
              </span>
              <span v-if="cluster.entry.objectType" class="text-slate-400 text-[9px] bg-carbon-elevated px-1 py-0.2 rounded border border-carbon-border ml-1 font-mono inline-block">
                {{ cluster.entry.objectType }}
              </span>
            </div>
          </div>

          <!-- Structured Details Grid -->
          <div v-if="cluster.entry.details && typeof cluster.entry.details === 'object'" class="mt-1.5 ml-16 p-2 rounded-lg bg-carbon-900/80 border border-carbon-border/70 grid grid-cols-2 sm:grid-cols-3 gap-1.5 text-[10px] font-mono">
            <div v-for="[k, v] in Object.entries(cluster.entry.details)" :key="k" class="flex items-center justify-between px-2 py-0.5 rounded bg-carbon-950 border border-carbon-border/50">
              <span class="text-carbon-400 truncate mr-2">{{ k }}:</span>
              <span :class="typeof v === 'boolean' ? (v ? 'text-emerald-400 font-semibold' : 'text-slate-500') : 'text-amber-400 font-medium'">{{ String(v) }}</span>
            </div>
          </div>
        </div>

        <!-- Clustered / Folded entries -->
        <div v-else class="border border-carbon-border/50 rounded bg-carbon-elevated/40 overflow-hidden my-1">
          <div
            @click="toggleCluster(cluster.id)"
            class="py-1 px-2 flex items-center justify-between cursor-pointer hover:bg-carbon-elevated transition-colors text-xs select-none"
          >
            <div class="flex items-center gap-2 overflow-hidden">
              <i class="pi text-[10px] text-slate-400" :class="cluster.expanded ? 'pi-chevron-down' : 'pi-chevron-right'"></i>
              <span class="text-slate-600 text-[10px] tabular-nums">{{ cluster.entry.time || '--:--:--' }}</span>
              <span :class="cluster.entry.levelClass || 'text-slate-400'" class="text-[10px] uppercase font-bold">
                [{{ (cluster.entry.level || 'INFO').trim() }}]
              </span>
              <span class="text-slate-300 truncate max-w-md">{{ cluster.entry.msg }}</span>
            </div>
            <span class="text-[10px] px-1.5 py-0.5 rounded bg-electric-blue/20 text-electric-blue font-bold border border-electric-blue/30 shrink-0">
              {{ cluster.count }}× repeated
            </span>
          </div>

          <div v-if="cluster.expanded" class="px-3 py-1.5 bg-carbon-canvas border-t border-carbon-border/40 space-y-1">
            <div v-for="(sub, sIdx) in cluster.entries" :key="sIdx" class="py-1 flex flex-col text-[11px] border-b border-carbon-border/20 last:border-b-0">
              <div class="flex items-start gap-2">
                <span class="text-slate-500 text-[10px] shrink-0 tabular-nums">[{{ sub.time || '--:--:--' }}]</span>
                <div class="text-slate-300 min-w-0 flex-1 break-words">
                  <span class="whitespace-pre-wrap">{{ sub.msg }}</span>
                  <button
                    v-if="getEntryPath(sub)"
                    type="button"
                    @click.stop="handleJump(getEntryPath(sub))"
                    class="inline-flex items-center gap-1 text-electric-300 hover:text-white text-[10px] bg-electric-950/60 hover:bg-electric-900 px-1.5 py-0.5 rounded border border-electric-800/60 hover:border-electric-500 ml-1.5 font-sans cursor-pointer transition-colors"
                    :title="'Jump to file in Explorer: ' + getEntryPath(sub)"
                  >
                    <i class="pi pi-external-link text-[9px] text-electric-400"></i>
                    <span>→ {{ getEntryPath(sub) }}</span>
                  </button>
                </div>
              </div>
              <div v-if="sub.details && typeof sub.details === 'object'" class="mt-1 ml-16 p-2 rounded-lg bg-carbon-900/80 border border-carbon-border/70 grid grid-cols-2 sm:grid-cols-3 gap-1.5 text-[10px] font-mono">
                <div v-for="[sk, sv] in Object.entries(sub.details)" :key="sk" class="flex items-center justify-between px-2 py-0.5 rounded bg-carbon-950 border border-carbon-border/50">
                  <span class="text-carbon-400 truncate mr-2">{{ sk }}:</span>
                  <span :class="typeof sv === 'boolean' ? (sv ? 'text-emerald-400 font-semibold' : 'text-slate-500') : 'text-amber-400 font-medium'">{{ String(sv) }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </template>
    </div>
  </Dialog>

</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import Dialog from 'primevue/dialog'
import { useLogsStore } from '../stores/logs'
import { useExplorerStore } from '../stores/explorer'
import { useAppStore } from '../stores/app'
import type { LogEntry, LogCluster } from '../types/logs'
import { clusterLogEntries, getEntryPath } from '../utils/logClustering'
import { computeLogSeverityCounts, useLogClipboardExport } from '../composables/useLogFilteringAndExport'

const logsStore = useLogsStore()
const explorerStore = useExplorerStore()
const appStore = useAppStore()
const { copiedRaw, copiedJSON, copyLogs: copyLogsExport } = useLogClipboardExport()

function handleJump(path: string | null) {
  if (!path) return
  logsStore.closeDaemonModal()
  appStore.switchTab('explorer')
  explorerStore.jumpToPath(path)
}

const levelFilter = ref<string>('all')
const searchQuery = ref<string>('')
const foldRepetitive = ref<boolean>(true)
const expandedClusters = ref<Record<string, boolean>>({})

const counts = computeLogSeverityCounts(() => logsStore.daemonLogs || [])

const activeRun = computed(() => logsStore.daemonActiveRun)

const modalTitle = computed(() => {
  const run = activeRun.value
  if (run?.type === 'dedupe' || (run?.run_id && run.run_id.startsWith('dedupe-'))) {
    return 'Correlated Deduplication Logs'
  }
  return 'Correlated Daemon Logs'
})

const subTitleText = computed(() => {
  const run = activeRun.value
  if (!run) return 'Continuous stream'
  const parts: string[] = []
  if (run.duration_sec !== undefined && run.duration_sec !== null) {
    parts.push(`${run.duration_sec.toFixed(1)}s`)
  }
  if (run.files_synced !== undefined && run.files_synced !== null) {
    parts.push(`${run.files_synced} transferred`)
  }
  return parts.length > 0 ? parts.join(' • ') : 'Continuous stream'
})

const runFlags = computed(() => {
  const run = activeRun.value
  const details = run?.details || {}
  let resync = !!(details.resync || (run as any)?.resync)
  let dryRun = !!(details.dry_run || details.dryRun || (run as any)?.dry_run || (run as any)?.dryRun)
  let force = !!(details.force || (run as any)?.force)

  const runMsg = (run?.msg || '').toLowerCase()
  const runAction = (run?.action || '').toLowerCase()
  if (!resync && (runMsg.includes('resync started') || runAction === 'resync')) resync = true
  if (!dryRun && (runMsg.includes('dry run started') || runAction === 'dry run')) dryRun = true

  const logs = logsStore.daemonLogs || []
  const scanLimit = Math.min(logs.length, 30)
  for (let i = 0; i < scanLimit; i++) {
    if (resync && dryRun && force) break
    const text = (logs[i].msg || '').toLowerCase()
    if (!resync && (text.includes('--resync') || text.includes('resync started'))) resync = true
    if (!dryRun && (text.includes('--dry-run') || text.includes('dry run started'))) dryRun = true
    if (!force && text.includes('--force')) force = true
  }

  return { resync, dryRun, force }
})

const filteredLogs = computed<LogEntry[]>(() => {
  let list = logsStore.daemonLogs || []
  const filter = levelFilter.value
  if (filter !== 'all') {
    list = list.filter(l => {
      const lvl = (l.level || '').toLowerCase().trim()
      if (filter === 'error') return lvl === 'error'
      if (filter === 'warning') return lvl === 'warning' || lvl === 'warn'
      if (filter === 'notice') return lvl === 'notice'
      if (filter === 'info') return lvl === 'info'
      return true
    })
  }

  const q = searchQuery.value.toLowerCase().trim()
  if (q) {
    list = list.filter(l => (l.msg || '').toLowerCase().includes(q) || (l.object || '').toLowerCase().includes(q) || (l.source || '').toLowerCase().includes(q))
  }
  return list
})

const clusteredLogs = computed(() => {
  return clusterLogEntries(filteredLogs.value, foldRepetitive.value, expandedClusters.value)
})

function toggleCluster(id: string) {
  expandedClusters.value[id] = !expandedClusters.value[id]
}

const areAllClustersExpanded = computed(() => {
  const expandable = clusteredLogs.value.filter((c: LogCluster) => c.isCluster)
  if (expandable.length === 0) return false
  return expandable.every((c: LogCluster) => !!expandedClusters.value[c.id])
})

function toggleAllClusters() {
  const nextState = !areAllClustersExpanded.value
  const next: Record<string, boolean> = {}
  for (const c of clusteredLogs.value as LogCluster[]) {
    if (c.isCluster) {
      next[c.id] = nextState
    }
  }
  expandedClusters.value = next
}


async function copyLogs(asJSON: boolean) {
  await copyLogsExport(filteredLogs.value, asJSON)
}
</script>

