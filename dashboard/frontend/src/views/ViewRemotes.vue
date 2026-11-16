<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRemotesStore } from '../stores/remotes'

const remotesStore = useRemotesStore()
const remotes = computed(() => remotesStore.remotes)
const summary = computed(() => remotesStore.summary)
const loading = computed(() => remotesStore.loading)

const quotaPct = computed(() => {
  const used = Number(summary.value?.total_used_bytes || 0)
  const total = Number(summary.value?.total_capacity_bytes || 0)
  if (total <= 0) return '0.0'
  return ((used / total) * 100).toFixed(1)
})

function formatBytes(bytes?: number): string {
  if (!bytes || bytes <= 0) return '-- GB'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`
}

function checkAuth(name?: string) {
  remotesStore.checkAuth(name)
}

onMounted(() => {
  remotesStore.fetchRemotes()
})
</script>

<template>
  <div class="space-y-6">
    <div class="grid grid-cols-2 sm:grid-cols-4 gap-4">
      <div class="bg-carbon-surface p-4 rounded-2xl border border-carbon-border">
        <div class="text-[11px] text-slate-400 uppercase font-semibold">Total Cloud Space</div>
        <div class="text-xl font-bold text-white mt-2 font-mono">{{ formatBytes(summary?.total_capacity_bytes) }}</div>
      </div>
      <div class="bg-carbon-surface p-4 rounded-2xl border border-carbon-border">
        <div class="text-[11px] text-slate-400 uppercase font-semibold">Utilized Storage</div>
        <div class="text-xl font-bold text-electric-blue mt-2 font-mono">{{ formatBytes(summary?.total_used_bytes) }}</div>
        <div class="text-[10px] text-slate-400 mt-1">{{ quotaPct }}% used</div>
      </div>
      <div class="bg-carbon-surface p-4 rounded-2xl border border-carbon-border">
        <div class="text-[11px] text-slate-400 uppercase font-semibold">Available Space</div>
        <div class="text-xl font-bold text-emerald-400 mt-2 font-mono">{{ formatBytes(summary?.total_free_bytes) }}</div>
      </div>
      <div class="bg-carbon-surface p-4 rounded-2xl border border-carbon-border">
        <div class="text-[11px] text-slate-400 uppercase font-semibold">Remotes Count</div>
        <div class="text-xl font-bold text-white mt-2 font-mono">{{ remotes.length }}</div>
      </div>
    </div>

    <div class="bg-carbon-surface p-4 rounded-2xl border border-carbon-border flex items-center justify-between">
      <div class="flex items-center gap-3">
        <div class="p-2.5 rounded-xl bg-electric-blue/10 border border-electric-blue/30 text-electric-blue">
          <i class="pi pi-cloud text-base"></i>
        </div>
        <div>
          <span class="font-bold text-white text-sm">Storage Remotes</span>
          <div class="text-[11px] text-slate-400">Cloud object storage endpoints</div>
        </div>
      </div>
      <button @click="checkAuth()" :disabled="loading" class="px-3 py-1.5 bg-carbon-elevated hover:bg-slate-800 text-electric-blue rounded-xl text-xs font-semibold border border-carbon-border transition-colors flex items-center gap-1.5 cursor-pointer disabled:opacity-50">
        <i class="pi pi-refresh text-xs" :class="{ 'pi-spin': loading }"></i>
        <span>Check Auth</span>
      </button>
    </div>

    <div v-if="remotes.length === 0" class="p-12 text-center text-slate-500 bg-carbon-surface border border-carbon-border rounded-2xl">
      <i class="pi pi-cloud text-4xl text-slate-600 mb-3 block"></i>
      <p class="text-sm font-semibold text-slate-400">No storage remotes found</p>
    </div>

    <div v-else class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <div v-for="remote in remotes" :key="remote.name" class="bg-carbon-surface border border-carbon-border rounded-2xl p-5 space-y-3">
        <div class="flex items-start justify-between">
          <div>
            <h4 class="font-bold text-white text-sm font-mono flex items-center gap-2">
              <span>{{ remote.name }}</span>
              <span class="text-[10px] px-1.5 py-0.5 rounded uppercase bg-carbon-elevated text-slate-400 border border-carbon-border">{{ remote.type || 'remote' }}</span>
            </h4>
            <div class="text-[11px] text-slate-400 font-mono mt-1">{{ remote.endpoint || remote.root || 'Configured endpoint' }}</div>
          </div>
          <span class="text-[10px] font-mono px-2 py-0.5 rounded-full font-bold uppercase" :class="remote.status === 'error' ? 'bg-rose-500/10 text-rose-400 border border-rose-500/30' : 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/30'">
            {{ remote.status === 'error' ? 'ERROR' : 'CONNECTED' }}
          </span>
        </div>
        <div class="pt-2 border-t border-carbon-border flex justify-end">
          <button @click="checkAuth(remote.name)" class="text-electric-blue hover:text-sky-300 text-xs font-mono flex items-center gap-1 cursor-pointer">
            <i class="pi pi-check-circle text-xs"></i><span>Test Auth</span>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
