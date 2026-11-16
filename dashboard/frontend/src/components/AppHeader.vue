<script setup lang="ts">
import { computed } from 'vue'
import FSMStateBadge from './FSMStateBadge.vue'
import { useProfilesStore } from '../stores/profiles'
import { useFSMStore } from '../stores/fsm'
import { useTelemetryStore } from '../stores/telemetry'

const props = defineProps<{
  activeTab: string
}>()

const emit = defineEmits<{
  (e: 'update:activeTab', tab: string): void
  (e: 'openReview'): void
}>()

const profilesStore = useProfilesStore()
const fsmStore = useFSMStore()
const telemetryStore = useTelemetryStore()

const tabs = [
  { id: 'fleet', label: 'Fleet Overview', icon: 'pi pi-server' },
  { id: 'explorer', label: 'Dual Explorer', icon: 'pi pi-folder' },
  { id: 'audit', label: 'Audit Trail', icon: 'pi pi-shield' },
]

const daemonStatus = computed(() => {
  return telemetryStore.daemonHealthy ? 'healthy' : 'degraded'
})

function onProfileChange(event: Event) {
  const target = event.target as HTMLSelectElement
  if (target && target.value) {
    profilesStore.selectProfile(target.value)
  }
}

async function handleRunSync() {
  if (!fsmStore.canTrigger) return
  try {
    await profilesStore.triggerSync({ dryRun: false })
  } catch (err) {
    console.error('Failed to run sync:', err)
  }
}
</script>

<template>
  <header class="bg-carbon-900 border-b border-carbon-700 px-6 py-3 flex flex-wrap items-center justify-between gap-4 sticky top-0 z-50 shadow-md">
    <!-- Brand & Control Plane Identity -->
    <div class="flex items-center gap-3">
      <div class="h-9 w-9 rounded-lg bg-electric-950 border border-electric-600 flex items-center justify-center text-electric-400 font-bold shadow-inner">
        <i class="pi pi-sync text-base"></i>
      </div>
      <div>
        <div class="flex items-center gap-2">
          <span class="font-bold text-carbon-50 text-base tracking-tight">Bisync Hub</span>
          <span class="text-[10px] font-mono px-1.5 py-0.5 rounded bg-carbon-800 text-carbon-400 border border-carbon-700">FSM v2</span>
        </div>
        <div class="flex items-center gap-2 text-xs text-carbon-400">
          <span>Storage Control Plane</span>
          <span class="inline-block w-1 h-1 rounded-full bg-carbon-600"></span>
          <span class="flex items-center gap-1 font-mono text-[11px]" :class="daemonStatus === 'healthy' ? 'text-emerald-400' : 'text-amber-400'">
            <span class="w-1.5 h-1.5 rounded-full" :class="daemonStatus === 'healthy' ? 'bg-emerald-400' : 'bg-amber-400 animate-pulse'"></span>
            {{ daemonStatus === 'healthy' ? 'RC Daemon Online' : 'Daemon Standby' }}
          </span>
        </div>
      </div>
    </div>

    <!-- Navigation Tabs -->
    <nav class="flex items-center bg-carbon-950/70 p-1 rounded-lg border border-carbon-700/80">
      <button
        v-for="tab in tabs"
        :key="tab.id"
        @click="emit('update:activeTab', tab.id)"
        class="flex items-center gap-2 px-3 py-1.5 rounded-md text-xs font-medium transition-all"
        :class="
          props.activeTab === tab.id
            ? 'bg-electric-500 text-white shadow-sm'
            : 'text-carbon-400 hover:text-carbon-50 hover:bg-carbon-800/60'
        "
      >
        <i :class="tab.icon" class="text-xs"></i>
        <span>{{ tab.label }}</span>
      </button>
    </nav>

    <!-- Profile Selector, FSM Badge & Action Center -->
    <div class="flex items-center gap-3">
      <!-- Profile Picker -->
      <div class="flex items-center gap-2 bg-carbon-850 px-2.5 py-1 rounded-md border border-carbon-700">
        <label for="profile-select" class="text-xs text-carbon-400 font-mono">Profile:</label>
        <select
          id="profile-select"
          :value="profilesStore.selectedProfileId"
          @change="onProfileChange"
          class="bg-transparent text-xs text-carbon-50 font-medium focus:outline-none cursor-pointer"
        >
          <option v-for="p in profilesStore.profiles" :key="p.id" :value="p.id" class="bg-carbon-900 text-carbon-50">
            {{ p.name }} ({{ p.remote }})
          </option>
        </select>
      </div>

      <!-- FSM State Badge -->
      <FSMStateBadge :state="fsmStore.currentState" :reason="fsmStore.currentStatus.last_transition?.reason" />

      <!-- Review Alert Trigger (if circuit breaker tripped) -->
      <button
        v-if="telemetryStore.hasReview || fsmStore.pendingReview"
        @click="emit('openReview')"
        class="flex items-center gap-1.5 px-2.5 py-1 rounded-md bg-amber-950 border border-amber-600 text-amber-300 text-xs font-semibold hover:bg-amber-900 transition-colors animate-pulse"
      >
        <i class="pi pi-exclamation-triangle"></i>
        <span>Resolve Conflicts</span>
      </button>

      <!-- Sync Action Button -->
      <button
        @click="handleRunSync"
        :disabled="!fsmStore.canTrigger"
        class="flex items-center gap-2 px-3.5 py-1.5 rounded-md text-xs font-semibold tracking-wide transition-all shadow-sm"
        :class="
          fsmStore.canTrigger
            ? 'bg-electric-500 hover:bg-electric-400 active:bg-electric-600 text-white shadow-[0_0_10px_rgba(14,165,233,0.3)] cursor-pointer'
            : 'bg-carbon-800 text-carbon-500 border border-carbon-700 cursor-not-allowed'
        "
      >
        <i :class="fsmStore.isActive ? 'pi pi-spin pi-spinner' : 'pi pi-play'" class="text-xs"></i>
        <span>{{ fsmStore.isActive ? 'Syncing...' : 'Sync Now' }}</span>
      </button>
    </div>
  </header>
</template>
