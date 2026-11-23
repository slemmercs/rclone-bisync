<script setup lang="ts">
import type { StatusFilterType, FleetStatusPill } from '../../types/fleet'

defineProps<{
  searchQuery: string
  statusFilter: StatusFilterType
  statusPills: FleetStatusPill[]
}>()

defineEmits<{
  'update:searchQuery': [value: string]
  'update:statusFilter': [value: StatusFilterType]
}>()
</script>

<template>
  <div class="flex flex-col md:flex-row md:items-center justify-between gap-3 bg-carbon-900 border border-carbon-700 rounded-lg p-3 shadow-sm">
    <!-- Search Input -->
    <div class="relative flex-1 max-w-md">
      <i class="pi pi-search absolute left-3 top-1/2 -translate-y-1/2 text-carbon-500 text-xs"></i>
      <input
        :value="searchQuery"
        @input="$emit('update:searchQuery', ($event.target as HTMLInputElement).value)"
        type="text"
        placeholder="Filter by profile name, remote, or path..."
        class="w-full pl-8 pr-8 py-1.5 bg-carbon-950 border border-carbon-700 rounded-md text-xs font-mono text-carbon-200 placeholder-carbon-500 focus:outline-none focus:border-electric-500 transition-colors"
      />
      <button
        v-if="searchQuery"
        type="button"
        @click="$emit('update:searchQuery', '')"
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
        type="button"
        @click="$emit('update:statusFilter', pill.value)"
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
</template>
