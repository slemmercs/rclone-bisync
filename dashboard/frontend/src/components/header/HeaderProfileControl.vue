<script setup lang="ts">
import FSMStateBadge from '../FSMStateBadge.vue'
import { useProfilesStore } from '../../stores/profiles'
import { useFSMStore } from '../../stores/fsm'

defineEmits<{
  (e: 'openProfiles'): void
  (e: 'openReview'): void
}>()

const profilesStore = useProfilesStore()
const fsmStore = useFSMStore()

function onProfileChange(event: Event) {
  const target = event.target as HTMLSelectElement
  if (target && target.value) {
    profilesStore.selectProfile(target.value)
  }
}
</script>

<template>
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
        type="button"
        @click="$emit('openProfiles')"
        class="text-carbon-400 hover:text-electric-blue text-xs p-1 ml-1 cursor-pointer transition-colors"
        title="Open Sync Profiles Manager"
      >
        <i class="pi pi-cog text-xs"></i>
      </button>
    </div>

    <!-- FSM State Badge -->
    <FSMStateBadge
      :state="fsmStore.currentState"
      :reason="fsmStore.currentStatus.last_transition?.reason"
      @click="fsmStore.currentState === 'REVIEW_REQUIRED' && $emit('openReview')"
    />
  </div>
</template>
