<template>
  <div class="space-y-6">
    <div class="bg-carbon-surface border border-carbon-border p-3.5 rounded-2xl flex flex-wrap items-center justify-between gap-3 shadow-lg shadow-black/40">
      <div class="flex items-center gap-2.5 px-1">
        <span class="w-2.5 h-2.5 rounded-full" :class="isSyncing ? 'bg-amber-400 animate-pulse' : 'bg-emerald-400'"></span>
        <span class="text-xs font-semibold uppercase tracking-wider text-slate-300 font-mono">
          Engine: {{ isSyncing ? 'SYNCING' : (fsmStore.currentState || 'IDLE') }}
        </span>
        <span v-if="activeProfile" class="text-[11px] font-mono px-2 py-0.5 rounded bg-carbon-elevated text-electric-blue border border-carbon-border ml-1">
          {{ activeProfile.name }}
        </span>
      </div>

      <div class="flex flex-wrap items-center gap-2">
        <button @click="handleAction('start_sync')" :disabled="actionLoading !== null" class="px-3 py-1.5 bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 rounded-xl hover:bg-emerald-500/20 text-xs font-semibold transition-colors flex items-center gap-1.5 disabled:opacity-50 cursor-pointer">
          <i v-if="actionLoading === 'start_sync'" class="pi pi-spin pi-spinner text-xs"></i>
          <i v-else class="pi pi-play text-xs"></i>
          <span>{{ actionLoading === 'start_sync' ? 'Executing...' : 'Start Sync' }}</span>
        </button>

        <button @click="handleAction('dry_run')" :disabled="actionLoading !== null" class="px-3 py-1.5 bg-sky-500/10 text-electric-blue border border-sky-500/20 rounded-xl hover:bg-sky-500/20 text-xs font-semibold transition-colors flex items-center gap-1.5 disabled:opacity-50 cursor-pointer">
          <i v-if="actionLoading === 'dry_run'" class="pi pi-spin pi-spinner text-xs"></i>
          <i v-else class="pi pi-eye text-xs"></i>
          <span>{{ actionLoading === 'dry_run' ? 'Simulating...' : 'Dry Run' }}</span>
        </button>

        <button @click="handleAction('resync')" :disabled="actionLoading !== null" class="px-3 py-1.5 bg-amber-500/10 text-amber-300 border border-amber-500/30 rounded-xl hover:bg-amber-500/20 text-xs font-semibold transition-colors flex items-center gap-1.5 disabled:opacity-50 cursor-pointer">
          <i v-if="actionLoading === 'resync'" class="pi pi-spin pi-spinner text-xs"></i>
          <i v-else class="pi pi-refresh text-xs"></i>
          <span>Resync</span>
        </button>

        <button @click="handleAction('force_sync')" :disabled="actionLoading !== null" class="px-3 py-1.5 bg-rose-500/10 text-rose-400 border border-rose-500/30 rounded-xl hover:bg-rose-500/20 text-xs font-semibold transition-colors flex items-center gap-1.5 disabled:opacity-50 cursor-pointer">
          <i v-if="actionLoading === 'force_sync'" class="pi pi-spin pi-spinner text-xs"></i>
          <i v-else class="pi pi-bolt text-xs"></i>
          <span>Force Sync</span>
        </button>
      </div>
    </div>

    <div v-if="actionNotice" class="p-3 bg-sky-950/40 border border-sky-500/30 rounded-xl text-xs text-sky-200 flex items-center justify-between">
      <div class="flex items-center gap-2"><i class="pi pi-info-circle text-sky-400"></i><span>{{ actionNotice }}</span></div>
      <button @click="actionNotice = null" class="text-sky-400 hover:text-sky-100 text-xs cursor-pointer">Dismiss</button>
    </div>

    <TelemetryChart />

    <div class="grid grid-cols-1 md:grid-cols-3 gap-4 text-xs font-mono">
      <div class="bg-carbon-surface p-4 rounded-2xl border border-carbon-border">
        <span class="text-slate-500 block uppercase tracking-wider text-[10px] font-bold mb-1">Local Source</span>
        <span class="text-indigo-300 break-all font-semibold">{{ activeProfile?.local_path || '--' }}</span>
      </div>
      <div class="bg-carbon-surface p-4 rounded-2xl border border-carbon-border">
        <span class="text-slate-500 block uppercase tracking-wider text-[10px] font-bold mb-1">Target Remote</span>
        <span class="text-emerald-400 break-all font-semibold">{{ activeProfile?.remote ? `${activeProfile.remote}:${activeProfile.remote_path || ''}` : '--' }}</span>
      </div>
      <div class="bg-carbon-surface p-4 rounded-2xl border border-carbon-border">
        <span class="text-slate-500 block uppercase tracking-wider text-[10px] font-bold mb-1">Schedule & Mode</span>
        <span class="text-white font-semibold">{{ activeProfile?.schedule || 'manual' }} (Review: {{ activeProfile?.review_mode || 'deletions' }})</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useProfilesStore } from '../stores/profiles'
import { useTelemetryStore } from '../stores/telemetry'
import { useFSMStore } from '../stores/fsm'
import TelemetryChart from '../components/TelemetryChart.vue'

const profilesStore = useProfilesStore()
const telemetryStore = useTelemetryStore()
const fsmStore = useFSMStore()

const activeProfile = computed(() => profilesStore.activeProfile)
const isSyncing = computed(() => telemetryStore.isSyncing)

const actionLoading = ref<string | null>(null)
const actionNotice = ref<string | null>(null)

async function handleAction(action: 'start_sync' | 'dry_run' | 'resync' | 'force_sync') {
  actionLoading.value = action
  actionNotice.value = null
  try {
    const res = await profilesStore.triggerSync({
      dryRun: action === 'dry_run',
      resync: action === 'resync',
      force: action === 'force_sync',
    })
    if (res?.requires_resync) {
      actionNotice.value = res.msg || 'Initial synchronization requires Resync to establish baseline.'
    } else {
      actionNotice.value = res?.msg || `Action ${action} triggered successfully.`
    }
  } catch (e: any) {
    actionNotice.value = `Error: ${e.message}`
  } finally {
    actionLoading.value = null
  }
}
</script>
