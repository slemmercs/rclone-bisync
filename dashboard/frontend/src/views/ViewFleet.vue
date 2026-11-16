<script setup lang="ts">
import { computed } from 'vue'
import FSMStateBadge from '../components/FSMStateBadge.vue'
import { useProfilesStore } from '../stores/profiles'
import { useFSMStore } from '../stores/fsm'
import type { Profile } from '../types/models'
import type { SyncFSMState } from '../types/fsm'

const profilesStore = useProfilesStore()
const fsmStore = useFSMStore()

const profiles = computed(() => profilesStore.profiles)

function getProfileState(profile: Profile): SyncFSMState {
  const fsm = fsmStore.statuses[profile.id]
  if (fsm) return fsm.current_state
  if (profile.state?.is_syncing) return 'RUNNING'
  if (profile.state?.last_run_status === 'failed') return 'ABORTED'
  if (profile.state?.last_run_status === 'success') return 'COMPLETED'
  return 'IDLE'
}

function handleSelect(id: string) {
  profilesStore.selectProfile(id)
}

async function handleSync(profile: Profile, dryRun: boolean = false) {
  profilesStore.selectProfile(profile.id)
  try {
    await profilesStore.triggerSync({ dryRun })
  } catch (err) {
    console.error('Trigger sync error:', err)
  }
}

function formatDate(iso?: string) {
  if (!iso) return 'Never'
  try {
    const d = new Date(iso)
    return d.toLocaleString()
  } catch {
    return iso
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
    <!-- Profile Fleet Header -->
    <div class="flex items-center justify-between">
      <div>
        <h2 class="text-base font-semibold text-carbon-50 tracking-tight">Synchronization Workloads</h2>
        <p class="text-xs text-carbon-400">Deterministic bidirectional sync bindings across local storage and remote targets.</p>
      </div>
    </div>

    <!-- Profiles Cards Grid -->
    <div v-if="profiles.length === 0" class="bg-carbon-900 border border-carbon-700 rounded-lg p-12 text-center">
      <i class="pi pi-folder-open text-4xl text-carbon-500 mb-3"></i>
      <h3 class="text-sm font-semibold text-carbon-300">No Sync Profiles Configured</h3>
      <p class="text-xs text-carbon-500 mt-1 max-w-sm mx-auto">Create your first bidirectional sync profile to start orchestrating file trees with full safety canaries.</p>
    </div>

    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-5">
      <div
        v-for="profile in profiles"
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
            <FSMStateBadge :state="getProfileState(profile)" />
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
            class="flex-1 py-1.5 px-2 rounded text-xs font-medium border text-center transition-colors"
            :class="profilesStore.selectedProfileId === profile.id ? 'bg-carbon-800 text-electric-400 border-electric-500/50' : 'bg-carbon-850 text-carbon-300 border-carbon-700 hover:bg-carbon-800'"
          >
            {{ profilesStore.selectedProfileId === profile.id ? 'Selected' : 'Select' }}
          </button>
          <button
            @click="handleSync(profile, true)"
            class="py-1.5 px-2.5 rounded text-xs font-mono border border-carbon-700 text-carbon-400 hover:text-carbon-200 hover:bg-carbon-800"
          >
            Dry Run
          </button>
          <button
            @click="handleSync(profile, false)"
            class="py-1.5 px-3 rounded text-xs font-semibold bg-electric-500 hover:bg-electric-400 text-white shadow-sm"
          >
            Sync
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
