<script setup lang="ts">
defineProps<{
  totalProfiles: number
  activeWorkloads: number
  engineState: string
  pendingReview: boolean
}>()
</script>

<template>
  <div class="grid grid-cols-1 md:grid-cols-4 gap-3.5">
    <!-- Total Profiles KPI -->
    <div class="bg-carbon-900 border border-carbon-700 rounded-lg p-3.5 flex items-center justify-between shadow-sm">
      <div>
        <span class="text-xs uppercase tracking-wider text-carbon-400 font-mono">Total Profiles</span>
        <p class="text-2xl font-bold text-carbon-50 mt-1">{{ totalProfiles }}</p>
      </div>
      <div class="h-10 w-10 rounded-md bg-carbon-800 border border-carbon-700 flex items-center justify-center text-electric-400">
        <i class="pi pi-database text-lg"></i>
      </div>
    </div>

    <!-- Active Workloads KPI -->
    <div class="bg-carbon-900 border border-carbon-700 rounded-lg p-3.5 flex items-center justify-between shadow-sm">
      <div>
        <span class="text-xs uppercase tracking-wider text-carbon-400 font-mono">Active Workloads</span>
        <p class="text-2xl font-bold text-emerald-400 mt-1">{{ activeWorkloads }}</p>
      </div>
      <div class="h-10 w-10 rounded-md bg-emerald-950/40 border border-emerald-600/40 flex items-center justify-center text-emerald-400">
        <i class="pi pi-check-circle text-lg"></i>
      </div>
    </div>

    <!-- Engine State KPI -->
    <div class="bg-carbon-900 border border-carbon-700 rounded-lg p-3.5 flex items-center justify-between shadow-sm">
      <div>
        <span class="text-xs uppercase tracking-wider text-carbon-400 font-mono">Engine State</span>
        <p class="text-xl font-bold text-electric-400 mt-1">{{ engineState }}</p>
      </div>
      <div class="h-10 w-10 rounded-md bg-electric-950/40 border border-electric-600/40 flex items-center justify-center text-electric-400">
        <i class="pi pi-sliders-h text-lg"></i>
      </div>
    </div>

    <!-- Safety Circuit KPI -->
    <div class="bg-carbon-900 border border-carbon-700 rounded-lg p-3.5 flex items-center justify-between shadow-sm">
      <div>
        <span class="text-xs uppercase tracking-wider text-carbon-400 font-mono">Safety Circuit</span>
        <p class="text-xl font-bold mt-1" :class="pendingReview ? 'text-amber-400' : 'text-carbon-300'">
          {{ pendingReview ? 'TRIPPED' : 'ARMED' }}
        </p>
      </div>
      <div
        class="h-10 w-10 rounded-md flex items-center justify-center"
        :class="pendingReview ? 'bg-amber-950/60 border border-amber-600 text-amber-400 animate-pulse' : 'bg-carbon-800 border border-carbon-700 text-carbon-400'"
      >
        <i :class="pendingReview ? 'pi pi-exclamation-triangle' : 'pi pi-shield'" class="text-lg"></i>
      </div>
    </div>
  </div>
</template>
