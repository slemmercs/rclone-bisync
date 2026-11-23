<script setup lang="ts">
import FSMStateBadge from '../FSMStateBadge.vue'
import { formatDate } from '../../composables/useFormatters'
import type { Profile } from '../../types/models'
import type { SyncFSMState } from '../../types/fsm'
import type { FleetProfileSyncEvent } from '../../types/fleet'

defineProps<{
  profile: Profile
  isSelected: boolean
  state: SyncFSMState
}>()

defineEmits<{
  select: [id: string]
  sync: [event: FleetProfileSyncEvent]
  openExplorer: [profile: Profile]
  openTelemetry: [profile: Profile]
  review: []
}>()
</script>

<template>
  <div
    class="bg-carbon-900 border rounded-lg p-4 sm:p-5 flex flex-col justify-between transition-all duration-150 hover:border-carbon-600 shadow-sm"
    :class="isSelected ? 'border-electric-500 ring-1 ring-electric-500/30' : 'border-carbon-700'"
  >
    <!-- Card Body -->
    <div class="space-y-2.5">
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
            @click.stop="$emit('openExplorer', profile)"
            class="p-1 rounded text-carbon-400 hover:text-electric-300 hover:bg-carbon-800 transition-colors cursor-pointer border border-carbon-800"
            title="Jump to Dual Explorer for this profile"
          >
            <i class="pi pi-folder text-xs"></i>
          </button>
          <button
            type="button"
            @click.stop="$emit('openTelemetry', profile)"
            class="p-1 rounded text-carbon-400 hover:text-purple-300 hover:bg-carbon-800 transition-colors cursor-pointer border border-carbon-800"
            title="Jump to Engine Telemetry for this profile"
          >
            <i class="pi pi-chart-line text-xs"></i>
          </button>
          <FSMStateBadge
            :state="state"
            @click="state === 'REVIEW_REQUIRED' && $emit('review')"
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
        type="button"
        @click="$emit('select', profile.id)"
        class="flex-1 py-1.5 px-2 rounded text-xs font-medium border text-center transition-colors cursor-pointer"
        :class="isSelected ? 'bg-carbon-800 text-electric-400 border-electric-500/50' : 'bg-carbon-850 text-carbon-300 border-carbon-700 hover:bg-carbon-800'"
      >
        {{ isSelected ? 'Selected' : 'Select' }}
      </button>
      <button
        type="button"
        @click="$emit('sync', { profile, dryRun: true })"
        class="py-1.5 px-2.5 rounded text-xs font-mono border border-carbon-700 text-carbon-400 hover:text-carbon-200 hover:bg-carbon-800 cursor-pointer"
      >
        Dry Run
      </button>
      <button
        type="button"
        @click="$emit('sync', { profile, dryRun: false })"
        class="py-1.5 px-3 rounded text-xs font-semibold bg-electric-500 hover:bg-electric-400 text-white shadow-sm cursor-pointer"
      >
        Sync
      </button>
    </div>
  </div>
</template>
