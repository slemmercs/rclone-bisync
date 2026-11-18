<script setup lang="ts">
import { ref, computed } from 'vue'
import FSMStateBadge from '../components/FSMStateBadge.vue'
import TelemetryChart from '../components/TelemetryChart.vue'
import { useProfilesStore } from '../stores/profiles'
import { useFSMStore } from '../stores/fsm'
import { useTelemetryStore } from '../stores/telemetry'
import { useReviewStore } from '../stores/review'
import { useAppStore } from '../stores/app'
import { formatDate } from '../composables/useFormatters'
import { useToastNotification } from '../composables/useToastNotification'
import type { Profile } from '../types/models'
import type { SyncFSMState } from '../types/fsm'

const profilesStore = useProfilesStore()
const fsmStore = useFSMStore()
const telemetryStore = useTelemetryStore()
const reviewStore = useReviewStore()
const appStore = useAppStore()
const toast = useToastNotification()

const profiles = computed(() => profilesStore.profiles)

type StatusFilterType = 'ALL' | 'ACTIVE' | 'IDLE' | 'FAILED' | 'REVIEW_REQUIRED'
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

const statusPills = computed(() => {
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
    { label: 'All', value: 'ALL' as StatusFilterType, count: all },
    { label: 'Active', value: 'ACTIVE' as StatusFilterType, count: active },
    { label: 'Idle', value: 'IDLE' as StatusFilterType, count: idle },
    { label: 'Failed', value: 'FAILED' as StatusFilterType, count: failed },
    { label: 'Review Required', value: 'REVIEW_REQUIRED' as StatusFilterType, count: review },
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

async function handleSync(profile: Profile, dryRun: boolean = false) {
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
  <div class="space-y-6">
    <!-- Top KPI Metrics Ribbon -->
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
      <div class="bg-carbon-900 border border-carbon-700 rounded-lg p-4 flex items-center justify-between shadow-sm">
        <div>
          <span class="text-xs uppercase tracking-wider text-carbon-400 font-mono">Total Profiles</span>
          <p class="text-2xl font-bold text-carbon-50 mt-1">{{ profiles.length }}</p>
        </div>
        <div class="h-10 w-10 rounded-md bg-carbon-800 border border-carbon-700 flex items-center justify-center text-electric-400">
          <i class="pi pi-database text-lg"></i>
        </div>
      </div>

      <div class="bg-carbon-900 border border-carbon-700 rounded-lg p-4 flex items-center justify-between shadow-sm">
        <div>
          <span class="text-xs uppercase tracking-wider text-carbon-400 font-mono">Active Workloads</span>
          <p class="text-2xl font-bold text-emerald-400 mt-1">{{ profilesStore.enabledProfiles.length }}</p>
        </div>
        <div class="h-10 w-10 rounded-md bg-emerald-950/40 border border-emerald-600/40 flex items-center justify-center text-emerald-400">
          <i class="pi pi-check-circle text-lg"></i>
        </div>
      </div>

      <div class="bg-carbon-900 border border-carbon-700 rounded-lg p-4 flex items-center justify-between shadow-sm">
        <div>
          <span class="text-xs uppercase tracking-wider text-carbon-400 font-mono">Engine State</span>
          <p class="text-xl font-bold text-electric-400 mt-1">{{ fsmStore.currentState }}</p>
        </div>
        <div class="h-10 w-10 rounded-md bg-electric-950/40 border border-electric-600/40 flex items-center justify-center text-electric-400">
          <i class="pi pi-sliders-h text-lg"></i>
        </div>
      </div>

      <div class="bg-carbon-900 border border-carbon-700 rounded-lg p-4 flex items-center justify-between shadow-sm">
        <div>
          <span class="text-xs uppercase tracking-wider text-carbon-400 font-mono">Safety Circuit</span>
          <p class="text-xl font-bold mt-1" :class="fsmStore.pendingReview ? 'text-amber-400' : 'text-carbon-300'">
            {{ fsmStore.pendingReview ? 'TRIPPED' : 'ARMED' }}
          </p>
        </div>
        <div
          class="h-10 w-10 rounded-md flex items-center justify-center"
          :class="fsmStore.pendingReview ? 'bg-amber-950/60 border border-amber-600 text-amber-400 animate-pulse' : 'bg-carbon-800 border border-carbon-700 text-carbon-400'"
        >
          <i :class="fsmStore.pendingReview ? 'pi pi-exclamation-triangle' : 'pi pi-shield'" class="text-lg"></i>
        </div>
      </div>
    </div>
    <!-- Real-Time Telemetry & Throughput Card -->
    <div class="bg-carbon-900 border border-carbon-700 rounded-lg p-4 shadow-sm">
      <div class="flex items-center justify-between mb-2">
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
        </div>
      </div>
      <TelemetryChart />
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
    <div v-if="profiles.length > 0" class="flex flex-col md:flex-row md:items-center justify-between gap-3 bg-carbon-900 border border-carbon-700 rounded-lg p-3 shadow-sm">
      <!-- Search Input -->
      <div class="relative flex-1 max-w-md">
        <i class="pi pi-search absolute left-3 top-1/2 -translate-y-1/2 text-carbon-500 text-xs"></i>
        <input
          v-model="searchQuery"
          type="text"
          placeholder="Filter by profile name, remote, or path..."
          class="w-full pl-8 pr-8 py-1.5 bg-carbon-950 border border-carbon-700 rounded-md text-xs font-mono text-carbon-200 placeholder-carbon-500 focus:outline-none focus:border-electric-500 transition-colors"
        />
        <button
          v-if="searchQuery"
          @click="searchQuery = ''"
          class="absolute right-2.5 top-1/2 -translate-y-1/2 text-carbon-500 hover:text-carbon-300 cursor-pointer"
          title="Clear search query"
        >
          <i class="pi pi-times text-xs"></i>
        </button>
      </div>

      <!-- Status Filter Segmented Pills -->
      <div class="flex items-center gap-1.5 flex-wrap font-mono text-xs">
        <button
          v-for="pill in statusPills"
          :key="pill.value"
          @click="statusFilter = pill.value"
          class="px-2.5 py-1 rounded-md text-[11px] font-semibold border transition-all cursor-pointer flex items-center gap-1.5 select-none"
          :class="statusFilter === pill.value
            ? 'bg-electric-950 text-electric-300 border-electric-500 shadow-sm'
            : 'bg-carbon-850 text-carbon-400 border-carbon-700 hover:bg-carbon-800 hover:text-carbon-200'"
        >
          <span>{{ pill.label }}</span>
          <span
            class="px-1.5 py-0.2 rounded text-[10px]"
            :class="statusFilter === pill.value ? 'bg-electric-900/80 text-electric-200' : 'bg-carbon-800 text-carbon-500'"
          >
            {{ pill.count }}
          </span>
        </button>
      </div>
    </div>

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

    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-5">
      <div
        v-for="profile in filteredProfiles"
        :key="profile.id"
        class="bg-carbon-900 border rounded-lg p-5 flex flex-col justify-between transition-all duration-150 hover:border-carbon-600 shadow-sm"
        :class="profilesStore.selectedProfileId === profile.id ? 'border-electric-500 ring-1 ring-electric-500/30' : 'border-carbon-700'"
      >
        <!-- Card Body -->
        <div class="space-y-3">
          <div class="flex items-start justify-between gap-2">
            <div>
              <div class="flex items-center gap-2">
                <span class="font-bold text-sm text-carbon-50">{{ profile.name }}</span>
                <span v-if="!profile.enabled" class="text-[10px] font-mono px-1.5 py-0.2 rounded bg-carbon-800 text-carbon-500">DISABLED</span>
              </div>
              <span class="text-[11px] font-mono text-carbon-500">{{ profile.id }}</span>
            </div>

            <div class="flex items-center gap-1.5">
              <button
                type="button"
                @click.stop="openInExplorer(profile)"
                class="p-1 rounded text-carbon-400 hover:text-electric-300 hover:bg-carbon-800 transition-colors cursor-pointer border border-carbon-800"
                title="Jump to Dual Explorer for this profile"
              >
                <i class="pi pi-folder text-xs"></i>
              </button>
              <button
                type="button"
                @click.stop="openInTelemetry(profile)"
                class="p-1 rounded text-carbon-400 hover:text-purple-300 hover:bg-carbon-800 transition-colors cursor-pointer border border-carbon-800"
                title="Jump to Engine Telemetry for this profile"
              >
                <i class="pi pi-chart-line text-xs"></i>
              </button>
              <FSMStateBadge
                :state="getProfileState(profile)"
                @click="getProfileState(profile) === 'REVIEW_REQUIRED' && reviewStore.openModal()"
              />
            </div>
          </div>

          <div class="bg-carbon-950/70 rounded-md p-2.5 border border-carbon-800 space-y-1.5 text-xs">
            <div class="flex items-center gap-1.5 text-carbon-400 truncate">
              <i class="pi pi-desktop text-carbon-500 text-[11px]"></i>
              <span class="font-mono truncate" :title="profile.local_path">{{ profile.local_path }}</span>
            </div>
            <div class="flex items-center gap-1.5 text-electric-300 font-semibold truncate">
              <i class="pi pi-cloud text-electric-400 text-[11px]"></i>
              <span class="font-mono truncate" :title="profile.remote">{{ profile.remote }}{{ profile.remote_path ? `:${profile.remote_path}` : '' }}</span>
            </div>
          </div>

          <div class="flex flex-wrap items-center gap-2 text-[11px] font-mono text-carbon-400">
            <span class="px-2 py-0.5 rounded bg-carbon-800 border border-carbon-700 flex items-center gap-1">
              <i class="pi pi-clock text-[10px]"></i> {{ profile.schedule || 'manual' }}
            </span>
            <span class="px-2 py-0.5 rounded bg-carbon-800 border border-carbon-700 flex items-center gap-1">
              <i class="pi pi-sliders-v text-[10px]"></i> Review: {{ profile.review_mode || 'disabled' }}
            </span>
          </div>

          <div class="text-[11px] text-carbon-400 border-t border-carbon-800 pt-2 space-y-1 font-mono">
            <div class="flex justify-between">
              <span>Last Run:</span>
              <span class="text-carbon-300">{{ formatDate(profile.state?.last_run_time) }}</span>
            </div>
            <div v-if="profile.state?.last_error" class="text-red-400 truncate" :title="profile.state.last_error">
              Err: {{ profile.state.last_error }}
            </div>
          </div>
        </div>

        <!-- Card Footer Actions -->
        <div class="flex items-center gap-2 pt-3 mt-3 border-t border-carbon-800/80">
          <button
            @click="handleSelect(profile.id)"
            class="flex-1 py-1.5 px-2 rounded text-xs font-medium border text-center transition-colors cursor-pointer"
            :class="profilesStore.selectedProfileId === profile.id ? 'bg-carbon-800 text-electric-400 border-electric-500/50' : 'bg-carbon-850 text-carbon-300 border-carbon-700 hover:bg-carbon-800'"
          >
            {{ profilesStore.selectedProfileId === profile.id ? 'Selected' : 'Select' }}
          </button>
          <button
            @click="handleSync(profile, true)"
            class="py-1.5 px-2.5 rounded text-xs font-mono border border-carbon-700 text-carbon-400 hover:text-carbon-200 hover:bg-carbon-800 cursor-pointer"
          >
            Dry Run
          </button>
          <button
            @click="handleSync(profile, false)"
            class="py-1.5 px-3 rounded text-xs font-semibold bg-electric-500 hover:bg-electric-400 text-white shadow-sm cursor-pointer"
          >
            Sync
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
