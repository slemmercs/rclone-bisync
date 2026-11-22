<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  summary?: Record<string, any> | null
  remotesCount: number
}>()

const quotaPct = computed(() => {
  const used = Number(props.summary?.total_used_bytes || 0)
  const total = Number(props.summary?.total_capacity_bytes || 0)
  if (total <= 0) return '0.0'
  return ((used / total) * 100).toFixed(1)
})

function formatBytes(bytes?: number): string {
  if (!bytes || bytes <= 0) return '-- GB'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`
}
</script>

<template>
  <div class="grid grid-cols-2 sm:grid-cols-4 gap-4">
    <div class="bg-carbon-surface p-4 rounded-2xl border border-carbon-border">
      <div class="text-[11px] text-slate-400 uppercase font-semibold">Total Cloud Space</div>
      <div class="text-xl font-bold text-white mt-2 font-mono">{{ formatBytes(summary?.total_capacity_bytes) }}</div>
    </div>
    <div class="bg-carbon-surface p-4 rounded-2xl border border-carbon-border">
      <div class="text-[11px] text-slate-400 uppercase font-semibold">Utilized Storage</div>
      <div class="text-xl font-bold text-electric-blue mt-2 font-mono">{{ formatBytes(summary?.total_used_bytes) }}</div>
      <div class="text-[10px] text-slate-400 mt-1">{{ quotaPct }}% used</div>
    </div>
    <div class="bg-carbon-surface p-4 rounded-2xl border border-carbon-border">
      <div class="text-[11px] text-slate-400 uppercase font-semibold">Available Space</div>
      <div class="text-xl font-bold text-emerald-400 mt-2 font-mono">{{ formatBytes(summary?.total_free_bytes) }}</div>
    </div>
    <div class="bg-carbon-surface p-4 rounded-2xl border border-carbon-border">
      <div class="text-[11px] text-slate-400 uppercase font-semibold">Remotes Count</div>
      <div class="text-xl font-bold text-white mt-2 font-mono">{{ remotesCount }}</div>
    </div>
  </div>
</template>
