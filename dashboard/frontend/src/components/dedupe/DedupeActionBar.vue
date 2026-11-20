<template>
  <div
    v-if="dedupeStore.stagedCount > 0"
    class="p-3 bg-carbon-surface border-t border-carbon-border flex flex-wrap items-center justify-between gap-3 shrink-0"
  >
    <div class="flex items-center gap-3">
      <button
        type="button"
        @click="dedupeStore.toggleStagedDrawer()"
        class="px-2.5 py-1 rounded-lg bg-amber-500/20 hover:bg-amber-500/30 text-amber-300 border border-amber-500/40 font-mono text-xs font-bold flex items-center gap-1.5 transition-colors cursor-pointer"
        title="Toggle Staged Actions Drawer"
      >
        <span>{{ dedupeStore.stagedCount }} staged</span>
        <i :class="dedupeStore.drawerOpen ? 'pi pi-chevron-down' : 'pi pi-chevron-up'" class="text-[10px]"></i>
      </button>
      <label class="flex items-center gap-1.5 text-[11px] font-mono text-slate-400 cursor-pointer select-none">
        <input
          type="checkbox"
          v-model="dedupeStore.triggerResync"
          class="rounded bg-carbon-canvas border-carbon-border text-amber-500 focus:ring-0 w-3.5 h-3.5"
        />
        <span>Trigger profile --resync</span>
      </label>
    </div>

    <div class="flex items-center gap-2">
      <button
        type="button"
        @click="dedupeStore.clearStagedActions()"
        class="px-3 py-1 rounded-lg bg-carbon-elevated hover:bg-slate-700 text-slate-300 text-xs font-mono border border-carbon-border transition-colors cursor-pointer"
      >
        Clear
      </button>
      <button
        type="button"
        @click="handleApplyBatch()"
        :disabled="dedupeStore.loading"
        class="px-3.5 py-1 rounded-lg bg-amber-500 hover:bg-amber-400 text-black font-bold text-xs font-mono shadow transition-colors flex items-center gap-1.5 cursor-pointer disabled:opacity-50"
      >
        <i class="pi pi-check"></i>
        <span>Apply Staged Batch</span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useDedupeStore } from '../../stores/dedupe'
import { useToastNotification } from '../../composables/useToastNotification'

const dedupeStore = useDedupeStore()
const toast = useToastNotification()

async function handleApplyBatch() {
  const res = await dedupeStore.applyBatch()
  if (res && res.status !== 'error') {
    toast.success(`Batch deduplication executed successfully (${res.success_count} actions applied).`, 'Dedupe Batch')
  } else if (dedupeStore.error) {
    toast.error(dedupeStore.error, 'Dedupe Batch Failed')
  }
}
</script>
