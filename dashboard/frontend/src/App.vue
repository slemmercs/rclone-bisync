<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import AppHeader from './components/AppHeader.vue'
import ViewFleet from './views/ViewFleet.vue'
import ViewExplorer from './views/ViewExplorer.vue'
import ViewAudit from './views/ViewAudit.vue'
import { useProfilesStore } from './stores/profiles'
import { useTelemetryStore } from './stores/telemetry'

const activeTab = ref<string>('fleet')
const profilesStore = useProfilesStore()
const telemetryStore = useTelemetryStore()

onMounted(async () => {
  await profilesStore.fetchProfiles()
  telemetryStore.startPolling(3000)
})

onUnmounted(() => {
  telemetryStore.stopPolling()
})
</script>

<template>
  <div class="min-h-screen bg-carbon-950 text-carbon-50 flex flex-col font-sans selection:bg-electric-500/30 selection:text-electric-300">
    <!-- Main Sticky Cockpit Header -->
    <AppHeader :activeTab="activeTab" @update:activeTab="(tab) => (activeTab = tab)" />

    <!-- View Workspace Container -->
    <main class="flex-1 p-6 max-w-7xl w-full mx-auto">
      <ViewFleet v-if="activeTab === 'fleet'" />
      <ViewExplorer v-else-if="activeTab === 'explorer'" />
      <ViewAudit v-else-if="activeTab === 'audit'" />
    </main>

    <!-- Cockpit Telemetry Footer -->
    <footer class="bg-carbon-900 border-t border-carbon-700/80 px-6 py-2.5 text-xs text-carbon-400 flex flex-wrap items-center justify-between gap-4">
      <div class="flex items-center gap-3">
        <span class="flex items-center gap-1.5 font-mono text-[11px]">
          <span class="w-2 h-2 rounded-full" :class="telemetryStore.isSyncing ? 'bg-electric-400 animate-ping' : 'bg-carbon-600'"></span>
          {{ telemetryStore.isSyncing ? 'Engine Active' : 'Engine Idle' }}
        </span>
        <span v-if="telemetryStore.transferStats" class="font-mono text-carbon-300 text-[11px]">
          Speed: {{ telemetryStore.transferStats.speed_str || '0 B/s' }} | {{ telemetryStore.transferStats.percentage.toFixed(1) }}%
        </span>
      </div>

      <div class="flex items-center gap-4 text-[11px] font-mono text-carbon-500">
        <span>FSM Storage Control Plane</span>
        <span>•</span>
        <span>PrimeVue v4 + Tailwind</span>
      </div>
    </footer>
  </div>
</template>

