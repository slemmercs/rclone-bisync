<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import FSMStateBadge from './FSMStateBadge.vue'
import BandwidthPopover from './BandwidthPopover.vue'
import { useProfilesStore } from '../stores/profiles'
import { useFSMStore } from '../stores/fsm'
import { useTelemetryStore } from '../stores/telemetry'
import { useLogsStore } from '../stores/logs'
import { useHelpStore } from '../stores/help'
import { useSettingsStore } from '../stores/settings'
import { useQueueStore } from '../stores/queue'
import { useBandwidthStore } from '../stores/bandwidth'
import { formatSpeed } from '../composables/useFormatters'

const props = defineProps<{
  activeTab: string
}>()

const emit = defineEmits<{
  (e: 'update:activeTab', tab: string): void
  (e: 'openReview'): void
  (e: 'openProfiles'): void
}>()

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


const isMcpActive = computed(() => {
  return settingsStore.mcpStatus?.status === 'active' || telemetryStore.telemetry?.mcp_status?.status === 'active'
})

const mcpStatusText = computed(() => {
  return isMcpActive.value ? 'MCP Online' : 'MCP Standby'
})


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


const tabs = [
  { id: 'fleet', label: 'Fleet', icon: 'pi pi-server' },
  { id: 'telemetry', label: 'Engine & Telemetry', icon: 'pi pi-chart-line' },
  { id: 'remotes', label: 'Storage Remotes', icon: 'pi pi-cloud' },
  { id: 'explorer', label: 'File Explorer', icon: 'pi pi-folder' },
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
          <button
            type="button"
            @click="settingsStore.openSystemHealthModal()"
            class="flex items-center gap-1 font-mono text-[11px] cursor-pointer hover:underline"
            :class="daemonStatus === 'healthy' ? 'text-emerald-400' : 'text-amber-400'"
            title="Open System Health & Engine Center"
          >
            <span class="w-1.5 h-1.5 rounded-full" :class="daemonStatus === 'healthy' ? 'bg-emerald-400' : 'bg-amber-400 animate-pulse'"></span>
            {{ daemonStatus === 'healthy' ? 'RC Daemon Online' : 'Daemon Standby' }}
          </button>
          <span class="inline-block w-1 h-1 rounded-full bg-carbon-600"></span>
          <button
            type="button"
            @click="settingsStore.openMcpModal()"
            class="flex items-center gap-1 font-mono text-[11px] cursor-pointer hover:underline"
            :class="isMcpActive ? 'text-violet-400' : 'text-slate-500'"
            title="Open MCP Inspector"
          >
            <span class="w-1.5 h-1.5 rounded-full" :class="isMcpActive ? 'bg-violet-400' : 'bg-slate-600'"></span>
            <span>{{ mcpStatusText }}</span>
          </button>
        </div>
      </div>
    </div>

    <!-- Navigation Tabs -->
    <nav class="flex items-center gap-1 bg-carbon-950 p-1 rounded-lg border border-carbon-800">
      <button
        v-for="tab in tabs"
        :key="tab.id"
        @click="emit('update:activeTab', tab.id)"
        class="flex items-center gap-2 px-3 py-1.5 rounded-md text-xs font-semibold transition-all cursor-pointer select-none"
        :class="activeTab === tab.id
          ? 'bg-carbon-800 text-electric-400 shadow-sm border border-carbon-700'
          : 'text-carbon-400 hover:text-carbon-200 hover:bg-carbon-900'"
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
        <button
          @click="emit('openProfiles')"
          class="text-carbon-400 hover:text-electric-blue text-xs p-1 ml-1 cursor-pointer transition-colors"
          title="Open Sync Profiles Manager"
        >
          <i class="pi pi-cog text-xs"></i>
        </button>
      </div>

      <!-- FSM State Badge -->
      <FSMStateBadge :state="fsmStore.currentState" :reason="fsmStore.currentStatus.last_transition?.reason" />

      <!-- Log Dock Toggle Button -->
      <button
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
      <!-- Help & Diagnostics Drawer Button -->
      <!-- Execution Queue Manager Pill -->
      <button
        @click="queueStore.openModal()"
        class="flex items-center gap-1.5 px-2.5 py-1 rounded-md border text-xs font-mono transition-colors cursor-pointer"
        :class="queueStore.isPaused
          ? 'bg-amber-950/80 border-amber-600 text-amber-300'
          : (queueStore.queuedCount > 0 || queueStore.runningCount > 0)
            ? 'bg-electric-950/80 border-electric-500 text-electric-300'
            : 'bg-carbon-850 hover:bg-carbon-800 border-carbon-700 text-carbon-300'"
        title="Open Execution Queue Manager"
      >
        <i class="pi pi-list text-xs" :class="queueStore.runningCount > 0 ? 'animate-spin' : ''"></i>
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

      <button
        @click="helpStore.openDrawer(undefined, '', activeContext)"
        class="flex items-center gap-1.5 px-2.5 py-1 rounded-md bg-carbon-850 hover:bg-carbon-800 border border-carbon-700 text-carbon-300 hover:text-white text-xs font-mono transition-colors cursor-pointer"
        title="Open Reference Manual & Diagnostics Drawer"
      >
        <i class="pi pi-question-circle text-xs text-electric-blue"></i>
        <span>Help</span>
      </button>
      <!-- Global Fleet Settings Button -->
      <button
        @click="settingsStore.openGlobalSettingsModal()"
        class="flex items-center gap-1.5 px-2.5 py-1 rounded-md bg-carbon-850 hover:bg-carbon-800 border border-carbon-700 text-carbon-300 hover:text-white text-xs font-mono transition-colors cursor-pointer"
        title="Open Global Fleet Settings"
      >
        <i class="pi pi-cog text-xs"></i>
        <span>Settings</span>
      </button>



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
        class="flex items-center gap-2 px-3 py-1.5 rounded-md text-xs font-bold transition-all shadow-md cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
        :class="fsmStore.canTrigger
          ? 'bg-electric-600 hover:bg-electric-500 text-white shadow-electric-900/30'
          : 'bg-carbon-800 text-carbon-500 border border-carbon-700'"
      >
        <i class="pi pi-play text-xs"></i>
        <span>Sync Now</span>
      </button>
    </div>

    <!-- Bandwidth Throttling Popover -->
    <BandwidthPopover ref="bandwidthPopoverRef" />
  </header>
</template>
