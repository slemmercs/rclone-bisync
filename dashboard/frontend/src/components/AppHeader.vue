<script setup lang="ts">
import HeaderBrandStatus from './header/HeaderBrandStatus.vue'
import HeaderNavigation from './header/HeaderNavigation.vue'
import HeaderProfileControl from './header/HeaderProfileControl.vue'
import HeaderActionToolbar from './header/HeaderActionToolbar.vue'

defineProps<{
  activeTab: string
}>()

defineEmits<{
  (e: 'update:activeTab', tab: string): void
  (e: 'openReview'): void
  (e: 'openProfiles'): void
  (e: 'openRemoteWizard'): void
}>()
</script>

<template>
  <header class="bg-carbon-900 border-b border-carbon-700 px-6 py-3 flex flex-wrap items-center justify-between gap-4 sticky top-0 z-50 shadow-md">
    <!-- Brand & Control Plane Identity -->
    <HeaderBrandStatus />

    <!-- Navigation Tabs -->
    <HeaderNavigation
      :active-tab="activeTab"
      @update:active-tab="$emit('update:activeTab', $event)"
    />

    <!-- Profile Selector, FSM Badge & Action Center -->
    <div class="flex items-center gap-3">
      <!-- Profile Picker & FSM State Badge -->
      <HeaderProfileControl
        @open-profiles="$emit('openProfiles')"
        @open-review="$emit('openReview')"
      />

      <!-- Action Center & Utility Toolbar -->
      <HeaderActionToolbar
        :active-tab="activeTab"
        @open-review="$emit('openReview')"
        @open-profiles="$emit('openProfiles')"
        @open-remote-wizard="$emit('openRemoteWizard')"
      />
    </div>
  </header>
</template>
