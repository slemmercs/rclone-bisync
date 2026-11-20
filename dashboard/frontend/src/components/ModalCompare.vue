<template>
  <Dialog
    :visible="visible"
    @update:visible="(val) => emit('update:visible', val)"
    modal
    header="Cryptographic Checksum Comparison"
    :style="{ width: '52rem', maxWidth: '95vw' }"
    :pt="{
      root: { class: 'bg-carbon-surface border border-carbon-border rounded-2xl text-slate-100 shadow-2xl p-5' },
      header: { class: 'border-b border-carbon-border pb-3 flex items-center justify-between text-white font-bold' },
      content: { class: 'pt-4 space-y-4' },
    }"
  >
    <!-- Loading State -->
    <div v-if="loading" class="py-16 text-center text-slate-400 space-y-3">
      <i class="pi pi-spin pi-spinner text-2xl text-electric-blue"></i>
      <div class="text-xs font-mono">Hashing files and negotiating algorithms across endpoints...</div>
    </div>

    <!-- Error State -->
    <div v-else-if="error" class="p-4 bg-rose-500/10 border border-rose-500/20 text-rose-300 rounded-xl text-xs font-mono">
      {{ error }}
    </div>

    <div v-else-if="comparison" class="space-y-4">
      <!-- Summary Banner -->
      <div
        class="p-3.5 rounded-xl border flex items-center justify-between gap-3 text-xs font-mono"
        :class="comparison.matches ? 'bg-emerald-500/10 border-emerald-500/30 text-emerald-300' : 'bg-amber-500/10 border-amber-500/30 text-amber-300'"
      >
        <div class="flex items-center gap-2">
          <i :class="comparison.matches ? 'pi pi-check-circle text-emerald-400' : 'pi pi-exclamation-triangle text-amber-400'"></i>
          <span class="font-bold">
            {{ comparison.matches ? 'Cryptographic Content Match' : 'Content Discrepancy Detected' }}
          </span>
        </div>
        <span
          class="px-2 py-0.5 rounded text-[10px] uppercase font-bold border"
          :class="comparison.algorithm === 'none'
            ? 'bg-amber-950/80 text-amber-300 border-amber-700/60'
            : 'bg-carbon-elevated text-electric-blue border-carbon-border'"
        >
          {{ comparison.algorithm === 'none' ? 'Fallback: Size Only (No Common Hash)' : `Algorithm: ${comparison.algorithm.toUpperCase()}` }}
        </span>
      </div>

      <!-- Clusters View -->
      <div class="space-y-3 max-h-[55vh] overflow-y-auto pr-1">
        <div
          v-for="(cluster, cIdx) in clusters"
          :key="cIdx"
          class="bg-carbon-canvas border border-carbon-border rounded-xl p-3.5 space-y-2.5"
        >
          <div class="flex items-center justify-between border-b border-carbon-border pb-2 text-xs">
            <div class="flex items-center gap-2">
              <span class="px-2 py-0.5 rounded text-[10px] font-mono font-bold uppercase bg-carbon-elevated text-slate-300 border border-carbon-border">
                Equivalence Group #{{ cIdx + 1 }}
              </span>
              <span class="text-slate-400 text-xs">{{ cluster.length }} file(s)</span>
            </div>
            <div v-if="cluster[0]?.primary_hash" class="font-mono text-[11px] text-slate-400 truncate max-w-sm">
              <span class="text-electric-blue font-semibold">{{ comparison.algorithm.toUpperCase() }}:</span>
              <span class="text-slate-300 ml-1 select-all font-mono">{{ cluster[0].primary_hash }}</span>
            </div>
            <div v-else-if="comparison.algorithm === 'none'" class="font-mono text-[11px] text-slate-500">
              <span>Size: {{ formatBytes(cluster[0]?.size) }}</span>
            </div>
          </div>

          <div class="space-y-1.5">
            <div
              v-for="item in cluster"
              :key="item.remote + ':' + item.path"
              class="px-3 py-2 bg-carbon-surface border border-carbon-border rounded-lg flex items-center justify-between gap-3 text-xs font-mono"
            >
              <div class="flex items-center gap-2 min-w-0 flex-1 truncate">
                <span
                  class="px-1.5 py-0.5 rounded text-[9px] font-bold uppercase border shrink-0"
                  :class="item.remote === 'local' ? 'bg-indigo-500/10 text-indigo-300 border-indigo-500/30' : 'bg-emerald-500/10 text-emerald-300 border-emerald-500/30'"
                >
                  {{ item.remote }}
                </span>
                <span class="text-white font-medium truncate">{{ item.name }}</span>
                <span class="text-slate-500 text-[11px] truncate">({{ item.path }})</span>
              </div>
              <div class="shrink-0 text-slate-400 text-[11px]">
                <span v-if="item.error" class="text-rose-400">{{ item.error }}</span>
                <span v-else>{{ formatBytes(item.size) }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Footer -->
    <div class="flex justify-end border-t border-carbon-border pt-3">
      <button
        @click="emit('update:visible', false)"
        class="px-4 py-1.5 bg-carbon-elevated text-slate-200 hover:bg-slate-800 rounded-xl text-xs font-medium cursor-pointer"
      >
        Close
      </button>
    </div>
  </Dialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import Dialog from 'primevue/dialog'
import type { FileCompareResponse, CompareItemResult } from '../types/explorer'

const props = defineProps<{
  visible: boolean
  comparison: FileCompareResponse | null
  loading?: boolean
  error?: string | null
}>()

const emit = defineEmits<{
  (e: 'update:visible', val: boolean): void
}>()

const clusters = computed<CompareItemResult[][]>(() => {
  if (!props.comparison || !Array.isArray(props.comparison.clusters) || !props.comparison.items) {
    return []
  }
  return props.comparison.clusters.map((indices) => {
    return indices
      .map((idx) => props.comparison!.items[idx])
      .filter(Boolean)
  })
})

function formatBytes(bytes?: number): string {
  if (bytes === undefined || bytes === null || bytes < 0) return '--'
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`
}
</script>
