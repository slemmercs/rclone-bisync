<template>
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
          @click="$emit('openCompare')"
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

      <!-- Upload Dock Launcher Button -->
      <button
        @click="explorerStore.openUploadDockModal(activeProfileId)"
        class="px-3 py-1.5 bg-carbon-800 hover:bg-carbon-750 text-carbon-200 border border-carbon-700 rounded-xl text-xs font-medium transition-colors flex items-center gap-1.5 cursor-pointer"
        :title="'Upload Dock: ' + (explorerStore.uploadDock.pending_count || 0) + ' pending'"
      >
        <i class="pi pi-upload text-xs text-electric-400"></i>
        <span>Uploads ({{ explorerStore.uploadDock.pending_count || 0 }})</span>
        <span
          v-if="explorerStore.uploadDock.is_sync_active"
          class="text-[9px] px-1 py-0.2 rounded bg-amber-950 text-amber-400 border border-amber-700 font-mono uppercase"
        >
          Deferred
        </span>
      </button>

      <!-- Swap Panes Button -->
      <button
        @click="$emit('swapPanes')"
        class="px-3 py-1.5 bg-carbon-800 hover:bg-carbon-750 text-carbon-200 border border-carbon-700 rounded-xl text-xs font-medium transition-colors flex items-center gap-1.5 cursor-pointer"
        title="Swap Left and Right Panes"
      >
        <i class="pi pi-arrows-h text-xs text-electric-400"></i>
        <span>Swap Panes</span>
      </button>

      <!-- Refresh Explorer Button -->
      <button
        @click="$emit('refreshAll')"
        class="px-3 py-1.5 bg-carbon-800 hover:bg-carbon-750 text-carbon-200 border border-carbon-700 rounded-xl text-xs font-medium transition-colors flex items-center gap-1.5 cursor-pointer"
        title="Refresh both storage panes"
      >
        <i :class="loading ? 'pi pi-spin pi-spinner' : 'pi pi-refresh'" class="text-xs"></i>
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
</template>

<script setup lang="ts">
import { useExplorerStore } from '../../stores/explorer'
import { useLogsStore } from '../../stores/logs'

defineProps<{
  isSwapped: boolean
  loading: boolean
  activeProfileId?: string
}>()

defineEmits<{
  (e: 'swapPanes'): void
  (e: 'refreshAll'): void
  (e: 'openCompare'): void
}>()

const explorerStore = useExplorerStore()
const logsStore = useLogsStore()
</script>
