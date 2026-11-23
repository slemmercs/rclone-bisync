<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import BandwidthPopover from '../BandwidthPopover.vue'
import { useAppStore } from '../../stores/app'
import { useProfilesStore } from '../../stores/profiles'
import { useFSMStore } from '../../stores/fsm'
import { useTelemetryStore } from '../../stores/telemetry'
import { useLogsStore } from '../../stores/logs'
import { useHelpStore } from '../../stores/help'
import { useSettingsStore } from '../../stores/settings'
import { useQueueStore } from '../../stores/queue'
import { useBandwidthStore } from '../../stores/bandwidth'
import { formatSpeed } from '../../composables/useFormatters'

const props = defineProps<{
  activeTab: string
}>()

const emit = defineEmits<{
  (e: 'openReview'): void
  (e: 'openProfiles'): void
  (e: 'openRemoteWizard'): void
}>()

const appStore = useAppStore()
const profilesStore = useProfilesStore()
const fsmStore = useFSMStore()
const telemetryStore = useTelemetryStore()
const logsStore = useLogsStore()
const helpStore = useHelpStore()
const settingsStore = useSettingsStore()
const queueStore = useQueueStore()
const bandwidthStore = useBandwidthStore()

const bandwidthPopoverRef = ref<InstanceType<typeof BandwidthPopover> | null>(null)

onMounted(() => {
  bandwidthStore.startPolling(5000)
})

onUnmounted(() => {
  bandwidthStore.stopPolling()
})

function handleToggleBandwidth(event: Event) {
  bandwidthPopoverRef.value?.toggle(event)
}

function handleQuickNewProfile() {
  appStore.setQuickCreate(false)
  emit('openProfiles')
}

function handleQuickNewRemote() {
  appStore.setQuickCreate(false)
  emit('openRemoteWizard')
}

function handleOpenShortcuts() {
  appStore.setQuickCreate(false)
  appStore.openShortcutsModal()
}

const activeContext = computed(() => {
  switch (props.activeTab) {
    case 'telemetry':
      return { id: 'view_telemetry', title: 'Engine & Telemetry' }
    case 'remotes':
      return { id: 'view_remotes', title: 'Storage Remotes' }
    case 'explorer':
      return { id: 'view_explorer', title: 'File Explorer' }
    case 'audit':
      return { id: 'view_audit', title: 'Audit Trail' }
    case 'fleet':
    default:
      return { id: 'global', title: 'Fleet Overview' }
  }
})

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
  <div class="flex items-center gap-3">
    <!-- Log Dock Toggle Button -->
    <button
      type="button"
      @click="logsStore.toggleDock()"
      class="flex items-center gap-1.5 px-2.5 py-1 rounded-md border text-xs font-mono transition-colors cursor-pointer"
      :class="logsStore.isDockOpen ? 'bg-electric-950/80 border-electric-500 text-electric-300' : 'bg-carbon-850 hover:bg-carbon-800 border-carbon-700 text-carbon-300'"
      title="Toggle Bisync Log Dock"
    >
      <i class="pi pi-terminal text-xs"></i>
      <span>Logs</span>
      <span class="px-1 py-0.2 rounded bg-carbon-900 border border-carbon-700 text-[10px] text-carbon-400">
        {{ logsStore.logs.length }}
      </span>
    </button>

    <!-- Execution Queue Manager Pill -->
    <button
      type="button"
      @click="queueStore.openModal()"
      class="flex items-center gap-1.5 px-2.5 py-1 rounded-md border text-xs font-mono transition-colors cursor-pointer"
      :class="queueStore.isPaused
        ? 'bg-amber-950/80 border-amber-600 text-amber-300'
        : (queueStore.queuedCount > 0 || queueStore.runningCount > 0)
          ? 'bg-electric-950/80 border-electric-500 text-electric-300'
          : 'bg-carbon-850 hover:bg-carbon-800 border-carbon-700 text-carbon-300'"
      title="Open Execution Queue Manager"
    >
      <i class="pi pi-list text-xs"></i>
      <span>Queue</span>
      <span
        v-if="queueStore.isPaused"
        class="px-1 py-0.2 rounded bg-amber-900/60 border border-amber-700/80 text-[10px] text-amber-300"
      >
        PAUSED
      </span>
      <span
        v-else-if="queueStore.runningCount > 0 || queueStore.queuedCount > 0"
        class="px-1 py-0.2 rounded bg-carbon-900 border border-carbon-700 text-[10px] text-electric-300"
      >
        {{ queueStore.runningCount }}/{{ queueStore.queuedCount }}
      </span>
    </button>

    <!-- Bandwidth Throttling & Live Transfer Button -->
    <button
      type="button"
      @click="handleToggleBandwidth"
      class="flex items-center gap-1.5 px-2.5 py-1 rounded-md border text-xs font-mono transition-colors cursor-pointer"
      :class="bandwidthStore.currentLimit !== 'off' && bandwidthStore.currentLimit !== ''
        ? 'bg-amber-950/80 border-amber-600 text-amber-300'
        : 'bg-carbon-850 hover:bg-carbon-800 border-carbon-700 text-carbon-300'"
      title="Toggle Bandwidth Throttling & Live Transfer Rates"
    >
      <i class="pi pi-gauge text-xs"></i>
      <span>BW:</span>
      <span class="text-white font-bold">{{ bandwidthStore.rateFormatted }}</span>
      <span
        v-if="bandwidthStore.speed > 0"
        class="px-1 py-0.2 rounded bg-electric-950/80 border border-electric-700/80 text-[10px] text-electric-300 font-sans"
      >
        {{ formatSpeed(bandwidthStore.speed) }}
      </span>
    </button>

    <!-- Help & Diagnostics Drawer Button -->
    <button
      type="button"
      @click="helpStore.openDrawer(undefined, '', activeContext)"
      class="flex items-center gap-1.5 px-2.5 py-1 rounded-md bg-carbon-850 hover:bg-carbon-800 border border-carbon-700 text-carbon-300 hover:text-white text-xs font-mono transition-colors cursor-pointer"
      title="Open Reference Manual & Diagnostics Drawer"
    >
      <i class="pi pi-question-circle text-xs text-electric-blue"></i>
      <span>Help</span>
    </button>

    <!-- Global Fleet Settings Button -->
    <button
      type="button"
      @click="settingsStore.openGlobalSettingsModal()"
      class="flex items-center gap-1.5 px-2.5 py-1 rounded-md bg-carbon-850 hover:bg-carbon-800 border border-carbon-700 text-carbon-300 hover:text-white text-xs font-mono transition-colors cursor-pointer"
      title="Open Global Fleet Settings"
    >
      <i class="pi pi-cog text-xs"></i>
      <span>Settings</span>
    </button>

    <!-- Keyboard Shortcuts Dialog Trigger -->
    <button
      type="button"
      @click="appStore.openShortcutsModal()"
      class="flex items-center gap-1 px-2 py-1 rounded-md bg-carbon-850 hover:bg-carbon-800 border border-carbon-700 text-carbon-400 hover:text-white text-xs font-mono transition-colors cursor-pointer"
      title="Open Keyboard Shortcuts Reference (?)"
    >
      <i class="pi pi-compass text-xs"></i>
    </button>
    <!-- Global Quick-Create Menu -->
    <div class="relative">
      <button
        type="button"
        @click="appStore.toggleQuickCreate()"
        class="flex items-center gap-1.5 px-3 py-1 rounded-md bg-electric-600 hover:bg-electric-500 text-white text-xs font-semibold shadow-sm transition-all cursor-pointer"
        title="Global Quick Action (C / Ctrl+K)"
      >
        <i class="pi pi-plus text-xs"></i>
        <span>Create</span>
        <i class="pi pi-chevron-down text-[10px] transition-transform duration-200" :class="appStore.quickCreateOpen ? 'rotate-180' : ''"></i>
      </button>

      <!-- Quick Create Dropdown Menu -->
      <div
        v-if="appStore.quickCreateOpen"
        class="absolute right-0 mt-2 w-56 bg-carbon-900 border border-carbon-700 rounded-xl shadow-2xl z-50 overflow-hidden text-xs py-1"
      >
        <div class="px-3 py-1.5 text-[10px] uppercase font-bold text-carbon-400 border-b border-carbon-800 tracking-wider">
          Quick Actions
        </div>
        <button
          type="button"
          @click="handleQuickNewProfile"
          class="w-full px-3 py-2 text-left hover:bg-carbon-800 text-carbon-100 flex items-center justify-between cursor-pointer transition-colors"
        >
          <div class="flex items-center gap-2">
            <i class="pi pi-folder-plus text-electric-400"></i>
            <span class="font-medium">New Sync Profile</span>
          </div>
          <kbd class="px-1.5 py-0.5 rounded bg-carbon-850 text-carbon-400 border border-carbon-700 text-[10px] font-mono">N</kbd>
        </button>
        <button
          type="button"
          @click="handleQuickNewRemote"
          class="w-full px-3 py-2 text-left hover:bg-carbon-800 text-carbon-100 flex items-center justify-between cursor-pointer transition-colors"
        >
          <div class="flex items-center gap-2">
            <i class="pi pi-cloud-upload text-emerald-400"></i>
            <span class="font-medium">Connect Remote</span>
          </div>
          <kbd class="px-1.5 py-0.5 rounded bg-carbon-850 text-carbon-400 border border-carbon-700 text-[10px] font-mono">R</kbd>
        </button>
        <div class="border-t border-carbon-800 my-1"></div>
        <button
          type="button"
          @click="handleOpenShortcuts"
          class="w-full px-3 py-1.5 text-left hover:bg-carbon-800 text-carbon-400 hover:text-carbon-200 flex items-center justify-between cursor-pointer transition-colors"
        >
          <div class="flex items-center gap-2">
            <i class="pi pi-compass text-carbon-400"></i>
            <span>Keyboard Shortcuts</span>
          </div>
          <kbd class="px-1.5 py-0.5 rounded bg-carbon-850 text-carbon-400 border border-carbon-700 text-[10px] font-mono">?</kbd>
        </button>
      </div>
    </div>

    <!-- Review Alert Trigger (if circuit breaker tripped) -->
    <button
      v-if="telemetryStore.hasReview || fsmStore.pendingReview"
      type="button"
      @click="$emit('openReview')"
      class="flex items-center gap-1.5 px-2.5 py-1 rounded-md bg-amber-950 border border-amber-600 text-amber-300 text-xs font-semibold hover:bg-amber-900 transition-colors animate-pulse cursor-pointer"
      title="Open Conflict & Circuit Breaker Review Modal"
    >
      <i class="pi pi-exclamation-triangle"></i>
      <span>Review Required</span>
    </button>

    <!-- Pending Deletions Safety Trigger -->
    <button
      v-if="telemetryStore.hasPendingDeletions"
      type="button"
      @click="telemetryStore.openDeletionsModal()"
      class="flex items-center gap-1.5 px-2.5 py-1 rounded-md bg-rose-950 border border-rose-600 text-rose-300 text-xs font-semibold hover:bg-rose-900 transition-colors animate-pulse cursor-pointer"
      title="Inspect and authorize pending mass deletions"
    >
      <i class="pi pi-shield"></i>
      <span>Review Deletions ({{ telemetryStore.pendingDeletions?.total_count || 0 }})</span>
    </button>

    <!-- Sync Action Button -->
    <button
      type="button"
      @click="handleRunSync"
      :disabled="!fsmStore.canTrigger"
      class="flex items-center gap-2 px-3 py-1.5 rounded-md text-xs font-bold transition-all shadow-md cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
      :class="fsmStore.canTrigger
        ? 'bg-electric-600 hover:bg-electric-500 text-white shadow-electric-900/30'
        : 'bg-carbon-800 text-carbon-500 border border-carbon-700'"
    >
      <i class="pi pi-play text-xs"></i>
      <span>Sync Now</span>
    </button>

    <!-- Bandwidth Throttling Popover -->
    <BandwidthPopover ref="bandwidthPopoverRef" />

  </div>
</template>
