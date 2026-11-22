<script setup lang="ts">
import type { Profile } from '../../types/models'

defineProps<{
  profiles: Profile[]
}>()

const emit = defineEmits<{
  (e: 'sync', p: Profile): void
  (e: 'history', p: Profile): void
  (e: 'clone', p: Profile): void
  (e: 'edit', p: Profile): void
  (e: 'delete', p: Profile): void
  (e: 'create'): void
}>()
</script>

<template>
  <div class="max-h-[60vh] overflow-y-auto space-y-2.5 pr-1">
    <div
      v-if="profiles.length === 0"
      class="p-8 text-center text-slate-500 text-xs border border-dashed border-carbon-border rounded-xl"
    >
      No synchronization profiles configured. Click "New Profile" to get started.
    </div>
    <div
      v-for="p in profiles"
      :key="p.id"
      class="p-3.5 rounded-xl border border-carbon-border bg-carbon-elevated/40 hover:border-slate-700 transition flex flex-col sm:flex-row sm:items-center justify-between gap-3"
    >
      <div class="space-y-1 min-w-0">
        <div class="flex items-center gap-2 flex-wrap">
          <span class="font-bold text-sm text-white">{{ p.name }}</span>
          <span class="text-[10px] px-1.5 py-0.5 rounded font-mono bg-carbon-elevated text-slate-400 border border-carbon-border">v{{ p.version || 1 }}</span>
          <span class="text-[9px] px-2 py-0.5 rounded font-mono font-bold uppercase bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
            {{ p.state?.last_run_status || 'idle' }}
          </span>
          <span class="text-[10px] px-1.5 py-0.5 rounded font-mono bg-slate-800 text-slate-300">
            {{ p.schedule || 'manual' }}
          </span>
        </div>
        <div class="text-xs font-mono text-slate-400 flex items-center gap-1 truncate">
          <span class="truncate">{{ p.local_path }}</span>
          <span class="text-slate-600">&harr;</span>
          <span class="text-emerald-400 truncate">{{ p.remote }}{{ p.remote_path ? ':' + p.remote_path : '' }}</span>
        </div>
      </div>

      <div class="flex items-center gap-1.5 shrink-0 self-end sm:self-center">
        <button
          type="button"
          @click="emit('sync', p)"
          class="px-2.5 py-1 bg-electric-blue/10 hover:bg-electric-blue/20 text-electric-blue rounded-lg text-xs font-mono border border-electric-blue/30 cursor-pointer transition flex items-center gap-1"
          title="Trigger profile synchronization"
        >
          <i class="pi pi-play text-[10px]"></i><span>Sync</span>
        </button>
        <button
          type="button"
          @click="emit('history', p)"
          class="px-2.5 py-1 bg-carbon-elevated hover:bg-slate-800 text-slate-300 rounded-lg text-xs font-mono border border-carbon-border cursor-pointer transition"
          title="View configuration version history & rollback"
        >
          <i class="pi pi-history text-[10px]"></i>
        </button>
        <button
          type="button"
          @click="emit('clone', p)"
          class="px-2.5 py-1 bg-carbon-elevated hover:bg-slate-800 text-slate-300 rounded-lg text-xs font-mono border border-carbon-border cursor-pointer transition"
          title="Clone profile workload"
        >
          <i class="pi pi-copy text-[10px]"></i>
        </button>
        <button
          type="button"
          @click="emit('edit', p)"
          class="px-2.5 py-1 bg-carbon-elevated hover:bg-slate-800 text-slate-200 rounded-lg text-xs font-mono border border-carbon-border cursor-pointer transition"
        >
          Edit
        </button>
        <button
          type="button"
          @click="emit('delete', p)"
          class="px-2.5 py-1 bg-rose-500/10 hover:bg-rose-500/20 text-rose-400 rounded-lg text-xs font-mono border border-rose-500/30 cursor-pointer transition"
        >
          Delete
        </button>
      </div>
    </div>
  </div>
</template>
