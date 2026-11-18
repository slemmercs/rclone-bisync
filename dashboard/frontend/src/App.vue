<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import AppHeader from './components/AppHeader.vue'
import ViewFleet from './views/ViewFleet.vue'
import ViewTelemetry from './views/ViewTelemetry.vue'
import ViewRemotes from './views/ViewRemotes.vue'
import ViewExplorer from './views/ViewExplorer.vue'
import ViewAudit from './views/ViewAudit.vue'
import LogDock from './components/LogDock.vue'
import ConflictReviewModal from './components/ConflictReviewModal.vue'
import ProfileManagerModal from './components/ProfileManagerModal.vue'
import ModalDaemonLogs from './components/ModalDaemonLogs.vue'
import HelpDrawer from './components/HelpDrawer.vue'
import ModalSystemHealth from './components/ModalSystemHealth.vue'
import ModalGlobalSettings from './components/ModalGlobalSettings.vue'
import ModalMcp from './components/ModalMcp.vue'
import ModalRemoteWizard from './components/ModalRemoteWizard.vue'
import ModalQueueManager from './components/ModalQueueManager.vue'
import ModalPendingDeletions from './components/ModalPendingDeletions.vue'
import ModalKeyboardShortcuts from './components/ModalKeyboardShortcuts.vue'
import Toast from 'primevue/toast'


import { useAppStore } from './stores/app'
import { useProfilesStore } from './stores/profiles'
import { useTelemetryStore } from './stores/telemetry'
import { useDedupeStore } from './stores/dedupe'
import { useQueueStore } from './stores/queue'
import { useReviewStore } from './stores/review'
import { useKeyboardShortcuts } from './composables/useKeyboardShortcuts'


const appStore = useAppStore()
const dedupeStore = useDedupeStore()
const reviewStore = useReviewStore()
const activeTab = computed({
  get: () => appStore.activeTab,
  set: (val: string) => appStore.switchTab(val)
})
const showReviewModal = ref<boolean>(false)
const showProfilesModal = ref<boolean>(false)
const profilesStore = useProfilesStore()
const telemetryStore = useTelemetryStore()
const queueStore = useQueueStore()

useKeyboardShortcuts({
  onOpenProfiles: () => {
    showProfilesModal.value = true
  },
  onOpenRemoteWizard: () => {
    dedupeStore.openWizard()
  },
  onOpenReview: () => {
    reviewStore.openModal()
  },
})

onMounted(async () => {
  appStore.initPolling()
  await profilesStore.fetchProfiles()
  if (!appStore.isPollingPaused) {
    telemetryStore.startPolling(3000)
    queueStore.startPolling(3000)
  }
})

onUnmounted(() => {
  telemetryStore.stopPolling()
  queueStore.stopPolling()
})
</script>

<template>
  <div class="min-h-screen bg-carbon-950 text-carbon-50 flex flex-col font-sans selection:bg-electric-500/30 selection:text-electric-300 pb-12">
    <!-- Main Sticky Cockpit Header -->
    <AppHeader
      :activeTab="activeTab"
      @update:activeTab="(tab) => (activeTab = tab)"
      @openReview="showReviewModal = true"
      @openProfiles="showProfilesModal = true"
      @openRemoteWizard="dedupeStore.openWizard()"
    />

    <!-- View Workspace Container -->
    <main class="flex-1 p-6 max-w-7xl w-full mx-auto">
      <ViewFleet v-if="activeTab === 'fleet'" />
      <ViewTelemetry v-else-if="activeTab === 'telemetry'" />
      <ViewRemotes v-else-if="activeTab === 'remotes'" />
      <ViewExplorer v-else-if="activeTab === 'explorer'" />
      <ViewAudit v-else-if="activeTab === 'audit'" />
    </main>

    <!-- Cockpit Telemetry Footer -->
    <footer class="bg-carbon-900 border-t border-carbon-700/80 px-6 py-2.5 text-xs text-carbon-400 flex flex-wrap items-center justify-between gap-4 mb-10">
      <div class="flex items-center gap-3">
        <span class="flex items-center gap-1.5 font-mono text-[11px]">
          <span class="w-2 h-2 rounded-full" :class="telemetryStore.isSyncing ? 'bg-electric-400 animate-ping' : 'bg-carbon-600'"></span>
          {{ telemetryStore.isSyncing ? 'Engine Active' : 'Engine Idle' }}
        </span>
        <span v-if="telemetryStore.transferStats" class="font-mono text-carbon-300 text-[11px]">
          Speed: {{ telemetryStore.transferStats.speed_str || '0 B/s' }} | {{ telemetryStore.transferStats.percentage.toFixed(1) }}%
        </span>
        <span class="text-carbon-600 select-none">•</span>
        <button
          type="button"
          @click="queueStore.openModal()"
          class="flex items-center gap-1.5 font-mono text-[11px] text-carbon-400 hover:text-carbon-200 cursor-pointer transition-colors"
          title="Open Execution Queue Manager"
        >
          <span class="w-1.5 h-1.5 rounded-full" :class="queueStore.isPaused ? 'bg-amber-400' : (queueStore.runningCount > 0 ? 'bg-electric-400 animate-pulse' : 'bg-carbon-600')"></span>
          <span>Queue: {{ queueStore.runningCount }} active, {{ queueStore.queuedCount }} queued{{ queueStore.isPaused ? ' (paused)' : '' }}</span>
        </button>

      </div>

      <div class="flex items-center gap-4 text-[11px] font-mono text-carbon-500">
        <span>FSM Storage Control Plane</span>
        <span>•</span>
        <span>PrimeVue v4 + Tailwind</span>
      </div>
    </footer>

    <!-- Pinned Streaming Log Dock -->
    <LogDock />

    <!-- Interactive Modals -->
    <ConflictReviewModal
      :visible="showReviewModal || reviewStore.modalOpen"
      @update:visible="(val) => { showReviewModal = val; reviewStore.modalOpen = val }"
    />
    <ProfileManagerModal
      v-model:visible="showProfilesModal"
      @open-remote-wizard="showProfilesModal = false; dedupeStore.openWizard()"
    />
    <ModalDaemonLogs />
    <HelpDrawer />
    <ModalSystemHealth />
    <ModalGlobalSettings />
    <ModalMcp />
    <ModalRemoteWizard />
    <ModalQueueManager />
    <ModalPendingDeletions />
    <ModalKeyboardShortcuts />
    <Toast position="bottom-right" />



  </div>

</template>


