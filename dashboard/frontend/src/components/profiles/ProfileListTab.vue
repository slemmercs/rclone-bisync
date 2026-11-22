<script setup lang="ts">
import { computed } from 'vue'
import { useRemotesStore } from '../../stores/remotes'
import type { Profile } from '../../types/models'

const props = defineProps<{
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

const remotesStore = useRemotesStore()

interface RemotePipelineInfo {
  isEncrypted: boolean
  isAccelerated: boolean
  chain: string[]
  underlyingPath?: string
}

function resolvePipeline(remoteName: string | undefined): RemotePipelineInfo {
  if (!remoteName) return { isEncrypted: false, isAccelerated: false, chain: [] }
  const clean = remoteName.replace(/:+$/, '').trim()
  if (!clean || clean.toLowerCase() === 'local') {
    return { isEncrypted: false, isAccelerated: false, chain: ['local'] }
  }

  let isEncrypted = false
  let isAccelerated = false
  const chain: string[] = [clean]
  let currentName = clean
  const visited = new Set<string>([clean.toLowerCase()])
  let underlyingPath: string | undefined

  while (currentName) {
    const entry = (remotesStore.remotes || []).find(
      r => (r.name || '').toLowerCase() === currentName.toLowerCase()
    )
    if (!entry) break
    const rtype = (entry.type || '').toLowerCase()
    if (rtype === 'crypt') isEncrypted = true
    if (rtype === 'hasher') isAccelerated = true

    if (entry.underlying_path && !underlyingPath) {
      underlyingPath = entry.underlying_path
    }

    if (entry.underlying_remote && !visited.has(entry.underlying_remote.toLowerCase())) {
      chain.push(entry.underlying_remote)
      visited.add(entry.underlying_remote.toLowerCase())
      currentName = entry.underlying_remote
    } else {
      break
    }
  }

  return { isEncrypted, isAccelerated, chain, underlyingPath }
}

const profilePipelines = computed(() => {
  const map: Record<string, RemotePipelineInfo> = {}
  for (const p of props.profiles) {
    map[p.id || p.name] = resolvePipeline(p.remote)
  }
  return map
})
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
          <!-- Virtual Overlay Badges -->
          <span
            v-if="profilePipelines[p.id || p.name]?.isEncrypted"
            class="text-[9px] px-1.5 py-0.5 rounded font-mono font-bold uppercase bg-cyan-500/10 text-cyan-400 border border-cyan-500/20 flex items-center gap-1"
            title="Encrypted overlay remote (Crypt)"
          >
            <i class="pi pi-shield text-[9px]"></i>
            <span>ENCRYPTED</span>
          </span>
          <span
            v-if="profilePipelines[p.id || p.name]?.isAccelerated"
            class="text-[9px] px-1.5 py-0.5 rounded font-mono font-bold uppercase bg-violet-500/10 text-violet-400 border border-violet-500/20 flex items-center gap-1"
            title="Accelerated overlay remote (Hasher)"
          >
            <i class="pi pi-bolt text-[9px]"></i>
            <span>ACCELERATED</span>
          </span>
        </div>
        <div class="text-xs font-mono text-slate-400 flex items-center gap-1 truncate">
          <span class="truncate">{{ p.local_path }}</span>
          <span class="text-slate-600">&harr;</span>
          <span class="text-emerald-400 truncate">{{ p.remote }}{{ p.remote_path ? ':' + p.remote_path : '' }}</span>
        </div>
        <!-- Pipeline Topology Breadcrumbs -->
        <div
          v-if="(profilePipelines[p.id || p.name]?.chain.length || 0) > 1"
          class="text-[10px] font-mono text-slate-400 flex items-center gap-1 flex-wrap pt-0.5"
        >
          <span class="text-slate-500 text-[9px]">Pipeline:</span>
          <span class="px-1.5 py-0.2 rounded bg-slate-800 text-slate-300 border border-slate-700 text-[10px]">local</span>
          <i class="pi pi-arrow-right text-[8px] text-slate-500"></i>
          <template v-for="(node, idx) in profilePipelines[p.id || p.name]?.chain" :key="node">
            <span
              class="px-1.5 py-0.2 rounded border text-[10px]"
              :class="idx === 0 && profilePipelines[p.id || p.name]?.isEncrypted ? 'bg-cyan-950/60 text-cyan-300 border-cyan-800/60' : idx === 0 && profilePipelines[p.id || p.name]?.isAccelerated ? 'bg-violet-950/60 text-violet-300 border-violet-800/60' : 'bg-slate-900 text-slate-400 border-slate-800'"
            >
              {{ node }}
            </span>
            <i v-if="idx < (profilePipelines[p.id || p.name]?.chain.length || 0) - 1" class="pi pi-arrow-right text-[8px] text-slate-500"></i>
          </template>
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
