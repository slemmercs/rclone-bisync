<template>
  <div
    v-if="dedupeStore.stagedCount > 0 && dedupeStore.drawerOpen"
    class="border-t border-carbon-border bg-carbon-surface/95 backdrop-blur-md flex flex-col max-h-64 shrink-0 overflow-hidden shadow-2xl transition-all duration-200"
  >
    <!-- Drawer Header -->
    <div class="px-4 py-2 border-b border-carbon-border flex items-center justify-between gap-3 bg-carbon-canvas/80 shrink-0">
      <div class="flex items-center gap-2 flex-wrap">
        <span class="text-xs font-bold text-slate-200 uppercase tracking-wider font-mono">Staged Review</span>
        <span class="px-2 py-0.5 rounded-full bg-carbon-elevated text-slate-300 font-mono text-[11px] border border-carbon-border">
          {{ dedupeStore.stagedCounts.total }} total
        </span>
        <div class="flex items-center gap-1.5 ml-2 text-[11px] font-mono">
          <span
            v-if="dedupeStore.stagedCounts.delete > 0"
            class="px-1.5 py-0.5 rounded bg-rose-950/60 text-rose-300 border border-rose-800/50"
          >
            {{ dedupeStore.stagedCounts.delete }} delete
          </span>
          <span
            v-if="dedupeStore.stagedCounts.rename > 0"
            class="px-1.5 py-0.5 rounded bg-amber-950/60 text-amber-300 border border-amber-800/50"
          >
            {{ dedupeStore.stagedCounts.rename }} rename
          </span>
          <span
            v-if="dedupeStore.stagedCounts.keep > 0"
            class="px-1.5 py-0.5 rounded bg-emerald-950/60 text-emerald-300 border border-emerald-800/50"
          >
            {{ dedupeStore.stagedCounts.keep }} keep
          </span>
        </div>
      </div>
      <button
        type="button"
        @click="dedupeStore.toggleStagedDrawer()"
        class="text-slate-400 hover:text-white p-1 rounded-lg hover:bg-carbon-elevated transition-colors cursor-pointer"
        title="Collapse Drawer"
      >
        <i class="pi pi-chevron-down text-xs"></i>
      </button>
    </div>

    <!-- Batch Progress Indicator -->
    <div
      v-if="dedupeStore.loading || (dedupeStore.progress && dedupeStore.progress.status !== 'idle')"
      class="px-4 py-2 bg-carbon-canvas border-b border-carbon-border space-y-1.5 shrink-0"
    >
      <div class="flex items-center justify-between text-xs font-mono">
        <span class="text-slate-300 truncate">
          {{ dedupeStore.progress.currentAction || 'Applying staged batch...' }}
        </span>
        <span class="text-amber-400 font-bold">
          {{ dedupeStore.progress.percent || 0 }}%
        </span>
      </div>
      <div class="w-full h-1.5 bg-carbon-elevated rounded-full overflow-hidden">
        <div
          class="h-full bg-amber-500 transition-all duration-300 rounded-full"
          :style="{ width: `${dedupeStore.progress.percent || 0}%` }"
          :class="{
            'bg-emerald-500': dedupeStore.progress.status === 'complete',
            'bg-rose-500': dedupeStore.progress.status === 'error',
            'animate-pulse': dedupeStore.loading,
          }"
        ></div>
      </div>
    </div>

    <!-- Staged Action Items List -->
    <div class="flex-1 overflow-y-auto p-2 space-y-1 divide-y divide-carbon-border/50">
      <div
        v-for="item in dedupeStore.stagedEntries"
        :key="item.key"
        class="pt-1 first:pt-0 flex items-center justify-between gap-3 text-xs px-2 py-1.5 rounded-lg hover:bg-carbon-elevated/50 transition-colors"
      >
        <div class="flex items-center gap-2.5 min-w-0">
          <span
            class="px-2 py-0.5 rounded font-mono font-bold text-[10px] uppercase shrink-0 border"
            :class="{
              'bg-rose-950/80 text-rose-300 border-rose-700/50': item.action === 'delete',
              'bg-amber-950/80 text-amber-300 border-amber-700/50': item.action === 'rename',
              'bg-emerald-950/80 text-emerald-300 border-emerald-700/50': item.action === 'keep',
            }"
          >
            {{ item.action }}
          </span>
          <div class="min-w-0">
            <div class="text-slate-200 font-mono truncate text-[11px]">
              <span>{{ item.path }}</span>
              <span class="text-slate-500 ml-1.5">(Copy #{{ item.candidate || 1 }})</span>
            </div>
            <div v-if="item.action === 'rename' && item.new_name" class="text-[10px] text-amber-300/90 font-mono truncate">
              ➔ rename to: <span class="font-bold text-amber-200">{{ item.new_name }}</span>
            </div>
          </div>
        </div>
        <div class="flex items-center gap-2 shrink-0">
          <span class="text-[10px] font-mono text-slate-400">
            {{ formatBytes(item.size || 0) }}
          </span>
          <button
            type="button"
            @click="dedupeStore.unstageCandidateAction(item.key)"
            class="p-1 rounded text-slate-500 hover:text-rose-400 hover:bg-rose-500/10 transition-colors cursor-pointer"
            title="Unstage candidate action"
          >
            <i class="pi pi-times text-xs"></i>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useDedupeStore } from '../../stores/dedupe'
import { formatBytes } from '../../composables/useFormatters'

const dedupeStore = useDedupeStore()
</script>
