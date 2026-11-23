<script setup lang="ts">
import { ref, computed } from 'vue'
import TelemetryChart from '../components/TelemetryChart.vue'
import FleetKpiRibbon from '../components/fleet/FleetKpiRibbon.vue'
import FleetToolbar from '../components/fleet/FleetToolbar.vue'
import FleetProfileCard from '../components/fleet/FleetProfileCard.vue'
import { useProfilesStore } from '../stores/profiles'
import { useFSMStore } from '../stores/fsm'
import { useTelemetryStore } from '../stores/telemetry'
import { useReviewStore } from '../stores/review'
import { useAppStore } from '../stores/app'
import { useToastNotification } from '../composables/useToastNotification'
import type { Profile } from '../types/models'
import type { SyncFSMState } from '../types/fsm'
import type { StatusFilterType, FleetStatusPill, FleetProfileSyncEvent } from '../types/fleet'

const profilesStore = useProfilesStore()
const fsmStore = useFSMStore()
const telemetryStore = useTelemetryStore()
const reviewStore = useReviewStore()
const appStore = useAppStore()
const toast = useToastNotification()

const isTelemetryCollapsed = ref<boolean>(localStorage.getItem('fleet_telemetry_collapsed') === 'true')

function toggleTelemetryCollapse() {
  isTelemetryCollapsed.value = !isTelemetryCollapsed.value
  try {
    localStorage.setItem('fleet_telemetry_collapsed', String(isTelemetryCollapsed.value))
  } catch {
    // Ignore storage quota or access errors
  }
}

const profiles = computed(() => profilesStore.profiles)

const statusFilter = ref<StatusFilterType>('ALL')
const searchQuery = ref<string>('')

function getProfileState(profile: Profile): SyncFSMState {
  const fsm = fsmStore.statuses[profile.id]
  if (fsm) return fsm.current_state
  if (profile.state?.is_syncing) return 'RUNNING'
  if (profile.state?.last_run_status === 'failed') return 'ABORTED'
  if (profile.state?.last_run_status === 'success') return 'COMPLETED'
  return 'IDLE'
}

const statusPills = computed<FleetStatusPill[]>(() => {
  const all = profiles.value.length
  let active = 0
  let idle = 0
  let failed = 0
  let review = 0

  for (const p of profiles.value) {
    const s = getProfileState(p)
    if (s === 'RUNNING' || s === 'QUEUED' || s === 'RESOLVING') {
      active++
    } else if (s === 'ABORTED' || p.state?.last_run_status === 'failed') {
      failed++
    } else if (s === 'REVIEW_REQUIRED') {
      review++
    } else {
      idle++
    }
  }

  return [
    { label: 'All', value: 'ALL', count: all },
    { label: 'Active', value: 'ACTIVE', count: active },
    { label: 'Idle', value: 'IDLE', count: idle },
    { label: 'Failed', value: 'FAILED', count: failed },
    { label: 'Review Required', value: 'REVIEW_REQUIRED', count: review },
  ]
})

const filteredProfiles = computed(() => {
  let list = profiles.value

  // Status filtering
  if (statusFilter.value !== 'ALL') {
    list = list.filter((p) => {
      const s = getProfileState(p)
      switch (statusFilter.value) {
        case 'ACTIVE':
          return s === 'RUNNING' || s === 'QUEUED' || s === 'RESOLVING'
        case 'FAILED':
          return s === 'ABORTED' || p.state?.last_run_status === 'failed'
        case 'REVIEW_REQUIRED':
          return s === 'REVIEW_REQUIRED'
        case 'IDLE':
          return s === 'IDLE' || s === 'COMPLETED'
        default:
          return true
      }
    })
  }

  // Search query filtering
  const q = searchQuery.value.trim().toLowerCase()
  if (q) {
    list = list.filter((p) => {
      return (
        (p.name && p.name.toLowerCase().includes(q)) ||
        (p.id && p.id.toLowerCase().includes(q)) ||
        (p.local_path && p.local_path.toLowerCase().includes(q)) ||
        (p.remote && p.remote.toLowerCase().includes(q)) ||
        (p.remote_path && p.remote_path.toLowerCase().includes(q))
      )
    })
  }

  return list
})

function handleSelect(id: string) {
  profilesStore.selectProfile(id)
}

function openInExplorer(profile: Profile) {
  profilesStore.selectProfile(profile.id)
  appStore.switchTab('explorer')
}

function openInTelemetry(profile: Profile) {
  profilesStore.selectProfile(profile.id)
  appStore.switchTab('telemetry')
}

function clearFilters() {
  searchQuery.value = ''
  statusFilter.value = 'ALL'
}

async function handleSync(event: FleetProfileSyncEvent) {
  const { profile, dryRun } = event
  profilesStore.selectProfile(profile.id)
  try {
    await profilesStore.triggerSync({ dryRun })
    toast.success(
      `${dryRun ? 'Dry-run' : 'Sync'} dispatched for "${profile.name || profile.id}"`,
      'Sync Triggered'
    )
  } catch (err: any) {
    console.error('Trigger sync error:', err)
    toast.error(err?.message || 'Failed to trigger synchronization', 'Sync Error')
  }
}
</script>

<template>
  <div class="space-y-4">
    <!-- Top KPI Metrics Ribbon -->
    <FleetKpiRibbon
      :total-profiles="profiles.length"
      :active-workloads="profilesStore.enabledProfiles.length"
      :engine-state="fsmStore.currentState"
      :pending-review="fsmStore.pendingReview"
    />
    <!-- Real-Time Telemetry & Throughput Card -->
    <div class="bg-carbon-900 border border-carbon-700 rounded-lg p-3.5 shadow-sm transition-all">
      <div class="flex items-center justify-between" :class="{ 'mb-2': !isTelemetryCollapsed }">
        <div class="flex items-center gap-2">
          <i class="pi pi-chart-line text-electric-400"></i>
          <span class="text-xs font-bold text-carbon-50 uppercase tracking-wide">Live Performance Telemetry</span>
          <span v-if="telemetryStore.isSyncing" class="text-[10px] font-mono px-2 py-0.5 rounded bg-electric-950 text-electric-300 border border-electric-600/50 flex items-center gap-1">
            <span class="w-1.5 h-1.5 rounded-full bg-electric-400 animate-ping"></span>
            Sync Active
          </span>
        </div>
        <div class="flex items-center gap-3 font-mono text-[11px] text-carbon-400">
          <span>Speed: <strong class="text-electric-300">{{ telemetryStore.transferStats?.speed_str || '0 B/s' }}</strong></span>
          <span>Latency: <strong class="text-purple-300">{{ telemetryStore.telemetry?.daemon_health?.response_time_ms ? telemetryStore.telemetry.daemon_health.response_time_ms.toFixed(1) + 'ms' : '0ms' }}</strong></span>
          <span>Status: <strong class="text-carbon-200">{{ telemetryStore.daemonHealthy ? 'ONLINE' : 'STANDBY' }}</strong></span>
          <button
            type="button"
            @click="toggleTelemetryCollapse"
            class="text-carbon-400 hover:text-carbon-100 p-1 rounded hover:bg-carbon-800 transition-colors cursor-pointer ml-1 flex items-center justify-center"
            :title="isTelemetryCollapsed ? 'Expand Telemetry Chart' : 'Collapse Telemetry Chart'"
            :aria-label="isTelemetryCollapsed ? 'Expand Telemetry Chart' : 'Collapse Telemetry Chart'"
          >
            <i :class="isTelemetryCollapsed ? 'pi pi-chevron-down' : 'pi pi-chevron-up'" class="text-xs"></i>
          </button>
        </div>
      </div>
      <TelemetryChart v-if="!isTelemetryCollapsed" />
    </div>

    <!-- Profile Fleet Header & Controls -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
      <div>
        <h2 class="text-base font-semibold text-carbon-50 tracking-tight">Synchronization Workloads</h2>
        <p class="text-xs text-carbon-400">Deterministic bidirectional sync bindings across local storage and remote targets.</p>
      </div>

      <div class="flex items-center gap-2 text-xs font-mono text-carbon-400">
        <span class="px-2 py-0.5 rounded bg-carbon-800 border border-carbon-700">
          Showing <strong class="text-electric-300">{{ filteredProfiles.length }}</strong> of {{ profiles.length }}
        </span>
      </div>
    </div>

    <!-- Search & Filter Controls Toolbar -->
    <FleetToolbar
      v-if="profiles.length > 0"
      v-model:search-query="searchQuery"
      v-model:status-filter="statusFilter"
      :status-pills="statusPills"
    />

    <!-- Profiles Cards Grid -->
    <div v-if="profiles.length === 0" class="bg-carbon-900 border border-carbon-700 rounded-lg p-12 text-center">
      <i class="pi pi-folder-open text-4xl text-carbon-500 mb-3"></i>
      <h3 class="text-sm font-semibold text-carbon-300">No Sync Profiles Configured</h3>
      <p class="text-xs text-carbon-500 mt-1 max-w-sm mx-auto">Create your first bidirectional sync profile to start orchestrating file trees with full safety canaries.</p>
    </div>

    <!-- Empty Search Results -->
    <div v-else-if="filteredProfiles.length === 0" class="bg-carbon-900 border border-carbon-700 rounded-lg p-10 text-center space-y-3">
      <i class="pi pi-filter-slash text-3xl text-carbon-500"></i>
      <div>
        <h3 class="text-sm font-semibold text-carbon-300">No Matching Workloads Found</h3>
        <p class="text-xs text-carbon-500 mt-1 max-w-md mx-auto">
          No synchronization profiles match the active search query or status filter.
        </p>
      </div>
      <button
        @click="clearFilters"
        class="px-3 py-1.5 bg-carbon-800 hover:bg-carbon-750 text-electric-300 border border-carbon-700 rounded-md text-xs font-mono transition-colors cursor-pointer"
      >
        Clear Filters
      </button>
    </div>

    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      <FleetProfileCard
        v-for="profile in filteredProfiles"
        :key="profile.id"
        :profile="profile"
        :is-selected="profilesStore.selectedProfileId === profile.id"
        :state="getProfileState(profile)"
        @select="handleSelect"
        @sync="handleSync"
        @open-explorer="openInExplorer"
        @open-telemetry="openInTelemetry"
        @review="reviewStore.openModal"
      />
    </div>
  </div>
</template>
