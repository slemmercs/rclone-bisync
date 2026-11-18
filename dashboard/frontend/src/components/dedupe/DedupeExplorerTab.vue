<template>
  <div class="flex-1 flex flex-col overflow-hidden">
    <!-- Breadcrumbs Bar -->
    <div class="px-3 py-2 bg-carbon-surface/60 border-b border-carbon-border flex items-center justify-between gap-2 text-xs shrink-0 font-mono">
      <div class="flex items-center gap-1.5 text-slate-300 truncate min-w-0 flex-wrap">
        <span class="text-amber-400 font-bold shrink-0">Path:</span>
        <div v-for="(crumb, idx) in dedupeStore.breadcrumbs" :key="crumb.path" class="flex items-center gap-1 shrink-0">
          <button
            type="button"
            @click="dedupeStore.loadExplorer(crumb.path)"
            class="hover:text-amber-300 hover:underline transition-colors cursor-pointer truncate max-w-[10rem]"
            :class="idx === dedupeStore.breadcrumbs.length - 1 ? 'text-amber-400 font-bold' : 'text-slate-400'"
          >
            {{ crumb.name }}
          </button>
          <span v-if="idx < dedupeStore.breadcrumbs.length - 1" class="text-slate-600">/</span>
        </div>
      </div>
      <div class="flex items-center gap-1 shrink-0">
        <button
          type="button"
          @click="dedupeStore.navigateExplorerParent()"
          :disabled="!dedupeStore.explorerPath"
          class="px-2.5 py-1 rounded bg-carbon-elevated hover:bg-slate-700 disabled:opacity-40 text-slate-300 text-[11px] border border-carbon-border transition-colors cursor-pointer"
        >
          ↑ Parent
        </button>
        <button
          type="button"
          @click="dedupeStore.loadExplorer(dedupeStore.explorerPath)"
          class="px-2.5 py-1 rounded bg-carbon-elevated hover:bg-slate-700 text-slate-300 text-[11px] border border-carbon-border transition-colors cursor-pointer"
        >
          Refresh
        </button>
      </div>
    </div>
    <!-- Items container -->
    <div class="flex-1 overflow-y-auto p-2 space-y-1">
      <div v-if="dedupeStore.explorerLoading" class="text-center py-12 text-slate-400 text-xs space-y-2">
        <i class="pi pi-spin pi-spinner text-amber-400 text-lg"></i>
        <div>Loading directory listing...</div>
      </div>

      <div v-else-if="dedupeStore.explorerError" class="p-6 text-center text-xs space-y-3">
        <div class="text-rose-400 font-mono">{{ dedupeStore.explorerError }}</div>
        <div class="flex items-center justify-center gap-2">
          <button type="button" @click="dedupeStore.loadExplorer(dedupeStore.explorerPath)" class="px-3 py-1 bg-carbon-elevated hover:bg-slate-700 text-slate-200 rounded-lg text-xs font-medium border border-carbon-border cursor-pointer">Retry</button>
          <button type="button" @click="dedupeStore.navigateExplorerParent()" :disabled="!dedupeStore.explorerPath" class="px-3 py-1 bg-carbon-elevated hover:bg-slate-700 text-slate-200 rounded-lg text-xs font-medium disabled:opacity-40 border border-carbon-border cursor-pointer">Go to Parent</button>
        </div>
      </div>

      <template v-else>
        <div
          v-if="dedupeStore.explorerPath"
          @click="dedupeStore.navigateExplorerParent()"
          class="px-2.5 py-1.5 rounded-lg border border-transparent hover:border-carbon-border hover:bg-carbon-elevated/40 text-xs text-slate-400 font-mono cursor-pointer flex items-center gap-2 transition-colors"
        >
          <i class="pi pi-arrow-up text-amber-400 text-xs"></i>
          <span>.. (Parent Directory)</span>
        </div>

        <div
          v-for="item in dedupeStore.explorerItems"
          :key="item.Path || item.Name"
          @click="dedupeStore.selectExplorerItem(item)"
          class="px-2.5 py-1.5 rounded-lg border border-carbon-border/40 hover:border-carbon-border hover:bg-carbon-elevated/50 flex items-center justify-between gap-3 text-xs font-mono transition-colors"
          :class="item.IsDir ? 'cursor-pointer text-amber-300' : 'text-slate-200'"
        >
          <div class="flex items-center gap-2.5 truncate min-w-0">
            <i :class="item.IsDir ? 'pi pi-folder text-amber-400' : 'pi pi-file text-slate-400'" class="text-xs shrink-0"></i>
            <span class="truncate font-medium">{{ item.Name }}</span>
            <span v-if="getCollisionForItem(item)" class="px-1.5 py-0.2 rounded text-[10px] uppercase font-bold bg-amber-950/80 text-amber-300 border border-amber-700/60 shrink-0">Duplicate</span>
          </div>

          <div class="flex items-center gap-3 shrink-0 text-slate-400 text-[11px]" @click.stop>
            <span v-if="!item.IsDir && item.Size !== undefined">{{ formatBytes(item.Size) }}</span>
            <span v-if="item.ModTime" class="hidden sm:inline text-slate-500">{{ formatDate(item.ModTime) }}</span>
            <div v-if="!item.IsDir" class="flex items-center gap-1.5">
              <button type="button" @click.stop="emit('preview', item)" class="p-1 rounded hover:bg-carbon-elevated text-slate-400 hover:text-white transition-colors cursor-pointer" title="Preview file">
                <i class="pi pi-eye text-xs text-indigo-400"></i>
              </button>
              <template v-if="getCollisionForItem(item) && getCandidateForExplorerItem(item)">
                <div v-if="!getStagedForExplorerItem(item)" class="flex items-center gap-1">
                  <button type="button" @click.stop="stageKeep(item)" class="px-1.5 py-0.5 bg-emerald-950/60 hover:bg-emerald-900/60 text-emerald-300 border border-emerald-700/50 rounded text-[10px] font-medium transition-colors cursor-pointer" title="Keep this copy">Keep</button>
                  <button type="button" @click.stop="promptRename(item)" class="px-1.5 py-0.5 bg-amber-950/60 hover:bg-amber-900/60 text-amber-300 border border-amber-700/50 rounded text-[10px] font-medium transition-colors cursor-pointer" title="Rename this copy">Rename</button>
                  <button type="button" @click.stop="stageDelete(item)" class="px-1.5 py-0.5 bg-rose-950/60 hover:bg-rose-900/60 text-rose-300 border border-rose-700/50 rounded text-[10px] font-medium transition-colors cursor-pointer" title="Delete this copy">Delete</button>
                </div>
                <div v-else class="flex items-center gap-1">
                  <span class="px-1.5 py-0.5 rounded text-[10px] font-mono font-bold uppercase border bg-amber-950/90 text-amber-300 border-amber-600/60">{{ getStagedForExplorerItem(item)?.action }}</span>
                  <button type="button" @click.stop="unstage(item)" class="p-0.5 text-slate-400 hover:text-rose-300 cursor-pointer" title="Clear staged action"><i class="pi pi-times text-[10px]"></i></button>
                </div>
              </template>
            </div>
          </div>
        </div>

        <div v-if="!dedupeStore.explorerItems || dedupeStore.explorerItems.length === 0" class="text-center py-12 text-slate-500 text-xs font-mono">Empty directory.</div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useDedupeStore } from '../../stores/dedupe'
import { formatBytes, formatDate } from '../../composables/useFormatters'
import type { BrowseItem } from '../../types/models'
import type { ParsedDedupeItem, ParsedCandidate } from '../../utils/dedupeParser'
import type { DedupeStageAction } from '../../types/dedupe'

const emit = defineEmits<{
  (e: 'preview', item: BrowseItem): void
}>()

const dedupeStore = useDedupeStore()

function getCollisionForItem(item: BrowseItem): ParsedDedupeItem | null {
  if (!item || item.IsDir) return null
  const normName = (item.Name || '').toLowerCase()
  const itemPath = (item.Path || item.Name || '').toLowerCase()
  return (
    dedupeStore.parsed.items.find(
      (c) => c.name.toLowerCase() === normName || c.path.toLowerCase() === itemPath
    ) || null
  )
}

function getCandidateForExplorerItem(item: BrowseItem): ParsedCandidate | null {
  const col = getCollisionForItem(item)
  if (!col || !col.candidates || col.candidates.length === 0) return null
  const itemId = item.ID || item.Id
  if (itemId) {
    const match = col.candidates.find((c) => c.id === itemId)
    if (match) return match
  }
  if (item.Size !== undefined) {
    const match = col.candidates.find((c) => c.size === item.Size)
    if (match) return match
  }
  return col.candidates[0] || null
}

function getStagedForExplorerItem(item: BrowseItem): DedupeStageAction | null {
  const col = getCollisionForItem(item)
  const cand = getCandidateForExplorerItem(item)
  if (!col || !cand) return null
  return dedupeStore.getStagedCandidateAction(col.path, cand)
}

function stageKeep(item: BrowseItem) {
  const col = getCollisionForItem(item)
  const cand = getCandidateForExplorerItem(item)
  if (!col || !cand) return
  dedupeStore.stageKeepOnly(col, cand)
}

function promptRename(item: BrowseItem) {
  const col = getCollisionForItem(item)
  const cand = getCandidateForExplorerItem(item)
  if (!col || !cand) return
  const orig = item.Name || 'file'
  const lastDot = orig.lastIndexOf('.')
  const ext = lastDot !== -1 ? orig.substring(lastDot) : ''
  const base = lastDot !== -1 ? orig.substring(0, lastDot) : orig
  const defaultNew = `${base}_dedup_${cand.index || 1}${ext}`
  const newName = window.prompt(`Enter new unique name for copy #${cand.index || 1}:`, defaultNew)
  if (newName && newName.trim() && newName.trim() !== orig) {
    dedupeStore.stageCandidateAction(col.path, cand, 'rename', newName.trim())
  }
}

function stageDelete(item: BrowseItem) {
  const col = getCollisionForItem(item)
  const cand = getCandidateForExplorerItem(item)
  if (!col || !cand) return
  dedupeStore.stageCandidateAction(col.path, cand, 'delete')
}

function unstage(item: BrowseItem) {
  const col = getCollisionForItem(item)
  const cand = getCandidateForExplorerItem(item)
  if (!col || !cand) return
  const key = dedupeStore.getCandidateKey(col.path, cand.id, cand.index)
  dedupeStore.unstageCandidateAction(key)
}
</script>
