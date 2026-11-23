<script setup lang="ts">
defineProps<{
  activeTab: string
}>()

defineEmits<{
  (e: 'update:activeTab', tab: string): void
}>()

const tabs = [
  { id: 'fleet', label: 'Fleet', icon: 'pi pi-server' },
  { id: 'telemetry', label: 'Engine & Telemetry', icon: 'pi pi-chart-line' },
  { id: 'remotes', label: 'Storage Remotes', icon: 'pi pi-cloud' },
  { id: 'explorer', label: 'File Explorer', icon: 'pi pi-folder' },
  { id: 'audit', label: 'Audit Trail', icon: 'pi pi-shield' },
]
</script>

<template>
  <nav class="flex items-center gap-1 bg-carbon-950 p-1 rounded-lg border border-carbon-800">
    <button
      v-for="tab in tabs"
      :key="tab.id"
      type="button"
      @click="$emit('update:activeTab', tab.id)"
      class="flex items-center gap-2 px-3 py-1.5 rounded-md text-xs font-semibold transition-all cursor-pointer select-none"
      :class="activeTab === tab.id
        ? 'bg-carbon-800 text-electric-400 shadow-sm border border-carbon-700'
        : 'text-carbon-400 hover:text-carbon-200 hover:bg-carbon-900'"
    >
      <i :class="tab.icon" class="text-xs"></i>
      <span>{{ tab.label }}</span>
    </button>
  </nav>
</template>
